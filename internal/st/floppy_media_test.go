package st

import (
	"bytes"
	"testing"

	"github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"
)

func TestFloppyMediaReceiptsWrapAndReset(t *testing.T) {
	var receipts floppyMediaReceipts
	for attempt := uint64(1); attempt <= 12; attempt++ {
		receipts.append(floppyMediaReceipt{ReadCommandClock: attempt * 100})
	}
	if receipts.Total != 12 || receipts.Count != floppyMediaReceiptCapacity || receipts.Next != 4 {
		t.Fatalf("ring total/count/next=%d/%d/%d", receipts.Total, receipts.Count, receipts.Next)
	}
	for attempt := uint64(1); attempt <= 12; attempt++ {
		receipt, ok := receipts.attempt(attempt)
		wantOK := attempt >= 5
		if ok != wantOK {
			t.Fatalf("attempt %d present=%v want %v", attempt, ok, wantOK)
		}
		if ok && (receipt.Attempt != attempt || receipt.ReadCommandClock != attempt*100) {
			t.Fatalf("attempt %d receipt=%+v", attempt, receipt)
		}
	}
	receipts.reset()
	if receipts != (floppyMediaReceipts{}) {
		t.Fatalf("reset retained receipts: %+v", receipts)
	}
}

func TestFloppyMediaFirstTransactionTrackPrefix(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.floppyReadStage = 68
	memory.dmaMode = 0x0080
	memory.beginFloppyMediaAtTrack()

	if err := memory.WriteWord(STDMAControl, 0x0082, 5); err != nil {
		t.Fatalf("track selector: %v", err)
	}
	if memory.floppyMediaPhase != floppyMediaTrackData || memory.dmaMode != 0x0082 {
		t.Fatalf("track selector phase/mode=%d/%04x", memory.floppyMediaPhase, memory.dmaMode)
	}
	if err := memory.WriteWord(STDiskController, 0, 5); err != nil {
		t.Fatalf("track data: %v", err)
	}
	if memory.floppyMediaPhase != floppyMediaDriveSelector || memory.floppyMediaCurrent.Track != 0 {
		t.Fatalf("track data phase/track=%d/%d", memory.floppyMediaPhase, memory.floppyMediaCurrent.Track)
	}
}

func TestFloppyMediaRecurringNoDiskTransactions(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.floppyReadStage = 68
	memory.psgDriveStage = 9
	memory.psgRegisterSelect = 14
	memory.psgRegisters[7] = 0xc0
	memory.psgRegisters[14] = 0x25
	memory.dmaMode = 0x0080
	memory.dmaAddress = 0x001004
	memory.mfpGPIPIn = 0xb1
	machine := &Machine{Memory: memory}
	before := append([]byte(nil), memory.ram[0x1004:0x1204]...)

	for attempt := uint64(1); attempt <= 12; attempt++ {
		base := attempt * 10_000
		byteAt := func(address uint32, value byte, offset uint64) {
			t.Helper()
			if _, err := memory.WriteByteAt(address, value,
				m68k.BusAccess{Clock: base + offset, FunctionCode: 5}); err != nil {
				t.Fatalf("attempt %d byte write %06x=%02x: %v", attempt, address, value, err)
			}
		}
		wordAt := func(address uint32, value uint16, offset uint64) {
			t.Helper()
			if _, err := memory.WriteWordAt(address, value,
				m68k.BusAccess{Clock: base + offset, FunctionCode: 5}); err != nil {
				t.Fatalf("attempt %d word write %06x=%04x: %v", attempt, address, value, err)
			}
		}

		byteAt(PSGRegisterSelect, 14, 0)
		if value, _, err := memory.ReadByteAt(PSGRegisterSelect,
			m68k.BusAccess{Clock: base + 100, FunctionCode: 5}); err != nil || value != 0x25 {
			t.Fatalf("attempt %d drive read=%02x err=%v", attempt, value, err)
		}
		byteAt(PSGRegisterData, 0x25, 200)
		wordAt(STDMAControl, 0x0084, 300)
		wordAt(STDiskController, 1, 400)
		byteAt(STDMAAddressLow, 0x04, 500)
		byteAt(STDMAAddressMiddle, 0x10, 600)
		byteAt(STDMAAddressHigh, 0, 700)
		wordAt(STDMAControl, 0x0190, 800)
		wordAt(STDMAControl, 0x0090, 900)
		wordAt(STDiskController, 1, 1000)
		wordAt(STDMAControl, 0x0080, 1100)
		wordAt(STDiskController, 0x0080, 1200)
		wordAt(STDMAControl, 0x0080, 1300)
		wordAt(STDiskController, 0x00d0, 1400)
		wordAt(STDMAControl, 0x0086, 1500)
		wordAt(STDiskController, 0, 1600)
		wordAt(STDMAControl, 0x0080, 1700)
		wordAt(STDiskController, 0x0013, 1800)
		for poll := 0; poll < 9; poll++ {
			if value, err := memory.ReadByteFC(MFPGPIP, 5); err != nil || value&0x20 == 0 {
				t.Fatalf("attempt %d inactive poll %d=%02x err=%v", attempt, poll, value, err)
			}
		}
		machine.Clocks = base + 2529
		machine.advanceClockedDevices()
		if memory.floppyMediaPhase != floppyMediaSeekIRQ || !memory.fdcIRQ {
			t.Fatalf("attempt %d seek completion phase/IRQ=%d/%v", attempt,
				memory.floppyMediaPhase, memory.fdcIRQ)
		}
		if value, err := memory.ReadByteFC(MFPGPIP, 5); err != nil || value&0x20 != 0 {
			t.Fatalf("attempt %d IRQ poll=%02x err=%v", attempt, value, err)
		}
		wordAt(STDMAControl, 0x0080, 2600)
		if value, _, err := memory.ReadWordAt(STDiskController,
			m68k.BusAccess{Clock: base + 2700, FunctionCode: 5}); err != nil || value != 0x00e4 {
			t.Fatalf("attempt %d status=%04x err=%v", attempt, value, err)
		}
		if memory.floppyMediaPhase != floppyMediaIdle || memory.floppyMediaReceipts.Total != attempt {
			t.Fatalf("attempt %d completion phase/total=%d/%d", attempt,
				memory.floppyMediaPhase, memory.floppyMediaReceipts.Total)
		}
	}

	if memory.floppyMediaReceipts.Count != floppyMediaReceiptCapacity ||
		!bytes.Equal(before, memory.ram[0x1004:0x1204]) {
		t.Fatalf("recurring ring count=%d or DMA buffer changed", memory.floppyMediaReceipts.Count)
	}
	for attempt := uint64(5); attempt <= 12; attempt++ {
		receipt, ok := memory.floppyMediaReceipts.attempt(attempt)
		if !ok || receipt.DrivePort != 0x25 || receipt.Sector != 1 ||
			receipt.DMAAddressStage != 3 || receipt.DMAResetCount != 2 ||
			receipt.ReadCommand != 0x80 || receipt.ForceInterrupt != 0xd0 ||
			receipt.SeekCommand != 0x13 || receipt.InactivePolls != 9 ||
			!receipt.IRQObserved || receipt.StatusReadClock != attempt*10_000+2700 {
			t.Fatalf("attempt %d receipt=%+v present=%v", attempt, receipt, ok)
		}
	}
	memory.ColdReset()
	if memory.floppyMediaPhase != floppyMediaIdle || memory.floppyMediaCurrent != (floppyMediaReceipt{}) ||
		memory.floppyMediaReceipts != (floppyMediaReceipts{}) {
		t.Fatal("cold reset retained recurring media transaction state")
	}
}
