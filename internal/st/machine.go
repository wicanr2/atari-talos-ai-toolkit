package st

import "github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"

const firstColorSTVBLClock uint64 = 263*508 + 64
const colorST60HzFrameClocks uint64 = 263 * 508
const colorST50HzFrameClocks uint64 = 313 * 512
const colorSTLineZero50HzExtension uint64 = 262 * (512 - 508)

type Machine struct {
	CPU                    m68k.CPU
	Memory                 *Memory
	Instructions           uint64
	Interrupts             uint64
	Clocks                 uint64
	nextVBLClock           uint64
	vblFrameClocks         uint64
	vblPending             bool
	aciaClockStarted       bool
	nextACIABitClock       uint64
	ikbdResetRXDeadline    uint64
	ikbdResetRXClock       uint64
	ikbdSecondTXClock      uint64
	timerCClockStarted     bool
	timerCPeriods          uint64
	nextTimerCClock        uint64
	timerDClockStarted     bool
	timerDPeriods          uint64
	nextTimerDClock        uint64
	fdcRestoreClockStarted bool
	nextFDCRestoreClock    uint64
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
	m.timerCClockStarted = false
	m.timerCPeriods = 0
	m.nextTimerCClock = 0
	m.timerDClockStarted = false
	m.timerDPeriods = 0
	m.nextTimerDClock = 0
	m.fdcRestoreClockStarted = false
	m.nextFDCRestoreClock = 0
	return nil
}

func (m *Machine) Step() (m68k.StepResult, error) {
	idle := uint64(0)
	if m.CPU.IsStopped() && !m.vblPending && m.Clocks < m.nextVBLClock {
		idle = m.nextVBLClock - m.Clocks
		m.raiseVBL()
	}
	if channel, pending := m.mfpBInterruptChannel(); pending {
		result, accepted, err := m.CPU.AcceptVectoredInterruptAt(6, m.Memory.mfpVector(channel), m.Clocks+idle)
		if err != nil {
			return result, err
		}
		if accepted {
			result = prependIdle(result, uint32(idle))
			m.Memory.acknowledgeMFPB(channel)
			m.Interrupts++
			m.Clocks += uint64(result.Clocks)
			m.advanceClockedDevices()
			return result, nil
		}
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
	if !m.fdcRestoreClockStarted && m.Memory != nil && m.Memory.fdcRestorePending &&
		m.Memory.fdcRestoreStartClock != 0 {
		m.fdcRestoreClockStarted = true
		m.nextFDCRestoreClock = fdcRestoreDeadline(m.Memory.fdcRestoreStartClock)
	}
	if m.fdcRestoreClockStarted && m.Memory != nil && m.Memory.fdcRestorePending &&
		m.Clocks >= m.nextFDCRestoreClock {
		m.Memory.completeFDCRestore()
		m.fdcRestoreClockStarted = false
		m.nextFDCRestoreClock = 0
	}
	if m.timerDClockStarted && m.Memory != nil && !m.Memory.mfpTimerDStart {
		m.timerDClockStarted = false
		m.timerDPeriods = 0
		m.nextTimerDClock = 0
	}
	if !m.timerCClockStarted && m.Memory != nil && m.Memory.mfpTimerCStartClock != 0 {
		m.timerCClockStarted = true
		m.timerCPeriods = 1
		m.nextTimerCClock = timerCDeadline(m.Memory.mfpTimerCStartClock, m.timerCPeriods)
	}
	if !m.timerCClockStarted && m.Memory != nil && m.Memory.mfpTimerCStart {
		m.Memory.mfpTimerCStartClock = m.Clocks
		m.timerCClockStarted = true
		m.timerCPeriods = 1
		m.nextTimerCClock = timerCDeadline(m.Memory.mfpTimerCStartClock, m.timerCPeriods)
	}
	for m.timerCClockStarted && m.Clocks >= m.nextTimerCClock {
		m.Memory.mfpIPRB |= 0x20
		m.timerCPeriods++
		m.nextTimerCClock = timerCDeadline(m.Memory.mfpTimerCStartClock, m.timerCPeriods)
	}
	if !m.timerDClockStarted && m.Memory != nil && m.Memory.mfpTimerDStart &&
		m.Memory.mfpTCDCR&0x07 == 2 && m.Memory.mfpTimerDStartClock != 0 {
		m.timerDClockStarted = true
		m.timerDPeriods = 1
		m.nextTimerDClock = timerDDeadline(m.Memory.mfpTimerDStartClock, m.timerDPeriods)
	}
	if !m.timerDClockStarted && m.Memory != nil && m.Memory.mfpTimerDSystemStage == 8 &&
		m.Memory.mfpTimerDStart && m.Memory.mfpTCDCR&0x07 == 2 {
		// MOVE.B has not yet migrated every effective-address path to TimedBus.
		// Keep this fallback local until that CPU slice supplies the access clock.
		m.Memory.mfpTimerDStartClock = m.Clocks
		m.timerDClockStarted = true
		m.timerDPeriods = 1
		m.nextTimerDClock = timerDDeadline(m.Memory.mfpTimerDStartClock, m.timerDPeriods)
	}
	for m.timerDClockStarted && m.Clocks >= m.nextTimerDClock {
		m.Memory.mfpIPRB |= 0x10
		m.timerDPeriods++
		m.nextTimerDClock = timerDDeadline(m.Memory.mfpTimerDStartClock, m.timerDPeriods)
	}
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

func fdcRestoreDeadline(start uint64) uint64 {
	const numerator uint64 = 728 * 8021248
	const denominator uint64 = 8000000
	return start + numerator/denominator
}

func (m *Machine) mfpBInterruptChannel() (uint8, bool) {
	if m.Memory == nil {
		return 0, false
	}
	requests := m.Memory.mfpIPRB & m.Memory.mfpIERB & m.Memory.mfpIMRB & 0x30
	for channel := uint8(5); ; channel-- {
		bit := byte(1 << channel)
		if requests&bit != 0 && m.Memory.mfpISRB&^(bit-1) == 0 {
			return channel, true
		}
		if channel == 4 {
			break
		}
	}
	return 0, false
}

func timerCDeadline(start, periods uint64) uint64 {
	const numerator uint64 = 12288 * 8021248
	const denominator uint64 = 2457600
	return start + periods*numerator/denominator
}

func timerDDeadline(start, periods uint64) uint64 {
	const numerator uint64 = 2560 * 8021248
	const denominator uint64 = 2457600
	return start + periods*numerator/denominator
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
