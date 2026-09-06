package st

import (
	"bytes"
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
	if err := machine.Memory.WriteByteFC(MMUConfig, 0x0a, 5); err != nil {
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
	if got, err := machine.Memory.ReadByteFC(MMUConfig, 5); err != nil || got != 0 {
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
	if got, err := machine.Memory.ReadByteFC(MMUConfig, 5); err != nil || got != 0 {
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
	if got, err := machine.Memory.ReadByteFC(MMUConfig, 5); err != nil || got != 0 {
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
	if got, err := machine.Memory.ReadByteFC(MFPGPIP, 5); err != nil || got != 0xa1 {
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
	if got, err := machine.Memory.ReadByteFC(MFPAER, 5); err != nil || got != 0 {
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
	if got, err := machine.Memory.ReadByteFC(MFPDDR, 5); err != nil || got != 0 {
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
		if got, err := machine.Memory.ReadByteFC(address, 5); err != nil || got != 0 {
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
		if got, err := machine.Memory.ReadByteFC(address, 5); err != nil || got != 0 {
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
		if got, err := machine.Memory.ReadByteFC(address, 5); err != nil || got != 0 {
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
		if got, err := machine.Memory.ReadByteFC(address, 5); err != nil || got != 0 {
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
	if got, err := machine.Memory.ReadByteFC(MFPVR, 5); err != nil || got != 0 {
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
		if got, err := machine.Memory.ReadByteFC(address, 5); err != nil || got != 0 {
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
		if got, err := machine.Memory.ReadByteFC(address, 5); err != nil || got != 0 {
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
		if err := memory.WriteByteFC(IKBDACIAControl, value, 5); err != nil {
			t.Fatal(err)
		}
	}
	if err := memory.WriteByteFC(IKBDACIAData, 0x80, 5); err != nil {
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
	memory.mfpIERB, memory.mfpIMRB, memory.mfpIPRB = 0x70, 0x70, 0x70
	machine := &Machine{Memory: memory}
	if channel, ok := machine.mfpBInterruptChannel(); !ok || channel != 6 {
		t.Fatalf("channel/ok=%d/%v want 6/true", channel, ok)
	}
	memory.mfpISRB = 0x40
	if channel, ok := machine.mfpBInterruptChannel(); ok {
		t.Fatalf("in-service channel 6 allowed channel %d", channel)
	}
	memory.mfpISRB = 0x20
	if channel, ok := machine.mfpBInterruptChannel(); !ok || channel != 6 {
		t.Fatalf("lower in-service channel/ok=%d/%v want 6/true", channel, ok)
	}
	memory.mfpIPRB = 0x30
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

func TestMachineEmuTOSStopsTimerD(t *testing.T) {
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
	for machine.Instructions < 400_000 && machine.Memory.mfpTimerDStopStage < 7 {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("Timer D stop instructions=%d interrupts=%d clocks=%d PC=%08x prefetch=%04x,%04x A0=%08x D0=%08x: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State.PC,
				machine.CPU.State.Prefetch[0], machine.CPU.State.Prefetch[1], machine.CPU.State.A[0],
				machine.CPU.State.D[0], err)
		}
	}
	if machine.Instructions != 289256 || machine.Interrupts != 234 || machine.Clocks != 2978730 ||
		machine.CPU.State.PC != 0x00fc61b4 ||
		machine.CPU.State.Prefetch != [2]uint16{0x4e75, 0x302f} ||
		machine.Memory.mfpTimerDStopStage != 7 || machine.Memory.mfpTimerDStart ||
		machine.timerDClockStarted || machine.nextTimerDClock != 0 || machine.Memory.mfpIERB != 0x60 ||
		machine.Memory.mfpIMRB != 0x60 || machine.Memory.mfpTCDCR != 0x50 {
		t.Fatalf("stop boundary instructions=%d interrupts=%d clocks=%d state=%+v stage/start/scheduler/next/IERB/IMRB/TCDCR=%d/%v/%v/%d/%02x/%02x/%02x",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
			machine.Memory.mfpTimerDStopStage, machine.Memory.mfpTimerDStart,
			machine.timerDClockStarted, machine.nextTimerDClock, machine.Memory.mfpIERB,
			machine.Memory.mfpIMRB, machine.Memory.mfpTCDCR)
	}
	high, err := machine.Memory.ReadWord(0x110, 5)
	if err != nil {
		t.Fatal(err)
	}
	low, err := machine.Memory.ReadWord(0x112, 5)
	if err != nil || uint32(high)<<16|uint32(low) != 0x00fc03ea {
		t.Fatalf("vector 68=%04x%04x err=%v want 00fc03ea", high, low, err)
	}
	for machine.Instructions < 400_000 && machine.Memory.mfpUSARTReconfigStage < 7 {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("USART reconfigure instructions=%d interrupts=%d clocks=%d PC=%08x prefetch=%04x,%04x stage=%d: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State.PC,
				machine.CPU.State.Prefetch[0], machine.CPU.State.Prefetch[1],
				machine.Memory.mfpUSARTReconfigStage, err)
		}
	}
	if machine.Memory.mfpUSARTReconfigStage != 7 || !machine.Memory.mfpTimerDStart ||
		machine.Memory.mfpTCDCR != 0x51 || machine.Memory.mfpTDDR != 2 ||
		machine.Memory.mfpUCR != 0x88 || machine.Memory.mfpRSR != 1 ||
		machine.Memory.mfpTSR != 1 || machine.Memory.mfpSCR != 0 ||
		machine.timerDClockStarted || machine.nextTimerDClock != 0 {
		t.Fatalf("USART boundary instructions=%d interrupts=%d clocks=%d state=%+v stage/start/TCDCR/TDDR/UCR/RSR/TSR/SCR/scheduler/next=%d/%v/%02x/%02x/%02x/%02x/%02x/%02x/%v/%d",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
			machine.Memory.mfpUSARTReconfigStage, machine.Memory.mfpTimerDStart,
			machine.Memory.mfpTCDCR, machine.Memory.mfpTDDR, machine.Memory.mfpUCR,
			machine.Memory.mfpRSR, machine.Memory.mfpTSR, machine.Memory.mfpSCR,
			machine.timerDClockStarted, machine.nextTimerDClock)
	}
	if machine.Instructions != 289342 || machine.Interrupts != 234 || machine.Clocks != 2979680 ||
		machine.CPU.State.PC != 0x00fc6b58 ||
		machine.CPU.State.Prefetch != [2]uint16{0x2002, 0x4cdf} {
		t.Fatalf("USART exact boundary instructions=%d interrupts=%d clocks=%d state=%+v",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State)
	}
	for machine.Instructions < 289521 {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("void DMA write instructions=%d interrupts=%d clocks=%d PC=%08x prefetch=%04x,%04x: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State.PC,
				machine.CPU.State.Prefetch[0], machine.CPU.State.Prefetch[1], err)
		}
	}
	state := machine.CPU.State
	wantD := [8]uint32{0x0000ff00, 200, 436, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0, 1, 0x3008, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 289521 || machine.Interrupts != 234 || machine.Clocks != 2982760 ||
		state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0f3c ||
		state.SR != 0x2304 || state.PC != 0x00fc3790 ||
		state.Prefetch != [2]uint16{0x0c2a, 0x0003} {
		t.Fatalf("void DMA exact boundary instructions=%d interrupts=%d clocks=%d state=%+v",
			machine.Instructions, machine.Interrupts, machine.Clocks, state)
	}
	for machine.Instructions < 400_000 && machine.Memory.psgDriveStage < 3 {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("PSG drive update instructions=%d interrupts=%d clocks=%d PC=%08x prefetch=%04x,%04x stage=%d: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State.PC,
				machine.CPU.State.Prefetch[0], machine.CPU.State.Prefetch[1],
				machine.Memory.psgDriveStage, err)
		}
	}
	state = machine.CPU.State
	wantD = [8]uint32{7, 5, 0x2300, 0, 0x00080000, 0x00100000, 5, 1}
	wantA = [7]uint32{0xffff8800, 1, 0x00fc3924, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 289556 || machine.Interrupts != 234 || machine.Clocks != 2983132 ||
		state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0f34 ||
		state.SR != 0x2700 || state.PC != 0x00fc36e4 ||
		state.Prefetch != [2]uint16{0x40c1, 0x46c2} || machine.Memory.psgDriveStage != 3 ||
		machine.Memory.psgRegisterSelect != 14 || machine.Memory.psgRegisters[14] != 5 {
		t.Fatalf("PSG drive exact boundary instructions=%d interrupts=%d clocks=%d state=%+v stage/select/R14=%d/%02x/%02x",
			machine.Instructions, machine.Interrupts, machine.Clocks, state,
			machine.Memory.psgDriveStage, machine.Memory.psgRegisterSelect, machine.Memory.psgRegisters[14])
	}
	for machine.Instructions < 400_000 && machine.Memory.fdcInitStage < 2 {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("FDC force interrupt instructions=%d interrupts=%d clocks=%d PC=%08x prefetch=%04x,%04x stage=%d: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State.PC,
				machine.CPU.State.Prefetch[0], machine.CPU.State.Prefetch[1],
				machine.Memory.fdcInitStage, err)
		}
	}
	state = machine.CPU.State
	wantD = [8]uint32{0xffffffff, 0x2700, 0x01b4, 0, 0x00080000, 0x00100000, 5, 1}
	wantA = [7]uint32{0xffff8800, 1, 0x00fc3924, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 289612 || machine.Interrupts != 234 || machine.Clocks != 2983704 ||
		state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0f38 ||
		state.SR != 0x2310 || state.PC != 0x00fc373a ||
		state.Prefetch != [2]uint16{0x4e75, 0x2f0a} || machine.Memory.fdcInitStage != 2 ||
		machine.Memory.dmaMode != 0x0080 || machine.Memory.fdcCommand != 0xd0 ||
		machine.Memory.fdcStatus != 0x80 || !machine.Memory.fdcStatusTypeI ||
		machine.Memory.fdcIRQ || machine.Memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("FDC force exact boundary instructions=%d interrupts=%d clocks=%d state=%+v stage/mode/command/status/typeI/IRQ/GPIP=%d/%04x/%02x/%02x/%v/%v/%02x",
			machine.Instructions, machine.Interrupts, machine.Clocks, state,
			machine.Memory.fdcInitStage, machine.Memory.dmaMode, machine.Memory.fdcCommand,
			machine.Memory.fdcStatus, machine.Memory.fdcStatusTypeI, machine.Memory.fdcIRQ,
			machine.Memory.mfpGPIPIn)
	}
	for machine.Instructions < 400_000 && !machine.Memory.fdcRestoreIRQObserved {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("FDC restore instructions=%d interrupts=%d clocks=%d PC=%08x prefetch=%04x,%04x stage=%d polls=%d: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State.PC,
				machine.CPU.State.Prefetch[0], machine.CPU.State.Prefetch[1],
				machine.Memory.fdcInitStage, machine.Memory.fdcRestoreInactivePolls, err)
		}
	}
	if !machine.Memory.fdcRestoreIRQObserved || machine.Memory.fdcInitStage != 5 ||
		machine.Memory.fdcRestoreInactivePolls != 9 || machine.Memory.fdcRestorePending ||
		machine.Memory.fdcStatus != 0x84 || !machine.Memory.fdcIRQ || machine.Memory.mfpGPIPIn&0x20 != 0 {
		t.Fatalf("FDC restore boundary instructions=%d interrupts=%d clocks=%d state=%+v stage/polls/pending/status/IRQ/GPIP/observed=%d/%d/%v/%02x/%v/%02x/%v",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
			machine.Memory.fdcInitStage, machine.Memory.fdcRestoreInactivePolls,
			machine.Memory.fdcRestorePending, machine.Memory.fdcStatus, machine.Memory.fdcIRQ,
			machine.Memory.mfpGPIPIn, machine.Memory.fdcRestoreIRQObserved)
	}
	state = machine.CPU.State
	wantD = [8]uint32{0x028a, 0x0091, 0x0258, 0, 0x00080000, 0x00100000, 5, 1}
	wantA = [7]uint32{0xffff8800, 1, 0x00fc3720, 0, 0x00003008, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 289803 || machine.Interrupts != 234 || machine.Clocks != 2985654 ||
		state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0f22 ||
		state.SR != 0x2308 || state.PC != 0x00fc6314 ||
		state.Prefetch != [2]uint16{0x0801, 0x0005} ||
		machine.Memory.fdcRestoreStartClock != 2984902 || machine.nextFDCRestoreClock != 0 {
		t.Fatalf("FDC restore exact boundary instructions=%d interrupts=%d clocks=%d state=%+v start/next=%d/%d",
			machine.Instructions, machine.Interrupts, machine.Clocks, state,
			machine.Memory.fdcRestoreStartClock, machine.nextFDCRestoreClock)
	}
	for steps := 0; steps < 128 && machine.Memory.fdcInitStage < 7; steps++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("FDC status read instructions=%d interrupts=%d clocks=%d state=%+v stage=%d: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
				machine.Memory.fdcInitStage, err)
		}
	}
	if machine.Memory.fdcInitStage != 7 || machine.Memory.fdcStatus != 0xe4 ||
		machine.Memory.fdcIRQ || machine.Memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("FDC status boundary instructions=%d interrupts=%d clocks=%d state=%+v stage/status/IRQ/GPIP=%d/%02x/%v/%02x",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
			machine.Memory.fdcInitStage, machine.Memory.fdcStatus, machine.Memory.fdcIRQ,
			machine.Memory.mfpGPIPIn)
	}
	state = machine.CPU.State
	wantD = [8]uint32{0xffff00e4, 0x0091, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantA = [7]uint32{0xffff8800, 1, 0, 0, 0x00003008, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 289865 || machine.Interrupts != 234 || machine.Clocks != 2986256 ||
		state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0f42 ||
		state.SR != 0x2310 || state.PC != 0x00fc38a0 ||
		state.Prefetch != [2]uint16{0x4e75, 0x2f0a} || machine.Memory.fdcStatusReadClock != 2986242 {
		t.Fatalf("FDC status exact boundary instructions=%d interrupts=%d clocks=%d state=%+v readClock=%d",
			machine.Instructions, machine.Interrupts, machine.Clocks, state,
			machine.Memory.fdcStatusReadClock)
	}
	for steps := 0; steps < 512 && machine.Memory.fdcInitStage < 14; steps++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("FDC seek instructions=%d interrupts=%d clocks=%d state=%+v stage=%d polls=%d: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
				machine.Memory.fdcInitStage, machine.Memory.fdcSeekInactivePolls, err)
		}
	}
	if machine.Memory.fdcInitStage != 14 || machine.Memory.fdcData != 0 ||
		machine.Memory.fdcCommand != 0x13 || machine.Memory.fdcStatus != 0xe4 ||
		machine.Memory.fdcSeekInactivePolls != 9 || !machine.Memory.fdcSeekIRQObserved ||
		machine.Memory.fdcSeekPending || machine.Memory.fdcIRQ || machine.Memory.mfpGPIPIn&0x20 == 0 ||
		machine.Memory.fdcSeekStatusReadClock == 0 {
		t.Fatalf("FDC seek boundary instructions=%d interrupts=%d clocks=%d state=%+v stage/data/command/status/polls/observed/pending/IRQ/GPIP/readClock=%d/%02x/%02x/%02x/%d/%v/%v/%v/%02x/%d",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
			machine.Memory.fdcInitStage, machine.Memory.fdcData, machine.Memory.fdcCommand,
			machine.Memory.fdcStatus, machine.Memory.fdcSeekInactivePolls,
			machine.Memory.fdcSeekIRQObserved, machine.Memory.fdcSeekPending,
			machine.Memory.fdcIRQ, machine.Memory.mfpGPIPIn,
			machine.Memory.fdcSeekStatusReadClock)
	}
	state = machine.CPU.State
	wantD = [8]uint32{0xffff00e4, 0x0091, 0x01b4, 0, 0x00080000, 0x00100000, 5, 1}
	wantA = [7]uint32{0, 0, 0x00003008, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 290223 || machine.Interrupts != 234 || machine.Clocks != 2989944 ||
		state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0f48 ||
		state.SR != 0x2310 || state.PC != 0x00fc38a0 ||
		state.Prefetch != [2]uint16{0x4e75, 0x2f0a} ||
		machine.Memory.fdcSeekStartClock != 2988614 ||
		machine.Memory.fdcSeekStatusReadClock != 2989930 || machine.nextFDCSeekClock != 0 {
		t.Fatalf("FDC seek exact boundary instructions=%d interrupts=%d clocks=%d state=%+v start/read/next=%d/%d/%d",
			machine.Instructions, machine.Interrupts, machine.Clocks, state,
			machine.Memory.fdcSeekStartClock, machine.Memory.fdcSeekStatusReadClock,
			machine.nextFDCSeekClock)
	}
	for steps := 0; steps < 128 && machine.Memory.psgDriveStage < 6; steps++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("drive-one PSG instructions=%d interrupts=%d clocks=%d state=%+v stage/R14=%d/%02x: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
				machine.Memory.psgDriveStage, machine.Memory.psgRegisters[14], err)
		}
	}
	if machine.Memory.psgDriveStage != 6 || machine.Memory.psgRegisterSelect != 14 ||
		machine.Memory.psgRegisters[7] != 0xc0 || machine.Memory.psgRegisters[14] != 3 {
		t.Fatalf("drive-one PSG boundary instructions=%d interrupts=%d clocks=%d state=%+v stage/select/R7/R14=%d/%02x/%02x/%02x",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
			machine.Memory.psgDriveStage, machine.Memory.psgRegisterSelect,
			machine.Memory.psgRegisters[7], machine.Memory.psgRegisters[14])
	}
	state = machine.CPU.State
	wantD = [8]uint32{5, 3, 0x2300, 0, 0x00080000, 0x00100000, 5, 1}
	wantA = [7]uint32{0xffff8800, 0, 0x00fc3924, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 290303 || machine.Interrupts != 234 || machine.Clocks != 2990890 ||
		state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0f32 ||
		state.SR != 0x2700 || state.PC != 0x00fc36e4 ||
		state.Prefetch != [2]uint16{0x40c1, 0x46c2} {
		t.Fatalf("drive-one PSG exact boundary instructions=%d interrupts=%d clocks=%d state=%+v",
			machine.Instructions, machine.Interrupts, machine.Clocks, state)
	}
	for steps := 0; steps < 1024 && !(machine.Memory.fdcProbeDrive == 1 && machine.Memory.fdcInitStage == 14); steps++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("drive-one FDC instructions=%d interrupts=%d clocks=%d state=%+v drive/stage/restorePolls/seekPolls=%d/%d/%d/%d: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
				machine.Memory.fdcProbeDrive, machine.Memory.fdcInitStage,
				machine.Memory.fdcRestoreInactivePolls, machine.Memory.fdcSeekInactivePolls, err)
		}
	}
	if machine.Memory.fdcProbeDrive != 1 || machine.Memory.fdcInitStage != 14 ||
		machine.Memory.fdcData != 0 || machine.Memory.fdcCommand != 0x13 ||
		machine.Memory.fdcStatus != 0xe4 || machine.Memory.fdcRestoreInactivePolls != 9 ||
		machine.Memory.fdcSeekInactivePolls != 9 || !machine.Memory.fdcRestoreIRQObserved ||
		!machine.Memory.fdcSeekIRQObserved || machine.Memory.fdcRestorePending ||
		machine.Memory.fdcSeekPending || machine.Memory.fdcIRQ || machine.Memory.mfpGPIPIn&0x20 == 0 ||
		machine.Memory.fdcRestoreStartClock == 0 || machine.Memory.fdcStatusReadClock == 0 ||
		machine.Memory.fdcSeekStartClock == 0 || machine.Memory.fdcSeekStatusReadClock == 0 ||
		machine.nextFDCRestoreClock != 0 || machine.nextFDCSeekClock != 0 {
		t.Fatalf("drive-one FDC boundary instructions=%d interrupts=%d clocks=%d state=%+v drive/stage/data/command/status/restorePolls/seekPolls/restoreObserved/seekObserved/restorePending/seekPending/IRQ/GPIP/start/read/seekStart/seekRead/nextRestore/nextSeek=%d/%d/%02x/%02x/%02x/%d/%d/%v/%v/%v/%v/%v/%02x/%d/%d/%d/%d/%d/%d",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
			machine.Memory.fdcProbeDrive, machine.Memory.fdcInitStage, machine.Memory.fdcData,
			machine.Memory.fdcCommand, machine.Memory.fdcStatus,
			machine.Memory.fdcRestoreInactivePolls, machine.Memory.fdcSeekInactivePolls,
			machine.Memory.fdcRestoreIRQObserved, machine.Memory.fdcSeekIRQObserved,
			machine.Memory.fdcRestorePending, machine.Memory.fdcSeekPending, machine.Memory.fdcIRQ,
			machine.Memory.mfpGPIPIn, machine.Memory.fdcRestoreStartClock,
			machine.Memory.fdcStatusReadClock, machine.Memory.fdcSeekStartClock,
			machine.Memory.fdcSeekStatusReadClock, machine.nextFDCRestoreClock,
			machine.nextFDCSeekClock)
	}
	state = machine.CPU.State
	wantD = [8]uint32{0xffff00e4, 0x0091, 0x01b4, 0, 0x00080000, 0x00100000, 5, 1}
	wantA = [7]uint32{1, 2, 0x0000301e, 0, 0, 0x00fc01f4, 0x00000ffc}
	if machine.Instructions != 290970 || machine.Interrupts != 234 || machine.Clocks != 2997708 ||
		state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0f46 ||
		state.SR != 0x2310 || state.PC != 0x00fc38a0 ||
		state.Prefetch != [2]uint16{0x4e75, 0x2f0a} ||
		machine.Memory.fdcRestoreStartClock != 2992662 ||
		machine.Memory.fdcStatusReadClock != 2994002 ||
		machine.Memory.fdcSeekStartClock != 2996378 ||
		machine.Memory.fdcSeekStatusReadClock != 2997694 {
		t.Fatalf("drive-one FDC exact boundary instructions=%d interrupts=%d clocks=%d state=%+v restore=%d/%d seek=%d/%d",
			machine.Instructions, machine.Interrupts, machine.Clocks, state,
			machine.Memory.fdcRestoreStartClock, machine.Memory.fdcStatusReadClock,
			machine.Memory.fdcSeekStartClock, machine.Memory.fdcSeekStatusReadClock)
	}
	for steps := 0; steps < 512 && machine.Memory.dmaAddressWriteStage < 3; steps++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("DMA address instructions=%d interrupts=%d clocks=%d state=%+v stage/address=%d/%06x: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
				machine.Memory.dmaAddressWriteStage, machine.Memory.dmaAddress, err)
		}
	}
	if machine.Memory.dmaAddressWriteStage != 3 || machine.Memory.dmaAddress != 0x001004 {
		t.Fatalf("DMA address boundary instructions=%d interrupts=%d clocks=%d state=%+v stage/address=%d/%06x",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
			machine.Memory.dmaAddressWriteStage, machine.Memory.dmaAddress)
	}
	state = machine.CPU.State
	wantD = [8]uint32{9, 0, 0, 0, 0x1004, 0x00100000, 0, 1}
	wantA = [7]uint32{0x0eb6, 2, 0x0e9c, 0x00fcd074, 0x00fc1116, 0x0e9c, 0x0eba}
	if machine.Instructions != 291294 || machine.Interrupts != 234 || machine.Clocks != 3001576 ||
		state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0e30 ||
		state.SR != 0x2314 || state.PC != 0x00fc360e ||
		state.Prefetch != [2]uint16{0x4e75, 0x326f} {
		t.Fatalf("DMA address exact boundary instructions=%d interrupts=%d clocks=%d state=%+v",
			machine.Instructions, machine.Interrupts, machine.Clocks, state)
	}
	for steps := 0; steps < 256 && machine.Memory.dmaInitStage < 3; steps++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("DMA reset instructions=%d interrupts=%d clocks=%d state=%+v stage/mode/count/resets=%d/%04x/%d/%d: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
				machine.Memory.dmaInitStage, machine.Memory.dmaMode,
				machine.Memory.dmaSectorCount, machine.Memory.dmaResetCount, err)
		}
	}
	if machine.Memory.dmaInitStage != 3 || machine.Memory.dmaMode != 0x0090 ||
		machine.Memory.dmaSectorCount != 0 || machine.Memory.dmaResetCount != 2 {
		t.Fatalf("DMA reset boundary instructions=%d interrupts=%d clocks=%d state=%+v stage/mode/count/resets=%d/%04x/%d/%d",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
			machine.Memory.dmaInitStage, machine.Memory.dmaMode,
			machine.Memory.dmaSectorCount, machine.Memory.dmaResetCount)
	}
	state = machine.CPU.State
	wantD = [8]uint32{0xffffffff, 0xffffffff, 6, 0, 0x1004, 0x00100080, 0, 1}
	wantA = [7]uint32{0x0eb6, 0x0e69, 0x0e9c, 6, 0x00fc1116, 0x0e9c, 0x0eba}
	if machine.Instructions != 291376 || machine.Interrupts != 234 || machine.Clocks != 3002468 ||
		state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0e38 ||
		state.SR != 0x2314 || state.PC != 0x00fc1248 ||
		state.Prefetch != [2]uint16{0x0045, 0x0008} {
		t.Fatalf("DMA reset exact boundary instructions=%d interrupts=%d clocks=%d state=%+v",
			machine.Instructions, machine.Interrupts, machine.Clocks, state)
	}
	for steps := 0; steps < 128 && machine.Memory.acsiStage < 3; steps++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("ACSI target zero instructions=%d interrupts=%d clocks=%d state=%+v stage/target/command/mode=%d/%d/%02x/%04x: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
				machine.Memory.acsiStage, machine.Memory.acsiTarget,
				machine.Memory.acsiCommand, machine.Memory.dmaMode, err)
		}
	}
	if machine.Memory.acsiStage != 3 || machine.Memory.acsiTarget != 0 ||
		machine.Memory.acsiCommand != 0 || machine.Memory.dmaMode != 0x008a ||
		machine.Memory.mfpGPIPIn&0x20 == 0 || machine.Memory.fdcIRQ ||
		machine.Memory.dmaResetCount != 2 {
		t.Fatalf("ACSI target zero boundary instructions=%d interrupts=%d clocks=%d state=%+v stage/target/command/mode/GPIP/IRQ/resets=%d/%d/%02x/%04x/%02x/%v/%d",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
			machine.Memory.acsiStage, machine.Memory.acsiTarget, machine.Memory.acsiCommand,
			machine.Memory.dmaMode, machine.Memory.mfpGPIPIn, machine.Memory.fdcIRQ,
			machine.Memory.dmaResetCount)
	}
	state = machine.CPU.State
	wantD = [8]uint32{0x8a, 0xffffffff, 0x8a, 0, 0x1004, 0x0010008a, 5, 1}
	wantA = [7]uint32{0x0e63, 0x0e69, 0x0e9c, 0x0e64, 0x00fc6300, 0x0e63, 0x00fcd074}
	if machine.Instructions != 291404 || machine.Interrupts != 234 || machine.Clocks != 3002700 ||
		state.D != wantD || state.A != wantA || state.USP != 0 || state.SSP != 0x0e38 ||
		state.SR != 0x2300 || state.PC != 0x00fc1292 ||
		state.Prefetch != [2]uint16{0x4878, 0x0014} {
		t.Fatalf("ACSI target zero exact boundary instructions=%d interrupts=%d clocks=%d state=%+v",
			machine.Instructions, machine.Interrupts, machine.Clocks, state)
	}
	for steps := 0; steps < 1_000_000 && machine.Memory.acsiStage != 5; steps++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("ACSI scan instructions=%d interrupts=%d clocks=%d state=%+v stage/target/command/mode=%d/%d/%02x/%04x: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
				machine.Memory.acsiStage, machine.Memory.acsiTarget,
				machine.Memory.acsiCommand, machine.Memory.dmaMode, err)
		}
	}
	wantCommands := [8]byte{0x00, 0x20, 0x40, 0x60, 0x80, 0xa0, 0xc0, 0xe0}
	wantTimeoutClocks := [8]uint64{3771064, 4853396, 5976920, 7099340,
		8222868, 9345296, 10468836, 11591272}
	if machine.Memory.acsiStage != 5 || machine.Memory.acsiTarget != 7 ||
		machine.Memory.acsiAttemptMask != 0xff || machine.Memory.acsiCommandReceipts != wantCommands ||
		machine.Memory.acsiTimeoutReturnClocks != wantTimeoutClocks ||
		machine.Memory.acsiTimeoutReturnClock != 11591272 ||
		machine.Memory.dmaMode != 0x0080 || machine.Memory.dmaInitStage != 0 ||
		machine.Memory.fdcInitStage != 14 || machine.Memory.fdcIRQ ||
		machine.Memory.mfpGPIPIn&0x20 == 0 || machine.Instructions != 866723 ||
		machine.Interrupts != 461 || machine.Clocks != 11591284 ||
		machine.CPU.State.D != [8]uint32{1, 0xe0, 0x8a, 0, 0x1004, 0x0010008a, 5, 1} ||
		machine.CPU.State.A != [7]uint32{0xffff8604, 0x0e69, 0x0e9c, 0x0e64, 0x00fc6300, 0x0e63, 0x00fcd074} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x0e38 ||
		machine.CPU.State.SR != 0x2300 || machine.CPU.State.PC != 0x00fc12a8 ||
		machine.CPU.State.Prefetch != [2]uint16{0x33fc, 0x0000} {
		t.Fatalf("ACSI completion instructions=%d interrupts=%d clocks=%d state=%+v stage/target/mask/commands/mode/init/FDC/IRQ/GPIP=%d/%d/%02x/%v/%04x/%d/%d/%v/%02x",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
			machine.Memory.acsiStage, machine.Memory.acsiTarget, machine.Memory.acsiAttemptMask,
			machine.Memory.acsiCommandReceipts, machine.Memory.dmaMode, machine.Memory.dmaInitStage,
			machine.Memory.fdcInitStage, machine.Memory.fdcIRQ, machine.Memory.mfpGPIPIn)
	}
	var nextGate error
	for steps := 0; steps < 100_000 && machine.Memory.psgDriveStage != 9; steps++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("parallel strobe init instructions=%d interrupts=%d clocks=%d state=%+v stage/select/R14=%d/%02x/%02x: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
				machine.Memory.psgDriveStage, machine.Memory.psgRegisterSelect,
				machine.Memory.psgRegisters[14], err)
		}
	}
	if machine.Memory.psgDriveStage != 9 || machine.Memory.psgRegisterSelect != 14 ||
		machine.Memory.psgRegisters[14] != 0x23 || machine.Instructions != 867260 ||
		machine.Interrupts != 462 || machine.Clocks != 11598144 ||
		machine.CPU.State.D != [8]uint32{0x04000023, 0x2310, 3, 0, 0x00080000, 0x00100000, 5, 1} ||
		machine.CPU.State.A != [7]uint32{0xffff8800, 0x2b26, 0, 0, 0, 0x00fc01f4, 0x00000ffc} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x0f82 ||
		machine.CPU.State.SR != 0x2700 || machine.CPU.State.PC != 0x00fc6e6c ||
		machine.CPU.State.Prefetch != [2]uint16{0x40c0, 0x46c1} {
		t.Fatalf("parallel strobe completion instructions=%d interrupts=%d clocks=%d state=%+v stage/select/R14=%d/%02x/%02x",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
			machine.Memory.psgDriveStage, machine.Memory.psgRegisterSelect,
			machine.Memory.psgRegisters[14])
	}
	for steps := 0; steps < 100_000 && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
		if machine.Memory.ikbdACIATDR == 0x1c {
			break
		}
	}
	if nextGate != nil || machine.Memory.ikbdACIATDR != 0x1c ||
		!machine.Memory.ikbdACIATXPending || machine.Memory.ikbdACIAStatus&2 != 0 ||
		machine.Instructions != 867321 || machine.Interrupts != 462 ||
		machine.Clocks != 11599204 || machine.nextACIABitClock != 11599710 ||
		machine.CPU.State.D != [8]uint32{0xffffffff, 0x03e8, 0x1c, 0, 0x00080000, 0x00100000, 5, 1} ||
		machine.CPU.State.A != [7]uint32{0xffff8800, 0x2b26, 0x00fc5132, 0x00fc5142, 0, 0x00fc01f4, 0x00000ffc} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x0f50 ||
		machine.CPU.State.SR != 0x2310 || machine.CPU.State.PC != 0x00fc515c ||
		machine.CPU.State.Prefetch != [2]uint16{0x241f, 0x245f} {
		t.Fatalf("clock request write instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State, nextGate)
	}
	for steps := 0; steps < 100_000 && !machine.Memory.ikbdClockRequestDone; steps++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("clock request frame instructions=%d interrupts=%d clocks=%d state=%+v pending/ticks/next=%v/%d/%d: %v",
				machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
				machine.Memory.ikbdACIATXPending, machine.Memory.ikbdACIATXShiftTicks,
				machine.nextACIABitClock, err)
		}
	}
	if !machine.Memory.ikbdClockRequestDone || !machine.Memory.ikbdClockRequestHandled ||
		machine.ikbdClockRequestTXClock != 11609950 || machine.nextACIABitClock != 11610974 ||
		machine.Instructions != 868214 || machine.Interrupts != 462 ||
		machine.Clocks != 11609966 ||
		machine.CPU.State.D != [8]uint32{0x0270, 0xe0, 0, 0, 0x00080000, 0x00100000, 5, 1} ||
		machine.CPU.State.A != [7]uint32{0xffff8800, 0x2b26, 0, 0, 0, 0x00fc01f4, 0x00000ffc} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x0f74 ||
		machine.CPU.State.SR != 0x2300 || machine.CPU.State.PC != 0x00fc21ae ||
		machine.CPU.State.Prefetch != [2]uint16{0xb280, 0x6dee} {
		t.Fatalf("clock request did not complete instructions=%d interrupts=%d clocks=%d state=%+v",
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State)
	}
	for steps := 0; steps < 100_000 && !machine.Memory.ikbdClockResponseComplete && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || !machine.Memory.ikbdClockResponseComplete ||
		machine.Memory.ikbdClockResponseDelivered != 7 || machine.Memory.ikbdClockResponseReadCount != 7 ||
		machine.Memory.ikbdClockResponseReads != ikbdClockResponse ||
		machine.ikbdClockResponseDeliveryClocks != [7]uint64{11626334, 11636574, 11646814, 11657054, 11667294, 11677534, 11687774} ||
		machine.Memory.ikbdClockResponseReadClocks != [7]uint64{11626630, 11636858, 11647106, 11657338, 11667578, 11677826, 11688058} ||
		machine.Instructions != 874579 || machine.Interrupts != 471 || machine.Clocks != 11688070 ||
		machine.CPU.State.D != [8]uint32{1, 0x83, 0, 0, 0x00080000, 0x00100000, 5, 1} ||
		machine.CPU.State.A != [7]uint32{0x00fc0794, 0x2b26, 0, 0, 0, 0x00fc01f4, 0x00000ffc} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x0f48 ||
		machine.CPU.State.SR != 0x2600 || machine.CPU.State.PC != 0x00fc07ae ||
		machine.CPU.State.Prefetch != [2]uint16{0x4eba, 0x0018} || machine.nextIKBDClockResponseClock != 0 {
		t.Fatalf("clock response complete=%v delivered/read=%d/%d values=%v delivery-clocks=%v read-clocks=%v instructions=%d interrupts=%d clocks=%d state=%+v next=%d err=%v",
			machine.Memory.ikbdClockResponseComplete, machine.Memory.ikbdClockResponseDelivered,
			machine.Memory.ikbdClockResponseReadCount, machine.Memory.ikbdClockResponseReads,
			machine.ikbdClockResponseDeliveryClocks, machine.Memory.ikbdClockResponseReadClocks,
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
			machine.nextIKBDClockResponseClock, nextGate)
	}
	for steps := 0; steps < 100_000 && !machine.Memory.ikbdClockReadbackComplete && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || !machine.Memory.ikbdClockReadbackComplete ||
		!machine.Memory.ikbdClockReadbackRequestWritten || !machine.Memory.ikbdClockReadbackRequestDone ||
		!machine.Memory.ikbdClockReadbackRequestHandled || machine.ikbdClockReadbackRequestTXClock != 11773790 ||
		!machine.Memory.ikbdSetClockComplete || machine.Memory.ikbdSetClockCompleteCount != 7 ||
		machine.Memory.ikbdSetClockCompletionClocks != [7]uint64{11702110, 11712350, 11722590, 11732830, 11743070, 11753310, 11763550} ||
		machine.Memory.ikbdClockReadbackDelivered != 7 || machine.Memory.ikbdClockReadbackReadCount != 7 ||
		machine.Memory.ikbdClockReadbackReads != ikbdClockReadback ||
		machine.Memory.ikbdClockReadbackDeliveryClocks != [7]uint64{11790174, 11800414, 11810654, 11820894, 11831134, 11841374, 11851614} ||
		machine.Memory.ikbdClockReadbackReadClocks != [7]uint64{11790462, 11800706, 11810938, 11821178, 11831430, 11841658, 11851898} ||
		machine.Instructions != 889609 || machine.Interrupts != 483 || machine.Clocks != 11851910 ||
		machine.CPU.State.D != [8]uint32{0, 0x83, 0, 0, 0x00080000, 0x00100000, 5, 1} ||
		machine.CPU.State.A != [7]uint32{0x00fc0794, 0x23b2, 0, 0, 0, 0x00fc01f4, 0x00000ffc} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x0f10 ||
		machine.CPU.State.SR != 0x2604 || machine.CPU.State.PC != 0x00fc07ae ||
		machine.CPU.State.Prefetch != [2]uint16{0x4eba, 0x0018} || machine.nextIKBDClockResponseClock != 0 {
		t.Fatalf("readback complete=%v request written/done/handled=%v/%v/%v request-clock=%d set complete/count/clocks=%v/%d/%v delivered/read=%d/%d values=%v delivery-clocks=%v read-clocks=%v instructions=%d interrupts=%d clocks=%d state=%+v next=%d err=%v",
			machine.Memory.ikbdClockReadbackComplete, machine.Memory.ikbdClockReadbackRequestWritten,
			machine.Memory.ikbdClockReadbackRequestDone, machine.Memory.ikbdClockReadbackRequestHandled,
			machine.ikbdClockReadbackRequestTXClock, machine.Memory.ikbdSetClockComplete,
			machine.Memory.ikbdSetClockCompleteCount, machine.Memory.ikbdSetClockCompletionClocks,
			machine.Memory.ikbdClockReadbackDelivered, machine.Memory.ikbdClockReadbackReadCount,
			machine.Memory.ikbdClockReadbackReads, machine.Memory.ikbdClockReadbackDeliveryClocks,
			machine.Memory.ikbdClockReadbackReadClocks, machine.Instructions, machine.Interrupts,
			machine.Clocks, machine.CPU.State, machine.nextIKBDClockResponseClock, nextGate)
	}
	for steps := 0; steps < 200_000 && !machine.Memory.flopVBLMediaComplete && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || !machine.Memory.flopVBLMediaComplete || machine.Memory.flopVBLMediaStage != 8 ||
		machine.Memory.flopVBLStatusReadClock != 13036978 || machine.Memory.psgRegisters[14] != 0x23 ||
		machine.Memory.dmaMode != 0x0080 || machine.Memory.fdcInitStage != 14 ||
		machine.Memory.fdcStatus != 0xe4 || machine.Memory.fdcIRQ || machine.Memory.mfpGPIPIn != 0xb1 ||
		machine.Instructions != 1005296 || machine.Interrupts != 521 || machine.Clocks != 13037306 ||
		machine.CPU.State.D != [8]uint32{0xffff0025, 0x23, 0x2400, 0x20, 0x00fc0003, 0x00100003, 0, 1} ||
		machine.CPU.State.A != [7]uint32{0xffff8800, 0x3010, 0x00fc36b8, 0, 0x100, 0x00fc58e8, 0x00fcc8d8} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x0e02 ||
		machine.CPU.State.SR != 0x2700 || machine.CPU.State.PC != 0x00fc36e4 ||
		machine.CPU.State.Prefetch != [2]uint16{0x40c1, 0x46c2} {
		t.Fatalf("flopvbl complete=%v stage=%d status-clock=%d port=%02x DMA/FDC/status/IRQ/GPIP=%04x/%d/%02x/%v/%02x instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
			machine.Memory.flopVBLMediaComplete, machine.Memory.flopVBLMediaStage,
			machine.Memory.flopVBLStatusReadClock, machine.Memory.psgRegisters[14],
			machine.Memory.dmaMode, machine.Memory.fdcInitStage, machine.Memory.fdcStatus,
			machine.Memory.fdcIRQ, machine.Memory.mfpGPIPIn, machine.Instructions,
			machine.Interrupts, machine.Clocks, machine.CPU.State, nextGate)
	}
	for steps := 0; steps < 200_000 && machine.Memory.ikbdClockPollCompleteCount == 0 && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || machine.Memory.ikbdClockPollRequestCount != 1 ||
		machine.Memory.ikbdClockPollCompleteCount != 1 || machine.ikbdClockPollRequestTXClock != 13937502 ||
		machine.Memory.ikbdClockPollResponseDelivered != 7 || machine.Memory.ikbdClockPollResponseReadCount != 7 ||
		machine.Memory.ikbdClockPollResponseReads != ikbdClockReadback ||
		machine.Memory.ikbdClockPollDeliveryClocks != [7]uint64{13953886, 13964126, 13974366, 13984606, 13994846, 14005086, 14015326} ||
		machine.Instructions != 1092926 || machine.Interrupts != 558 || machine.Clocks != 14015626 ||
		machine.CPU.State.D != [8]uint32{0, 0x183, 0x0f, 0, 0, 1, 5, 1} ||
		machine.CPU.State.A != [7]uint32{0x00fc0794, 0x23b2, 0x00fc52e4, 0x00fc5916, 0x00fc5916, 0x00fc5276, 0x00000ffc} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x0ed6 ||
		machine.CPU.State.SR != 0x2604 || machine.CPU.State.PC != 0x00fc07ae ||
		machine.CPU.State.Prefetch != [2]uint16{0x4eba, 0x0018} || machine.nextIKBDClockResponseClock != 0 {
		t.Fatalf("clock poll complete requests=%d responses=%d tx=%d delivered/read=%d/%d values=%v delivery=%v read=%v instructions=%d interrupts=%d clocks=%d state=%+v next=%d err=%v",
			machine.Memory.ikbdClockPollRequestCount, machine.Memory.ikbdClockPollCompleteCount,
			machine.ikbdClockPollRequestTXClock, machine.Memory.ikbdClockPollResponseDelivered,
			machine.Memory.ikbdClockPollResponseReadCount, machine.Memory.ikbdClockPollResponseReads,
			machine.Memory.ikbdClockPollDeliveryClocks, machine.Memory.ikbdClockPollReadClocks,
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State,
			machine.nextIKBDClockResponseClock, nextGate)
	}
	for steps := 0; steps < 100_000 && machine.Memory.flopVBLMediaChecks < 2 && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || machine.Memory.flopVBLMediaChecks != 2 || machine.Memory.flopVBLMediaDrive != 1 ||
		machine.Memory.flopVBLMediaStage != 8 || machine.Memory.flopVBLStatusReadClock != 14319166 ||
		machine.Memory.psgRegisters[14] != 0x23 || machine.Instructions != 1120734 ||
		machine.Interrupts != 568 || machine.Clocks != 14319494 ||
		machine.CPU.State.D != [8]uint32{0xffff0023, 0x23, 0x2400, 0x20, 0x00fc0003, 3, 0, 0x11} ||
		machine.CPU.State.A != [7]uint32{0xffff8800, 0x3010, 0x00fc36b8, 2, 0x100, 0x00fc58e8, 0x00fcc8d8} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x0df6 ||
		machine.CPU.State.SR != 0x2700 || machine.CPU.State.PC != 0x00fc36e4 ||
		machine.CPU.State.Prefetch != [2]uint16{0x40c1, 0x46c2} {
		t.Fatalf("second flopvbl checks=%d drive=%d stage=%d status-clock=%d port=%02x instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
			machine.Memory.flopVBLMediaChecks, machine.Memory.flopVBLMediaDrive,
			machine.Memory.flopVBLMediaStage, machine.Memory.flopVBLStatusReadClock,
			machine.Memory.psgRegisters[14], machine.Instructions, machine.Interrupts,
			machine.Clocks, machine.CPU.State, nextGate)
	}
	for steps := 0; steps < 200_000 && machine.Memory.floppyReadStage < 15 && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || machine.Memory.floppyReadStage != 15 || machine.Memory.floppyMediaLegacy[0].Sector != 1 ||
		machine.Memory.dmaAddress != 0x001004 || machine.Memory.floppyMediaLegacy[0].DMAAddressStage != 3 ||
		machine.Memory.floppyMediaLegacy[0].DMAResetCount != 2 || machine.Memory.dmaSectorCount != 1 ||
		machine.Memory.floppyMediaLegacy[0].ReadCommand != 0x80 || machine.Memory.floppyMediaLegacy[0].ReadCommandClock != 106340810 ||
		machine.Memory.flopVBLMediaChecks != 73 || machine.Instructions != 1286164 ||
		machine.Interrupts != 1761 || machine.Clocks != 106340824 ||
		machine.CPU.State.D != [8]uint32{0xffffffff, 0x80, 0x12c, 0x1004, 0x00fc3a88, 0x00100000, 0x00fc37ea, 1} ||
		machine.CPU.State.A != [7]uint32{0x00fc37ea, 0x2f44, 0x00fc3720, 1, 0x1004, 0, 0x00fcccf0} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x0f2a ||
		machine.CPU.State.SR != 0x2310 || machine.CPU.State.PC != 0x00fc373a ||
		machine.CPU.State.Prefetch != [2]uint16{0x4e75, 0x2f0a} {
		t.Fatalf("sector-read setup stage=%d sector=%02x address=%06x address-stage=%d resets=%d DMA-count=%d command=%02x command-clock=%d checks=%d instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
			machine.Memory.floppyReadStage, machine.Memory.floppyMediaLegacy[0].Sector, machine.Memory.dmaAddress,
			machine.Memory.floppyMediaLegacy[0].DMAAddressStage, machine.Memory.floppyMediaLegacy[0].DMAResetCount,
			machine.Memory.dmaSectorCount, machine.Memory.floppyMediaLegacy[0].ReadCommand,
			machine.Memory.floppyMediaLegacy[0].ReadCommandClock, machine.Memory.flopVBLMediaChecks,
			machine.Instructions, machine.Interrupts, machine.Clocks, machine.CPU.State, nextGate)
	}
	for steps := 0; steps < 1_500_000 && machine.Memory.floppyReadStage < 17 && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || machine.Memory.floppyReadStage != 17 ||
		machine.Memory.floppyMediaLegacy[0].TimeoutSelectorClock != 118354092 ||
		machine.Memory.floppyMediaLegacy[0].ForceInterrupt != 0xd0 ||
		machine.Memory.floppyMediaLegacy[0].ForceInterruptClock != 118354530 ||
		machine.Memory.fdcCommand != 0xd0 || machine.Memory.fdcStatus != 0x80 ||
		machine.Memory.fdcStatusTypeI || machine.Memory.fdcIRQ || machine.Memory.mfpGPIPIn != 0xb1 ||
		machine.Instructions != 2370884 || machine.Interrupts != 2136 || machine.Clocks != 118354544 ||
		machine.CPU.State.D != [8]uint32{0xffffffff, 0x4bb, 0x12c, 0x1004, 0x00fc3a88, 0x00100000, 0x00fc37ea, 1} ||
		machine.CPU.State.A != [7]uint32{0x00fc37ea, 0x2f44, 0x00fc3720, 1, 0x1004, 0, 0x00fcccf0} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x0f2a ||
		machine.CPU.State.SR != 0x2310 || machine.CPU.State.PC != 0x00fc373a ||
		machine.CPU.State.Prefetch != [2]uint16{0x4e75, 0x2f0a} {
		t.Fatalf("force-interrupt complete stage=%d selector-clock=%d force=%02x force-clock=%d FDC/status/type/IRQ/GPIP=%02x/%02x/%v/%v/%02x instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
			machine.Memory.floppyReadStage, machine.Memory.floppyMediaLegacy[0].TimeoutSelectorClock,
			machine.Memory.floppyMediaLegacy[0].ForceInterrupt, machine.Memory.floppyMediaLegacy[0].ForceInterruptClock,
			machine.Memory.fdcCommand, machine.Memory.fdcStatus, machine.Memory.fdcStatusTypeI,
			machine.Memory.fdcIRQ, machine.Memory.mfpGPIPIn, machine.Instructions,
			machine.Interrupts, machine.Clocks, machine.CPU.State, nextGate)
	}
	for steps := 0; steps < 10_000 && machine.Memory.floppyReadStage < 24 && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || machine.Memory.floppyReadStage != 24 ||
		machine.Memory.floppyMediaLegacy[0].SeekData != 0 || machine.Memory.floppyMediaLegacy[0].SeekCommand != 0x13 ||
		machine.Memory.floppyMediaLegacy[0].SeekStartClock != 118356450 ||
		machine.Memory.floppyMediaLegacy[0].InactivePolls != 9 || !machine.Memory.floppyMediaLegacy[0].IRQObserved ||
		machine.Memory.floppyMediaLegacy[0].StatusReadClock != 118357766 ||
		machine.Memory.fdcCommand != 0x13 || machine.Memory.fdcStatus != 0xe4 ||
		!machine.Memory.fdcStatusTypeI || machine.Memory.fdcIRQ || machine.Memory.mfpGPIPIn != 0xb1 ||
		machine.Instructions != 2371204 || machine.Interrupts != 2136 || machine.Clocks != 118357780 ||
		machine.CPU.State.D != [8]uint32{0xffff00e4, 0x491, 0, 0x1004, 0x00fcfffe, 0x00100000, 0x00fc37ea, 1} ||
		machine.CPU.State.A != [7]uint32{0, 0, 0x3008, 1, 0x1004, 0, 0x00fcccf0} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x0f2a ||
		machine.CPU.State.SR != 0x2310 || machine.CPU.State.PC != 0x00fc38a0 ||
		machine.CPU.State.Prefetch != [2]uint16{0x4e75, 0x2f0a} {
		t.Fatalf("retry seek complete stage=%d data=%02x command=%02x start=%d polls=%d observed=%v status-read=%d FDC/status/type/IRQ/GPIP=%02x/%02x/%v/%v/%02x instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
			machine.Memory.floppyReadStage, machine.Memory.floppyMediaLegacy[0].SeekData,
			machine.Memory.floppyMediaLegacy[0].SeekCommand, machine.Memory.floppyMediaLegacy[0].SeekStartClock,
			machine.Memory.floppyMediaLegacy[0].InactivePolls, machine.Memory.floppyMediaLegacy[0].IRQObserved,
			machine.Memory.floppyMediaLegacy[0].StatusReadClock, machine.Memory.fdcCommand,
			machine.Memory.fdcStatus, machine.Memory.fdcStatusTypeI, machine.Memory.fdcIRQ,
			machine.Memory.mfpGPIPIn, machine.Instructions, machine.Interrupts, machine.Clocks,
			machine.CPU.State, nextGate)
	}
	for steps := 0; steps < 10_000 && machine.Memory.floppyReadStage < 27 && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || machine.Memory.floppyReadStage != 27 ||
		machine.Memory.floppyMediaLegacy[1].DrivePort != 0x25 ||
		machine.Memory.floppyMediaLegacy[1].DriveWriteClock != 118369158 ||
		machine.Memory.psgRegisters[14] != 0x25 || machine.Memory.flopVBLMediaChecks != 73 ||
		machine.Instructions != 2371990 || machine.Interrupts != 2136 || machine.Clocks != 118369170 ||
		machine.CPU.State.D != [8]uint32{0x25, 0x25, 0x2300, 0x1020, 0, 0, 0, 0x0a} ||
		machine.CPU.State.A != [7]uint32{0xffff8800, 0x2f44, 6, 1, 0x1200, 5, 0x00fc413a} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x6880 ||
		machine.CPU.State.SR != 0x2700 || machine.CPU.State.PC != 0x00fc36e4 ||
		machine.CPU.State.Prefetch != [2]uint16{0x40c1, 0x46c2} {
		t.Fatalf("retry drive reselect stage=%d port/clock=%02x/%d R14=%02x checks=%d instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
			machine.Memory.floppyReadStage, machine.Memory.floppyMediaLegacy[1].DrivePort,
			machine.Memory.floppyMediaLegacy[1].DriveWriteClock, machine.Memory.psgRegisters[14],
			machine.Memory.flopVBLMediaChecks, machine.Instructions, machine.Interrupts,
			machine.Clocks, machine.CPU.State, nextGate)
	}
	for steps := 0; steps < 1_000 && machine.Memory.floppyReadStage < 37 && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || machine.Memory.floppyReadStage != 37 ||
		machine.Memory.floppyMediaLegacy[1].Sector != 1 || machine.Memory.dmaAddress != 0x001004 ||
		machine.Memory.floppyMediaLegacy[1].DMAAddressStage != 3 ||
		machine.Memory.floppyMediaLegacy[1].DMAResetCount != 2 || machine.Memory.dmaSectorCount != 1 ||
		machine.Memory.floppyMediaLegacy[1].ReadCommand != 0x80 ||
		machine.Memory.floppyMediaLegacy[1].ReadCommandClock != 118371398 ||
		machine.Memory.fdcCommand != 0x80 || machine.Memory.fdcStatus != 0x81 ||
		machine.Memory.fdcStatusTypeI || machine.Memory.fdcIRQ || machine.Memory.mfpGPIPIn != 0xb1 ||
		machine.Memory.flopVBLMediaChecks != 73 ||
		!bytes.Equal(machine.Memory.ram[0x1004:0x1204], make([]byte, 0x200)) ||
		machine.Instructions != 2372203 || machine.Interrupts != 2136 || machine.Clocks != 118371412 ||
		machine.CPU.State.D != [8]uint32{0xffffffff, 0x80, 0x12c, 0x1004, 0x00fc3a88, 0, 0x00fc37ea, 1} ||
		machine.CPU.State.A != [7]uint32{0x00fc37ea, 0x2f44, 0x00fc3720, 1, 0x1004, 0, 0x00fcccf0} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x687c ||
		machine.CPU.State.SR != 0x2310 || machine.CPU.State.PC != 0x00fc373a ||
		machine.CPU.State.Prefetch != [2]uint16{0x4e75, 0x2f0a} {
		t.Fatalf("retry setup stage=%d sector=%02x address=%06x address-stage=%d resets=%d count=%d command/clock=%02x/%d FDC/status/type/IRQ/GPIP=%02x/%02x/%v/%v/%02x checks=%d instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
			machine.Memory.floppyReadStage, machine.Memory.floppyMediaLegacy[1].Sector, machine.Memory.dmaAddress,
			machine.Memory.floppyMediaLegacy[1].DMAAddressStage, machine.Memory.floppyMediaLegacy[1].DMAResetCount,
			machine.Memory.dmaSectorCount, machine.Memory.floppyMediaLegacy[1].ReadCommand,
			machine.Memory.floppyMediaLegacy[1].ReadCommandClock, machine.Memory.fdcCommand, machine.Memory.fdcStatus,
			machine.Memory.fdcStatusTypeI, machine.Memory.fdcIRQ, machine.Memory.mfpGPIPIn,
			machine.Memory.flopVBLMediaChecks, machine.Instructions, machine.Interrupts, machine.Clocks,
			machine.CPU.State, nextGate)
	}
	for steps := 0; steps < 1_500_000 && machine.Memory.floppyReadStage < 39 && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || machine.Memory.floppyReadStage != 39 ||
		machine.Memory.floppyMediaLegacy[1].TimeoutSelectorClock != 130385964 ||
		machine.Memory.floppyMediaLegacy[1].ForceInterrupt != 0xd0 ||
		machine.Memory.floppyMediaLegacy[1].ForceInterruptClock != 130386402 ||
		machine.Memory.fdcCommand != 0xd0 || machine.Memory.fdcStatus != 0x80 ||
		machine.Memory.fdcStatusTypeI || machine.Memory.fdcIRQ || machine.Memory.mfpGPIPIn != 0xb1 ||
		machine.Memory.floppyMediaLegacy[0].TimeoutSelectorClock != 118354092 ||
		machine.Memory.floppyMediaLegacy[0].ForceInterruptClock != 118354530 ||
		machine.Instructions != 3457037 || machine.Interrupts != 2511 || machine.Clocks != 130386416 ||
		machine.CPU.State.D != [8]uint32{0xffffffff, 0x5e7, 0x12c, 0x1004, 0x00fc3a88, 0, 0x00fc37ea, 1} ||
		machine.CPU.State.A != [7]uint32{0x00fc37ea, 0x2f44, 0x00fc3720, 1, 0x1004, 0, 0x00fcccf0} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x687c ||
		machine.CPU.State.SR != 0x2310 || machine.CPU.State.PC != 0x00fc373a ||
		machine.CPU.State.Prefetch != [2]uint16{0x4e75, 0x2f0a} {
		t.Fatalf("retry timeout stage=%d selector-clock=%d force/clock=%02x/%d FDC/status/type/IRQ/GPIP=%02x/%02x/%v/%v/%02x instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
			machine.Memory.floppyReadStage, machine.Memory.floppyMediaLegacy[1].TimeoutSelectorClock,
			machine.Memory.floppyMediaLegacy[1].ForceInterrupt, machine.Memory.floppyMediaLegacy[1].ForceInterruptClock,
			machine.Memory.fdcCommand, machine.Memory.fdcStatus, machine.Memory.fdcStatusTypeI,
			machine.Memory.fdcIRQ, machine.Memory.mfpGPIPIn, machine.Instructions, machine.Interrupts,
			machine.Clocks, machine.CPU.State, nextGate)
	}
	for steps := 0; steps < 10_000 && machine.Memory.floppyReadStage < 46 && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || machine.Memory.floppyReadStage != 46 || machine.Memory.floppyMediaLegacy[1].SeekData != 0 ||
		machine.Memory.floppyMediaLegacy[1].SeekCommand != 0x13 ||
		machine.Memory.floppyMediaLegacy[1].SeekStartClock != 130388322 ||
		machine.Memory.floppyMediaLegacy[1].InactivePolls != 9 || !machine.Memory.floppyMediaLegacy[1].IRQObserved ||
		machine.Memory.floppyMediaLegacy[1].StatusReadClock != 130389638 ||
		machine.Memory.fdcCommand != 0x13 || machine.Memory.fdcStatus != 0xe4 ||
		!machine.Memory.fdcStatusTypeI || machine.Memory.fdcIRQ || machine.Memory.mfpGPIPIn != 0xb1 ||
		machine.Memory.floppyMediaLegacy[0].SeekStartClock != 118356450 ||
		machine.Memory.floppyMediaLegacy[0].StatusReadClock != 118357766 ||
		machine.Instructions != 3457357 || machine.Interrupts != 2511 || machine.Clocks != 130389652 ||
		machine.CPU.State.D != [8]uint32{0xffff00e4, 0x591, 0, 0x1004, 0x00fcfffe, 0, 0x00fc37ea, 1} ||
		machine.CPU.State.A != [7]uint32{0, 0, 0x3008, 1, 0x1004, 0, 0x00fcccf0} ||
		machine.CPU.State.USP != 0 || machine.CPU.State.SSP != 0x687c ||
		machine.CPU.State.SR != 0x2310 || machine.CPU.State.PC != 0x00fc38a0 ||
		machine.CPU.State.Prefetch != [2]uint16{0x4e75, 0x2f0a} {
		t.Fatalf("second dummy seek stage=%d data=%02x command/start=%02x/%d polls=%d observed=%v status-read=%d FDC/status/type/IRQ/GPIP=%02x/%02x/%v/%v/%02x instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
			machine.Memory.floppyReadStage, machine.Memory.floppyMediaLegacy[1].SeekData,
			machine.Memory.floppyMediaLegacy[1].SeekCommand, machine.Memory.floppyMediaLegacy[1].SeekStartClock,
			machine.Memory.floppyMediaLegacy[1].InactivePolls, machine.Memory.floppyMediaLegacy[1].IRQObserved,
			machine.Memory.floppyMediaLegacy[1].StatusReadClock, machine.Memory.fdcCommand,
			machine.Memory.fdcStatus, machine.Memory.fdcStatusTypeI, machine.Memory.fdcIRQ,
			machine.Memory.mfpGPIPIn, machine.Instructions, machine.Interrupts, machine.Clocks,
			machine.CPU.State, nextGate)
	}
	for steps := 0; steps < 100_000 && machine.Memory.floppyReadStage < 59 && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || machine.Memory.floppyReadStage != 59 ||
		machine.Memory.floppyMediaLegacy[2].DrivePort != 0x25 ||
		machine.Memory.floppyMediaLegacy[2].DriveWriteClock != 130971538 ||
		machine.Memory.floppyMediaLegacy[2].Sector != 1 || machine.Memory.floppyMediaLegacy[2].DMAAddressStage != 3 ||
		machine.Memory.floppyMediaLegacy[2].DMAResetCount != 2 || machine.Memory.dmaSectorCount != 1 ||
		machine.Memory.floppyMediaLegacy[2].ReadCommand != 0x80 || machine.Memory.floppyMediaLegacy[2].ReadCommandClock != 130973778 ||
		machine.Memory.fdcCommand != 0x80 || machine.Memory.fdcStatus != 0x81 ||
		machine.Memory.fdcStatusTypeI || machine.Memory.fdcIRQ || machine.Memory.mfpGPIPIn != 0xb1 ||
		machine.Memory.psgRegisters[14] != 0x25 || machine.Memory.flopVBLMediaChecks != 73 ||
		machine.Instructions != 3516426 || machine.Interrupts != 2528 || machine.Clocks != 130973792 ||
		machine.CPU.State.D != [8]uint32{0xffffffff, 0x80, 0x12c, 0x1004, 0x00fc3a88, 0, 0x00fc37ea, 1} ||
		machine.CPU.State.A != [7]uint32{0x00fc37ea, 0x2f44, 0x00fc3720, 1, 0x1004, 0, 0x00fcccf0} ||
		machine.CPU.State.USP != 0x7d22 || machine.CPU.State.SSP != 0x68a2 ||
		machine.CPU.State.SR != 0x2310 || machine.CPU.State.PC != 0x00fc373a ||
		machine.CPU.State.Prefetch != [2]uint16{0x4e75, 0x2f0a} {
		t.Fatalf("third retry stage=%d drive/clock=%02x/%d sector=%d address-stage=%d resets=%d count=%d command/clock=%02x/%d FDC/status/type/IRQ/GPIP=%02x/%02x/%v/%v/%02x R14/checks=%02x/%d instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
			machine.Memory.floppyReadStage, machine.Memory.floppyMediaLegacy[2].DrivePort,
			machine.Memory.floppyMediaLegacy[2].DriveWriteClock, machine.Memory.floppyMediaLegacy[2].Sector,
			machine.Memory.floppyMediaLegacy[2].DMAAddressStage, machine.Memory.floppyMediaLegacy[2].DMAResetCount,
			machine.Memory.dmaSectorCount, machine.Memory.floppyMediaLegacy[2].ReadCommand,
			machine.Memory.floppyMediaLegacy[2].ReadCommandClock, machine.Memory.fdcCommand, machine.Memory.fdcStatus,
			machine.Memory.fdcStatusTypeI, machine.Memory.fdcIRQ, machine.Memory.mfpGPIPIn,
			machine.Memory.psgRegisters[14], machine.Memory.flopVBLMediaChecks, machine.Instructions,
			machine.Interrupts, machine.Clocks, machine.CPU.State, nextGate)
	}
	for steps := 0; steps < 1_500_000 && machine.Memory.floppyReadStage < 61 && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || machine.Memory.floppyReadStage != 61 ||
		machine.Memory.floppyMediaLegacy[2].TimeoutSelectorClock != 142979300 ||
		machine.Memory.floppyMediaLegacy[2].ForceInterrupt != 0xd0 ||
		machine.Memory.floppyMediaLegacy[2].ForceInterruptClock != 142979738 ||
		machine.Memory.fdcCommand != 0xd0 || machine.Memory.fdcStatus != 0x80 ||
		machine.Memory.fdcStatusTypeI || machine.Memory.fdcIRQ || machine.Memory.mfpGPIPIn != 0xb1 ||
		machine.Instructions != 4600435 || machine.Interrupts != 2903 || machine.Clocks != 142979752 ||
		machine.CPU.State.D != [8]uint32{0xffffffff, 0x721, 0x12c, 0x1004, 0x00fc3a88, 0, 0x00fc37ea, 1} ||
		machine.CPU.State.A != [7]uint32{0x00fc37ea, 0x2f44, 0x00fc3720, 1, 0x1004, 0, 0x00fcccf0} ||
		machine.CPU.State.USP != 0x7d22 || machine.CPU.State.SSP != 0x68a2 ||
		machine.CPU.State.SR != 0x2310 || machine.CPU.State.PC != 0x00fc373a ||
		machine.CPU.State.Prefetch != [2]uint16{0x4e75, 0x2f0a} {
		t.Fatalf("third timeout stage=%d selector-clock=%d force/clock=%02x/%d FDC/status/type/IRQ/GPIP=%02x/%02x/%v/%v/%02x instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
			machine.Memory.floppyReadStage, machine.Memory.floppyMediaLegacy[2].TimeoutSelectorClock,
			machine.Memory.floppyMediaLegacy[2].ForceInterrupt, machine.Memory.floppyMediaLegacy[2].ForceInterruptClock,
			machine.Memory.fdcCommand, machine.Memory.fdcStatus, machine.Memory.fdcStatusTypeI,
			machine.Memory.fdcIRQ, machine.Memory.mfpGPIPIn, machine.Instructions, machine.Interrupts,
			machine.Clocks, machine.CPU.State, nextGate)
	}
	for steps := 0; steps < 10_000 && machine.Memory.floppyReadStage < 68 && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	if nextGate != nil || machine.Memory.floppyReadStage != 68 || machine.Memory.floppyMediaLegacy[2].SeekData != 0 ||
		machine.Memory.floppyMediaLegacy[2].SeekCommand != 0x13 ||
		machine.Memory.floppyMediaLegacy[2].SeekStartClock != 142981658 ||
		machine.Memory.floppyMediaLegacy[2].InactivePolls != 9 || !machine.Memory.floppyMediaLegacy[2].IRQObserved ||
		machine.Memory.floppyMediaLegacy[2].StatusReadClock != 142982974 ||
		machine.Memory.fdcCommand != 0x13 || machine.Memory.fdcStatus != 0xe4 ||
		!machine.Memory.fdcStatusTypeI || machine.Memory.fdcIRQ || machine.Memory.mfpGPIPIn != 0xb1 ||
		machine.Instructions != 4600755 || machine.Interrupts != 2903 || machine.Clocks != 142982988 ||
		machine.CPU.State.D != [8]uint32{0xffff00e4, 0x791, 0, 0x1004, 0x00fcfffe, 0, 0x00fc37ea, 1} ||
		machine.CPU.State.A != [7]uint32{0, 0, 0x3008, 1, 0x1004, 0, 0x00fcccf0} ||
		machine.CPU.State.USP != 0x7d22 || machine.CPU.State.SSP != 0x68a2 ||
		machine.CPU.State.SR != 0x2310 || machine.CPU.State.PC != 0x00fc38a0 ||
		machine.CPU.State.Prefetch != [2]uint16{0x4e75, 0x2f0a} {
		t.Fatalf("third dummy stage=%d data=%02x command/start=%02x/%d polls=%d observed=%v status-read=%d FDC/status/type/IRQ/GPIP=%02x/%02x/%v/%v/%02x instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
			machine.Memory.floppyReadStage, machine.Memory.floppyMediaLegacy[2].SeekData,
			machine.Memory.floppyMediaLegacy[2].SeekCommand, machine.Memory.floppyMediaLegacy[2].SeekStartClock,
			machine.Memory.floppyMediaLegacy[2].InactivePolls, machine.Memory.floppyMediaLegacy[2].IRQObserved,
			machine.Memory.floppyMediaLegacy[2].StatusReadClock, machine.Memory.fdcCommand,
			machine.Memory.fdcStatus, machine.Memory.fdcStatusTypeI, machine.Memory.fdcIRQ,
			machine.Memory.mfpGPIPIn, machine.Instructions, machine.Interrupts, machine.Clocks,
			machine.CPU.State, nextGate)
	}
	for attempt := uint64(1); attempt <= 3; attempt++ {
		receipt, ok := machine.Memory.floppyMediaReceipts.attempt(attempt)
		receipt.Attempt = 0
		if !ok || receipt != machine.Memory.floppyMediaLegacy[attempt-1] {
			t.Fatalf("legacy migration attempt=%d receipt=%+v legacy=%+v present=%v",
				attempt, receipt, machine.Memory.floppyMediaLegacy[attempt-1], ok)
		}
	}
	for steps := 0; steps < 5_000_000 && machine.Memory.floppyMediaReceipts.Total < 6 && nextGate == nil; steps++ {
		_, nextGate = machine.Step()
	}
	last, ok := machine.Memory.floppyMediaReceipts.attempt(5)
	if nextGate == nil || nextGate.Error() != "st: write 1-byte bus fault at 0xfffc02 fc=5: unsupported_device_state" ||
		machine.Memory.floppyMediaPhase != floppyMediaIdle ||
		machine.Memory.floppyMediaReceipts.Total != 5 || machine.Memory.floppyMediaReceipts.Count != 5 ||
		machine.Memory.floppyMediaReceipts.Next != 5 || !ok || last.DrivePort != 0x25 ||
		last.DriveWriteClock != 155070654 || last.Sector != 1 || last.DMAAddressStage != 3 ||
		last.DMAResetCount != 2 || last.ReadCommand != 0x80 || last.ReadCommandClock != 155072894 ||
		last.TimeoutSelectorClock != 167083840 || last.ForceInterrupt != 0xd0 ||
		last.ForceInterruptClock != 167084278 || last.SeekCommand != 0x13 ||
		last.SeekStartClock != 167086198 || last.InactivePolls != 9 || !last.IRQObserved ||
		last.StatusReadClock != 167087514 || machine.Instructions != 6779282 ||
		machine.Interrupts != 3656 || machine.Clocks != 167143396 {
		t.Fatalf("recurring gate phase=%d total/count/next=%d/%d/%d current=%+v last=%+v present=%v instructions=%d interrupts=%d clocks=%d state=%+v err=%v",
			machine.Memory.floppyMediaPhase, machine.Memory.floppyMediaReceipts.Total,
			machine.Memory.floppyMediaReceipts.Count, machine.Memory.floppyMediaReceipts.Next,
			machine.Memory.floppyMediaCurrent, last, ok, machine.Instructions,
			machine.Interrupts, machine.Clocks, machine.CPU.State, nextGate)
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
	memory.ikbdACIATXShift = 1
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

func TestMachineSchedulesIKBDClockResponseAfterRequestFrame(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.ikbdACIAConfigured = true
	memory.ikbdACIAStatus = 2
	memory.ikbdACIATDR = 0x1c
	memory.ikbdACIATXShift = 0x1c
	memory.ikbdACIATXShiftTicks = 1
	machine := &Machine{Memory: memory, Clocks: 1000, aciaClockStarted: true, nextACIABitClock: 1000}
	machine.advanceClockedDevices()
	if machine.ikbdClockRequestTXClock != 1000 || machine.nextIKBDClockResponseClock != 17384 {
		t.Fatalf("request/response clocks=%d/%d", machine.ikbdClockRequestTXClock,
			machine.nextIKBDClockResponseClock)
	}
	machine.Clocks = 17383
	machine.advanceClockedDevices()
	if memory.ikbdClockResponseDelivered != 0 {
		t.Fatal("clock response arrived before deadline")
	}
	machine.Clocks = 17384
	machine.advanceClockedDevices()
	if memory.ikbdClockResponseDelivered != 1 || memory.ikbdACIARDR != 0xfc ||
		machine.ikbdClockResponseDeliveryClocks[0] != 17384 ||
		machine.nextIKBDClockResponseClock != 27624 {
		t.Fatalf("delivery/RDR/clocks/next=%d/%02x/%v/%d",
			memory.ikbdClockResponseDelivered, memory.ikbdACIARDR,
			machine.ikbdClockResponseDeliveryClocks, machine.nextIKBDClockResponseClock)
	}
}

func TestMachineSchedulesIKBDClockReadbackAfterBufferedRequest(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.ikbdACIAConfigured = true
	memory.ikbdACIAStatus = 2
	memory.ikbdClockRequestHandled = true
	memory.ikbdSetClockComplete = true
	memory.ikbdClockReadbackRequestWritten = true
	memory.ikbdACIATXShift = 0x1c
	memory.ikbdACIATXShiftTicks = 1
	machine := &Machine{Memory: memory, Clocks: 1000, aciaClockStarted: true, nextACIABitClock: 1000}
	machine.advanceClockedDevices()
	if !memory.ikbdClockReadbackRequestDone || !memory.ikbdClockReadbackRequestHandled ||
		machine.ikbdClockReadbackRequestTXClock != 1000 || machine.ikbdClockResponseRound != 2 ||
		machine.nextIKBDClockResponseClock != 17384 {
		t.Fatalf("readback done/handled/request/round/response=%v/%v/%d/%d/%d",
			memory.ikbdClockReadbackRequestDone, memory.ikbdClockReadbackRequestHandled,
			machine.ikbdClockReadbackRequestTXClock, machine.ikbdClockResponseRound,
			machine.nextIKBDClockResponseClock)
	}
}

func TestMachineSchedulesRecurringIKBDClockPoll(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.ikbdACIAConfigured = true
	memory.ikbdACIAStatus = 2
	memory.ikbdClockRequestHandled = true
	memory.ikbdSetClockComplete = true
	memory.ikbdClockReadbackRequestHandled = true
	memory.ikbdClockReadbackComplete = true
	memory.flopVBLMediaComplete = true
	memory.ikbdClockPollRequestWritten = true
	memory.ikbdACIATXShift = 0x1c
	memory.ikbdACIATXShiftTicks = 1
	machine := &Machine{Memory: memory, Clocks: 1000, aciaClockStarted: true, nextACIABitClock: 1000}
	machine.advanceClockedDevices()
	if memory.ikbdClockPollRequestCount != 1 || machine.ikbdClockPollScheduledCount != 1 ||
		machine.ikbdClockPollRequestTXClock != 1000 || machine.ikbdClockResponseRound != 3 ||
		machine.nextIKBDClockResponseClock != 17384 {
		t.Fatalf("poll completion/scheduled/tx/round/response=%d/%d/%d/%d/%d",
			memory.ikbdClockPollRequestCount, machine.ikbdClockPollScheduledCount,
			machine.ikbdClockPollRequestTXClock, machine.ikbdClockResponseRound,
			machine.nextIKBDClockResponseClock)
	}
	machine.Clocks = 17384
	machine.advanceClockedDevices()
	if memory.ikbdClockPollResponseDelivered != 1 || memory.ikbdACIARDR != 0xfc ||
		memory.ikbdClockPollDeliveryClocks[0] != 17384 || machine.nextIKBDClockResponseClock != 27624 {
		t.Fatalf("poll delivery/RDR/clocks/next=%d/%02x/%v/%d",
			memory.ikbdClockPollResponseDelivered, memory.ikbdACIARDR,
			memory.ikbdClockPollDeliveryClocks, machine.nextIKBDClockResponseClock)
	}
}

func TestMachineCompletesFDCRestoreAtDeadline(t *testing.T) {
	rom := testROM()
	copy(rom[:8], []byte{0x00, 0x00, 0x10, 0x00, 0x00, 0xfc, 0x00, 0x00})
	memory, err := NewMemory(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	memory.fdcInitStage = 2
	memory.dmaMode = 0x0080
	memory.fdcCommand = 0xd0
	memory.fdcStatus = 0x80
	memory.fdcStatusTypeI = true
	if err := memory.WriteWord(STDiskController, 0x000b, 5); err == nil ||
		memory.fdcInitStage != 2 || memory.fdcCommand != 0xd0 {
		t.Fatal("restore before second DMA mode unexpectedly accepted")
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0080,
		m68k.BusAccess{Clock: 900, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.fdcInitStage != 3 {
		t.Fatalf("second mode wait/err/stage=%d/%v/%d", wait, err, memory.fdcInitStage)
	}
	if err := memory.WriteWord(STDiskController, 0x000a, 5); err == nil ||
		memory.fdcInitStage != 3 || memory.fdcRestorePending {
		t.Fatal("wrong restore command unexpectedly accepted")
	}
	if wait, err := memory.WriteWordAt(STDiskController, 0x000b,
		m68k.BusAccess{Clock: 1000, FunctionCode: 5}); err != nil || wait != 4 {
		t.Fatalf("restore wait/err=%d/%v", wait, err)
	}
	if memory.fdcInitStage != 4 || memory.fdcCommand != 0x0b || memory.fdcStatus != 0x81 ||
		!memory.fdcStatusTypeI || memory.fdcIRQ || memory.mfpGPIPIn&0x20 == 0 ||
		!memory.fdcRestorePending || memory.fdcRestoreStartClock != 1000 {
		t.Fatalf("restore start stage/command/status/typeI/IRQ/GPIP/pending/start=%d/%02x/%02x/%v/%v/%02x/%v/%d",
			memory.fdcInitStage, memory.fdcCommand, memory.fdcStatus, memory.fdcStatusTypeI,
			memory.fdcIRQ, memory.mfpGPIPIn, memory.fdcRestorePending, memory.fdcRestoreStartClock)
	}
	machine := &Machine{Memory: memory, Clocks: 1728}
	machine.CPU.Bus = memory
	machine.advanceClockedDevices()
	if !machine.fdcRestoreClockStarted || machine.nextFDCRestoreClock != 1729 ||
		!memory.fdcRestorePending || memory.fdcStatus != 0x81 || memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("early scheduler/next/pending/status/GPIP=%v/%d/%v/%02x/%02x",
			machine.fdcRestoreClockStarted, machine.nextFDCRestoreClock,
			memory.fdcRestorePending, memory.fdcStatus, memory.mfpGPIPIn)
	}
	if got, err := memory.ReadByteFC(MFPGPIP, 5); err != nil || got&0x20 == 0 ||
		memory.fdcRestoreInactivePolls != 1 || !memory.fdcRestorePending {
		t.Fatalf("inactive poll=%02x err=%v polls=%d pending=%v", got, err,
			memory.fdcRestoreInactivePolls, memory.fdcRestorePending)
	}
	machine.Clocks = 1729
	machine.advanceClockedDevices()
	if machine.fdcRestoreClockStarted || machine.nextFDCRestoreClock != 0 ||
		memory.fdcRestorePending || memory.fdcInitStage != 5 || memory.fdcStatus != 0x84 ||
		!memory.fdcIRQ || memory.mfpGPIPIn&0x20 != 0 {
		t.Fatalf("complete scheduler/next/pending/stage/status/IRQ/GPIP=%v/%d/%v/%d/%02x/%v/%02x",
			machine.fdcRestoreClockStarted, machine.nextFDCRestoreClock, memory.fdcRestorePending,
			memory.fdcInitStage, memory.fdcStatus, memory.fdcIRQ, memory.mfpGPIPIn)
	}
	if got, err := memory.ReadByteFC(MFPGPIP, 5); err != nil || got&0x20 != 0 ||
		!memory.fdcRestoreIRQObserved {
		t.Fatalf("active poll=%02x err=%v observed=%v", got, err, memory.fdcRestoreIRQObserved)
	}
	machine.fdcRestoreClockStarted = true
	machine.nextFDCRestoreClock = 2000
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	if machine.fdcRestoreClockStarted || machine.nextFDCRestoreClock != 0 ||
		memory.fdcRestorePending || memory.fdcRestoreStartClock != 0 ||
		memory.fdcRestoreInactivePolls != 0 || memory.fdcRestoreIRQObserved {
		t.Fatalf("reset scheduler/next/pending/start/polls/observed=%v/%d/%v/%d/%d/%v",
			machine.fdcRestoreClockStarted, machine.nextFDCRestoreClock, memory.fdcRestorePending,
			memory.fdcRestoreStartClock, memory.fdcRestoreInactivePolls, memory.fdcRestoreIRQObserved)
	}
}

func TestFDCStatusReadClearsIRQ(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.dmaMode = 0x0080
	memory.fdcCommand = 0x0b
	memory.fdcStatus = 0x84
	memory.fdcStatusTypeI = true
	memory.fdcIRQ = true
	memory.mfpGPIPIn &^= 0x20
	memory.fdcInitStage = 5
	if err := memory.WriteWord(STDMAControl, 0x0082, 5); err == nil ||
		memory.fdcInitStage != 5 || memory.dmaMode != 0x0080 || !memory.fdcIRQ {
		t.Fatal("wrong post-restore mode unexpectedly accepted")
	}
	if _, err := memory.ReadWord(STDiskController, 5); err == nil ||
		memory.fdcInitStage != 5 || !memory.fdcIRQ || memory.mfpGPIPIn&0x20 != 0 {
		t.Fatal("status read before mode unexpectedly accepted")
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0080,
		m68k.BusAccess{Clock: 2000, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.fdcInitStage != 6 || memory.fdcStatus != 0x84 || !memory.fdcIRQ {
		t.Fatalf("mode wait/err/stage/status/IRQ=%d/%v/%d/%02x/%v", wait, err,
			memory.fdcInitStage, memory.fdcStatus, memory.fdcIRQ)
	}
	if _, err := memory.ReadWord(STDiskController, 1); err == nil ||
		memory.fdcInitStage != 6 || !memory.fdcIRQ {
		t.Fatal("user status read unexpectedly accepted")
	}
	value, wait, err := memory.ReadWordAt(STDiskController,
		m68k.BusAccess{Clock: 2000, FunctionCode: 5})
	if err != nil || wait != 4 || value != 0x00e4 || memory.fdcInitStage != 7 ||
		memory.fdcStatus != 0xe4 || memory.fdcIRQ || memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("status value/wait/err/stage/state/IRQ/GPIP=%04x/%d/%v/%d/%02x/%v/%02x",
			value, wait, err, memory.fdcInitStage, memory.fdcStatus, memory.fdcIRQ,
			memory.mfpGPIPIn)
	}
	if memory.fdcStatusReadClock != 2000 {
		t.Fatalf("status read clock=%d want 2000", memory.fdcStatusReadClock)
	}
	if _, err := memory.ReadWord(STDiskController, 5); err == nil || memory.fdcInitStage != 7 {
		t.Fatal("repeated status read unexpectedly accepted")
	}
	memory.ColdReset()
	if memory.fdcInitStage != 0 || memory.fdcStatus != 0 || memory.fdcIRQ ||
		memory.mfpGPIPIn&0x20 == 0 || memory.fdcStatusReadClock != 0 {
		t.Fatalf("reset stage/status/IRQ/GPIP=%d/%02x/%v/%02x", memory.fdcInitStage,
			memory.fdcStatus, memory.fdcIRQ, memory.mfpGPIPIn)
	}
}

func TestFDCSeekTrackZeroDeadlineAndStatus(t *testing.T) {
	rom := testROM()
	copy(rom[:8], []byte{0x00, 0x00, 0x10, 0x00, 0x00, 0xfc, 0x00, 0x00})
	memory, err := NewMemory(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	memory.dmaMode = 0x0080
	memory.fdcCommand = 0x0b
	memory.fdcStatus = 0xe4
	memory.fdcStatusTypeI = true
	memory.fdcInitStage = 7
	if err := memory.WriteWord(STDiskController, 0, 5); err == nil || memory.fdcInitStage != 7 {
		t.Fatal("data write before selector unexpectedly accepted")
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0086,
		m68k.BusAccess{Clock: 2000, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.fdcInitStage != 8 || memory.dmaMode != 0x0086 {
		t.Fatalf("data selector wait/err/stage/mode=%d/%v/%d/%04x", wait, err,
			memory.fdcInitStage, memory.dmaMode)
	}
	if wait, err := memory.WriteWordAt(STDiskController, 0,
		m68k.BusAccess{Clock: 2100, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.fdcInitStage != 9 || memory.fdcData != 0 {
		t.Fatalf("data wait/err/stage/value=%d/%v/%d/%02x", wait, err,
			memory.fdcInitStage, memory.fdcData)
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0080,
		m68k.BusAccess{Clock: 2200, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.fdcInitStage != 10 || memory.dmaMode != 0x0080 {
		t.Fatalf("command selector wait/err/stage/mode=%d/%v/%d/%04x", wait, err,
			memory.fdcInitStage, memory.dmaMode)
	}
	if err := memory.WriteWord(STDiskController, 0x0012, 5); err == nil ||
		memory.fdcInitStage != 10 || memory.fdcSeekPending {
		t.Fatal("wrong seek command unexpectedly accepted")
	}
	if wait, err := memory.WriteWordAt(STDiskController, 0x0013,
		m68k.BusAccess{Clock: 2300, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.fdcInitStage != 11 || !memory.fdcSeekPending ||
		memory.fdcSeekStartClock != 2300 || memory.fdcStatus != 0xe5 || memory.fdcIRQ {
		t.Fatalf("seek wait/err/stage/pending/start/status/IRQ=%d/%v/%d/%v/%d/%02x/%v",
			wait, err, memory.fdcInitStage, memory.fdcSeekPending,
			memory.fdcSeekStartClock, memory.fdcStatus, memory.fdcIRQ)
	}
	for i := 0; i < 9; i++ {
		if got, err := memory.ReadByteFC(MFPGPIP, 5); err != nil || got&0x20 == 0 {
			t.Fatalf("inactive poll %d=%02x err=%v", i, got, err)
		}
	}
	if memory.fdcSeekInactivePolls != 9 || !memory.fdcSeekPending {
		t.Fatalf("polls/pending=%d/%v", memory.fdcSeekInactivePolls, memory.fdcSeekPending)
	}
	machine := &Machine{Memory: memory, Clocks: 3028}
	machine.CPU.Bus = memory
	machine.advanceClockedDevices()
	if !machine.fdcSeekClockStarted || machine.nextFDCSeekClock != 3029 ||
		!memory.fdcSeekPending || memory.fdcStatus != 0xe5 {
		t.Fatalf("early scheduler/next/pending/status=%v/%d/%v/%02x",
			machine.fdcSeekClockStarted, machine.nextFDCSeekClock,
			memory.fdcSeekPending, memory.fdcStatus)
	}
	machine.Clocks = 3029
	machine.advanceClockedDevices()
	if machine.fdcSeekClockStarted || machine.nextFDCSeekClock != 0 || memory.fdcSeekPending ||
		memory.fdcInitStage != 12 || memory.fdcStatus != 0xe4 || !memory.fdcIRQ ||
		memory.mfpGPIPIn&0x20 != 0 {
		t.Fatalf("complete scheduler/next/pending/stage/status/IRQ/GPIP=%v/%d/%v/%d/%02x/%v/%02x",
			machine.fdcSeekClockStarted, machine.nextFDCSeekClock, memory.fdcSeekPending,
			memory.fdcInitStage, memory.fdcStatus, memory.fdcIRQ, memory.mfpGPIPIn)
	}
	if got, err := memory.ReadByteFC(MFPGPIP, 5); err != nil || got&0x20 != 0 ||
		!memory.fdcSeekIRQObserved {
		t.Fatalf("active poll=%02x err=%v observed=%v", got, err, memory.fdcSeekIRQObserved)
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0080,
		m68k.BusAccess{Clock: 3100, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.fdcInitStage != 13 {
		t.Fatalf("status selector wait/err/stage=%d/%v/%d", wait, err, memory.fdcInitStage)
	}
	value, wait, err := memory.ReadWordAt(STDiskController,
		m68k.BusAccess{Clock: 3200, FunctionCode: 5})
	if err != nil || wait != 4 || value != 0xe4 || memory.fdcInitStage != 14 ||
		memory.fdcIRQ || memory.mfpGPIPIn&0x20 == 0 || memory.fdcSeekStatusReadClock != 3200 {
		t.Fatalf("status value/wait/err/stage/IRQ/GPIP/clock=%04x/%d/%v/%d/%v/%02x/%d",
			value, wait, err, memory.fdcInitStage, memory.fdcIRQ, memory.mfpGPIPIn,
			memory.fdcSeekStatusReadClock)
	}
	machine.fdcSeekClockStarted = true
	machine.nextFDCSeekClock = 4000
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	if machine.fdcSeekClockStarted || machine.nextFDCSeekClock != 0 || memory.fdcData != 0 ||
		memory.fdcSeekPending || memory.fdcSeekStartClock != 0 || memory.fdcSeekInactivePolls != 0 ||
		memory.fdcSeekIRQObserved || memory.fdcSeekStatusReadClock != 0 {
		t.Fatalf("reset scheduler/next/data/pending/start/polls/observed/readClock=%v/%d/%02x/%v/%d/%d/%v/%d",
			machine.fdcSeekClockStarted, machine.nextFDCSeekClock, memory.fdcData,
			memory.fdcSeekPending, memory.fdcSeekStartClock, memory.fdcSeekInactivePolls,
			memory.fdcSeekIRQObserved, memory.fdcSeekStatusReadClock)
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
		want := byte(0)
		if address == MFPTSR {
			want = 0x80
		}
		if got, err := machine.Memory.ReadByteFC(address, 5); err != nil || got != want {
			t.Fatalf("MFP USART %06x=%02x/%v want %02x", address, got, err, want)
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
