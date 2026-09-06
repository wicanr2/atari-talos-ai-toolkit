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
	}}, Clocks: firstColorSTVBLClock - 4, nextVBLClock: firstColorSTVBLClock, vblFrameClocks: colorST60HzFrameClocks}
	if _, err := machine.Step(); err != nil {
		t.Fatal(err)
	}
	if machine.nextVBLClock != firstColorSTVBLClock+colorST60HzFrameClocks || !machine.vblPending || machine.Instructions != 1 || machine.Interrupts != 0 {
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
	machine := Machine{CPU: cpu, Clocks: 200000, nextVBLClock: 267272, vblFrameClocks: colorST60HzFrameClocks}
	result, err := machine.Step()
	if err != nil {
		t.Fatal(err)
	}
	if result.Clocks != 67332 || machine.Clocks != 267332 || machine.Interrupts != 1 ||
		machine.Instructions != 0 || machine.vblPending || machine.nextVBLClock != 400876 {
		t.Fatalf("machine=%+v result=%+v", machine, result)
	}
	if len(result.Timeline) == 0 || result.Timeline[0] != (m68k.BusPhase{Cycles: 67288}) {
		t.Fatalf("timeline=%+v", result.Timeline)
	}
}

func TestMachineStoppedVBLReloadsProgrammedVideoBase(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	for address, value := range map[uint32]uint16{
		0x70: 0x0000, 0x72: 0x2000,
		0x2000: 0x4e71, 0x2002: 0x4e71,
	} {
		if err := memory.WriteWord(address, value, 5); err != nil {
			t.Fatal(err)
		}
	}
	if err := memory.WriteByteFC(VideoBaseHigh, 0x0f, 5); err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteByteFC(VideoBaseMiddle, 0x80, 5); err != nil {
		t.Fatal(err)
	}
	cpu := m68k.CPU{Bus: memory, State: m68k.State{
		SSP: 0x3000, SR: 0x2300, PC: 0x1004, Prefetch: [2]uint16{0x4e72, 0x2300},
	}}
	if _, err := cpu.Step(); err != nil {
		t.Fatal(err)
	}
	machine := Machine{CPU: cpu, Memory: memory, Clocks: 200000,
		nextVBLClock: 267272, vblFrameClocks: colorST60HzFrameClocks}
	if _, err := machine.Step(); err != nil {
		t.Fatal(err)
	}
	if got := memory.ActiveVideoBase(); got != 0x0f8000 {
		t.Fatalf("STOP-fast-forward active video base=%06x want 0f8000", got)
	}
}

func TestVideoIACKExtraClocks(t *testing.T) {
	for _, test := range []struct {
		epoch uint64
		want  uint32
	}{{267272, 16}, {400876, 12}, {133672, 16}, {177950, 18}} {
		if got := videoIACKExtraClocks(test.epoch); got != test.want {
			t.Fatalf("epoch=%d got=%d want=%d", test.epoch, got, test.want)
		}
	}
}

func TestMachineEmuTOSInitializesShifterLowResolution(t *testing.T) {
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
	for machine.CPU.State.Prefetch != [2]uint16{0x11c0, 0x8260} {
		if machine.Instructions > 7700 {
			t.Fatal("EmuTOS Shifter resolution write was not reached")
		}
		if _, err := machine.Step(); err != nil {
			t.Fatalf("step %d clocks=%d: %v", machine.Instructions+1, machine.Clocks, err)
		}
	}
	wantD := [8]uint32{0, 0x18, 1, 0, 0x00080000, 0x00100000, 5, 1}
	wantA := [7]uint32{0xfffffa2f, 0x3156, 0x00fc68fa, 0, 0, 0x00fc01f4, 0x0ffc}
	before := machine.CPU.State
	if machine.Instructions != 7654 || machine.Interrupts != 3 || machine.Clocks != 401192 ||
		before.D != wantD || before.A != wantA || before.SSP != 0x0f84 || before.SR != 0x2714 ||
		before.PC != 0x00fc69ea {
		t.Fatalf("Shifter pre-state instructions=%d interrupts=%d clocks=%d state=%+v",
			machine.Instructions, machine.Interrupts, machine.Clocks, before)
	}
	result, err := machine.Step()
	if err != nil {
		t.Fatal(err)
	}
	after := machine.CPU.State
	if result.Clocks != 12 || machine.Instructions != 7655 || machine.Clocks != 401204 ||
		after.D != wantD || after.A != wantA || after.SSP != 0x0f84 || after.SR != 0x2714 ||
		after.PC != 0x00fc69ee || after.Prefetch != [2]uint16{0x3239, 0x00fc} {
		t.Fatalf("Shifter post-state result=%+v machine=%+v", result, machine)
	}
	if got, err := machine.Memory.ReadByteFC(ShifterResolution, 5); err != nil || got != 0xfc {
		t.Fatalf("Shifter resolution=%02x/%v want fc", got, err)
	}
	for machine.CPU.State.Prefetch != [2]uint16{0x11c1, 0x820a} {
		if machine.Instructions > 7700 {
			t.Fatal("EmuTOS video sync write was not reached")
		}
		if _, err := machine.Step(); err != nil {
			t.Fatalf("post-Shifter step %d clocks=%d: %v", machine.Instructions+1, machine.Clocks, err)
		}
	}
	wantSyncD := [8]uint32{0, 2, 1, 0, 0x00080000, 0x00100000, 5, 1}
	before = machine.CPU.State
	if machine.Instructions != 7662 || machine.Interrupts != 3 || machine.Clocks != 401270 ||
		before.D != wantSyncD || before.A != wantA || before.SSP != 0x0f84 || before.SR != 0x2710 ||
		before.PC != 0x00fc6a06 || machine.nextVBLClock != 534480 ||
		machine.vblFrameClocks != colorST60HzFrameClocks {
		t.Fatalf("sync pre-state machine=%+v", machine)
	}
	result, err = machine.Step()
	if err != nil {
		t.Fatal(err)
	}
	after = machine.CPU.State
	if result.Clocks != 12 || machine.Instructions != 7663 || machine.Clocks != 401282 ||
		after.D != wantSyncD || after.A != wantA || after.SSP != 0x0f84 || after.SR != 0x2710 ||
		after.PC != 0x00fc6a0a || after.Prefetch != [2]uint16{0x4267, 0x3f00} ||
		machine.nextVBLClock != 535528 || machine.vblFrameClocks != colorST50HzFrameClocks {
		t.Fatalf("sync post-state result=%+v machine=%+v", result, machine)
	}
	if got, err := machine.Memory.ReadByteFC(VideoSyncMode, 5); err != nil || got != 0xfe {
		t.Fatalf("video sync=%02x/%v want fe", got, err)
	}
	for !(machine.CPU.State.Prefetch[0] == 0x30c1 && machine.CPU.State.A[0] == 0xffff8240) {
		if machine.Instructions > 7800 {
			t.Fatal("EmuTOS palette loop was not reached")
		}
		if _, err := machine.Step(); err != nil {
			t.Fatalf("pre-palette step %d clocks=%d: %v", machine.Instructions+1, machine.Clocks, err)
		}
	}
	wantPaletteD := [8]uint32{0, 0x0777, 1, 0, 0x00080000, 0x00100000, 5, 1}
	wantPaletteA := [7]uint32{0xffff8240, 0x00fc6a68, 0x00fc68fa, 0, 0, 0x00fc01f4, 0x0ffc}
	before = machine.CPU.State
	if machine.Instructions != 7671 || machine.Interrupts != 3 || machine.Clocks != 401366 ||
		before.D != wantPaletteD || before.A != wantPaletteA || before.SSP != 0x0f7c ||
		before.SR != 0x2710 || before.PC != 0x00fc671e || before.Prefetch != [2]uint16{0x30c1, 0xb0fc} {
		t.Fatalf("palette first pre-state machine=%+v", machine)
	}
	result, err = machine.Step()
	if err != nil {
		t.Fatal(err)
	}
	wantPaletteA[0] = 0xffff8242
	after = machine.CPU.State
	if result.Clocks != 8 || machine.Instructions != 7672 || machine.Clocks != 401374 ||
		after.D != wantPaletteD || after.A != wantPaletteA || after.SSP != 0x0f7c ||
		after.SR != 0x2710 || after.PC != 0x00fc6720 || after.Prefetch != [2]uint16{0xb0fc, 0x8260} ||
		machine.Memory.shifterPalette[0] != 0x0777 {
		t.Fatalf("palette first post-state result=%+v machine=%+v", result, machine)
	}
	for !(machine.CPU.State.Prefetch[0] == 0x0c40 && machine.CPU.State.PC == 0x00fc6726) {
		if machine.Instructions > 7800 {
			t.Fatal("EmuTOS palette loop did not finish")
		}
		if _, err := machine.Step(); err != nil {
			t.Fatalf("palette loop step %d clocks=%d: %v", machine.Instructions+1, machine.Clocks, err)
		}
	}
	wantPalette := [16]uint16{
		0x0777, 0x0700, 0x0070, 0x0770, 0x0007, 0x0707, 0x0077, 0x0555,
		0x0333, 0x0733, 0x0373, 0x0773, 0x0337, 0x0737, 0x0377, 0x0000,
	}
	wantLoopD := [8]uint32{0, 0, 1, 0, 0x00080000, 0x00100000, 5, 1}
	wantLoopA := [7]uint32{0xffff8260, 0x00fc6a86, 0x00fc68fa, 0, 0, 0x00fc01f4, 0x0ffc}
	after = machine.CPU.State
	if machine.Instructions != 7749 || machine.Clocks != 402052 || after.D != wantLoopD ||
		after.A != wantLoopA || after.SSP != 0x0f7c || after.SR != 0x2714 ||
		after.Prefetch != [2]uint16{0x0c40, 0x0001} || machine.Memory.shifterPalette != wantPalette {
		t.Fatalf("palette loop final machine=%+v palette=%#v", machine, machine.Memory.shifterPalette)
	}
	for machine.CPU.State.Prefetch != [2]uint16{0x11c1, 0x8201} {
		if machine.Instructions > 7920 {
			t.Fatal("EmuTOS framebuffer base write was not reached")
		}
		if _, err := machine.Step(); err != nil {
			t.Fatalf("pre-video-base step %d clocks=%d: %v", machine.Instructions+1, machine.Clocks, err)
		}
	}
	wantBaseD := [8]uint32{0x000f8000, 0x0f, 0, 0, 0x00080000, 0x00100000, 5, 1}
	wantBaseA := [7]uint32{0x00100000, 0x000f2358, 0, 0, 0, 0x00fc01f4, 0x0ffc}
	before = machine.CPU.State
	if machine.Instructions != 7896 || machine.Clocks != 403900 || before.D != wantBaseD ||
		before.A != wantBaseA || before.SSP != 0x0f86 || before.SR != 0x2700 ||
		before.PC != 0x00fc67fe {
		t.Fatalf("video-base high pre-state machine=%+v", machine)
	}
	result, err = machine.Step()
	if err != nil {
		t.Fatal(err)
	}
	if result.Clocks != 12 || machine.Clocks != 403912 || machine.Memory.ProgrammedVideoBase() != 0x0f0000 ||
		machine.CPU.State.PC != 0x00fc6802 || machine.CPU.State.Prefetch != [2]uint16{0xe088, 0x11c0} {
		t.Fatalf("video-base high post-state result=%+v machine=%+v", result, machine)
	}
	result, err = machine.Step()
	if err != nil {
		t.Fatal(err)
	}
	wantBaseD[0] = 0x00000f80
	if result.Clocks != 24 || machine.Clocks != 403936 || machine.CPU.State.D != wantBaseD ||
		machine.CPU.State.PC != 0x00fc6804 || machine.CPU.State.Prefetch != [2]uint16{0x11c0, 0x8203} {
		t.Fatalf("video-base shift post-state result=%+v machine=%+v", result, machine)
	}
	result, err = machine.Step()
	if err != nil {
		t.Fatal(err)
	}
	if result.Clocks != 12 || machine.Instructions != 7899 || machine.Clocks != 403948 ||
		machine.CPU.State.D != wantBaseD || machine.CPU.State.A != wantBaseA ||
		machine.CPU.State.SSP != 0x0f86 || machine.CPU.State.SR != 0x2708 ||
		machine.CPU.State.PC != 0x00fc6808 || machine.CPU.State.Prefetch != [2]uint16{0x5c8f, 0x4e75} ||
		machine.Memory.ProgrammedVideoBase() != 0x0f8000 {
		t.Fatalf("video-base middle post-state result=%+v machine=%+v", result, machine)
	}
	for machine.Clocks < 535520 {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("pre-VBL4 step %d clocks=%d: %v", machine.Instructions+1, machine.Clocks, err)
		}
	}
	if machine.Clocks != 535520 || machine.Memory.ActiveVideoBase() != 0 ||
		machine.Memory.ProgrammedVideoBase() != 0x0f8000 || machine.vblPending {
		t.Fatalf("VBL4 pre-boundary machine=%+v programmed=%06x active=%06x", machine,
			machine.Memory.ProgrammedVideoBase(), machine.Memory.ActiveVideoBase())
	}
	before = machine.CPU.State
	result, err = machine.Step()
	if err != nil {
		t.Fatal(err)
	}
	if result.Clocks != 10 || machine.Clocks != 535530 || !machine.vblPending ||
		machine.nextVBLClock != 695784 || machine.Memory.ActiveVideoBase() != 0x0f8000 ||
		machine.Memory.ProgrammedVideoBase() != 0x0f8000 {
		t.Fatalf("VBL4 post-boundary result=%+v machine=%+v programmed=%06x active=%06x before=%+v",
			result, machine, machine.Memory.ProgrammedVideoBase(), machine.Memory.ActiveVideoBase(), before)
	}
	frame, err := machine.Memory.LowResolutionFrame()
	if err != nil {
		t.Fatal(err)
	}
	if frame.Width != 320 || frame.Height != 200 || frame.Palette != wantPalette {
		t.Fatalf("VBL4 indexed frame metadata=%+v", frame)
	}
	for offset, index := range frame.Indices {
		if index != 0 {
			t.Fatalf("VBL4 indexed frame pixel %d=%d want 0", offset, index)
		}
	}
}
