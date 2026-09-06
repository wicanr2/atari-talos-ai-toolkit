package st

import "testing"

// TestEmuTOSLineFProbeTakesVector11 is spec 059's end-to-end receipt: EmuTOS
// probes for a 68030 PMMU with `PMOVE (A0),TT0` ($F010 $0800) at $FC00BE, and
// on an MC68000 that is a line-F emulator exception through vector 11. It is
// the first stop the boot path used to hit.
//
// The clock check is on this one step, not on the accumulated count: the
// 26-versus-24 question for `MOVE.L #imm,(xxx).W` is still open and two of
// those run before this point, so the running total is four short of Hatari's.
// That is its own item; borrowing this slice to absorb it would hide both.
func TestEmuTOSLineFProbeTakesVector11(t *testing.T) {
	machine := emuTOSMachine(t)
	// 18 instructions reach $FC00BE; the line-F opcode is the 19th.
	for step := 0; step < 18; step++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("step %d: %v", step+1, err)
		}
	}
	if machine.CPU.State.PC != 0xfc00be+4 {
		t.Fatalf("before the probe: PC=%#08x, want %#08x", machine.CPU.State.PC, uint32(0xfc00be+4))
	}
	if machine.CPU.State.Prefetch != [2]uint16{0xf010, 0x0800} {
		t.Fatalf("before the probe: prefetch=%#04x, want f010 0800", machine.CPU.State.Prefetch)
	}
	if machine.CPU.State.SSP != 0x0fe6 || machine.CPU.State.SR != 0x2700 {
		t.Fatalf("before the probe: SSP=%#06x SR=%#04x, want 0fe6/2700 (Hatari trace, spec 059)",
			machine.CPU.State.SSP, machine.CPU.State.SR)
	}
	before := machine.Clocks
	if _, err := machine.Step(); err != nil {
		t.Fatalf("the line-F probe: %v", err)
	}
	if spent := machine.Clocks - before; spent != 36 {
		t.Errorf("the probe took %d clocks, want 36 (Hatari 532 − 496)", spent)
	}
	// The handler is $FC00D4, which $FC00B2 wrote into vector 11's slot at $2C.
	if machine.CPU.State.PC != 0xfc00d4+4 {
		t.Errorf("after the probe: PC=%#08x, want %#08x", machine.CPU.State.PC, uint32(0xfc00d4+4))
	}
	if machine.CPU.State.Prefetch != [2]uint16{0x21fc, 0x00fc} {
		t.Errorf("after the probe: prefetch=%#04x, want 21fc 00fc", machine.CPU.State.Prefetch)
	}
	if machine.CPU.State.SSP != 0x0fe0 || machine.CPU.State.SR != 0x2700 {
		t.Errorf("after the probe: SSP=%#06x SR=%#04x, want 0fe0/2700",
			machine.CPU.State.SSP, machine.CPU.State.SR)
	}
	// The frame keeps the opcode address, not the next PC.
	for index, want := range []uint16{0x2700, 0x00fc, 0x00be} {
		got, err := machine.Memory.ReadWord(0x0fe0+uint32(index*2), 5)
		if err != nil || got != want {
			t.Errorf("frame[%d]=%#04x/%v, want %#04x", index, got, err, want)
		}
	}
}
