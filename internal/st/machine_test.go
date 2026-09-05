package st

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
)

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
