package st

import (
	"github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"
	"testing"
)

func timerAMemory(t *testing.T, data, mode byte) *Memory {
	t.Helper()
	m, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []struct {
		a uint32
		v byte
	}{{MFPTADR, data}, {MFPTACR, mode}} {
		if _, err := m.WriteByteAt(w.a, w.v, m68k.BusAccess{FunctionCode: 5}); err != nil {
			t.Fatal(err)
		}
	}
	return m
}

func TestTimerAPrescalersAndRationalPhase(t *testing.T) {
	for mode := byte(1); mode <= 7; mode++ {
		for _, data := range []byte{0, 1, 112, 255} {
			m := timerAMemory(t, data, mode)
			want := (timerACount(data)*timerAPrescalers[mode]*15667 + 4799) / 4800
			if got := m.timerADeadline(); got != want {
				t.Fatalf("deadline %d want %d", got, want)
			}
			m.advanceTimerA(want - 1)
			if m.mfpTimerATimeouts != 0 {
				t.Fatal("early timeout")
			}
			m.advanceTimerA(want)
			if m.mfpTimerATimeouts != 1 || m.mfpTAMain != data || !m.mfpTimerAOutput {
				t.Fatal("reload/output")
			}
			one := timerAMemory(t, data, mode)
			split := timerAMemory(t, data, mode)
			one.advanceTimerA(1_000_000)
			for clock := uint64(1); clock < 1_000_000; clock += 71 {
				split.advanceTimerA(clock)
			}
			split.advanceTimerA(1_000_000)
			if one.mfpTimerAFraction != split.mfpTimerAFraction || one.mfpTAMain != split.mfpTAMain || one.mfpTimerATimeouts != split.mfpTimerATimeouts {
				t.Fatal("phase depends on batching")
			}
		}
	}
}

func TestTimerAActiveReloadStopRestartAndReentry(t *testing.T) {
	m := timerAMemory(t, 3, 1)
	write := func(a uint32, v byte, clock uint64) {
		t.Helper()
		if wait, err := m.WriteByteAt(a, v, m68k.BusAccess{Clock: clock, FunctionCode: 5}); err != nil || wait != 4 {
			t.Fatalf("write: %d %v", wait, err)
		}
	}
	write(MFPTADR, 7, 14)
	if m.mfpTAMain != 2 || m.mfpTADR != 7 {
		t.Fatal("active write replaced main counter")
	}
	deadline := m.timerADeadline()
	write(MFPTACR, 1, 20)
	if m.timerADeadline() != deadline {
		t.Fatal("same mode restarted timer")
	}
	m.advanceTimerA(40)
	if m.mfpTAMain != 7 || m.mfpTimerATimeouts != 1 {
		t.Fatal("deferred reload missing")
	}
	write(MFPTACR, 0, 55)
	held := m.mfpTAMain
	m.advanceTimerA(1000)
	if m.mfpTAMain != held || m.mfpTimerAFraction != 0 {
		t.Fatal("stopped timer advanced")
	}
	write(MFPTACR, 1, 1000)
	if m.timerADeadline() != 1000+(timerACount(held)*4*15667+4799)/4800 {
		t.Fatal("restart counter/phase")
	}
	before := m.mfpTACR
	if err := m.WriteByteFC(MFPTACR, 2, 5); err == nil || m.mfpTACR != before {
		t.Fatal("active prescaler switch accepted")
	}
	if err := m.WriteByteFC(MFPTACR, 8, 5); err == nil || m.mfpTACR != before {
		t.Fatal("event mode accepted")
	}
	if err := m.WriteByteFC(MFPTACR, 0, 1); err == nil || m.mfpTACR != before {
		t.Fatal("user write accepted")
	}
	if err := m.WriteWord(MFPTADR, 0, 5); err == nil {
		t.Fatal("word accepted")
	}
	m.mfpTimerAOutput = true
	write(MFPTACR, 0x11, 1000)
	if m.mfpTimerAOutput || m.mfpTACR != 1 {
		t.Fatal("output reset changed mode")
	}
	m.ColdReset()
	if m.mfpTimerATimeouts != 0 || m.mfpTimerAFraction != 0 || m.mfpTimerAOutput || m.mfpTACR != 0 {
		t.Fatal("reset retained timer")
	}
}

func TestTimerAInterruptEnableMaskAndPriority(t *testing.T) {
	m := timerAMemory(t, 3, 1)
	m.advanceTimerA(40)
	if m.mfpIPRA != 0 {
		t.Fatal("disabled channel became pending")
	}
	if err := m.WriteByteFC(MFPIERA, 0x20, 5); err != nil {
		t.Fatal(err)
	}
	m.advanceTimerA(80)
	if m.mfpIPRA != 0x20 {
		t.Fatal("masked event not pending")
	}
	if err := m.WriteByteFC(MFPIMRA, 0x20, 5); err != nil {
		t.Fatal(err)
	}
	machine := &Machine{Memory: m}
	m.mfpIERB, m.mfpIMRB, m.mfpIPRB = 0x70, 0x70, 0x70
	if ch, ok := machine.mfpInterruptChannel(); !ok || ch != 13 {
		t.Fatal("A did not outrank B")
	}
	m.mfpVR = 0x48
	if m.mfpVector(13) != 77 {
		t.Fatal("wrong vector")
	}
	m.acknowledgeMFP(13)
	if m.mfpIPRA != 0 || m.mfpISRA != 0x20 {
		t.Fatal("IACK side effects")
	}
	if _, ok := machine.mfpInterruptChannel(); ok {
		t.Fatal("ISR did not block B")
	}
	m.mfpIPRA = 0x20
	if _, ok := machine.mfpInterruptChannel(); ok {
		t.Fatal("ISR did not block A")
	}
	if err := m.WriteByteFC(MFPIERA, 0, 5); err != nil {
		t.Fatal(err)
	}
	if m.mfpIPRA != 0 || m.mfpISRA != 0x20 {
		t.Fatal("disable corrupted pending/ISR")
	}
	if err := m.WriteByteFC(MFPISRA, 0xdf, 5); err != nil {
		t.Fatal(err)
	}
	if ch, ok := machine.mfpInterruptChannel(); !ok || ch != 6 {
		t.Fatal("EOI did not release B")
	}
}

func TestTimerAWakesStoppedCPUAtDeadline(t *testing.T) {
	m, err := NewMachine(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	// 合成 CPU 路徑：STOP #$2300，Timer A vector 77 指向 NOP handler。
	m.Memory.mmuConfig = 0x05
	for _, w := range []struct {
		a uint32
		v uint16
	}{{77 * 4, 0}, {77*4 + 2, 0x2000}, {0x2000, 0x4e71}, {0x2002, 0x4e71}, {0x2004, 0x4e71}} {
		if err := m.Memory.WriteWord(w.a, w.v, 5); err != nil {
			t.Fatal(err)
		}
	}
	m.CPU.State = m68k.State{PC: 0x1004, SR: 0x2300, SSP: 0x8000, Prefetch: [2]uint16{0x4e72, 0x2300}}
	m.nextVBLClock = 100000
	m.vblFrameClocks = colorST50HzFrameClocks
	m.Memory.mfpVR = 0x48
	m.Memory.mfpIERA = 0x20
	m.Memory.mfpIMRA = 0x20
	if err := m.Memory.WriteByteFC(MFPTADR, 112, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Memory.WriteByteAt(MFPTACR, 1, m68k.BusAccess{FunctionCode: 5}); err != nil {
		t.Fatal(err)
	}
	deadline := m.Memory.timerADeadline()
	if _, err := m.Step(); err != nil || !m.CPU.IsStopped() {
		t.Fatalf("STOP: %v", err)
	}
	result, err := m.Step()
	if err != nil {
		t.Fatal(err)
	}
	if m.CPU.IsStopped() || m.Interrupts != 1 || m.CPU.State.PC != 0x2004 || m.Memory.mfpISRA != 0x20 || m.Memory.mfpIPRA != 0 {
		t.Fatalf("wake result=%+v state=%+v IRQ=%d", result, m.CPU.State, m.Interrupts)
	}
	if m.Clocks < deadline || m.Clocks >= m.nextVBLClock || m.Memory.mfpTimerATimeouts != 1 {
		t.Fatalf("wake clocks=%d deadline=%d", m.Clocks, deadline)
	}
}
