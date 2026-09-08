package st

import (
	"bytes"
	"crypto/sha256"
	"os"
	"testing"
)

// TestPrivateRawFloppy validates a caller-supplied image without changing it.
// This is a media check, not a game boot or behavioral parity receipt.
func TestPrivateRawFloppy(t *testing.T) {
	path := os.Getenv("TALOS_RAW_FLOPPY")
	if path == "" {
		t.Skip("設定 TALOS_RAW_FLOPPY 以驗證私人磁片")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("bytes=%d sha256=%x", len(data), sha256.Sum256(data))
	floppy, err := NewRawFloppy(data)
	if err != nil {
		t.Fatalf("磁片掛載前驗證失敗：%v", err)
	}
	tracks, sides, sectors := floppy.Geometry()
	t.Logf("tracks=%d sides=%d sectors_per_track=%d", tracks, sides, sectors)
	offset := 0
	for track := uint16(0); track < tracks; track++ {
		for side := uint16(0); side < sides; side++ {
			for sector := uint16(1); sector <= sectors; sector++ {
				got, err := floppy.Sector(track, side, sector)
				if err != nil || !bytes.Equal(got, data[offset:offset+512]) {
					t.Fatalf("CHS=%d/%d/%d 資料不符：%v", track, side, sector, err)
				}
				offset += 512
			}
		}
	}
	if offset != len(data) {
		t.Fatalf("讀取 %d bytes，輸入有 %d bytes", offset, len(data))
	}
}
