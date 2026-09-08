package m68k

import "testing"

func TestSingleStepMOVEPWord(t *testing.T) { testSingleStepCorpus(t, "MOVEP.w.json.bin") }
func TestSingleStepMOVEPLong(t *testing.T) { testSingleStepCorpus(t, "MOVEP.l.json.bin") }

func TestMOVEPTimedWaitPropagation(t *testing.T) {
	b := &timedRecordingBus{SparseMemory: SparseMemory{0x104: 0x4e, 0x105: 0x71, 0x106: 0x4e, 0x107: 0x71, 0x201: 0x12, 0x203: 0x34}, wait: 2}
	c := CPU{Bus: b, State: State{SR: supervisor, PC: 0x104, Prefetch: [2]uint16{0x0108, 1}}}
	c.State.A[0] = 0x200
	r, err := c.StepAt(100)
	if err != nil || r.Clocks != 24 || c.State.D[0] != 0x1234 {
		t.Fatalf("result=%+v D0=%x err=%v", r, c.State.D[0], err)
	}
	if len(b.accesses) != 4 {
		t.Fatalf("accesses=%v", b.accesses)
	}
	for i, a := range b.accesses {
		if a.Clock != 100+uint64(i)*6 {
			t.Fatalf("phase %d clock=%d", i, a.Clock)
		}
	}
}
