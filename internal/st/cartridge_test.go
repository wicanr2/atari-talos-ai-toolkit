package st

import "testing"

// TestCartridgePortReadsIdleBus covers spec 057: with no cartridge plugged in
// nothing drives the data bus, so reads see $FF rather than a bus fault.
func TestCartridgePortReadsIdleBus(t *testing.T) {
	machine, err := NewMachine(RAM1M, make([]byte, TOSROMSize))
	if err != nil {
		t.Fatal(err)
	}
	for _, address := range []uint32{CartridgeBase, CartridgeBase + 2, CartridgeEnd - 1} {
		got, err := machine.Memory.ReadWord(address, 5)
		if err != nil {
			t.Errorf("read %#08x: %v", address, err)
			continue
		}
		if got != 0xffff {
			t.Errorf("read %#08x = %#04x, want 0xffff", address, got)
		}
	}
	// The region has to stay bounded: below it is unmapped RAM space, above it
	// is TOS ROM. Widening the case would hide both.
	if _, err := machine.Memory.ReadWord(CartridgeBase-2, 5); err == nil {
		t.Error("read below the cartridge port should still fault")
	}
	if _, err := machine.Memory.ReadWord(TOSROMBase, 5); err != nil {
		t.Errorf("read at TOS ROM base should still be ROM: %v", err)
	}
}

// TestEmuTOSPassesCartridgeCheck is the end-to-end receipt: the CMPI.L at
// $FC008A must complete and take the branch, matching the Hatari cycle counts
// recorded in spec 057.
func TestEmuTOSPassesCartridgeCheck(t *testing.T) {
	machine := emuTOSMachine(t)
	// 11 instructions reach $FC008A (spec 053); the CMPI.L is the 12th.
	for step := 0; step < 11; step++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
	}
	if machine.Clocks != 352 {
		t.Fatalf("before CMPI.L: clocks=%d, want 352", machine.Clocks)
	}
	// Stops at $FC00A0. The next instruction (MOVE.L #imm,(xxx).W) is 26 cycles
	// in Hatari and 24 here, and the corpus has no case for that addressing pair
	// to settle it — that is its own open item, not something this slice may
	// absorb by relaxing the expectation.
	for _, want := range []struct {
		clocks uint64
		label  string
	}{{380, "$FC0094 BNE"}, {390, "$FC00A0"}} {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("stepping to %s: %v", want.label, err)
		}
		if uint64(machine.Clocks) != want.clocks {
			t.Errorf("at %s: clocks=%d, want %d (Hatari trace, spec 057)",
				want.label, machine.Clocks, want.clocks)
		}
	}
}
