package st

import "fmt"

// 相對滑鼠位置紀錄的表頭：`%111110xy`，x 是左鍵（bit 1）、y 是右鍵（bit 0）。
// 見規格 142 引的兩份 IKBD 協定文件。
const (
	ikbdMousePacketFlag  = 0xf8
	ikbdMouseLeftBit     = 0x02
	ikbdMouseRightBit    = 0x01
	ikbdUplinkCapacity   = 6 // 兩個封包
	ikbdUplinkByteClocks = 10 * 1024
)

// QueueMouseMotion 把一次滑鼠事件交給 IKBD：兩軸的相對位移與左右鍵狀態。
// 門檻 1 之下，只要有位移或按鍵狀態改變就會排出一個三位元組的封包（規格 142）。
func (m *Memory) QueueMouseMotion(deltaX, deltaY int, left, right bool) error {
	if !m.ikbdRelativeMouse {
		return fmt.Errorf("st: ikbd is not in relative mouse mode")
	}
	if !m.ikbdYAxisUp {
		return fmt.Errorf("st: ikbd Y origin is not at the top")
	}
	if m.ikbdMouseThreshold != [2]byte{1, 1} {
		return fmt.Errorf("st: ikbd mouse threshold %v is not modeled", m.ikbdMouseThreshold)
	}
	m.ikbdMouseAccumX += deltaX
	m.ikbdMouseAccumY += deltaY
	moved := m.ikbdMouseAccumX != 0 || m.ikbdMouseAccumY != 0
	changed := left != m.ikbdMouseLeft || right != m.ikbdMouseRight
	m.ikbdMouseLeft, m.ikbdMouseRight = left, right
	if !moved && !changed {
		return nil
	}
	if m.ikbdMouseAccumX < -128 || m.ikbdMouseAccumX > 127 ||
		m.ikbdMouseAccumY < -128 || m.ikbdMouseAccumY > 127 {
		return fmt.Errorf("st: ikbd mouse delta %d,%d does not fit one packet",
			m.ikbdMouseAccumX, m.ikbdMouseAccumY)
	}
	header := byte(ikbdMousePacketFlag)
	if left {
		header |= ikbdMouseLeftBit
	}
	if right {
		header |= ikbdMouseRightBit
	}
	packet := [3]byte{header, byte(int8(m.ikbdMouseAccumX)), byte(int8(m.ikbdMouseAccumY))}
	if int(m.ikbdUplinkCount)+len(packet) > ikbdUplinkCapacity {
		return fmt.Errorf("st: ikbd uplink queue is full")
	}
	for _, value := range packet {
		m.ikbdUplink[(int(m.ikbdUplinkHead)+int(m.ikbdUplinkCount))%ikbdUplinkCapacity] = value
		m.ikbdUplinkCount++
	}
	m.ikbdMouseAccumX, m.ikbdMouseAccumY = 0, 0
	return nil
}

// nextIKBDUplinkByte 回報佇列最前面那個位元組。
func (m *Memory) nextIKBDUplinkByte() (byte, bool) {
	if m.ikbdUplinkCount == 0 {
		return 0, false
	}
	return m.ikbdUplink[m.ikbdUplinkHead], true
}

// deliverIKBDUplinkByte 把佇列最前面那個位元組推進 RDR。RDRF 還沒被主機清掉就
// 代表 overrun，本切片不做，失敗即關閉（規格 142）。
func (m *Memory) deliverIKBDUplinkByte() error {
	value, ok := m.nextIKBDUplinkByte()
	if !ok {
		return nil
	}
	if !m.ikbdACIAConfigured {
		return fmt.Errorf("st: ikbd uplink byte %#02x with the ACIA unconfigured", value)
	}
	if m.ikbdACIAStatus&1 != 0 {
		return fmt.Errorf("st: ikbd uplink byte %#02x while RDRF is still set", value)
	}
	m.ikbdUplinkHead = (m.ikbdUplinkHead + 1) % ikbdUplinkCapacity
	m.ikbdUplinkCount--
	m.ikbdUplinkActive = true
	m.ikbdACIARDR = value
	m.ikbdACIAStatus |= 0x81
	m.mfpGPIPIn &^= 0x10
	m.mfpIPRB |= 0x40
	return nil
}

// QueueMouseMotion 是 Machine 這一層的入口：排好封包之後，第一個位元組排在
// 十個位元時間之後送出。
func (m *Machine) QueueMouseMotion(deltaX, deltaY int, left, right bool) error {
	if m.Memory == nil {
		return fmt.Errorf("st: machine has no memory")
	}
	if err := m.Memory.QueueMouseMotion(deltaX, deltaY, left, right); err != nil {
		return err
	}
	if m.Memory.ikbdUplinkCount != 0 && m.nextIKBDUplinkClock == 0 {
		m.nextIKBDUplinkClock = m.Clocks + ikbdUplinkByteClocks
	}
	return nil
}

// QueueMouseButtons 只送按鍵狀態的變化。IKBD 在相對模式下，按鍵按下或放開一樣
// 會發一個相對位置封包，位移就是零（規格 143）。
func (m *Memory) QueueMouseButtons(left, right bool) error {
	return m.QueueMouseMotion(0, 0, left, right)
}

func (m *Machine) QueueMouseButtons(left, right bool) error {
	return m.QueueMouseMotion(0, 0, left, right)
}
