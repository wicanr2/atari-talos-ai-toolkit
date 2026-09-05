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
	machine.Instructions, machine.Interrupts, machine.Clocks = 99, 3, 1234
	machine.nextVBLClock, machine.vblFrameClocks, machine.vblPending = 9999, 8888, true
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
	if machine.Instructions != 0 || machine.Interrupts != 0 || machine.Clocks != 0 ||
		machine.nextVBLClock != firstColorSTVBLClock || machine.vblFrameClocks != colorST60HzFrameClocks || machine.vblPending {
		t.Fatalf("reset counters/events instructions=%d interrupts=%d clocks=%d next=%d period=%d pending=%v",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.nextVBLClock, machine.vblFrameClocks, machine.vblPending)
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
	if got, err := machine.Memory.ReadByte(MFPGPIP, 5); err != nil || got != 0xa1 {
		t.Fatalf("MFP GPIP sampled=%02x/%v want a1", got, err)
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
}

func TestMachineEmuTOSWritesMFPResetDDR(t *testing.T) {
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
	for i := 0; i < 7483; i++ {
		result, stepErr := machine.Step()
		if stepErr != nil {
			t.Fatalf("step %d: %v", i+1, stepErr)
		}
		if i == 7482 && result.Clocks != 16 {
			t.Fatalf("MFP DDR write clocks=%d want 16", result.Clocks)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0x1e, 2, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xfffffa05, 0x3156, 0, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 7483 || machine.Clocks != 176726 || state.D != wantD || state.A != wantA ||
		state.USP != 0 || state.SSP != 0x0f8c || state.SR != 0x2714 || state.PC != 0x00fc6152 ||
		state.Prefetch != [2]uint16{0x5488, 0xb0fc} {
		t.Fatalf("MFP DDR boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	if got, err := machine.Memory.ReadByte(MFPDDR, 5); err != nil || got != 0 {
		t.Fatalf("MFP DDR=%02x/%v want 00", got, err)
	}
}

func TestMachineEmuTOSWritesMFPResetIERs(t *testing.T) {
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
	for i := 0; i < 7491; i++ {
		result, stepErr := machine.Step()
		if stepErr != nil {
			t.Fatalf("step %d: %v", i+1, stepErr)
		}
		if (i == 7486 || i == 7490) && result.Clocks != 16 {
			t.Fatalf("MFP IER write step %d clocks=%d want 16", i+1, result.Clocks)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0x1e, 2, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xfffffa09, 0x3156, 0, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 7491 || machine.Clocks != 176814 || state.D != wantD || state.A != wantA ||
		state.USP != 0 || state.SSP != 0x0f8c || state.SR != 0x2714 || state.PC != 0x00fc6152 ||
		state.Prefetch != [2]uint16{0x5488, 0xb0fc} {
		t.Fatalf("MFP IER boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	for _, address := range []uint32{MFPIERA, MFPIERB} {
		if got, err := machine.Memory.ReadByte(address, 5); err != nil || got != 0 {
			t.Fatalf("MFP IER %06x=%02x/%v want 00", address, got, err)
		}
	}
}

func TestMachineEmuTOSClearsMFPResetIPRs(t *testing.T) {
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
	for i := 0; i < 7499; i++ {
		result, stepErr := machine.Step()
		if stepErr != nil {
			t.Fatalf("step %d: %v", i+1, stepErr)
		}
		if (i == 7494 || i == 7498) && result.Clocks != 16 {
			t.Fatalf("MFP IPR write step %d clocks=%d want 16", i+1, result.Clocks)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0x1e, 2, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xfffffa0d, 0x3156, 0, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 7499 || machine.Clocks != 176902 || state.D != wantD || state.A != wantA ||
		state.USP != 0 || state.SSP != 0x0f8c || state.SR != 0x2714 || state.PC != 0x00fc6152 ||
		state.Prefetch != [2]uint16{0x5488, 0xb0fc} {
		t.Fatalf("MFP IPR boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	for _, address := range []uint32{MFPIPRA, MFPIPRB} {
		if got, err := machine.Memory.ReadByte(address, 5); err != nil || got != 0 {
			t.Fatalf("MFP IPR %06x=%02x/%v want 00", address, got, err)
		}
	}
}

func TestMachineEmuTOSClearsMFPResetISRs(t *testing.T) {
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
	for i := 0; i < 7507; i++ {
		result, stepErr := machine.Step()
		if stepErr != nil {
			t.Fatalf("step %d: %v", i+1, stepErr)
		}
		if (i == 7502 || i == 7506) && result.Clocks != 16 {
			t.Fatalf("MFP ISR write step %d clocks=%d want 16", i+1, result.Clocks)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0x1e, 2, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xfffffa11, 0x3156, 0, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 7507 || machine.Clocks != 176990 || state.D != wantD || state.A != wantA ||
		state.USP != 0 || state.SSP != 0x0f8c || state.SR != 0x2714 || state.PC != 0x00fc6152 ||
		state.Prefetch != [2]uint16{0x5488, 0xb0fc} {
		t.Fatalf("MFP ISR boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	for _, address := range []uint32{MFPISRA, MFPISRB} {
		if got, err := machine.Memory.ReadByte(address, 5); err != nil || got != 0 {
			t.Fatalf("MFP ISR %06x=%02x/%v want 00", address, got, err)
		}
	}
}

func TestMachineEmuTOSMasksMFPResetInterrupts(t *testing.T) {
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
	for i := 0; i < 7515; i++ {
		result, stepErr := machine.Step()
		if stepErr != nil {
			t.Fatalf("step %d: %v", i+1, stepErr)
		}
		if (i == 7510 || i == 7514) && result.Clocks != 16 {
			t.Fatalf("MFP IMR write step %d clocks=%d want 16", i+1, result.Clocks)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0x1e, 2, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xfffffa15, 0x3156, 0, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 7515 || machine.Clocks != 177078 || state.D != wantD || state.A != wantA ||
		state.USP != 0 || state.SSP != 0x0f8c || state.SR != 0x2714 || state.PC != 0x00fc6152 ||
		state.Prefetch != [2]uint16{0x5488, 0xb0fc} {
		t.Fatalf("MFP IMR boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	for _, address := range []uint32{MFPIMRA, MFPIMRB} {
		if got, err := machine.Memory.ReadByte(address, 5); err != nil || got != 0 {
			t.Fatalf("MFP IMR %06x=%02x/%v want 00", address, got, err)
		}
	}
}

func TestMachineEmuTOSResetsMFPVectorRegister(t *testing.T) {
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
	for i := 0; i < 7519; i++ {
		result, stepErr := machine.Step()
		if stepErr != nil {
			t.Fatalf("step %d: %v", i+1, stepErr)
		}
		if i == 7518 && result.Clocks != 16 {
			t.Fatalf("MFP VR write step %d clocks=%d want 16", i+1, result.Clocks)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0x1e, 2, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xfffffa17, 0x3156, 0, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 7519 || machine.Clocks != 177122 || state.D != wantD || state.A != wantA ||
		state.USP != 0 || state.SSP != 0x0f8c || state.SR != 0x2714 || state.PC != 0x00fc6152 ||
		state.Prefetch != [2]uint16{0x5488, 0xb0fc} {
		t.Fatalf("MFP VR boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	if got, err := machine.Memory.ReadByte(MFPVR, 5); err != nil || got != 0 {
		t.Fatalf("MFP VR=%02x/%v want 00", got, err)
	}
}

func TestMachineEmuTOSStopsMFPResetTimers(t *testing.T) {
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
	for i := 0; i < 7531; i++ {
		result, stepErr := machine.Step()
		if stepErr != nil {
			t.Fatalf("step %d: %v", i+1, stepErr)
		}
		if (i == 7522 || i == 7526 || i == 7530) && result.Clocks != 16 {
			t.Fatalf("MFP timer control write step %d clocks=%d want 16", i+1, result.Clocks)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0x1e, 2, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xfffffa1d, 0x3156, 0, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 7531 || machine.Clocks != 177254 || state.D != wantD || state.A != wantA ||
		state.USP != 0 || state.SSP != 0x0f8c || state.SR != 0x2714 || state.PC != 0x00fc6152 ||
		state.Prefetch != [2]uint16{0x5488, 0xb0fc} {
		t.Fatalf("MFP timer control boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	for _, address := range []uint32{MFPTACR, MFPTBCR, MFPTCDCR} {
		if got, err := machine.Memory.ReadByte(address, 5); err != nil || got != 0 {
			t.Fatalf("MFP timer control %06x=%02x/%v want 00", address, got, err)
		}
	}
}

func TestMachineEmuTOSLoadsMFPResetTimerData(t *testing.T) {
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
	for i := 0; i < 7547; i++ {
		result, stepErr := machine.Step()
		if stepErr != nil {
			t.Fatalf("step %d: %v", i+1, stepErr)
		}
		if (i == 7534 || i == 7538 || i == 7542 || i == 7546) && result.Clocks != 16 {
			t.Fatalf("MFP timer data write step %d clocks=%d want 16", i+1, result.Clocks)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0x1e, 2, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xfffffa25, 0x3156, 0, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 7547 || machine.Clocks != 177430 || state.D != wantD || state.A != wantA ||
		state.USP != 0 || state.SSP != 0x0f8c || state.SR != 0x2714 || state.PC != 0x00fc6152 ||
		state.Prefetch != [2]uint16{0x5488, 0xb0fc} {
		t.Fatalf("MFP timer data boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	for _, address := range []uint32{MFPTADR, MFPTBDR, MFPTCDR, MFPTDDR} {
		if got, err := machine.Memory.ReadByte(address, 5); err != nil || got != 0 {
			t.Fatalf("MFP timer data %06x=%02x/%v want 00", address, got, err)
		}
	}
	for machine.Instructions < 7550 {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("post-TDDR step %d: %v", machine.Instructions+1, err)
		}
	}
	if _, err := machine.Step(); err != nil || machine.Instructions != 7551 ||
		machine.CPU.State.A[0] != 0xfffffa27 {
		t.Fatalf("SCR successor instructions=%d A0=%08x err=%v",
			machine.Instructions, machine.CPU.State.A[0], err)
	}
}

func TestMachineEmuTOSStartsTimerCDelayMode(t *testing.T) {
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
	for machine.Instructions < 68103 {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("step %d: %v", machine.Instructions+1, err)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0xffff_fffe, 0x00fc_0005, 5, 0x00fc_04de, 0x0008_0000, 0x0010_0000, 5, 1}
	wantA := [7]uint32{2, 0x00fc_634e, 0, 0, 0, 0x00fc_01f4, 0x0000_0ffc}
	if machine.Interrupts != 3 || machine.Clocks != 963104 || state.D != wantD || state.A != wantA ||
		state.USP != 0 || state.SSP != 0x0f70 || state.SR != 0x2708 || state.PC != 0x00fc6196 ||
		state.Prefetch != [2]uint16{0xe378, 0x1238} || machine.Memory.mfpTCDCR != 0x50 ||
		machine.Memory.mfpTCDR != 0xc0 || machine.Memory.mfpTCMain != 0xc0 || !machine.Memory.mfpTimerCStart {
		t.Fatalf("Timer C successor boundary instructions=%d interrupts=%d clocks=%d state=%+v control/data/main/start=%02x/%02x/%02x/%v",
			machine.Instructions, machine.Interrupts, machine.Clocks, state, machine.Memory.mfpTCDCR,
			machine.Memory.mfpTCDR, machine.Memory.mfpTCMain, machine.Memory.mfpTimerCStart)
	}
	if result, err := machine.Step(); err != nil || result.Clocks != 16 || machine.Instructions != 68104 ||
		machine.Clocks != 963120 || machine.CPU.State.PC != 0x00fc6198 ||
		machine.CPU.State.Prefetch != [2]uint16{0x1238, 0xfa15} {
		t.Fatalf("ROL.W result=%+v instructions=%d clocks=%d state=%+v err=%v",
			result, machine.Instructions, machine.Clocks, machine.CPU.State, err)
	}
	sawSecondTransfer := false
	sawResetResponse := false
	sawMIDIConfigured := false
	sawMFPACIAEnabled := false
	for machine.Instructions < 200000 {
		if _, err := machine.Step(); err != nil {
			if sawResetResponse {
				state = machine.CPU.State
				wantD = [8]uint32{0xffff_ffef, 0x60, 0, 0, 0x0008_0000, 0x0010_0000, 5, 1}
				wantA = [7]uint32{0x8c, 0x257c, 0, 0x00fc_615e, 0, 0x00fc_01f4, 0x0000_0ffc}
				if err.Error() != "st: write 1-byte bus fault at 0xfffa09 fc=5: unsupported_device_state" ||
					machine.Instructions != 136182 || machine.Interrupts != 8 || machine.Clocks != 1578882 ||
					state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0f7a ||
					state.SR != 0x2300 || state.PC != 0x00fc61aa ||
					state.Prefetch != [2]uint16{0xfa09, 0x11c0} ||
					machine.Memory.midiACIAControl != 0x95 || !machine.Memory.midiACIAConfigured ||
					machine.Memory.mfpIERB != 0x60 || machine.Memory.mfpIMRB != 0x60 ||
					machine.Memory.mfpACIAEnableStage != 5 || machine.Memory.ikbdStaleRDRReads != 0 {
					t.Fatalf("post-IKBD gate instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
						machine.Instructions, machine.Interrupts, machine.Clocks, state, err)
				}
				return
			}
			state = machine.CPU.State
			wantD = [8]uint32{0xffff_ffff, 0x10, 1, 0, 0x0008_0000, 0x0010_0000, 5, 1}
			wantA = [7]uint32{0x94, 0x3216, 0x00fc_5132, 0, 0, 0x00fc_01f4, 0x0000_0ffc}
			if err.Error() != "st: write 1-byte bus fault at 0xfffc02 fc=5: unsupported_device_state" ||
				machine.Instructions != 68645 || machine.Interrupts != 4 || machine.Clocks != 969640 ||
				state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0f7a ||
				state.SR != 0x2308 || state.PC != 0x00fc515a || state.Prefetch != [2]uint16{0xfc02, 0x241f} ||
				machine.Memory.mfpIERB != 0x20 || machine.Memory.mfpIMRB != 0x20 ||
				machine.Memory.mfpIERA != 0x14 || machine.Memory.mfpIMRA != 0x14 ||
				machine.Memory.mfpTCDCR != 0x51 || machine.Memory.mfpTDDR != 2 ||
				machine.Memory.mfpTDMain != 2 || !machine.Memory.mfpTimerDStart ||
				machine.Memory.mfpUCR != 0x88 || machine.Memory.mfpRSR != 1 || machine.Memory.mfpTSR != 1 ||
				machine.Memory.psgRegisterSelect != 14 || machine.Memory.psgRegisters[7] != 0xc0 ||
				machine.Memory.psgRegisters[14] != 7 {
				t.Fatalf("post-ROL gate instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
					machine.Instructions, machine.Interrupts, machine.Clocks, state, err)
			}
			return
		}
		if machine.ikbdSecondTXClock != 0 && !sawSecondTransfer {
			if machine.Memory.ikbdACIATDR != 1 || machine.Memory.ikbdACIATXPending ||
				machine.Memory.ikbdACIAStatus != 2 {
				t.Fatalf("second transfer device state clock=%d TDR/pending/status=%02x/%v/%02x",
					machine.ikbdSecondTXClock, machine.Memory.ikbdACIATDR,
					machine.Memory.ikbdACIATXPending, machine.Memory.ikbdACIAStatus)
			}
			sawSecondTransfer = true
		}
		if machine.Memory.ikbdResetResponseRead && !sawResetResponse {
			state = machine.CPU.State
			wantD = [8]uint32{0xffff_00f1, 0xffff_ff83, 0, 0, 0x0008_0000, 0x0010_0000, 5, 1}
			wantA = [7]uint32{0x12c, 0x3216, 0x00fc_48a2, 0, 0, 0x00fc_01f4, 0x0000_0ffc}
			if !sawSecondTransfer || machine.ikbdSecondTXClock != 979806 ||
				machine.Instructions != 128378 || machine.Interrupts != 21 || machine.Clocks != 1509022 ||
				state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0f82 ||
				state.SR != 0x2308 || state.PC != 0x00fc48bc ||
				state.Prefetch != [2]uint16{0x4e75, 0x2239} || machine.Memory.ikbdACIAStatus != 2 ||
				machine.Memory.ikbdACIARDR != 0xf1 || machine.ikbdResetRXDeadline != 0 ||
				machine.ikbdResetRXClock != 1503070 {
				t.Fatalf("IKBD response read boundary sawSecond=%v secondClock=%d instructions=%d interrupts=%d clocks=%d state=%+v status/RDR/deadline/rxClock=%02x/%02x/%d/%d",
					sawSecondTransfer, machine.ikbdSecondTXClock, machine.Instructions, machine.Interrupts,
					machine.Clocks, state, machine.Memory.ikbdACIAStatus, machine.Memory.ikbdACIARDR,
					machine.ikbdResetRXDeadline, machine.ikbdResetRXClock)
			}
			sawResetResponse = true
		}
		if machine.Memory.midiACIAConfigured && !sawMIDIConfigured {
			if machine.Instructions != 136125 || machine.Interrupts != 23 || machine.Clocks != 1579268 {
				t.Fatalf("MIDI configured receipt instructions=%d interrupts=%d clocks=%d",
					machine.Instructions, machine.Interrupts, machine.Clocks)
			}
			sawMIDIConfigured = true
		}
		if machine.Memory.mfpACIAEnableStage == 5 && !sawMFPACIAEnabled {
			if machine.Instructions != 136236 || machine.Interrupts != 23 || machine.Clocks != 1580634 {
				t.Fatalf("MFP ACIA enabled receipt instructions=%d interrupts=%d clocks=%d",
					machine.Instructions, machine.Interrupts, machine.Clocks)
			}
			sawMFPACIAEnabled = true
		}
		if machine.Memory.mfpTimerDSystemStage == 8 {
			state = machine.CPU.State
			wantD = [8]uint32{0x2f8, 0x11d00, 0x128e0, 0, 0x0008_0000, 0x0010_0000, 5, 1}
			wantA = [7]uint32{0xffff_fa1d, 0x00fc_03ea, 0, 0x00fc_615e, 0, 0x00fc_01f4, 0x0000_0ffc}
			if machine.Instructions != 136285 || machine.Interrupts != 23 || machine.Clocks != 1581256 ||
				state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0f76 ||
				state.SR != 0x2300 || state.PC != 0x00fc7848 ||
				state.Prefetch != [2]uint16{0x202f, 4} || machine.Memory.mfpIERB != 0x70 ||
				machine.Memory.mfpIMRB != 0x70 || machine.Memory.mfpTCDCR != 0x52 ||
				machine.Memory.mfpTDDR != 0 || machine.Memory.mfpTDMain != 0 ||
				!machine.Memory.mfpTimerDStart {
				t.Fatalf("Timer D system start boundary instructions=%d interrupts=%d clocks=%d state=%+v IERB/IMRB/control/data/main/start=%02x/%02x/%02x/%02x/%02x/%v",
					machine.Instructions, machine.Interrupts, machine.Clocks, state,
					machine.Memory.mfpIERB, machine.Memory.mfpIMRB, machine.Memory.mfpTCDCR,
					machine.Memory.mfpTDDR, machine.Memory.mfpTDMain, machine.Memory.mfpTimerDStart)
			}
			return
		}
	}
	t.Fatalf("no typed successor gate before 200000 instructions: interrupts=%d clocks=%d state=%+v ACIA=%02x/%02x/%v/%d",
		machine.Interrupts, machine.Clocks, machine.CPU.State, machine.Memory.ikbdACIATDR,
		machine.Memory.ikbdACIAStatus, machine.Memory.ikbdACIATXPending, machine.Memory.ikbdACIATXShiftTicks)
}

func TestMachineAdvancesIKBDACIAAtFirstDeadline(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []byte{3, 0x96} {
		if err := memory.WriteByte(IKBDACIAControl, value, 5); err != nil {
			t.Fatal(err)
		}
	}
	if err := memory.WriteByte(IKBDACIAData, 0x80, 5); err != nil {
		t.Fatal(err)
	}
	machine := &Machine{Memory: memory, aciaClockStarted: true, nextACIABitClock: 2024, Clocks: 2023}
	machine.advanceDueACIAClocks()
	if !memory.ikbdACIATXPending || memory.ikbdACIAStatus != 0 {
		t.Fatalf("before deadline pending/status=%v/%02x", memory.ikbdACIATXPending, memory.ikbdACIAStatus)
	}
	machine.Clocks = 2024
	machine.advanceDueACIAClocks()
	if memory.ikbdACIATXPending || memory.ikbdACIAStatus != 2 || memory.ikbdACIATXShiftTicks != 10 ||
		machine.nextACIABitClock != 3048 {
		t.Fatalf("at deadline pending/status/shift/next=%v/%02x/%d/%d", memory.ikbdACIATXPending,
			memory.ikbdACIAStatus, memory.ikbdACIATXShiftTicks, machine.nextACIABitClock)
	}
}

func TestTimerDDeadlineKeepsRationalPhase(t *testing.T) {
	const start = uint64(1000)
	if got := timerDDeadline(start, 1); got != 9355 {
		t.Fatalf("first deadline=%d want 9355", got)
	}
	if got := timerDDeadline(start, 3); got != 26066 {
		t.Fatalf("third deadline=%d want 26066", got)
	}
	if timerDDeadline(start, 3)-timerDDeadline(start, 2) != 8356 {
		t.Fatalf("third period did not carry the rational remainder")
	}
}

func TestTimerCDeadlineKeepsRationalPhase(t *testing.T) {
	const start = uint64(1000)
	if got := timerCDeadline(start, 1); got != 41106 {
		t.Fatalf("first deadline=%d want 41106", got)
	}
	if timerCDeadline(start, 5)-timerCDeadline(start, 4) != 40107 {
		t.Fatalf("fifth period did not carry the rational remainder")
	}
}

func TestMFPBInterruptPriorityAndInService(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.mfpIERB, memory.mfpIMRB, memory.mfpIPRB = 0x30, 0x30, 0x30
	machine := &Machine{Memory: memory}
	if channel, ok := machine.mfpBInterruptChannel(); !ok || channel != 5 {
		t.Fatalf("channel/ok=%d/%v want 5/true", channel, ok)
	}
	memory.mfpISRB = 0x20
	if channel, ok := machine.mfpBInterruptChannel(); ok {
		t.Fatalf("in-service channel 5 allowed channel %d", channel)
	}
	memory.mfpISRB = 0x10
	if channel, ok := machine.mfpBInterruptChannel(); !ok || channel != 5 {
		t.Fatalf("lower in-service channel/ok=%d/%v want 5/true", channel, ok)
	}
}

func TestMachineEmuTOSEntersTimerCHandler(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	for machine.Interrupts < 5 && machine.Instructions < 100000 {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("step instructions=%d interrupts=%d clocks=%d PC=%08x: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State.PC, err)
		}
	}
	state := machine.CPU.State
	if machine.Instructions != 72342 || machine.Interrupts != 5 || machine.Clocks != 1003004 ||
		machine.Memory.mfpTimerCStartClock != 962844 || machine.nextTimerCClock != 1043056 ||
		state.PC != 0x00fc04e2 ||
		state.Prefetch != [2]uint16{0x52b8, 0x04ba} || state.SR>>8&7 != 6 ||
		machine.Memory.mfpIPRB&0x20 != 0 || machine.Memory.mfpISRB&0x20 == 0 {
		t.Fatalf("Timer C handler boundary instructions=%d interrupts=%d clocks=%d state=%+v IPRB/ISRB=%02x/%02x start/next=%d/%d",
			machine.Instructions, machine.Interrupts, machine.Clocks, state,
			machine.Memory.mfpIPRB, machine.Memory.mfpISRB,
			machine.Memory.mfpTimerCStartClock, machine.nextTimerCClock)
	}
	for i := 0; i < 32 && machine.Memory.mfpISRB&0x20 != 0; i++ {
		if _, err := machine.Step(); err != nil {
			t.Fatal(err)
		}
	}
	if machine.Memory.mfpISRB&0x20 != 0 {
		t.Fatalf("guest did not clear Timer C in-service: ISRB=%02x", machine.Memory.mfpISRB)
	}
}

func TestMachineEmuTOSEntersTimerDHandler(t *testing.T) {
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
	for machine.Instructions < 160000 {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("step instructions=%d interrupts=%d clocks=%d PC=%08x: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State.PC, err)
		}
		if machine.CPU.State.PC == 0x00fc7888 && machine.Memory.mfpISRB&0x10 != 0 {
			break
		}
	}
	state := machine.CPU.State
	if machine.Instructions != 137213 || machine.Interrupts != 24 || machine.Clocks != 1589660 ||
		machine.Memory.mfpTimerDStartClock != 1581256 || machine.nextTimerDClock != 1597966 ||
		state.PC != 0x00fc7888 ||
		state.Prefetch != [2]uint16{0x52b9, 0x0000} || state.SR>>8&7 != 6 ||
		machine.Memory.mfpIPRB&0x10 != 0 || machine.Memory.mfpISRB&0x10 == 0 {
		t.Fatalf("Timer D handler boundary instructions=%d interrupts=%d clocks=%d state=%+v IPRB/ISRB=%02x/%02x start/next=%d/%d",
			machine.Instructions, machine.Interrupts, machine.Clocks, state,
			machine.Memory.mfpIPRB, machine.Memory.mfpISRB,
			machine.Memory.mfpTimerDStartClock, machine.nextTimerDClock)
	}
	for i := 0; i < 8 && machine.Memory.mfpISRB&0x10 != 0; i++ {
		if _, err := machine.Step(); err != nil {
			t.Fatal(err)
		}
	}
	if machine.Memory.mfpISRB&0x10 != 0 {
		t.Fatalf("guest did not clear Timer D in-service: ISRB=%02x", machine.Memory.mfpISRB)
	}
}

func TestMachineDeliversIKBDResetResponseAtDeadline(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.ikbdACIAConfigured = true
	memory.ikbdACIAStatus = 2
	memory.ikbdACIATDR = 1
	memory.ikbdACIATXShiftTicks = 1
	machine := &Machine{Memory: memory, Clocks: 1000, aciaClockStarted: true, nextACIABitClock: 1000}
	machine.advanceClockedDevices()
	if machine.ikbdResetRXDeadline != 514024 || memory.ikbdResetCommandDone ||
		!memory.ikbdResetCommandHandled || memory.ikbdACIAStatus != 2 {
		t.Fatalf("scheduled deadline/done/status=%d/%v/%02x", machine.ikbdResetRXDeadline,
			memory.ikbdResetCommandDone, memory.ikbdACIAStatus)
	}
	machine.Clocks = 514023
	machine.advanceClockedDevices()
	if memory.ikbdACIAStatus != 2 {
		t.Fatalf("early status=%02x", memory.ikbdACIAStatus)
	}
	machine.Clocks = 514024
	machine.advanceClockedDevices()
	if machine.ikbdResetRXDeadline != 0 || machine.ikbdResetRXClock != 514024 || memory.ikbdACIARDR != 0xf1 ||
		memory.ikbdACIAStatus != 0x83 {
		t.Fatalf("delivered deadline/RDR/status=%d/%02x/%02x", machine.ikbdResetRXDeadline,
			memory.ikbdACIARDR, memory.ikbdACIAStatus)
	}
}

func TestMachineEmuTOSWritesMFPUSARTResetRegisters(t *testing.T) {
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
	for machine.Instructions < 7563 {
		result, stepErr := machine.Step()
		if stepErr != nil {
			t.Fatalf("USART reset step instructions=%d clocks=%d PC=%08x A0=%08x err=%v",
				machine.Instructions, machine.Clocks, machine.CPU.State.PC, machine.CPU.State.A[0], stepErr)
		}
		if (machine.Instructions == 7551 || machine.Instructions == 7555 ||
			machine.Instructions == 7559 || machine.Instructions == 7563) && result.Clocks != 16 {
			t.Fatalf("MFP USART write step %d clocks=%d want 16", machine.Instructions, result.Clocks)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0x1e, 2, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xfffffa2d, 0x3156, 0, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Clocks != 177606 || state.D != wantD || state.A != wantA || state.USP != 0 ||
		state.SSP != 0x0f8c || state.SR != 0x2714 || state.PC != 0x00fc6152 ||
		state.Prefetch != [2]uint16{0x5488, 0xb0fc} {
		t.Fatalf("MFP USART boundary instructions=%d clocks=%d state=%+v",
			machine.Instructions, machine.Clocks, state)
	}
	for _, address := range []uint32{MFPSCR, MFPUCR, MFPRSR, MFPTSR} {
		if got, err := machine.Memory.ReadByte(address, 5); err != nil || got != 0 {
			t.Fatalf("MFP USART %06x=%02x/%v want 00", address, got, err)
		}
	}
	for machine.CPU.State.Prefetch[0] != 0x4e72 {
		if machine.Instructions > 7650 {
			t.Fatal("STOP was not reached after first VBL handler")
		}
		if _, err := machine.Step(); err != nil {
			t.Fatalf("post-USART step %d clocks=%d PC=%08x prefetch=%04x,%04x interrupts=%d: %v",
				machine.Instructions+1, machine.Clocks, machine.CPU.State.PC,
				machine.CPU.State.Prefetch[0], machine.CPU.State.Prefetch[1], machine.Interrupts, err)
		}
	}
	stopPreState := machine.CPU.State
	wantSTOPD := [8]uint32{0x2304, 0x18, 0x2710, 1, 0x00080000, 0x00100000, 5, 1}
	wantSTOPA := [7]uint32{0xfffffa2f, 0x3156, 0x00fcd074, 0, 0, 0x00fc01f4, 0x00000ffc}
	if stopPreState.D != wantSTOPD || stopPreState.A != wantSTOPA || stopPreState.USP != 0 ||
		stopPreState.SSP != 0x0f70 || stopPreState.SR != 0x2300 || stopPreState.PC != 0x00fcd09e ||
		stopPreState.Prefetch != [2]uint16{0x4e72, 0x2300} {
		t.Fatalf("STOP pre-state mismatch instructions=%d interrupts=%d clocks=%d: %+v", machine.Instructions, machine.Interrupts, machine.Clocks, stopPreState)
	}
	result, err := machine.Step()
	if err != nil || result.Clocks != 4 || machine.Instructions != 7604 || machine.Interrupts != 1 || machine.Clocks != 178244 ||
		machine.CPU.State.SR != 0x2300 || !machine.CPU.IsStopped() ||
		machine.CPU.State.PC != 0x00fcd09e || machine.CPU.State.Prefetch != [2]uint16{0x4e72, 0x2300} {
		t.Fatalf("STOP result=%+v instructions=%d clocks=%d pre=%+v state=%+v err=%v",
			result, machine.Instructions, machine.Clocks, stopPreState, machine.CPU.State, err)
	}
	result, err = machine.Step()
	state = machine.CPU.State
	if err != nil || result.Clocks != 89088 || machine.Instructions != 7604 || machine.Interrupts != 2 ||
		machine.Clocks != 267332 || machine.CPU.IsStopped() || state.D != wantSTOPD || state.A != wantSTOPA ||
		state.SSP != 0x0f6a || state.SR != 0x2400 || state.PC != 0x00fc044a ||
		state.Prefetch != [2]uint16{0x52b8, 0x0466} {
		t.Fatalf("second VBL result=%+v instructions=%d clocks=%d state=%+v err=%v",
			result, machine.Instructions, machine.Clocks, state, err)
	}
	wantSecondFrame := []uint16{0x2300, 0x00fc, 0xd09e}
	for i, want := range wantSecondFrame {
		got, readErr := machine.Memory.ReadWord(0x0f6a+uint32(i*2), 5)
		if readErr != nil || got != want {
			t.Fatalf("second VBL frame[%d]=%04x/%v want %04x", i, got, readErr, want)
		}
	}
	if _, err := machine.Step(); err != nil {
		t.Fatal(err)
	}
	frHigh, err := machine.Memory.ReadWord(0x466, 5)
	if err != nil {
		t.Fatal(err)
	}
	frLow, err := machine.Memory.ReadWord(0x468, 5)
	if err != nil {
		t.Fatal(err)
	}
	if got := uint32(frHigh)<<16 | uint32(frLow); got != 2 {
		t.Fatalf("guest second VBL handler frclock=%d want 2", got)
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
