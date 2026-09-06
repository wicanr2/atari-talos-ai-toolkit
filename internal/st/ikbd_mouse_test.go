package st

import "testing"

// ikbdMouseReady 把機器設到 EmuTOS 開機送完 Initmous 四條命令之後的狀態：
// 相對回報、Y 原點在上、兩軸門檻 1、ACIA 已設定。這是「直接跳到事件」的做法，
// 不從 reset 跑一千四百萬條指令去等它。
func ikbdMouseReady(t *testing.T) *Memory {
	t.Helper()
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.ikbdACIAConfigured = true
	memory.ikbdACIAStatus = 2
	memory.ikbdRelativeMouse = true
	memory.ikbdYAxisUp = true
	memory.ikbdMouseThreshold = [2]byte{1, 1}
	return memory
}

func ikbdDrain(t *testing.T, memory *Memory, want ...byte) {
	t.Helper()
	for index, expected := range want {
		if err := memory.deliverIKBDUplinkByte(); err != nil {
			t.Fatalf("第 %d 個位元組投遞失敗：%v", index, err)
		}
		if memory.ikbdACIAStatus&0x81 != 0x81 || memory.mfpGPIPIn&0x10 != 0 ||
			memory.mfpIPRB&0x40 == 0 {
			t.Fatalf("第 %d 個位元組的中斷效果：status=%02x gpip=%02x iprb=%02x",
				index, memory.ikbdACIAStatus, memory.mfpGPIPIn, memory.mfpIPRB)
		}
		got, err := memory.ReadByteFC(IKBDACIAData, 5)
		if err != nil || got != expected {
			t.Fatalf("第 %d 個位元組讀到 %#02x（要 %#02x）err=%v", index, got, expected, err)
		}
		if memory.ikbdUplinkActive {
			t.Fatalf("第 %d 個位元組讀走之後 uplink 還是 active", index)
		}
	}
	if memory.ikbdUplinkCount != 0 {
		t.Fatalf("佇列還剩 %d 個位元組", memory.ikbdUplinkCount)
	}
}

// TestIKBDRelativeMousePacket covers the three-byte record: header %111110xy
// with the left button in bit 1 and the right button in bit 0, then two
// two's-complement deltas.
func TestIKBDRelativeMousePacket(t *testing.T) {
	for _, test := range []struct {
		name        string
		dx, dy      int
		left, right bool
		want        [3]byte
	}{
		{"沒按鍵、正位移", 20, 10, false, false, [3]byte{0xf8, 0x14, 0x0a}},
		{"只有左鍵", 1, 0, true, false, [3]byte{0xfa, 0x01, 0x00}},
		{"只有右鍵", 0, 1, false, true, [3]byte{0xf9, 0x00, 0x01}},
		{"兩鍵都按", 0, 0, true, true, [3]byte{0xfb, 0x00, 0x00}},
		{"負位移取二補數", -1, -128, false, false, [3]byte{0xf8, 0xff, 0x80}},
		{"兩軸各到上界", 127, 127, false, false, [3]byte{0xf8, 0x7f, 0x7f}},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory := ikbdMouseReady(t)
			if err := memory.QueueMouseMotion(test.dx, test.dy, test.left, test.right); err != nil {
				t.Fatal(err)
			}
			ikbdDrain(t, memory, test.want[0], test.want[1], test.want[2])
		})
	}
}

// 位移累加到送出為止；沒有位移也沒有按鍵變化就不發封包。
func TestIKBDMouseAccumulatesUntilItSends(t *testing.T) {
	memory := ikbdMouseReady(t)
	if err := memory.QueueMouseMotion(0, 0, false, false); err != nil {
		t.Fatal(err)
	}
	if memory.ikbdUplinkCount != 0 {
		t.Fatalf("沒動也沒按鍵竟然發了封包：%d 個位元組", memory.ikbdUplinkCount)
	}
	if err := memory.QueueMouseMotion(3, -4, false, false); err != nil {
		t.Fatal(err)
	}
	ikbdDrain(t, memory, 0xf8, 0x03, 0xfc)
	if memory.ikbdMouseAccumX != 0 || memory.ikbdMouseAccumY != 0 {
		t.Errorf("送出後累加器沒歸零：%d,%d", memory.ikbdMouseAccumX, memory.ikbdMouseAccumY)
	}
}

// 本切片只做門檻 1、Y 原點在上、相對模式；其餘一律失敗即關閉。
func TestIKBDMouseRefusesWhatItDoesNotModel(t *testing.T) {
	t.Run("門檻不是 1", func(t *testing.T) {
		memory := ikbdMouseReady(t)
		memory.ikbdMouseThreshold = [2]byte{1, 2}
		if err := memory.QueueMouseMotion(1, 1, false, false); err == nil {
			t.Error("門檻 1,2 竟然被接受")
		}
	})
	t.Run("Y 原點在下", func(t *testing.T) {
		memory := ikbdMouseReady(t)
		memory.ikbdYAxisUp = false
		if err := memory.QueueMouseMotion(1, 1, false, false); err == nil {
			t.Error("Y 原點在下竟然被接受")
		}
	})
	t.Run("不是相對模式", func(t *testing.T) {
		memory := ikbdMouseReady(t)
		memory.ikbdRelativeMouse = false
		if err := memory.QueueMouseMotion(1, 1, false, false); err == nil {
			t.Error("非相對模式竟然被接受")
		}
	})
	t.Run("一個位元組裝不下", func(t *testing.T) {
		memory := ikbdMouseReady(t)
		if err := memory.QueueMouseMotion(128, 0, false, false); err == nil {
			t.Error("位移 128 竟然被接受")
		}
		if err := memory.QueueMouseMotion(0, -129, false, false); err == nil {
			t.Error("位移 -129 竟然被接受")
		}
	})
	t.Run("佇列滿", func(t *testing.T) {
		memory := ikbdMouseReady(t)
		for i := 0; i < 2; i++ {
			if err := memory.QueueMouseMotion(1, 0, false, false); err != nil {
				t.Fatal(err)
			}
		}
		if err := memory.QueueMouseMotion(1, 0, false, false); err == nil {
			t.Error("第三個封包竟然排得進去")
		}
	})
	t.Run("RDRF 還沒清掉", func(t *testing.T) {
		memory := ikbdMouseReady(t)
		if err := memory.QueueMouseMotion(1, 0, false, false); err != nil {
			t.Fatal(err)
		}
		if err := memory.deliverIKBDUplinkByte(); err != nil {
			t.Fatal(err)
		}
		if err := memory.deliverIKBDUplinkByte(); err == nil {
			t.Error("主機還沒讀走就送下一個，竟然被接受")
		}
	})
}

func TestIKBDMouseColdResetClearsTheQueue(t *testing.T) {
	memory := ikbdMouseReady(t)
	if err := memory.QueueMouseMotion(5, 5, true, false); err != nil {
		t.Fatal(err)
	}
	memory.ColdReset()
	if memory.ikbdUplinkCount != 0 || memory.ikbdUplinkHead != 0 || memory.ikbdUplinkActive ||
		memory.ikbdMouseAccumX != 0 || memory.ikbdMouseAccumY != 0 ||
		memory.ikbdMouseLeft || memory.ikbdMouseRight {
		t.Fatal("cold reset 之後上行狀態沒清乾淨")
	}
}
