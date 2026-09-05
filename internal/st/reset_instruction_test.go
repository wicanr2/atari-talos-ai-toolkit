package st

import "testing"

// TestEmuTOSExecutesResetInstruction moves the boot path's stopping point past
// $FC0088. Spec 051 recorded that both Atari Talos and Hatari reach that RESET
// in 10 instructions / 220 clocks; before spec 053 the step itself failed with
// "opcode 0x4e70 is not implemented".
func TestEmuTOSExecutesResetInstruction(t *testing.T) {
	machine := emuTOSMachine(t)
	for step := 0; step < 10; step++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("step %d: %v", step, err)
		}
	}
	if machine.Instructions != 10 || machine.Clocks != 220 || machine.CPU.State.PC != 0x00fc008c {
		t.Fatalf("at RESET: instructions=%d clocks=%d PC=%#08x, want 10/220/0x00fc008c",
			machine.Instructions, machine.Clocks, machine.CPU.State.PC)
	}

	before := machine.CPU.State
	result, err := machine.Step()
	if err != nil {
		t.Fatalf("RESET: %v", err)
	}
	if result.Clocks != 132 {
		t.Errorf("RESET clocks = %d, want 132", result.Clocks)
	}
	if machine.Instructions != 11 || machine.Clocks != 352 {
		t.Errorf("after RESET: instructions=%d clocks=%d, want 11/352",
			machine.Instructions, machine.Clocks)
	}
	// RESET drives an external line; the CPU's own registers must be untouched.
	after := machine.CPU.State
	if after.SR != before.SR || after.D != before.D || after.A != before.A ||
		after.SSP != before.SSP || after.USP != before.USP {
		t.Errorf("RESET changed CPU state:\n before=%+v\n after =%+v", before, after)
	}
	if after.PC != before.PC+2 {
		t.Errorf("PC = %#08x, want %#08x (prefetch advanced by one word)", after.PC, before.PC+2)
	}
}
