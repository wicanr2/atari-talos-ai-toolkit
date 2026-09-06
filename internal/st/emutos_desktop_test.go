package st

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

// TestEmuTOSReachesTheDesktop is the fixed-ROM receipt for spec 140: with the
// last port A gate gone, EmuTOS 1.3 boots all the way into GEM. The screen is
// finished well before this many instructions and then stays put — nothing is
// moving because nothing is feeding the machine any input.
//
// The hash is over the 32,000 bytes at $F8000: the menu bar (Desk File View
// Options), the two disk icons, the trash can and the mouse pointer.
func TestEmuTOSReachesTheDesktop(t *testing.T) {
	const (
		steps      = 14_000_000
		wantScreen = "1de1eb45e862218844abe07ae05fda4c4a9453817ed0ab348a374bca67768f78"
	)
	machine := emuTOSMachine(t)
	for step := 0; step < steps; step++ {
		if _, gate := machine.Step(); gate != nil {
			t.Fatalf("開機路徑在第 %d 條停住：%v", step, gate)
		}
	}
	m := machine.Memory
	if base := m.ProgrammedVideoBase(); base != 0x000f8000 {
		t.Errorf("video base=%#08x，應該是 $F8000", base)
	}
	if m.shifterResolution != 0 {
		t.Errorf("解析度=%d，應該是 0（320×200 四平面）", m.shifterResolution)
	}
	frame := make([]byte, 32000)
	for i := range frame {
		value, err := m.ReadByte(m.ProgrammedVideoBase()+uint32(i), 5)
		if err != nil {
			t.Fatalf("讀畫面第 %d 個 byte：%v", i, err)
		}
		frame[i] = value
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(frame)); got != wantScreen {
		t.Errorf("桌面畫面 SHA-256=%s，應該是 %s", got, wantScreen)
	}
	// 預設的十六色色盤：EmuTOS 開機時自己寫進去的那一組。
	want := [16]uint16{0x777, 0x700, 0x070, 0x770, 0x007, 0x707, 0x077, 0x555,
		0x333, 0x733, 0x373, 0x773, 0x337, 0x737, 0x377, 0x000}
	if m.shifterPalette != want {
		t.Errorf("色盤=%v，應該是 %v", m.shifterPalette, want)
	}
}
