package st

import (
	"crypto/sha256"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"
)

type epochRecordingMemory struct {
	*Memory
	clocks []uint64
	wait   uint32
}

func (m *epochRecordingMemory) ReadWordAt(address uint32, access m68k.BusAccess) (uint16, uint32, error) {
	m.clocks = append(m.clocks, access.Clock)
	value, err := m.Memory.ReadWord(address, access.FunctionCode)
	return value, m.wait, err
}

func TestMachinePassesCurrentClockAsInstructionEpoch(t *testing.T) {
	rom := make([]byte, TOSROMSize)
	rom[2], rom[3] = 0x12, 0x34
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	bus := &epochRecordingMemory{Memory: machine.Memory, wait: 2}
	machine.CPU.Bus = bus
	machine.CPU.State = m68k.State{SR: 0x2000, PC: TOSROMBase + 2,
		Prefetch: [2]uint16{0x4e71, 0xabcd}}
	machine.Instructions, machine.Clocks = 9, 390
	result, err := machine.Step()
	if err != nil {
		t.Fatal(err)
	}
	if len(bus.clocks) != 1 || bus.clocks[0] != 390 {
		t.Fatalf("access clocks=%v want [390]", bus.clocks)
	}
	if result.Clocks != 6 || machine.Instructions != 10 || machine.Clocks != 396 {
		t.Fatalf("result=%+v counters=%d/%d", result, machine.Instructions, machine.Clocks)
	}
}

func TestMOVELongImmediateAbsoluteShortBusPhases(t *testing.T) {
	for _, test := range []struct {
		name        string
		epoch       uint64
		wantClocks  uint32
		wantOffsets []uint32
	}{
		{name: "phase0", epoch: 12, wantClocks: 24, wantOffsets: []uint32{0, 4, 8, 12, 16, 20}},
		{name: "phase2", epoch: 10, wantClocks: 26, wantOffsets: []uint32{0, 2, 6, 10, 14, 18, 22}},
	} {
		t.Run(test.name, func(t *testing.T) {
			rom := make([]byte, TOSROMSize)
			copy(rom[0x44:], []byte{0x00, 0xb2, 0x00, 0x10, 0x4e, 0x71, 0x60, 0xfe})
			machine, err := NewMachine(RAM1M, rom)
			if err != nil {
				t.Fatal(err)
			}
			machine.CPU.State = m68k.State{SR: 0x2000, PC: TOSROMBase + 0x44,
				Prefetch: [2]uint16{0x21fc, 0x00fc}}
			result, err := machine.CPU.StepAt(test.epoch)
			if err != nil {
				t.Fatal(err)
			}
			if result.Clocks != test.wantClocks || machine.CPU.State.PC != TOSROMBase+0x4c ||
				machine.CPU.State.Prefetch != [2]uint16{0x4e71, 0x60fe} {
				t.Fatalf("result=%+v state=%+v", result, machine.CPU.State)
			}
			if got, err := machine.Memory.ReadWord(0x10, 5); err != nil || got != 0x00fc {
				t.Fatalf("RAM[10]=%04x err=%v", got, err)
			}
			if got, err := machine.Memory.ReadWord(0x12, 5); err != nil || got != 0x00b2 {
				t.Fatalf("RAM[12]=%04x err=%v", got, err)
			}
			gotOffsets := make([]uint32, len(result.Timeline))
			for i, phase := range result.Timeline {
				gotOffsets[i] = phase.Offset
			}
			if !reflect.DeepEqual(gotOffsets, test.wantOffsets) {
				t.Fatalf("timeline offsets=%v want %v", gotOffsets, test.wantOffsets)
			}
		})
	}
}

func TestMachineResetFromROMShadow(t *testing.T) {
	rom := make([]byte, TOSROMSize)
	copy(rom[:8], []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0xfc, 0x00, 0x30})
	copy(rom[0x30:0x34], []byte{0x60, 0x00, 0x00, 0x1c})
	copy(rom[0x4e:0x52], []byte{0x46, 0xfc, 0x27, 0x00})
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	machine.CPU.State.D[0] = 0x12345678
	machine.CPU.State.A[0] = 0x87654321
	machine.CPU.State.USP = 0x00abcdef
	machine.Instructions, machine.Clocks = 99, 1234
	if err := machine.Memory.WriteByte(MMUConfig, 0x0a, 5); err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	state := machine.CPU.State
	if state.SSP != 0x00010000 || state.SR != 0x2700 || state.PC != 0x00fc0034 ||
		state.Prefetch != [2]uint16{0x6000, 0x001c} {
		t.Fatalf("reset state=%+v", state)
	}
	if state.D[0] != 0x12345678 || state.A[0] != 0x87654321 || state.USP != 0x00abcdef {
		t.Fatalf("reset changed unspecified registers: %+v", state)
	}
	if machine.Instructions != 0 || machine.Clocks != 0 {
		t.Fatalf("reset counters=%d/%d", machine.Instructions, machine.Clocks)
	}
	if got, err := machine.Memory.ReadByte(MMUConfig, 5); err != nil || got != 0 {
		t.Fatalf("machine cold-reset MMU=%02x err=%v", got, err)
	}
	result, err := machine.Step()
	if err != nil {
		t.Fatal(err)
	}
	if result.Clocks != 10 || machine.Instructions != 1 || machine.Clocks != 10 ||
		machine.CPU.State.PC != 0x00fc0052 || machine.CPU.State.Prefetch != [2]uint16{0x46fc, 0x2700} {
		t.Fatalf("first step result=%+v machine=%+v", result, machine)
	}
}

func TestMachineEmuTOSReachesMOVECProbe(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
	if got := fmt.Sprintf("%x", sha256.Sum256(rom)); got != wantHash {
		t.Fatalf("EmuTOS SHA-256=%s want %s", got, wantHash)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 7; i++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	state := machine.CPU.State
	if machine.Instructions != 7 || machine.Clocks != 92 || state.PC != 0x00fc0074 ||
		state.Prefetch != [2]uint16{0x4e7b, 0x0801} || state.SSP != 0x1000 || state.SR != 0x2704 {
		t.Fatalf("MOVEC probe boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	if got, err := machine.Memory.ReadByte(MMUConfig, 5); err != nil || got != 0 {
		t.Fatalf("MMU config at MOVEC probe=%02x err=%v", got, err)
	}
}

func TestMachineEmuTOSMOVECEntersIllegalVector4(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
	if got := fmt.Sprintf("%x", sha256.Sum256(rom)); got != wantHash {
		t.Fatalf("EmuTOS SHA-256=%s want %s", got, wantHash)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	state := machine.CPU.State
	if machine.Instructions != 8 || machine.Clocks != 128 || state.PC != 0x00fc0078 ||
		state.Prefetch != [2]uint16{0x21fc, 0x00fc} || state.SSP != 0x0ffa || state.SR != 0x2704 {
		t.Fatalf("vector-4 boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	wantFrame := []uint16{0x2704, 0x00fc, 0x0070}
	for index, want := range wantFrame {
		got, readErr := machine.Memory.ReadWord(0x0ffa+uint32(index*2), 5)
		if readErr != nil || got != want {
			t.Fatalf("frame[%d]=%04x/%v want %04x", index, got, readErr, want)
		}
	}
}

func TestMachineEmuTOSAbsoluteShortBusErrorPreservesFaultAddress(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
	if got := fmt.Sprintf("%x", sha256.Sum256(rom)); got != wantHash {
		t.Fatalf("EmuTOS SHA-256=%s want %s", got, wantHash)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	state := machine.CPU.State
	if machine.Instructions != 10 || machine.Clocks != 220 || state.PC != 0x00fc008c ||
		state.Prefetch != [2]uint16{0x4e70, 0x0cb9} || state.SSP != 0x0fec || state.SR != 0x2700 {
		t.Fatalf("vector-2 boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	wantFrame := []uint16{0x4a75, 0xffff, 0x8006, 0x4a78, 0x2700, 0x00fc, 0x0080}
	for index, want := range wantFrame {
		got, readErr := machine.Memory.ReadWord(0x0fec+uint32(index*2), 5)
		if readErr != nil || got != want {
			t.Fatalf("frame[%d]=%04x/%v want %04x", index, got, readErr, want)
		}
	}
}

func TestMachineEmuTOSExecutesRESET(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
	if got := fmt.Sprintf("%x", sha256.Sum256(rom)); got != wantHash {
		t.Fatalf("EmuTOS SHA-256=%s want %s", got, wantHash)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 11; i++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	state := machine.CPU.State
	if machine.Instructions != 11 || machine.Clocks != 352 || state.PC != 0x00fc008e ||
		state.Prefetch != [2]uint16{0x0cb9, 0xfa52} || state.SSP != 0x0fec ||
		state.USP != 0 || state.SR != 0x2700 || state.D != [8]uint32{} || state.A != [7]uint32{} {
		t.Fatalf("RESET boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	if got, err := machine.Memory.ReadByte(MMUConfig, 5); err != nil || got != 0 {
		t.Fatalf("MMU config after RESET=%02x err=%v", got, err)
	}
}

func TestMachineEmuTOSEmptyCartridgeProbe(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
	if got := fmt.Sprintf("%x", sha256.Sum256(rom)); got != wantHash {
		t.Fatalf("EmuTOS SHA-256=%s want %s", got, wantHash)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	state := machine.CPU.State
	if machine.Instructions != 12 || machine.Clocks != 380 || state.PC != 0x00fc0098 ||
		state.Prefetch != [2]uint16{0x660a, 0x4dfa} || state.SSP != 0x0fec ||
		state.USP != 0 || state.SR != 0x2700 || state.D != [8]uint32{} || state.A != [7]uint32{} {
		t.Fatalf("cartridge-probe boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
}

func TestMachineEmuTOSAlignsFourteenthInstructionToBusSlot(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
	if got := fmt.Sprintf("%x", sha256.Sum256(rom)); got != wantHash {
		t.Fatalf("EmuTOS SHA-256=%s want %s", got, wantHash)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 14; i++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
	state := machine.CPU.State
	if machine.Instructions != 14 || machine.Clocks != 416 || state.PC != 0x00fc00ac ||
		state.Prefetch != [2]uint16{0x203c, 0x0000} || state.SSP != 0x0fec || state.SR != 0x2700 {
		t.Fatalf("instruction-14 boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	for address, want := range map[uint32]uint16{0x10: 0x00fc, 0x12: 0x00b2} {
		got, readErr := machine.Memory.ReadWord(address, 5)
		if readErr != nil || got != want {
			t.Fatalf("RAM[%x]=%04x/%v want %04x", address, got, readErr, want)
		}
	}
}

func TestMachineEmuTOSReachesLineFBoundary(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
	if got := fmt.Sprintf("%x", sha256.Sum256(rom)); got != wantHash {
		t.Fatalf("EmuTOS SHA-256=%s want %s", got, wantHash)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	wantClocks := []uint64{416, 428, 464, 488, 496}
	for i := 0; i < 18; i++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if i >= 13 && machine.Clocks != wantClocks[i-13] {
			t.Fatalf("step %d clocks=%d want %d", i+1, machine.Clocks, wantClocks[i-13])
		}
	}
	state := machine.CPU.State
	if state.PC != 0x00fc00c2 || state.Prefetch != [2]uint16{0xf010, 0x0800} ||
		state.D[0] != 0x00000808 || state.A[0] != 0x00fc0152 || state.SSP != 0x0fe6 || state.SR != 0x2700 {
		t.Fatalf("line-F boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
}

func TestMachineEmuTOSLineFVector11(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
	if got := fmt.Sprintf("%x", sha256.Sum256(rom)); got != wantHash {
		t.Fatalf("EmuTOS SHA-256=%s want %s", got, wantHash)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 19; i++ {
		result, err := machine.Step()
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if i == 18 {
			wantOffsets := []uint32{0, 6, 8, 12, 16, 20, 24, 28, 32}
			if result.Clocks != 36 || len(result.Timeline) != len(wantOffsets) {
				t.Fatalf("line-F timing result=%+v", result)
			}
			for phase, want := range wantOffsets {
				if result.Timeline[phase].Offset != want {
					t.Fatalf("line-F phase %d offset=%d want %d", phase, result.Timeline[phase].Offset, want)
				}
			}
		}
	}
	state := machine.CPU.State
	if machine.Instructions != 19 || machine.Clocks != 532 || state.PC != 0x00fc00d8 ||
		state.Prefetch != [2]uint16{0x21fc, 0x00fc} || state.SSP != 0x0fe0 || state.SR != 0x2700 {
		t.Fatalf("line-F handler instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	for address, want := range map[uint32]uint16{0x0fe0: 0x2700, 0x0fe2: 0x00fc, 0x0fe4: 0x00be} {
		got, readErr := machine.Memory.ReadWord(address, 5)
		if readErr != nil || got != want {
			t.Fatalf("exception frame[%x]=%04x/%v want %04x", address, got, readErr, want)
		}
	}
}

func TestMachineEmuTOSSTRicohVoidDMAByteRead(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
	if got := fmt.Sprintf("%x", sha256.Sum256(rom)); got != wantHash {
		t.Fatalf("EmuTOS SHA-256=%s want %s", got, wantHash)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6851; i++ {
		result, stepErr := machine.Step()
		if stepErr != nil {
			t.Fatalf("step %d: %v", i+1, stepErr)
		}
		if i == 6850 && result.Clocks != 8 {
			t.Fatalf("void DMA byte TST clocks=%d want 8", result.Clocks)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0, 0x0f84, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xffff860f, 0x00fc036e, 0, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 6851 || machine.Clocks != 167382 || state.D != wantD || state.A != wantA ||
		state.USP != 0 || state.SSP != 0x0f84 || state.SR != 0x2708 || state.PC != 0x00fc063c ||
		state.Prefetch != [2]uint16{0x4e71, 0x7001} {
		t.Fatalf("void DMA byte boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
}

func TestMachineEmuTOSSTWithoutMegaRTC(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
	if got := fmt.Sprintf("%x", sha256.Sum256(rom)); got != wantHash {
		t.Fatalf("EmuTOS SHA-256=%s want %s", got, wantHash)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6879; i++ {
		result, stepErr := machine.Step()
		if stepErr != nil {
			t.Fatalf("step %d: %v", i+1, stepErr)
		}
		if i == 6878 && result.Clocks != 8 {
			t.Fatalf("void RTC byte TST clocks=%d want 8", result.Clocks)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0, 0x0f80, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xfffffc21, 0x00fc036e, 0, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 6879 || machine.Clocks != 167750 || state.D != wantD || state.A != wantA ||
		state.USP != 0 || state.SSP != 0x0f80 || state.SR != 0x2708 || state.PC != 0x00fc063c ||
		state.Prefetch != [2]uint16{0x4e71, 0x7001} {
		t.Fatalf("void RTC boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
}

func TestMachineEmuTOSBlitterProbeBusError(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
	if got := fmt.Sprintf("%x", sha256.Sum256(rom)); got != wantHash {
		t.Fatalf("EmuTOS SHA-256=%s want %s", got, wantHash)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6917; i++ {
		result, stepErr := machine.Step()
		if stepErr != nil {
			t.Fatalf("step %d: %v", i+1, stepErr)
		}
		if i == 6916 && result.Clocks != 64 {
			t.Fatalf("Blitter probe bus error clocks=%d want 64", result.Clocks)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0, 0x0f84, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xffff8a3c, 0x00fc036e, 0, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 6917 || machine.Clocks != 168242 || state.D != wantD || state.A != wantA ||
		state.USP != 0 || state.SSP != 0x0f76 || state.SR != 0x2704 || state.PC != 0x00fc0640 ||
		state.Prefetch != [2]uint16{0x21c9, 0x0008} {
		t.Fatalf("Blitter probe handler instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	wantFrame := []uint16{0x4a15, 0xffff, 0x8a3c, 0x4a10, 0x2704, 0x00fc, 0x0638}
	for index, want := range wantFrame {
		got, readErr := machine.Memory.ReadWord(state.SSP+uint32(index*2), 5)
		if readErr != nil || got != want {
			t.Fatalf("Blitter probe frame[%d]=%04x/%v want %04x", index, got, readErr, want)
		}
	}
}

func TestMachineEmuTOSWritesMFPResetGPIP(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
	if got := fmt.Sprintf("%x", sha256.Sum256(rom)); got != wantHash {
		t.Fatalf("EmuTOS SHA-256=%s want %s", got, wantHash)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 7475; i++ {
		result, stepErr := machine.Step()
		if stepErr != nil {
			t.Fatalf("step %d: %v", i+1, stepErr)
		}
		if i == 7474 && result.Clocks != 16 {
			t.Fatalf("MFP GPIP write clocks=%d want 16", result.Clocks)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0x1e, 2, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xfffffa01, 0x3156, 0, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 7475 || machine.Clocks != 176638 || state.D != wantD || state.A != wantA ||
		state.USP != 0 || state.SSP != 0x0f8c || state.SR != 0x2714 || state.PC != 0x00fc6152 ||
		state.Prefetch != [2]uint16{0x5488, 0xb0fc} {
		t.Fatalf("MFP GPIP boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	if got, err := machine.Memory.ReadByte(MFPGPIP, 5); err != nil || got != 0 {
		t.Fatalf("MFP GPIP=%02x/%v want 00", got, err)
	}
}

func TestMachineEmuTOSWritesMFPResetAER(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
	if got := fmt.Sprintf("%x", sha256.Sum256(rom)); got != wantHash {
		t.Fatalf("EmuTOS SHA-256=%s want %s", got, wantHash)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 7479; i++ {
		result, stepErr := machine.Step()
		if stepErr != nil {
			t.Fatalf("step %d: %v", i+1, stepErr)
		}
		if i == 7478 && result.Clocks != 16 {
			t.Fatalf("MFP AER write clocks=%d want 16", result.Clocks)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0x1e, 2, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xfffffa03, 0x3156, 0, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 7479 || machine.Clocks != 176682 || state.D != wantD || state.A != wantA ||
		state.USP != 0 || state.SSP != 0x0f8c || state.SR != 0x2714 || state.PC != 0x00fc6152 ||
		state.Prefetch != [2]uint16{0x5488, 0xb0fc} {
		t.Fatalf("MFP AER boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	if got, err := machine.Memory.ReadByte(MFPAER, 5); err != nil || got != 0 {
		t.Fatalf("MFP AER=%02x/%v want 00", got, err)
	}
	for machine.Instructions < 7482 {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("post-AER step %d: %v", machine.Instructions+1, err)
		}
	}
	if _, err := machine.Step(); err == nil || machine.Instructions != 7482 ||
		machine.CPU.State.A[0] != 0xfffffa05 {
		t.Fatalf("next DDR stop instructions=%d A0=%08x err=%v",
			machine.Instructions, machine.CPU.State.A[0], err)
	}
}

func TestMachineResetWithEmuTOS(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set; external EmuTOS reset oracle unavailable")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
	if got := fmt.Sprintf("%x", sha256.Sum256(rom)); got != wantHash {
		t.Fatalf("EmuTOS SHA-256=%s want %s", got, wantHash)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	state := machine.CPU.State
	if state.SSP != 0x602e0104 || state.SR != 0x2700 || state.PC != 0x00fc0034 ||
		state.Prefetch != [2]uint16{0x6000, 0x001c} {
		t.Fatalf("EmuTOS reset state=%+v", state)
	}
	result, err := machine.Step()
	if err != nil {
		t.Fatal(err)
	}
	state = machine.CPU.State
	if result.Clocks != 10 || machine.Instructions != 1 || machine.Clocks != 10 ||
		state.SSP != 0x602e0104 || state.SR != 0x2700 || state.PC != 0x00fc0052 ||
		state.Prefetch != [2]uint16{0x46fc, 0x2700} {
		t.Fatalf("EmuTOS first step result=%+v state=%+v", result, state)
	}
}
