package st

import "fmt"

// IKBD 的鍵盤掃描碼：make 從 1 開始，break 是 make 或上 $80（規格 144）。
const (
	ikbdScanCodeMin   = 0x01
	ikbdScanCodeMax   = 0x72
	ikbdScanCodeBreak = 0x80
)

// QueueKey 把一次按鍵事件交給 IKBD。鍵盤一次只送一個位元組：按下送 make、
// 放開送 make | $80。
func (m *Memory) QueueKey(scanCode byte, pressed bool) error {
	if scanCode < ikbdScanCodeMin || scanCode > ikbdScanCodeMax {
		return fmt.Errorf("st: ikbd scan code %#02x is out of range", scanCode)
	}
	if m.ikbdUplinkCount >= ikbdUplinkCapacity {
		return fmt.Errorf("st: ikbd uplink queue is full")
	}
	value := scanCode
	if !pressed {
		value |= ikbdScanCodeBreak
	}
	m.ikbdUplink[(int(m.ikbdUplinkHead)+int(m.ikbdUplinkCount))%ikbdUplinkCapacity] = value
	m.ikbdUplinkCount++
	return nil
}

func (m *Machine) QueueKey(scanCode byte, pressed bool) error {
	if m.Memory == nil {
		return fmt.Errorf("st: machine has no memory")
	}
	if err := m.Memory.QueueKey(scanCode, pressed); err != nil {
		return err
	}
	if m.nextIKBDUplinkClock == 0 {
		m.nextIKBDUplinkClock = m.Clocks + ikbdUplinkByteClocks
	}
	return nil
}
