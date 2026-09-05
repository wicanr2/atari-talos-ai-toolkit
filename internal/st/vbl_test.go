package st

import (
	"os"
	"testing"

	"github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"
)

func TestMachineFirstVBLPendingSurvivesMask(t *testing.T) {
	bus := m68k.SparseMemory{
		0x70: 0x00, 0x71: 0x00, 0x72: 0x20, 0x73: 0x00,
		0x1000: 0x4e, 0x1001: 0x71, 0x1002: 0x4e, 0x1003: 0x71,
		0x1004: 0x4e, 0x1005: 0x71, 0x1006: 0x4e, 0x1007: 0x71,
		0x2000: 0x4e, 0x2001: 0x71, 0x2002: 0x4e, 0x2003: 0x71,
	}
	machine := Machine{CPU: m68k.CPU{Bus: bus, State: m68k.State{
		SSP: 0x3000, SR: 0x2700, PC: 0x1004, Prefetch: [2]uint16{0x4e71, 0x4e71},
	}}, Clocks: firstColorSTVBLClock - 4, nextVBLClock: firstColorSTVBLClock}
	if _, err := machine.Step(); err != nil {
		t.Fatal(err)
	}
	if machine.nextVBLClock != firstColorSTVBLClock+colorST50HzFrameClocks || !machine.vblPending || machine.Instructions != 1 || machine.Interrupts != 0 {
		t.Fatalf("deadline state=%+v", machine)
	}
	if _, err := machine.Step(); err != nil {
		t.Fatal(err)
	}
	if !machine.vblPending || machine.Instructions != 2 || machine.Interrupts != 0 {
		t.Fatalf("masked pending state=%+v", machine)
	}
	machine.CPU.State.SR = 0x2300
	result, err := machine.Step()
	if err != nil {
		t.Fatal(err)
	}
	if result.Clocks != 60 || machine.vblPending || machine.Instructions != 2 || machine.Interrupts != 1 {
		t.Fatalf("accepted state=%+v result=%+v", machine, result)
	}
}

func TestMachineEmuTOSFirstVBLExecutesGuestFrclockIncrement(t *testing.T) {
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
	for machine.Interrupts == 0 {
		if _, err := machine.Step(); err != nil {
			t.Fatal(err)
		}
	}
	wantD := [8]uint32{1, 0x0f, 0x2710, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xfffffa2f, 0x3156, 0x00fc68fa, 0, 0, 0x00fc01f4, 0x0ffc}
	state := machine.CPU.State
	if machine.Interrupts != 1 || machine.Clocks != 178012 || state.D != wantD || state.A != wantA ||
		state.SSP != 0x0f6e || state.SR != 0x2400 || state.PC != 0xfc044a ||
		state.Prefetch != [2]uint16{0x52b8, 0x0466} {
		t.Fatalf("first VBL entry interrupts=%d clocks=%d state=%+v", machine.Interrupts, machine.Clocks, state)
	}
	wantFrame := []uint16{0x2300, 0x00fc, 0x6904}
	for i, want := range wantFrame {
		got, readErr := machine.Memory.ReadWord(0x0f6e+uint32(i*2), 5)
		if readErr != nil || got != want {
			t.Fatalf("frame[%d]=%04x/%v want %04x", i, got, readErr, want)
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
	if got := uint32(frHigh)<<16 | uint32(frLow); got != 1 {
		t.Fatalf("guest VBL handler frclock=%d want 1", got)
	}
}

func TestMachineStoppedAdvancesToRecurringVBL(t *testing.T) {
	bus := m68k.SparseMemory{
		0x70: 0x00, 0x71: 0x00, 0x72: 0x20, 0x73: 0x00,
		0x2000: 0x4e, 0x2001: 0x71, 0x2002: 0x4e, 0x2003: 0x71,
	}
	cpu := m68k.CPU{Bus: bus, State: m68k.State{
		SSP: 0x3000, SR: 0x2300, PC: 0x1004, Prefetch: [2]uint16{0x4e72, 0x2300},
	}}
	if _, err := cpu.Step(); err != nil {
		t.Fatal(err)
	}
	machine := Machine{CPU: cpu, Clocks: 200000, nextVBLClock: 293924}
	result, err := machine.Step()
	if err != nil {
		t.Fatal(err)
	}
	if result.Clocks != 93982 || machine.Clocks != 293982 || machine.Interrupts != 1 ||
		machine.Instructions != 0 || machine.vblPending || machine.nextVBLClock != 454180 {
		t.Fatalf("machine=%+v result=%+v", machine, result)
	}
	if len(result.Timeline) == 0 || result.Timeline[0] != (m68k.BusPhase{Cycles: 93938}) {
		t.Fatalf("timeline=%+v", result.Timeline)
	}
}

func TestVideoIACKExtraClocks(t *testing.T) {
	for _, test := range []struct {
		epoch uint64
		want  uint32
	}{{293924, 14}, {133672, 16}, {177950, 18}} {
		if got := videoIACKExtraClocks(test.epoch); got != test.want {
			t.Fatalf("epoch=%d got=%d want=%d", test.epoch, got, test.want)
		}
	}
}
