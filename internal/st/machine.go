package st

import "github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"

const firstColorSTVBLClock uint64 = 263*508 + 64
const colorST60HzFrameClocks uint64 = 263 * 508
const colorST50HzFrameClocks uint64 = 313 * 512
const colorSTLineZero50HzExtension uint64 = 262 * (512 - 508)

type Machine struct {
	CPU                 m68k.CPU
	Memory              *Memory
	Instructions        uint64
	Interrupts          uint64
	Clocks              uint64
	nextVBLClock        uint64
	vblFrameClocks      uint64
	vblPending          bool
	aciaClockStarted    bool
	nextACIABitClock    uint64
	ikbdResetRXDeadline uint64
	ikbdResetRXClock    uint64
	ikbdSecondTXClock   uint64
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
	m.aciaClockStarted = false
	m.nextACIABitClock = 0
	m.ikbdResetRXDeadline = 0
	m.ikbdResetRXClock = 0
	m.ikbdSecondTXClock = 0
	return nil
}

func (m *Machine) Step() (m68k.StepResult, error) {
	idle := uint64(0)
	if m.CPU.IsStopped() && !m.vblPending && m.Clocks < m.nextVBLClock {
		idle = m.nextVBLClock - m.Clocks
		m.raiseVBL()
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
			m.advanceClockedDevices()
			return result, nil
		}
	}
	result, err := m.CPU.StepAt(m.Clocks)
	if err != nil {
		return result, err
	}
	m.Instructions++
	m.Clocks += uint64(result.Clocks)
	m.advanceClockedDevices()
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

func (m *Machine) advanceClockedDevices() {
	if !m.aciaClockStarted && m.Memory != nil && m.Memory.ikbdACIAConfigured {
		m.aciaClockStarted = true
		m.nextACIABitClock = m.Clocks + 1024
	}
	m.advanceDueACIAClocks()
	if m.ikbdResetRXDeadline != 0 && m.Clocks >= m.ikbdResetRXDeadline {
		m.ikbdResetRXClock = m.ikbdResetRXDeadline
		m.Memory.deliverIKBDResetResponse()
		m.ikbdResetRXDeadline = 0
	}
}

func (m *Machine) advanceDueACIAClocks() {
	for m.aciaClockStarted && m.nextACIABitClock != 0 && m.Clocks >= m.nextACIABitClock {
		secondPending := m.Memory.ikbdACIATDR == 1 && m.Memory.ikbdACIATXPending
		m.Memory.advanceIKBDACIAClock()
		if secondPending && !m.Memory.ikbdACIATXPending {
			m.ikbdSecondTXClock = m.nextACIABitClock
		}
		if m.Memory.ikbdResetCommandDone {
			m.ikbdResetRXDeadline = m.nextACIABitClock + 513024
			m.Memory.ikbdResetCommandDone = false
		}
		m.nextACIABitClock += 1024
	}
}

func (m *Machine) raiseDueVBL() {
	if m.nextVBLClock != 0 && m.Clocks >= m.nextVBLClock {
		m.raiseVBL()
	}
}

func (m *Machine) raiseVBL() {
	if m.Memory != nil {
		m.Memory.reloadVideoBaseOnVBL()
	}
	m.vblPending = true
	m.nextVBLClock += m.vblFrameClocks
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
