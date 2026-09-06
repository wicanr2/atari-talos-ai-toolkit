package st

import (
	"testing"

	"github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"
)

// flopVBLReady puts the machine where flopvbl() takes its turn: everything the
// gate needs is set, and port A holds whatever the previous caller left. That
// is the "jump straight to the event" shortcut — the boot path takes eight
// million instructions to reach the fourth entry value and none of them are
// what this slice is about.
func flopVBLReady(t *testing.T, entry byte, checks uint32) *Memory {
	t.Helper()
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.psgDriveStage = 9
	memory.psgRegisterSelect = 14
	memory.psgRegisters[7], memory.psgRegisters[14] = 0xc0, entry
	memory.flopVBLMediaStage = 8
	memory.flopVBLMediaChecks = checks
	memory.ikbdClockReadbackComplete = true
	memory.fdcInitStage = 14
	memory.acsiStage = 5
	memory.dmaMode = 0x0080
	memory.fdcStatusTypeI, memory.fdcIRQ = true, true
	memory.mfpGPIPIn = 0xb1 &^ 0x20
	return memory
}

// TestFlopVBLRestoresTheEntryPort covers spec 139: every turn reads the old
// port A, sets the drive it wants and puts the old value back. The entry value
// is not a constant — the Hatari trace shows $23, $25 and $27 all turning up.
func TestFlopVBLRestoresTheEntryPort(t *testing.T) {
	for _, test := range []struct {
		name   string
		entry  byte
		checks uint32
		target byte
	}{
		{"進場 $23，這一輪選 drive 0", 0x23, 74, 0x25},
		{"進場 $25，這一輪選 drive 1", 0x25, 73, 0x23},
		{"進場 $27，這一輪選 drive 1", 0x27, 73, 0x23},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory := flopVBLReady(t, test.entry, test.checks)
			if err := memory.WriteByte(PSGRegisterSelect, 14, 5); err != nil ||
				memory.flopVBLMediaStage != 1 {
				t.Fatalf("選 R14：stage=%d err=%v", memory.flopVBLMediaStage, err)
			}
			value, err := memory.ReadByte(PSGRegisterSelect, 5)
			if err != nil || value != test.entry || memory.flopVBLMediaStage != 2 ||
				memory.flopVBLMediaEntryPort != test.entry {
				t.Fatalf("讀回 %02x stage=%d entry=%02x err=%v",
					value, memory.flopVBLMediaStage, memory.flopVBLMediaEntryPort, err)
			}
			if err := memory.WriteByte(PSGRegisterData, test.target, 5); err != nil ||
				memory.flopVBLMediaStage != 3 || memory.psgRegisters[14] != test.target {
				t.Fatalf("選 drive：stage=%d port=%02x err=%v",
					memory.flopVBLMediaStage, memory.psgRegisters[14], err)
			}
			if err := memory.WriteWord(STDMAControl, 0x0080, 5); err != nil ||
				memory.flopVBLMediaStage != 4 {
				t.Fatalf("選 status：stage=%d err=%v", memory.flopVBLMediaStage, err)
			}
			if status, err := memory.ReadWord(STDiskController, 5); err != nil ||
				status != 0xe4 || memory.flopVBLMediaStage != 5 {
				t.Fatalf("讀 status=%04x stage=%d err=%v", status, memory.flopVBLMediaStage, err)
			}
			if err := memory.WriteByte(PSGRegisterSelect, 14, 5); err != nil ||
				memory.flopVBLMediaStage != 6 {
				t.Fatalf("收尾選 R14：stage=%d err=%v", memory.flopVBLMediaStage, err)
			}
			if _, err := memory.ReadByte(PSGRegisterSelect, 5); err != nil ||
				memory.flopVBLMediaStage != 7 {
				t.Fatalf("收尾讀回：stage=%d err=%v", memory.flopVBLMediaStage, err)
			}
			// 還原的是進場值，不是固定的 $23。
			if err := memory.WriteByte(PSGRegisterData, test.entry, 5); err != nil ||
				memory.flopVBLMediaStage != 8 || memory.psgRegisters[14] != test.entry ||
				memory.flopVBLMediaChecks != test.checks+1 {
				t.Fatalf("還原：stage=%d port=%02x checks=%d err=%v",
					memory.flopVBLMediaStage, memory.psgRegisters[14],
					memory.flopVBLMediaChecks, err)
			}
		})
	}
}

// Restoring anything other than the entry value fails closed — that is the
// whole point of remembering it.
func TestFlopVBLRejectsAWrongRestore(t *testing.T) {
	memory := flopVBLReady(t, 0x25, 73)
	for _, step := range []func() error{
		func() error { return memory.WriteByte(PSGRegisterSelect, 14, 5) },
		func() error { _, err := memory.ReadByte(PSGRegisterSelect, 5); return err },
		func() error { return memory.WriteByte(PSGRegisterData, 0x23, 5) },
		func() error { return memory.WriteWord(STDMAControl, 0x0080, 5) },
		func() error { _, err := memory.ReadWord(STDiskController, 5); return err },
		func() error { return memory.WriteByte(PSGRegisterSelect, 14, 5) },
		func() error { _, err := memory.ReadByte(PSGRegisterSelect, 5); return err },
	} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	if err := memory.WriteByte(PSGRegisterData, 0x23, 5); err == nil {
		t.Error("把進場值 $25 還原成 $23 竟然被接受")
	}
	if memory.flopVBLMediaStage != 7 || memory.psgRegisters[14] != 0x23 {
		t.Errorf("拒絕之後狀態變了：stage=%d port=%02x",
			memory.flopVBLMediaStage, memory.psgRegisters[14])
	}
}

// An entry value that is not one of the three drive-select combinations fails
// closed at the read, before anything is remembered.
func TestFlopVBLRejectsAnUnknownEntryPort(t *testing.T) {
	for _, entry := range []byte{0x20, 0x21, 0x22, 0x24, 0x26, 0x00} {
		memory := flopVBLReady(t, entry, 73)
		if err := memory.WriteByte(PSGRegisterSelect, 14, 5); err == nil &&
			memory.flopVBLMediaStage == 1 {
			if _, err := memory.ReadByte(PSGRegisterSelect, 5); err == nil {
				t.Errorf("進場值 %02x 被接受了", entry)
			}
		}
		if memory.flopVBLMediaEntryPort != 0 {
			t.Errorf("進場值 %02x 被記下來了", entry)
		}
	}
}

// The media check borrows the same three-step preamble. Which of the two is
// running only shows up at the DMA control that follows: $0084 is the media
// check's sector selector, $0080 is flopvbl()'s status read.
func TestTheMediaCheckBorrowsTheSharedPreamble(t *testing.T) {
	memory := flopVBLReady(t, 0x25, 74) // checks even → this turn wants drive 0 ($25)
	memory.floppyMediaLocked = true
	memory.dmaAddress = 0x001004
	for _, step := range []func() error{
		func() error { return memory.WriteByte(PSGRegisterSelect, 14, 5) },
		func() error { _, err := memory.ReadByte(PSGRegisterSelect, 5); return err },
	} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := memory.WriteByteAt(PSGRegisterData, 0x25,
		m68k.BusAccess{Clock: 900, FunctionCode: 5}); err != nil {
		t.Fatal(err)
	}
	if memory.flopVBLMediaStage != 3 || memory.floppyMediaPhase != floppyMediaIdle {
		t.Fatalf("data 那一步就分派了：stage=%d phase=%d",
			memory.flopVBLMediaStage, memory.floppyMediaPhase)
	}
	if err := memory.WriteWord(STDMAControl, 0x0084, 5); err != nil {
		t.Fatal(err)
	}
	if memory.floppyMediaPhase != floppyMediaSectorData || memory.flopVBLMediaStage != 8 ||
		memory.floppyMediaCurrent.DrivePort != 0x25 ||
		memory.floppyMediaCurrent.DriveWriteClock != 900 {
		t.Fatalf("$0084 沒有把這一輪交給媒體確認：phase=%d stage=%d 收據=%+v",
			memory.floppyMediaPhase, memory.flopVBLMediaStage, memory.floppyMediaCurrent)
	}
	// 借走的那一輪 flopvbl() 不算數。
	if memory.flopVBLMediaChecks != 74 {
		t.Errorf("借用的那一輪被算進 checks：%d", memory.flopVBLMediaChecks)
	}
}
