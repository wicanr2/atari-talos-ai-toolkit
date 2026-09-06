package st

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"
)

func TestRawFloppyGeometryAndCHS(t *testing.T) {
	image := testRawFloppy(80, 2, 9)
	floppy, err := NewRawFloppy(image)
	if err != nil {
		t.Fatal(err)
	}
	if tracks, sides, sectors := floppy.Geometry(); tracks != 80 || sides != 2 || sectors != 9 {
		t.Fatalf("geometry=%d/%d/%d, want 80/2/9", tracks, sides, sectors)
	}
	checks := []struct {
		track, side, sector uint16
		want                byte
	}{
		{0, 0, 1, 0},
		{0, 1, 1, 9},
		{1, 0, 1, 18},
		{79, 1, 9, 0x9f},
	}
	for _, check := range checks {
		got, err := floppy.Sector(check.track, check.side, check.sector)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != rawFloppySectorSize || got[0] != check.want {
			t.Errorf("CHS %d/%d/%d: len=%d first=%02x, want 512/%02x",
				check.track, check.side, check.sector, len(got), got[0], check.want)
		}
	}
}

func TestMountedFloppyReadSectorCompletesDMAAndRaisesIRQ(t *testing.T) {
	machine, err := NewMachine(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	image := testRawFloppy(80, 2, 9)
	for index := range rawFloppySectorSize {
		image[index] = byte(index*37 + 11)
	}
	// Restore the BPB fields that make the patterned first sector a valid image.
	image[0x0b], image[0x0c] = 0, 2
	image[0x13], image[0x14] = 0xa0, 5
	image[0x18], image[0x19] = 9, 0
	image[0x1a], image[0x1b] = 2, 0
	if err := machine.AttachFloppyA(image); err != nil {
		t.Fatal(err)
	}
	memory := machine.Memory
	memory.floppyMediaCurrent = floppyMediaReceipt{Drive: 0, Track: 0, Sector: 1}
	memory.floppyMediaPhase = floppyMediaReadBusy
	memory.dmaMode = 0x0080
	memory.dmaAddress = 0x001004
	memory.dmaSectorCount = 1
	before := append([]byte(nil), memory.ram[0x1004:0x1204]...)
	if wait, err := memory.WriteWordAt(STDiskController, 0x0080,
		m68k.BusAccess{Clock: 200, FunctionCode: 5}); err != nil || wait != 4 ||
		!memory.fdcReadPending || memory.fdcReadStartClock != 200 ||
		memory.floppyMediaPhase != floppyMediaReadTransfer {
		t.Fatalf("read submission wait/pending/start/phase=%d/%v/%d/%d err=%v", wait,
			memory.fdcReadPending, memory.fdcReadStartClock, memory.floppyMediaPhase, err)
	}
	machine.Clocks = 200 + fdcReadSectorLatencyClocks - 1
	machine.advanceClockedDevices()
	if !bytes.Equal(memory.ram[0x1004:0x1204], before) || memory.dmaAddress != 0x001004 ||
		memory.dmaSectorCount != 1 || memory.fdcIRQ {
		t.Fatal("read became visible before its deterministic deadline")
	}
	machine.Clocks++
	machine.advanceClockedDevices()
	if !bytes.Equal(memory.ram[0x1004:0x1204], image[:rawFloppySectorSize]) ||
		memory.dmaAddress != 0x001204 || memory.dmaSectorCount != 0 || memory.fdcStatus != 0x80 ||
		memory.fdcStatusTypeI || !memory.fdcIRQ || memory.mfpGPIPIn&0x20 != 0 ||
		memory.floppyMediaPhase != floppyMediaReadIRQReset {
		t.Fatalf("completed DMA addr/count/status/type/IRQ/GPIP/phase=%06x/%d/%02x/%v/%v/%02x/%d",
			memory.dmaAddress, memory.dmaSectorCount, memory.fdcStatus, memory.fdcStatusTypeI,
			memory.fdcIRQ, memory.mfpGPIPIn, memory.floppyMediaPhase)
	}
	if err := memory.WriteWord(STDMAControl, 0x0090, 5); err != nil ||
		memory.floppyMediaPhase != floppyMediaReadDMAStatus {
		t.Fatalf("DMA status reset phase=%d err=%v", memory.floppyMediaPhase, err)
	}
	if status, err := memory.ReadWord(STDMAControl, 5); err != nil || status != 1 ||
		memory.floppyMediaPhase != floppyMediaReadIRQSelector {
		t.Fatalf("DMA status=%04x phase=%d err=%v", status, memory.floppyMediaPhase, err)
	}
	if err := memory.WriteWord(STDMAControl, 0x0080, 5); err != nil ||
		memory.floppyMediaPhase != floppyMediaReadStatusRead {
		t.Fatalf("status selector phase=%d err=%v", memory.floppyMediaPhase, err)
	}
	if status, err := memory.ReadWord(STDiskController, 5); err != nil || status != 0x0080 ||
		memory.fdcIRQ || memory.mfpGPIPIn&0x20 == 0 ||
		memory.floppyMediaPhase != floppyMediaPostRead {
		t.Fatalf("status=%04x IRQ/GPIP/phase=%v/%02x/%d err=%v", status, memory.fdcIRQ,
			memory.mfpGPIPIn, memory.floppyMediaPhase, err)
	}
	if err := memory.WriteWord(STDMAControl, 0x0086, 5); err != nil ||
		memory.floppyMediaPhase != floppyMediaSeekData {
		t.Fatalf("dummy-seek selector phase=%d err=%v", memory.floppyMediaPhase, err)
	}
}

func TestMountedFloppyReadSideOneUsesReceiptCHS(t *testing.T) {
	machine, err := NewMachine(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	image := testRawFloppy(80, 2, 9)
	want := bytes.Repeat([]byte{0xa5}, rawFloppySectorSize)
	copy(image[9*rawFloppySectorSize:10*rawFloppySectorSize], want)
	if err := machine.AttachFloppyA(image); err != nil {
		t.Fatal(err)
	}
	memory := machine.Memory
	memory.floppyMediaCurrent = floppyMediaReceipt{Drive: 0, Side: 1, Track: 0, Sector: 1}
	memory.floppyMediaPhase = floppyMediaReadBusy
	memory.dmaMode = 0x0080
	memory.dmaAddress = 0x001004
	memory.dmaSectorCount = 1
	if _, err := memory.WriteWordAt(STDiskController, 0x0080,
		m68k.BusAccess{Clock: 200, FunctionCode: 5}); err != nil {
		t.Fatal(err)
	}
	machine.Clocks = 200 + fdcReadSectorLatencyClocks
	machine.advanceClockedDevices()
	if !bytes.Equal(memory.ram[0x1004:0x1204], want) || memory.floppyMediaCurrent.Side != 1 {
		t.Fatalf("side-one DMA mismatch: receipt=%+v", memory.floppyMediaCurrent)
	}
}

func TestMountedSingleSidedFloppyRejectsSideOneAtomically(t *testing.T) {
	machine, err := NewMachine(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.AttachFloppyA(testRawFloppy(80, 1, 9)); err != nil {
		t.Fatal(err)
	}
	memory := machine.Memory
	memory.floppyMediaCurrent = floppyMediaReceipt{Drive: 0, Side: 1, Track: 0, Sector: 1}
	memory.floppyMediaPhase = floppyMediaReadBusy
	memory.dmaMode = 0x0080
	memory.dmaAddress = 0x001004
	memory.dmaSectorCount = 1
	before := append([]byte(nil), memory.ram[0x1004:0x1204]...)
	if err := memory.WriteWord(STDiskController, 0x0080, 5); err == nil {
		t.Fatal("single-sided image unexpectedly accepted side one")
	}
	if !bytes.Equal(memory.ram[0x1004:0x1204], before) || memory.fdcReadPending ||
		memory.fdcCommand != 0 || memory.floppyMediaPhase != floppyMediaReadBusy {
		t.Fatal("rejected side-one read mutated controller or RAM")
	}
}

func TestMountedFloppySideOnePortStartsSectorRead(t *testing.T) {
	machine, err := NewMachine(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory := machine.Memory
	memory.psgDriveStage = 9
	memory.flopVBLMediaStage = 2
	memory.flopVBLMediaEntryPort = 0x25
	memory.psgRegisterSelect = 14
	memory.psgRegisters[7] = 0xc0
	memory.psgRegisters[14] = 0x25
	memory.floppyMediaLocked = true
	memory.floppyMediaPhase = floppyMediaIdle
	if _, err := memory.WriteByteAt(PSGRegisterData, 0x24,
		m68k.BusAccess{Clock: 300, FunctionCode: 5}); err != nil ||
		memory.flopVBLMediaStage != 3 || memory.flopVBLMediaDrive != 0 {
		t.Fatalf("side-one port dispatch stage/drive=%d/%d err=%v",
			memory.flopVBLMediaStage, memory.flopVBLMediaDrive, err)
	}
	if _, err := memory.WriteWordAt(STDMAControl, 0x0084,
		m68k.BusAccess{Clock: 400, FunctionCode: 5}); err != nil ||
		memory.floppyMediaPhase != floppyMediaSectorData ||
		memory.floppyMediaCurrent.Drive != 0 || memory.floppyMediaCurrent.Side != 1 ||
		memory.floppyMediaCurrent.DrivePort != 0x24 {
		t.Fatalf("side-one receipt phase/drive/side/port=%d/%d/%d/%02x err=%v",
			memory.floppyMediaPhase, memory.floppyMediaCurrent.Drive,
			memory.floppyMediaCurrent.Side, memory.floppyMediaCurrent.DrivePort, err)
	}
}

func TestMountedFloppySideOnePortReentersSharedPrefix(t *testing.T) {
	machine, err := NewMachine(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory := machine.Memory
	memory.psgDriveStage = 9
	memory.flopVBLMediaStage = 8
	memory.ikbdClockReadbackComplete = true
	memory.fdcInitStage = 14
	memory.acsiStage = 5
	memory.psgRegisterSelect = 14
	memory.psgRegisters[7] = 0xc0
	memory.psgRegisters[14] = 0x24
	memory.floppyMediaLocked = true
	memory.floppyMediaPhase = floppyMediaIdle
	if err := memory.WriteByteFC(PSGRegisterSelect, 14, 5); err != nil ||
		memory.flopVBLMediaStage != 1 {
		t.Fatalf("side-one selector reentry stage=%d err=%v", memory.flopVBLMediaStage, err)
	}
	if got, err := memory.ReadByteFC(PSGRegisterSelect, 5); err != nil || got != 0x24 ||
		memory.flopVBLMediaStage != 2 {
		t.Fatalf("side-one readback value/stage=%02x/%d err=%v", got,
			memory.flopVBLMediaStage, err)
	}
	if err := memory.WriteByteFC(PSGRegisterData, 0x24, 5); err != nil ||
		memory.flopVBLMediaStage != 3 {
		t.Fatalf("side-one same-port write stage=%d err=%v", memory.flopVBLMediaStage, err)
	}
	if err := memory.WriteWord(STDMAControl, 0x0084, 5); err != nil ||
		memory.floppyMediaPhase != floppyMediaSectorData || memory.floppyMediaCurrent.Side != 1 ||
		memory.floppyMediaCurrent.DrivePort != 0x24 {
		t.Fatalf("side-one reentry receipt phase/side/port=%d/%d/%02x err=%v",
			memory.floppyMediaPhase, memory.floppyMediaCurrent.Side,
			memory.floppyMediaCurrent.DrivePort, err)
	}
}

func TestMountedFloppySeeksTrackThenReadsSector(t *testing.T) {
	machine, err := NewMachine(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	image := testRawFloppy(80, 2, 9)
	want := bytes.Repeat([]byte{0x6d}, rawFloppySectorSize)
	trackOneOffset := 18 * rawFloppySectorSize
	copy(image[trackOneOffset:trackOneOffset+rawFloppySectorSize], want)
	if err := machine.AttachFloppyA(image); err != nil {
		t.Fatal(err)
	}
	memory := machine.Memory
	memory.psgDriveStage = 9
	memory.flopVBLMediaStage = 3
	memory.psgRegisters[14] = 0x25
	memory.floppyMediaLocked = true
	memory.floppyMediaPhase = floppyMediaIdle
	if _, err := memory.WriteWordAt(STDMAControl, 0x0086,
		m68k.BusAccess{Clock: 100, FunctionCode: 5}); err != nil ||
		memory.floppyMediaPhase != floppyMediaTrackSeekData {
		t.Fatalf("track seek selector phase=%d err=%v", memory.floppyMediaPhase, err)
	}
	if err := memory.WriteWord(STDiskController, 1, 5); err != nil ||
		memory.floppyMediaCurrent.Track != 1 || memory.fdcHeadTrack != 0 {
		t.Fatalf("track data receipt/head=%d/%d err=%v",
			memory.floppyMediaCurrent.Track, memory.fdcHeadTrack, err)
	}
	if err := memory.WriteWord(STDMAControl, 0x0080, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := memory.WriteWordAt(STDiskController, 0x0013,
		m68k.BusAccess{Clock: 400, FunctionCode: 5}); err != nil ||
		!memory.fdcSeekPending || memory.fdcHeadTrack != 0 {
		t.Fatalf("track seek command pending/head=%v/%d err=%v",
			memory.fdcSeekPending, memory.fdcHeadTrack, err)
	}
	deadline := fdcTrackSeekDeadline(400, 0, 1)
	machine.Clocks = deadline - 1
	machine.advanceClockedDevices()
	if memory.fdcHeadTrack != 0 || !memory.fdcSeekPending {
		t.Fatal("track seek committed before its deterministic deadline")
	}
	machine.Clocks = deadline
	machine.advanceClockedDevices()
	if memory.fdcHeadTrack != 1 || memory.fdcSeekPending || !memory.fdcIRQ ||
		memory.mfpGPIPIn&0x20 != 0 || memory.floppyMediaPhase != floppyMediaTrackSeekIRQ {
		t.Fatalf("track seek completion head/pending/IRQ/GPIP/phase=%d/%v/%v/%02x/%d",
			memory.fdcHeadTrack, memory.fdcSeekPending, memory.fdcIRQ,
			memory.mfpGPIPIn, memory.floppyMediaPhase)
	}
	if err := memory.WriteWord(STDMAControl, 0x0084, 5); err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteWord(STDiskController, 1, 5); err != nil {
		t.Fatal(err)
	}
	for _, write := range []struct {
		address uint32
		value   byte
	}{{STDMAAddressLow, 0x04}, {STDMAAddressMiddle, 0x10}, {STDMAAddressHigh, 0x00}} {
		if err := memory.WriteByteFC(write.address, write.value, 5); err != nil {
			t.Fatal(err)
		}
	}
	for _, write := range []struct {
		address uint32
		value   uint16
	}{{STDMAControl, 0x0190}, {STDMAControl, 0x0090}, {STDiskController, 1},
		{STDMAControl, 0x0080}} {
		if err := memory.WriteWord(write.address, write.value, 5); err != nil {
			t.Fatal(err)
		}
	}
	readStart := deadline + 100
	if _, err := memory.WriteWordAt(STDiskController, 0x0080,
		m68k.BusAccess{Clock: readStart, FunctionCode: 5}); err != nil {
		t.Fatal(err)
	}
	machine.Clocks = readStart + fdcReadSectorLatencyClocks
	machine.advanceClockedDevices()
	if !bytes.Equal(memory.ram[0x1004:0x1204], want) ||
		memory.floppyMediaCurrent.Track != 1 || memory.floppyMediaCurrent.SectorsRead != 1 {
		t.Fatalf("track-one DMA mismatch: receipt=%+v", memory.floppyMediaCurrent)
	}
	memory.floppyMediaPhase = floppyMediaSeekData
	memory.dmaMode = 0x0086
	if err := memory.WriteWord(STDiskController, 0, 5); err == nil ||
		memory.floppyMediaPhase != floppyMediaSeekData {
		t.Fatal("track-one dummy seek accepted track zero")
	}
	if err := memory.WriteWord(STDiskController, 1, 5); err != nil ||
		memory.floppyMediaPhase != floppyMediaSeekCommandSelector || memory.fdcData != 1 {
		t.Fatalf("track-one dummy seek data phase/data=%d/%d err=%v",
			memory.floppyMediaPhase, memory.fdcData, err)
	}
}

func TestMountedFloppyRejectsOutOfRangeTrackAtomically(t *testing.T) {
	machine, err := NewMachine(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.AttachFloppyA(testRawFloppy(1, 2, 9)); err != nil {
		t.Fatal(err)
	}
	memory := machine.Memory
	memory.floppyMediaPhase = floppyMediaTrackSeekData
	memory.dmaMode = 0x0086
	if err := memory.WriteWord(STDiskController, 1, 5); err == nil ||
		memory.floppyMediaPhase != floppyMediaTrackSeekData || memory.fdcData != 0 ||
		memory.fdcHeadTrack != 0 || memory.fdcSeekPending || memory.fdcCommand != 0 {
		t.Fatal("out-of-range track was accepted or mutated controller state")
	}
}

func TestMountedFloppySectorReadUsesCurrentHeadTrack(t *testing.T) {
	machine, err := NewMachine(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory := machine.Memory
	memory.psgDriveStage = 9
	memory.flopVBLMediaStage = 3
	memory.psgRegisters[14] = 0x24
	memory.floppyMediaLocked = true
	memory.floppyMediaPhase = floppyMediaIdle
	memory.fdcHeadTrack = 7
	if err := memory.WriteWord(STDMAControl, 0x0084, 5); err != nil ||
		memory.floppyMediaPhase != floppyMediaSectorData ||
		memory.floppyMediaCurrent.Track != 7 || memory.floppyMediaCurrent.Side != 1 {
		t.Fatalf("current CHS phase/track/side=%d/%d/%d err=%v",
			memory.floppyMediaPhase, memory.floppyMediaCurrent.Track,
			memory.floppyMediaCurrent.Side, err)
	}
}

func TestMountedFloppyReadRejectsDMAOverflowAtomically(t *testing.T) {
	machine, err := NewMachine(RAM512K, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.AttachFloppyA(testRawFloppy(80, 2, 9)); err != nil {
		t.Fatal(err)
	}
	memory := machine.Memory
	memory.floppyMediaCurrent = floppyMediaReceipt{Drive: 0, Track: 0, Sector: 1}
	memory.floppyMediaPhase = floppyMediaReadBusy
	memory.dmaMode = 0x0080
	memory.dmaAddress = RAM512K - 2
	memory.dmaSectorCount = 1
	before := append([]byte(nil), memory.ram...)
	if err := memory.WriteWord(STDiskController, 0x0080, 5); err == nil {
		t.Fatal("DMA overflow unexpectedly accepted")
	}
	if !bytes.Equal(memory.ram, before) || memory.fdcReadPending ||
		memory.floppyMediaPhase != floppyMediaReadBusy || memory.fdcCommand != 0 {
		t.Fatal("rejected DMA overflow mutated controller or RAM")
	}
	memory.dmaAddress = 0x1004
	memory.floppyMediaCurrent.Sector = 10
	if err := memory.WriteWord(STDiskController, 0x0080, 5); err == nil || memory.fdcReadPending ||
		memory.floppyMediaPhase != floppyMediaReadBusy || memory.fdcCommand != 0 {
		t.Fatal("out-of-range sector unexpectedly accepted or mutated controller")
	}
	memory.floppyMediaPhase = floppyMediaSectorData
	if err := memory.WriteWord(STDiskController, 0, 5); err == nil ||
		memory.floppyMediaPhase != floppyMediaSectorData {
		t.Fatal("sector zero unexpectedly accepted")
	}
}

func TestEmuTOSMountedFloppyCompletesSequentialMediaRead(t *testing.T) {
	path := os.Getenv("TALOS_TOS_ROM")
	if path == "" {
		t.Skip("TALOS_TOS_ROM is not set")
	}
	rom, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	image := testRawFloppy(80, 2, 9)
	for index := range rawFloppySectorSize {
		image[index] = byte(index*37 + 11)
	}
	image[0x0b], image[0x0c] = 0, 2
	image[0x13], image[0x14] = 0xa0, 5
	image[0x18], image[0x19] = 9, 0
	image[0x1a], image[0x1b] = 2, 0
	if err := machine.AttachFloppyA(image); err != nil {
		t.Fatal(err)
	}
	if err := machine.Reset(); err != nil {
		t.Fatal(err)
	}
	var gate error
	for steps := 0; steps < 3_000_000 && machine.Memory.floppyMediaReceipts.Total == 0 && gate == nil; steps++ {
		_, gate = machine.Step()
	}
	if gate != nil {
		t.Fatalf("mounted-media normal path reached unsupported gate: %v", gate)
	}
	if machine.Memory.floppyMediaReceipts.Total != 1 {
		t.Fatalf("first mounted-media read did not finish: phase=%d mode=%04x status=%02x IRQ=%v GPIP=%02x instructions=%d clocks=%d state=%+v",
			machine.Memory.floppyMediaPhase, machine.Memory.dmaMode, machine.Memory.fdcStatus,
			machine.Memory.fdcIRQ, machine.Memory.mfpGPIPIn, machine.Instructions, machine.Clocks,
			machine.CPU.State)
	}
	receipt, ok := machine.Memory.floppyMediaReceipts.attempt(1)
	t.Logf("mounted media receipt: %+v; instructions=%d interrupts=%d clocks=%d", receipt,
		machine.Instructions, machine.Interrupts, machine.Clocks)
	if !ok || receipt.ReadCompleteClock == 0 || receipt.TimeoutSelectorClock != 0 ||
		receipt.ForceInterrupt != 0 || !bytes.Equal(machine.Memory.ram[0x1004:0x1204], image[:rawFloppySectorSize]) {
		t.Fatalf("first mounted receipt=%+v ok=%v", receipt, ok)
	}
	for steps := 0; steps < 3_000_000 && machine.Memory.floppyMediaReceipts.Total < 2 && gate == nil; steps++ {
		_, gate = machine.Step()
	}
	if gate != nil {
		t.Fatalf("multi-sector normal path reached unsupported gate: %v", gate)
	}
	second, ok := machine.Memory.floppyMediaReceipts.attempt(2)
	t.Logf("second mounted media receipt: %+v; instructions=%d interrupts=%d clocks=%d", second,
		machine.Instructions, machine.Interrupts, machine.Clocks)
	if !ok || second.Sector != 6 || second.SectorsRead != 6 || second.BytesRead != 6*rawFloppySectorSize ||
		second.ReadCompleteClock != 107499042 || second.StatusReadClock != 107502734 ||
		second.TimeoutSelectorClock != 0 || second.ForceInterrupt != 0 ||
		!bytes.Equal(machine.Memory.ram[0x1004:0x1c04], image[:6*rawFloppySectorSize]) {
		t.Fatalf("multi-sector receipt=%+v ok=%v", second, ok)
	}
}

func TestRawFloppyCopiesInputAndSectorOutput(t *testing.T) {
	image := testRawFloppy(1, 1, 2)
	floppy, err := NewRawFloppy(image)
	if err != nil {
		t.Fatal(err)
	}
	image[0x100] ^= 0xff
	first, err := floppy.Sector(0, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := first[0x100]
	first[0x100] ^= 0xff
	again, err := floppy.Sector(0, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if again[0x100] != want {
		t.Fatalf("mounted image or returned sector aliases caller memory")
	}
}

func TestRawFloppyRejectsMalformedImages(t *testing.T) {
	valid := testRawFloppy(80, 2, 9)
	tests := []struct {
		name string
		edit func([]byte) []byte
	}{
		{"short", func([]byte) []byte { return make([]byte, 511) }},
		{"sector-size", func(b []byte) []byte { b[0x0b], b[0x0c] = 0, 4; return b }},
		{"zero-total", func(b []byte) []byte { b[0x13], b[0x14] = 0, 0; return b }},
		{"zero-sectors-per-track", func(b []byte) []byte { b[0x18], b[0x19] = 0, 0; return b }},
		{"invalid-sides", func(b []byte) []byte { b[0x1a], b[0x1b] = 3, 0; return b }},
		{"truncated", func(b []byte) []byte { return b[:len(b)-1] }},
		{"trailing", func(b []byte) []byte { return append(b, 0) }},
		{"partial-cylinder", func(b []byte) []byte {
			b = b[:len(b)-rawFloppySectorSize]
			total := uint16(len(b) / rawFloppySectorSize)
			b[0x13], b[0x14] = byte(total), byte(total>>8)
			return b
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			image := append([]byte(nil), valid...)
			if _, err := NewRawFloppy(test.edit(image)); err == nil {
				t.Fatal("malformed image unexpectedly accepted")
			}
		})
	}
}

func TestRawFloppyRejectsOutOfRangeCHS(t *testing.T) {
	floppy, err := NewRawFloppy(testRawFloppy(80, 2, 9))
	if err != nil {
		t.Fatal(err)
	}
	for _, chs := range [][3]uint16{{80, 0, 1}, {0, 2, 1}, {0, 0, 0}, {0, 0, 10}} {
		if _, err := floppy.Sector(chs[0], chs[1], chs[2]); err == nil ||
			!strings.Contains(err.Error(), "CHS out of range") {
			t.Fatalf("CHS %v error=%v", chs, err)
		}
	}
}

func TestMachineAttachFloppyAIsAtomicAndSurvivesReset(t *testing.T) {
	machine, err := NewMachine(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	image := testRawFloppy(80, 2, 9)
	if err := machine.AttachFloppyA(image); err != nil {
		t.Fatal(err)
	}
	image[rawFloppySectorSize] ^= 0xff
	want, err := machine.Memory.floppyA.Sector(0, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.AttachFloppyA(make([]byte, 511)); err == nil {
		t.Fatal("invalid replacement unexpectedly accepted")
	}
	machine.Memory.ColdReset()
	got, err := machine.Memory.floppyA.Sector(0, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatal("failed replacement or cold reset changed mounted media")
	}
}

func testRawFloppy(tracks, sides, sectorsPerTrack uint16) []byte {
	total := tracks * sides * sectorsPerTrack
	image := make([]byte, int(total)*rawFloppySectorSize)
	image[0x0b], image[0x0c] = 0, 2
	image[0x13], image[0x14] = byte(total), byte(total>>8)
	image[0x18], image[0x19] = byte(sectorsPerTrack), byte(sectorsPerTrack>>8)
	image[0x1a], image[0x1b] = byte(sides), byte(sides>>8)
	for sector := uint16(0); sector < total; sector++ {
		image[int(sector)*rawFloppySectorSize] = byte(sector)
	}
	return image
}
