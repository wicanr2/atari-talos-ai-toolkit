package st

import (
	"bytes"
	"testing"
)

func TestExplicitRawGeometryPreservesExtraTracks(t *testing.T) {
	data := append(testRawFloppy(80, 2, 10), bytes.Repeat([]byte{0xa7}, 2*2*10*512)...)
	if _, err := NewRawFloppy(data); err == nil {
		t.Fatal("BPB mismatch accepted")
	}
	f, err := NewRawFloppyGeometry(data, 82, 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-1] = 0
	last, err := f.Sector(81, 1, 10)
	if err != nil || !bytes.Equal(last, bytes.Repeat([]byte{0xa7}, 512)) {
		t.Fatalf("last sector: %v", err)
	}
	last[0] = 0
	again, _ := f.Sector(81, 1, 10)
	if again[0] != 0xa7 {
		t.Fatal("returned sector aliases medium")
	}
	for _, g := range [][3]uint16{{80, 2, 10}, {0, 2, 10}, {82, 0, 10}, {82, 3, 10}, {82, 2, 0}, {65535, 2, 65535}} {
		if _, err := NewRawFloppyGeometry(data, g[0], g[1], g[2]); err == nil {
			t.Fatalf("accepted %v", g)
		}
	}
	if _, err := f.Sector(82, 0, 1); err == nil {
		t.Fatal("out of range accepted")
	}
}
