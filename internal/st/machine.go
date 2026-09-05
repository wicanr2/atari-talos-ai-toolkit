package st

import "github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"

const firstColorSTVBLClock uint64 = 263*508 + 64
const colorST60HzFrameClocks uint64 = 263 * 508
const colorST50HzFrameClocks uint64 = 313 * 512
const colorSTLineZero50HzExtension uint64 = 262 * (512 - 508)

type Machine struct {
	CPU            m68k.CPU
	Memory         *Memory
	Instructions   uint64
	Interrupts     uint64
	Clocks         uint64
	nextVBLClock   uint64
	vblFrameClocks uint64
	vblPending     bool
}

func NewMachine(ramSize int, tosROM []byte) (*Machine, error) {
	memory, err := NewMemory(ramSize, tosROM)
	if err != nil {
		return nil, err
	}
	machine := &Machine{Memory: memory}
	machine.CPU.Bus = memory
	return machine, nil
}

func (m *Machine) Reset() error {
	m.Memory.ColdReset()
	if err := m.CPU.Reset(); err != nil {
		return err
	}
	m.Instructions = 0
	m.Interrupts = 0
	m.Clocks = 0
	m.nextVBLClock = firstColorSTVBLClock
	m.vblFrameClocks = colorST60HzFrameClocks
	m.vblPending = false
	return nil
}

func (m *Machine) Step() (m68k.StepResult, error) {
	idle := uint64(0)
	if m.CPU.IsStopped() && !m.vblPending && m.Clocks < m.nextVBLClock {
		idle = m.nextVBLClock - m.Clocks
		m.vblPending = true
		m.nextVBLClock += m.vblFrameClocks
	}
	if m.vblPending {
		acceptEpoch := m.Clocks + idle
		iack := videoIACKExtraClocks(acceptEpoch)
		result, accepted, err := m.CPU.AcceptAutovectorAt(4, acceptEpoch+uint64(iack))
		if err != nil {
			return result, err
		}
		if accepted {
			result = prependIdle(result, uint32(idle)+iack)
			m.vblPending = false
			m.Interrupts++
			m.Clocks += uint64(result.Clocks)
			return result, nil
		}
	}
	result, err := m.CPU.StepAt(m.Clocks)
	if err != nil {
		return result, err
	}
	m.Instructions++
	m.Clocks += uint64(result.Clocks)
	if m.Memory != nil && m.Memory.videoSync50Transition {
		if m.nextVBLClock != firstColorSTVBLClock+3*colorST60HzFrameClocks {
			return result, &BusFault{Address: VideoSyncMode, FunctionCode: 5, Write: true, Size: 1, Reason: FaultUnsupportedDeviceState}
		}
		m.Memory.videoSync50Transition = false
		m.nextVBLClock += colorSTLineZero50HzExtension
		m.vblFrameClocks = colorST50HzFrameClocks
	}
	m.raiseDueVBL()
	return result, nil
}

func (m *Machine) raiseDueVBL() {
	if m.nextVBLClock != 0 && m.Clocks >= m.nextVBLClock {
		m.vblPending = true
		m.nextVBLClock += m.vblFrameClocks
	}
}

func videoIACKExtraClocks(epoch uint64) uint32 {
	wait := (10 - (epoch+12)%10) % 10
	return uint32(10 + wait)
}

func prependIdle(result m68k.StepResult, clocks uint32) m68k.StepResult {
	if clocks == 0 {
		return result
	}
	timeline := make([]m68k.BusPhase, 0, len(result.Timeline)+1)
	timeline = append(timeline, m68k.BusPhase{Cycles: clocks})
	for _, phase := range result.Timeline {
		phase.Offset += clocks
		timeline = append(timeline, phase)
	}
	result.Clocks += clocks
	result.Timeline = timeline
	return result
}
