package st

import (
	"strings"
	"testing"
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
