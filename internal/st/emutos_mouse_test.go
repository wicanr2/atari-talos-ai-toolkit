package st

import "testing"

// emuTOSFrame 把作用中的 framebuffer 解成 320×200 的 palette index。
func emuTOSFrame(t *testing.T, m *Memory) []byte {
	t.Helper()
	base := m.ProgrammedVideoBase()
	frame := make([]byte, 320*200)
	for y := 0; y < 200; y++ {
		for word := 0; word < 20; word++ {
			var planes [4]uint16
			for plane := 0; plane < 4; plane++ {
				hi, err := m.ReadByteFC(base+uint32(y*160+word*8+plane*2), 5)
				if err != nil {
					t.Fatal(err)
				}
				lo, err := m.ReadByteFC(base+uint32(y*160+word*8+plane*2+1), 5)
				if err != nil {
					t.Fatal(err)
				}
				planes[plane] = uint16(hi)<<8 | uint16(lo)
			}
			for bit := 0; bit < 16; bit++ {
				index := byte(0)
				for plane := 0; plane < 4; plane++ {
					index |= byte((planes[plane]>>(15-bit))&1) << plane
				}
				frame[y*320+word*16+bit] = index
			}
		}
	}
	return frame
}

// changedBox 回報兩張畫面相異像素的外接矩形。桌面上唯一會動的東西是游標，
// 所以移動之後這個矩形就是「舊游標∪新游標」——寬高各等於游標尺寸加上位移量。
func changedBox(a, b []byte) (x0, y0, x1, y1 int, count int) {
	x0, y0, x1, y1 = 320, 200, -1, -1
	for i := range a {
		if a[i] == b[i] {
			continue
		}
		x, y := i%320, i/320
		if x < x0 {
			x0 = x
		}
		if x > x1 {
			x1 = x
		}
		if y < y0 {
			y0 = y
		}
		if y > y1 {
			y1 = y
		}
		count++
	}
	return
}

func runSteps(t *testing.T, machine *Machine, steps int) {
	t.Helper()
	for step := 0; step < steps; step++ {
		if _, err := machine.Step(); err != nil {
			t.Fatalf("第 %d 步：%v", step, err)
		}
	}
}

func moveMouse(t *testing.T, machine *Machine, deltaX, deltaY int) {
	t.Helper()
	if err := machine.QueueMouseMotion(deltaX, deltaY, false, false); err != nil {
		t.Fatalf("排封包 (%d,%d)：%v", deltaX, deltaY, err)
	}
	runSteps(t, machine, 200_000)
}

// TestEmuTOSMovesThePointer is the fixed-ROM receipt for spec 142: EmuTOS's own
// VDI is the oracle. The arrow is 10×16 and sits at (159,99) on a freshly drawn
// desktop, so a move of (dx,dy) has to grow the changed box to
// (10+|dx|)×(16+|dy|) — that pins both magnitudes without depending on any
// ROM-version-specific variable address.
func TestEmuTOSMovesThePointer(t *testing.T) {
	machine := emuTOSMachine(t)
	runSteps(t, machine, 14_000_000)
	base := emuTOSFrame(t, machine.Memory)

	for _, test := range []struct {
		name          string
		deltaX        int
		deltaY        int
		width, height int
	}{
		{"只往右 30", 30, 0, 10 + 30, 16},
		{"只往下 40", 0, 40, 10, 16 + 40},
		{"右下 (20,10)", 20, 10, 10 + 20, 16 + 10},
	} {
		t.Run(test.name, func(t *testing.T) {
			moveMouse(t, machine, test.deltaX, test.deltaY)
			moved := emuTOSFrame(t, machine.Memory)
			x0, y0, x1, y1, count := changedBox(base, moved)
			if x1-x0+1 != test.width || y1-y0+1 != test.height {
				t.Errorf("變動框 (%d,%d)-(%d,%d) 是 %dx%d，應該是 %dx%d（count=%d）",
					x0, y0, x1, y1, x1-x0+1, y1-y0+1, test.width, test.height, count)
			}
			// 游標的原點在 (159,99)：往右下移動只會讓框往右下長。
			if x0 != 159 || y0 != 99 {
				t.Errorf("變動框左上角是 (%d,%d)，應該還在游標原點 (159,99)", x0, y0)
			}
			moveMouse(t, machine, -test.deltaX, -test.deltaY)
			back := emuTOSFrame(t, machine.Memory)
			if _, _, _, _, count := changedBox(base, back); count != 0 {
				t.Errorf("移回原點之後還有 %d 個像素和基準不同", count)
			}
		})
	}

	// 方向由畫面邊界釘死：往負方向推得夠遠，游標會停在左上角；往正方向推，
	// 會停在右下角。這證明負是左／上、正是右／下，不必去讀 ROM 的滑鼠變數。
	// 前面每個子測試都把游標移回原點了，所以這裡的起點仍是 (159,99)。
	t.Run("推到畫面邊界", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			moveMouse(t, machine, -100, -100)
		}
		if x0, y0, _, _, _ := changedBox(base, emuTOSFrame(t, machine.Memory)); x0 != 0 || y0 > 2 {
			t.Errorf("往負方向推到底，變動框左上角是 (%d,%d)，應該貼著畫面左上", x0, y0)
		}
		for i := 0; i < 8; i++ {
			moveMouse(t, machine, 100, 100)
		}
		if _, _, x1, y1, _ := changedBox(base, emuTOSFrame(t, machine.Memory)); x1 != 319 || y1 != 199 {
			t.Errorf("往正方向推到底，變動框右下角是 (%d,%d)，應該貼著畫面右下", x1, y1)
		}
	})
}
