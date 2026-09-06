package st

import "testing"

// psgSoundReady 把機器設到開機之後、`flopvbl()` 兩輪之間的狀態：port 方向
// 已經是輸出，port A 停在 `$23`。按鍵聲就是在這種空檔彈出來的。
func psgSoundReady(t *testing.T) *Memory {
	t.Helper()
	return flopVBLReady(t, 0x23, 73)
}

// TestPSGSoundRegistersStoreAndReadBack covers spec 144: EmuTOS's key click
// writes the tone, mixer, amplitude and envelope registers and reads R7 back
// before rewriting it.
func TestPSGSoundRegistersStoreAndReadBack(t *testing.T) {
	memory := psgSoundReady(t)
	// EmuTOS 按鍵聲那一整串。
	writes := []struct {
		register byte
		value    byte
	}{
		{0, 0x3b}, {1, 0x00}, {2, 0x00}, {3, 0x00}, {4, 0x00}, {5, 0x00}, {6, 0x00},
		{8, 0x10}, {13, 0x03}, {11, 0x80}, {12, 0x01},
	}
	for _, write := range writes {
		if err := memory.WriteByteFC(PSGRegisterSelect, write.register, 5); err != nil {
			t.Fatalf("選 R%d：%v", write.register, err)
		}
		if err := memory.WriteByteFC(PSGRegisterData, write.value, 5); err != nil {
			t.Fatalf("寫 R%d=%02x：%v", write.register, write.value, err)
		}
		got, err := memory.ReadByteFC(PSGRegisterSelect, 5)
		if err != nil || got != write.value {
			t.Fatalf("讀回 R%d=%02x（要 %02x）err=%v", write.register, got, write.value, err)
		}
	}
	// R7 是讀改寫：讀到 $C0，寫回 $FE 只動混音位元。
	if err := memory.WriteByteFC(PSGRegisterSelect, 7, 5); err != nil {
		t.Fatal(err)
	}
	if got, err := memory.ReadByteFC(PSGRegisterSelect, 5); err != nil || got != 0xc0 {
		t.Fatalf("讀 R7=%02x err=%v", got, err)
	}
	if err := memory.WriteByteFC(PSGRegisterData, 0xfe, 5); err != nil {
		t.Fatalf("寫 R7=$FE：%v", err)
	}
	if memory.psgRegisters[7] != 0xfe || !memory.psgPortsAreOutputs() {
		t.Fatalf("R7=%02x", memory.psgRegisters[7])
	}
	// 混音位元變了不影響 port A：接著 `flopvbl()` 照樣選得到 R14，
	// 讀回拿到的是 port A 而不是剛剛選過的聲音暫存器。
	if err := memory.WriteByteFC(PSGRegisterSelect, 14, 5); err != nil {
		t.Fatalf("按鍵聲之後選 R14：%v", err)
	}
	if memory.psgRegisterSelect != 14 {
		t.Fatalf("選了 R14 之後 psgRegisterSelect 還是 %d", memory.psgRegisterSelect)
	}
	if got, err := memory.ReadByteFC(PSGRegisterSelect, 5); err != nil || got != 0x23 {
		t.Fatalf("按鍵聲之後讀 port A=%02x err=%v", got, err)
	}
}

// 把 port 的方向位元關掉會讓 port A 變成輸入，還沒建模；R15 也還沒建模。
func TestPSGRefusesWhatItDoesNotModel(t *testing.T) {
	for _, test := range []struct {
		name     string
		register byte
		value    byte
	}{
		{"清掉 port A 的方向位元", 7, 0xbe},
		{"清掉 port B 的方向位元", 7, 0x7e},
		{"兩個都清掉", 7, 0x3e},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory := psgSoundReady(t)
			if err := memory.WriteByteFC(PSGRegisterSelect, test.register, 5); err != nil {
				t.Fatal(err)
			}
			if err := memory.WriteByteFC(PSGRegisterData, test.value, 5); err == nil {
				t.Errorf("R7=%02x 竟然被接受", test.value)
			}
			if memory.psgRegisters[7] != 0xc0 {
				t.Errorf("拒絕之後 R7 變成 %02x", memory.psgRegisters[7])
			}
		})
	}
	t.Run("R15 還沒建模", func(t *testing.T) {
		memory := psgSoundReady(t)
		if err := memory.WriteByteFC(PSGRegisterSelect, 15, 5); err == nil {
			t.Error("選 R15 竟然被接受")
		}
	})
}
