package st

import "testing"

// TestIKBDKeyboardScanCodes covers spec 145: make is the scan code, break is
// the scan code with bit 7 set, one byte each.
func TestIKBDKeyboardScanCodes(t *testing.T) {
	memory := ikbdMouseReady(t)
	for _, test := range []struct {
		name     string
		scanCode byte
		want     [2]byte
	}{
		{"Return", 0x1c, [2]byte{0x1c, 0x9c}},
		{"數字 1", 0x02, [2]byte{0x02, 0x82}},
		{"最小的掃描碼", 0x01, [2]byte{0x01, 0x81}},
		{"最大的掃描碼", 0x72, [2]byte{0x72, 0xf2}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := memory.QueueKey(test.scanCode, true); err != nil {
				t.Fatal(err)
			}
			ikbdDrain(t, memory, test.want[0])
			if err := memory.QueueKey(test.scanCode, false); err != nil {
				t.Fatal(err)
			}
			ikbdDrain(t, memory, test.want[1])
		})
	}
}

func TestIKBDKeyboardRefusesWhatItDoesNotModel(t *testing.T) {
	memory := ikbdMouseReady(t)
	for _, scanCode := range []byte{0x00, 0x73, 0x74, 0x75, 0x80, 0xff} {
		if err := memory.QueueKey(scanCode, true); err == nil {
			t.Errorf("掃描碼 %#02x 竟然被接受", scanCode)
		}
	}
	if memory.ikbdUplinkCount != 0 {
		t.Fatalf("被拒絕的掃描碼還是排進佇列了：%d", memory.ikbdUplinkCount)
	}
	// 佇列滿了就不收
	for i := 0; i < ikbdUplinkCapacity; i++ {
		if err := memory.QueueKey(0x1c, true); err != nil {
			t.Fatal(err)
		}
	}
	if err := memory.QueueKey(0x1c, true); err == nil {
		t.Error("佇列滿了還收得下")
	}
}

// 鍵盤與滑鼠共用同一條線，先進先出。
func TestIKBDKeyboardAndMouseShareTheQueue(t *testing.T) {
	memory := ikbdMouseReady(t)
	if err := memory.QueueKey(0x1c, true); err != nil {
		t.Fatal(err)
	}
	if err := memory.QueueMouseMotion(2, -2, false, false); err != nil {
		t.Fatal(err)
	}
	if err := memory.QueueKey(0x1c, false); err != nil {
		t.Fatal(err)
	}
	ikbdDrain(t, memory, 0x1c, 0xf8, 0x02, 0xfe, 0x9c)
}
