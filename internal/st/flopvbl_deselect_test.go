package st

import "testing"

// flopVBLRound 走完一輪完整的輪詢：共用前置、選 drive、讀 status、收尾還原。
// 回傳 nil 表示每一步都被接受。
func flopVBLRound(memory *Memory, entry, target byte) error {
	steps := []func() error{
		func() error { return memory.WriteByteFC(PSGRegisterSelect, 14, 5) },
		func() error { _, err := memory.ReadByteFC(PSGRegisterSelect, 5); return err },
		func() error { return memory.WriteByteFC(PSGRegisterData, target, 5) },
		func() error { return memory.WriteWord(STDMAControl, 0x0080, 5) },
		func() error { _, err := memory.ReadWord(STDiskController, 5); return err },
		func() error { return memory.WriteByteFC(PSGRegisterSelect, 14, 5) },
		func() error { _, err := memory.ReadByteFC(PSGRegisterSelect, 5); return err },
		func() error { return memory.WriteByteFC(PSGRegisterData, entry, 5) },
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// TestFlopVBLTakesTheDriveTheROMPicks covers spec 140: which drive this turn
// checks comes out of a ROM-side counter that free-runs over VBLs, including
// the ones where the poll is skipped. The machine cannot see that counter, so
// it reads the choice off the data write instead of predicting it from how
// many turns have run.
func TestFlopVBLTakesTheDriveTheROMPicks(t *testing.T) {
	for _, test := range []struct {
		name   string
		target byte
		drive  int8
	}{
		{"同一個 checks 值選 drive 0", 0x25, 0},
		{"同一個 checks 值選 drive 1", 0x23, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory := flopVBLReady(t, 0x27, 74)
			if err := flopVBLRound(memory, 0x27, test.target); err != nil {
				t.Fatalf("這一輪走不完：%v", err)
			}
			if memory.flopVBLMediaDrive != test.drive {
				t.Errorf("記下的 drive=%d，應該是 %d", memory.flopVBLMediaDrive, test.drive)
			}
			if memory.psgRegisters[14] != 0x27 || memory.flopVBLMediaStage != 8 ||
				memory.flopVBLMediaChecks != 75 {
				t.Errorf("收尾：port=%02x stage=%d checks=%d",
					memory.psgRegisters[14], memory.flopVBLMediaStage, memory.flopVBLMediaChecks)
			}
		})
	}
}

// The shared data step takes both side-0 drive selects, the drive-A side-1
// select used by media reads, and the deselect covered below. Other values
// fail closed and leave the state alone; the following DMA access decides
// whether an accepted shared prefix belongs to flopvbl or flop_mediach.
func TestFlopVBLRejectsANonDriveSelectWrite(t *testing.T) {
	for _, value := range []byte{0x21, 0x22, 0x26, 0x00, 0x35} {
		memory := flopVBLReady(t, 0x25, 74)
		if err := memory.WriteByteFC(PSGRegisterSelect, 14, 5); err != nil {
			t.Fatal(err)
		}
		if _, err := memory.ReadByteFC(PSGRegisterSelect, 5); err != nil {
			t.Fatal(err)
		}
		if err := memory.WriteByteFC(PSGRegisterData, value, 5); err == nil {
			t.Errorf("data 那一步寫 %02x 竟然被接受", value)
		}
		if memory.flopVBLMediaStage != 2 || memory.psgRegisters[14] != 0x25 {
			t.Errorf("拒絕之後狀態變了：stage=%d port=%02x",
				memory.flopVBLMediaStage, memory.psgRegisters[14])
		}
	}
}

// TestFlopVBLDeselectsBothDrives covers the one-off call that puts $27 into
// port A: it ends after the data write — no status read — and does not count as
// a turn. Every turn after it comes in on $27 and goes back out on $27.
func TestFlopVBLDeselectsBothDrives(t *testing.T) {
	memory := flopVBLReady(t, 0x25, 76)
	if err := memory.WriteByteFC(PSGRegisterSelect, 14, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := memory.ReadByteFC(PSGRegisterSelect, 5); err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteByteFC(PSGRegisterData, 0x27, 5); err != nil {
		t.Fatalf("deselect 被拒絕：%v", err)
	}
	if memory.psgRegisters[14] != 0x27 || memory.flopVBLMediaStage != 8 {
		t.Fatalf("deselect 之後：port=%02x stage=%d",
			memory.psgRegisters[14], memory.flopVBLMediaStage)
	}
	if memory.flopVBLMediaChecks != 76 {
		t.Errorf("deselect 被算成一輪：checks=%d", memory.flopVBLMediaChecks)
	}
	// 之後每一輪的進場值都是 $27。
	if err := flopVBLRound(memory, 0x27, 0x23); err != nil {
		t.Fatalf("deselect 之後的下一輪走不完：%v", err)
	}
	if memory.psgRegisters[14] != 0x27 || memory.flopVBLMediaEntryPort != 0x27 ||
		memory.flopVBLMediaChecks != 77 {
		t.Errorf("下一輪：port=%02x entry=%02x checks=%d",
			memory.psgRegisters[14], memory.flopVBLMediaEntryPort, memory.flopVBLMediaChecks)
	}
}

// deselect 之後不接 status 讀取——這一步在 stage 8，不是 stage 4。
func TestFlopVBLDeselectHasNoStatusRead(t *testing.T) {
	memory := flopVBLReady(t, 0x25, 76)
	for _, step := range []func() error{
		func() error { return memory.WriteByteFC(PSGRegisterSelect, 14, 5) },
		func() error { _, err := memory.ReadByteFC(PSGRegisterSelect, 5); return err },
		func() error { return memory.WriteByteFC(PSGRegisterData, 0x27, 5) },
	} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := memory.ReadWord(STDiskController, 5); err == nil {
		t.Error("deselect 之後竟然讀得到 FDC status")
	}
}
