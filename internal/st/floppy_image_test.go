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
		memory.floppyMediaPhase != floppyMediaSeekDataSelector {
		t.Fatalf("status=%04x IRQ/GPIP/phase=%v/%02x/%d err=%v", status, memory.fdcIRQ,
			memory.mfpGPIPIn, memory.floppyMediaPhase, err)
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
}

func TestEmuTOSMountedFloppyCompletesFirstMediaRead(t *testing.T) {
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
