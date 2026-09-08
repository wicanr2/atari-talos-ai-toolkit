package st

// 規格 153：MC68901 §6。晶振起始相位採 hardware-spec approximation。
var timerAPrescalers = [8]uint64{0, 4, 10, 16, 50, 64, 100, 200}

func timerACount(value byte) uint64 {
	if value == 0 {
		return 256
	}
	return uint64(value)
}

func (m *Memory) advanceTimerA(clock uint64) {
	if m == nil || clock <= m.mfpTimerALastClock {
		return
	}
	delta := clock - m.mfpTimerALastClock
	m.mfpTimerALastClock = clock
	if m.mfpTACR == 0 || m.mfpTACR > 7 {
		return
	}
	quantum := timerAPrescalers[m.mfpTACR] * 15667
	// 先除再乘，避免長時間跨度的 delta*4800 溢位。
	partial := delta%quantum*4800 + m.mfpTimerAFraction
	pulses := delta/quantum*4800 + partial/quantum
	m.mfpTimerAFraction = partial % quantum
	remaining := timerACount(m.mfpTAMain)
	if pulses < remaining {
		m.mfpTAMain = byte(remaining - pulses)
		return
	}
	pulses -= remaining
	reload := timerACount(m.mfpTADR)
	timeouts := 1 + pulses/reload
	m.mfpTimerATimeouts += timeouts
	if timeouts&1 != 0 {
		m.mfpTimerAOutput = !m.mfpTimerAOutput
	}
	m.mfpTAMain = byte(reload - pulses%reload)
	if m.mfpIERA&0x20 != 0 {
		m.mfpIPRA |= 0x20
	}
}

func (m *Memory) timerADeadline() uint64 {
	if m == nil || m.mfpTACR == 0 || m.mfpTACR > 7 {
		return 0
	}
	numerator := timerACount(m.mfpTAMain)*timerAPrescalers[m.mfpTACR]*15667 - m.mfpTimerAFraction
	return m.mfpTimerALastClock + (numerator+4799)/4800
}

func (m *Machine) timerACanWake() bool {
	return m.Memory != nil && m.CPU.State.SR&0x0700 < 0x0600 &&
		m.Memory.mfpIERA&m.Memory.mfpIMRA&0x20 != 0 && m.Memory.mfpISRA&0xe0 == 0
}

func (m *Machine) mfpInterruptChannel() (uint8, bool) {
	if m.Memory != nil && m.Memory.mfpIPRA&m.Memory.mfpIERA&m.Memory.mfpIMRA&0x20 != 0 &&
		m.Memory.mfpISRA&0xe0 == 0 {
		return 13, true
	}
	return m.mfpBInterruptChannel()
}

func (m *Memory) acknowledgeMFP(channel uint8) {
	if channel == 13 {
		m.mfpTimerAAcknowledged++
		m.mfpIPRA &^= 0x20
		if m.mfpVR&8 != 0 {
			m.mfpISRA |= 0x20
		}
		return
	}
	m.acknowledgeMFPB(channel)
}
