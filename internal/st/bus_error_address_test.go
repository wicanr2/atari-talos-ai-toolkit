package st

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
)

// emuTOSMachine boots the pinned EmuTOS 1.3 UK ROM that specs 049-051 use.
func emuTOSMachine(t *testing.T) *Machine {
	t.Helper()
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const wantHash = "ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135"
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
	return machine
}

// buildFaultROM lays out a ROM whose reset vector runs one instruction at
// $FC0030 and whose vector-2 handler sits at $FC0100.
func buildFaultROM(instruction []byte) []byte {
	rom := make([]byte, TOSROMSize)
	// SSP = $00010000, PC = $00FC0030.
	copy(rom[0:8], []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0xfc, 0x00, 0x30})
	copy(rom[0x30:0x30+len(instruction)], instruction)
	// Handler: two NOPs, enough for the prefetch the frame entry performs.
	copy(rom[0x100:0x104], []byte{0x4e, 0x71, 0x4e, 0x71})
	return rom
}

func machineAtFault(t *testing.T, instruction []byte, prepare func(m *Machine)) *Machine {
	t.Helper()
	machine, err := NewMachine(RAM1M, buildFaultROM(instruction))
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	// Vector 2 lives in RAM and has to be planted after reset.
	if err := machine.Memory.WriteWord(0x08, 0x00fc, 5); err != nil {
		t.Fatal(err)
	}
	if err := machine.Memory.WriteWord(0x0a, 0x0100, 5); err != nil {
		t.Fatal(err)
	}
	if prepare != nil {
		prepare(machine)
	}
	return machine
}

// frameAccessAddress reads the access-address long out of the 14-byte frame the
// exception just pushed. The frame starts at the new SSP; the access address is
// the second and third words (spec 007 §4).
func frameAccessAddress(t *testing.T, m *Machine) uint32 {
	t.Helper()
	high, err := m.Memory.ReadWord(m.CPU.State.SSP+2, 5)
	if err != nil {
		t.Fatal(err)
	}
	low, err := m.Memory.ReadWord(m.CPU.State.SSP+4, 5)
	if err != nil {
		t.Fatal(err)
	}
	return uint32(high)<<16 | uint32(low)
}

// TestBusErrorFrameKeepsSignExtendedAbsoluteShort is the EmuTOS $FC0080 case:
// TST.W $8006 sign-extends to $FFFF8006, and that is what the frame must hold.
// The same step must still put the masked $FF8006 on the bus — checking only the
// frame would let "stop masking entirely" pass.
func TestBusErrorFrameKeepsSignExtendedAbsoluteShort(t *testing.T) {
	machine := machineAtFault(t, []byte{0x4a, 0x78, 0x80, 0x06}, nil)
	result, err := machine.Step()
	if err != nil {
		t.Fatalf("step: %v", err)
	}
	if got := frameAccessAddress(t, machine); got != 0xffff8006 {
		t.Errorf("frame access address = %#08x, want 0xffff8006", got)
	}
	var faults int
	for _, transaction := range result.Transactions {
		if transaction.Kind != "re" {
			continue
		}
		faults++
		if transaction.Address != 0x00ff8006 {
			t.Errorf("bus address = %#08x, want 0x00ff8006 (A1-A23 only)", transaction.Address)
		}
	}
	if faults != 1 {
		t.Errorf("fault transactions = %d, want 1", faults)
	}
	if machine.CPU.State.PC != 0x00fc0104 {
		t.Errorf("PC = %#08x, want 0x00fc0104 (handler + 4)", machine.CPU.State.PC)
	}
}

// TestBusErrorFrameKeepsAddressRegisterHighBits is the negative control: the
// high bits come from an address register rather than sign extension, so a fix
// that merely stopped masking would look identical on the previous test.
func TestBusErrorFrameKeepsAddressRegisterHighBits(t *testing.T) {
	machine := machineAtFault(t, []byte{0x4a, 0x50, 0x4e, 0x71}, func(m *Machine) {
		m.CPU.State.A[0] = 0x01ff8006
	})
	result, err := machine.Step()
	if err != nil {
		t.Fatalf("step: %v", err)
	}
	if got := frameAccessAddress(t, machine); got != 0x01ff8006 {
		t.Errorf("frame access address = %#08x, want 0x01ff8006", got)
	}
	for _, transaction := range result.Transactions {
		if transaction.Kind == "re" && transaction.Address != 0x00ff8006 {
			t.Errorf("bus address = %#08x, want 0x00ff8006", transaction.Address)
		}
	}
}

// TestEmuTOSBusErrorFrameKeepsIOAddress is the case spec 051 left open: EmuTOS
// 1.3 probes the Shifter with TST.W $8006 at $FC0080, and Hatari's frame holds
// $FFFF8006. Reaching it from reset with the real ROM is what makes this an
// end-to-end receipt rather than a synthetic one.
func TestEmuTOSBusErrorFrameKeepsIOAddress(t *testing.T) {
	machine := emuTOSMachine(t)
	var faulted bool
	for step := 0; step < 64 && !faulted; step++ {
		result, err := machine.Step()
		if err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
		for _, transaction := range result.Transactions {
			if transaction.Kind != "re" {
				continue
			}
			faulted = true
			if transaction.Address != 0x00ff8006 {
				t.Errorf("bus address = %#08x, want 0x00ff8006", transaction.Address)
			}
		}
	}
	if !faulted {
		t.Fatal("EmuTOS never took the $8006 bus error within 64 steps")
	}
	if got := frameAccessAddress(t, machine); got != 0xffff8006 {
		t.Errorf("frame access address = %#08x, want 0xffff8006 (Hatari, spec 051)", got)
	}
}
