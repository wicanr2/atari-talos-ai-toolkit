package st

// 規格 155：50 Hz 標準畫面 DE 下降緣，非 HBL 的全部掃描線。
func (m *Memory) nextTimerBEdge(clock uint64) uint64 {
	origin := m.mfpTimerBFrameOrigin
	if clock >= origin {
		origin += (clock - origin) / colorST50HzFrameClocks * colorST50HzFrameClocks
	}
	first := origin + 63*512 + 400
	if clock < first {
		return first
	}
	n := (clock-first)/512 + 1
	if n < 200 {
		return first + n*512
	}
	return first + colorST50HzFrameClocks
}

func (m *Memory) advanceTimerB(clock uint64) {
	if m == nil || clock <= m.mfpTimerBLastClock {
		return
	}
	m.mfpTimerBLastClock = clock
	if m.mfpTBCR != 8 {
		return
	}
	for m.mfpTimerBNextEdge != 0 && m.mfpTimerBNextEdge <= clock {
		edge := m.mfpTimerBNextEdge
		m.mfpTimerBEvents++
		count := timerACount(m.mfpTBMain)
		if count == 1 {
			m.mfpTBMain = m.mfpTBDR
			m.mfpTimerBOutput = !m.mfpTimerBOutput
			if m.mfpIERA&1 != 0 {
				m.mfpIPRA |= 1
			}
		} else {
			m.mfpTBMain = byte(count - 1)
		}
		m.mfpTimerBNextEdge = m.nextTimerBEdge(edge)
	}
}

func (m *Machine) timerBCanWake() bool {
	return m.Memory != nil && m.CPU.State.SR&0x700 < 0x600 &&
		m.Memory.mfpIERA&m.Memory.mfpIMRA&1 != 0 && m.Memory.mfpISRA == 0
}

func (m *Memory) timerBDeadline() uint64 {
	if m == nil || m.mfpTBCR != 8 || m.mfpTimerBNextEdge == 0 {
		return 0
	}
	edge := m.mfpTimerBNextEdge
	for n := uint64(1); n < timerACount(m.mfpTBMain); n++ {
		edge = m.nextTimerBEdge(edge)
	}
	return edge
}
