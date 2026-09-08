package st

import "testing"

// 按鍵狀態改變就發一個位移為零的封包；狀態沒變就不發（規格 143）。
func TestIKBDMouseButtonPackets(t *testing.T) {
	memory := ikbdMouseReady(t)
	if err := memory.QueueMouseButtons(true, false); err != nil {
		t.Fatal(err)
	}
	ikbdDrain(t, memory, 0xfa, 0x00, 0x00)
	if err := memory.QueueMouseButtons(true, false); err != nil {
		t.Fatal(err)
	}
	if memory.ikbdUplinkCount != 0 {
		t.Fatalf("同樣的按鍵狀態又發了封包：%d 個位元組", memory.ikbdUplinkCount)
	}
	if err := memory.QueueMouseButtons(false, false); err != nil {
		t.Fatal(err)
	}
	ikbdDrain(t, memory, 0xf8, 0x00, 0x00)
}

func clickHold(t *testing.T, machine *Machine, left, right bool, steps int) {
	t.Helper()
	if err := machine.QueueMouseButtons(left, right); err != nil {
		t.Fatalf("排按鍵封包 left=%v right=%v：%v", left, right, err)
	}
	runSteps(t, machine, steps)
}

// TestEmuTOSSelectsAnIconWithTheLeftButton pins the header's button bits by
// behaviour rather than by documentation: EmuTOS's desktop selects the DISK A
// icon on a left click and ignores the right button completely.
func TestEmuTOSSelectsAnIconWithTheLeftButton(t *testing.T) {
	machine := emuTOSMachine(t)
	runSteps(t, machine, 14_000_000)

	// 游標從 (159,99) 移到 DISK A 圖示上。
	moveMouse(t, machine, -127, -67)
	onIcon := emuTOSFrame(t, machine.Memory)

	// 按住不放：圖示反白。
	clickHold(t, machine, true, false, 300_000)
	held := emuTOSFrame(t, machine.Memory)
	x0, y0, x1, y1, count := changedBox(onIcon, held)
	if count == 0 || x0 != 0 || y0 != 11 || x1 != 72 || y1 != 51 {
		t.Errorf("左鍵按住：變動框 (%d,%d)-(%d,%d) count=%d，應該是 (0,11)-(72,51) 且非空",
			x0, y0, x1, y1, count)
	}

	// 按太久又沒移動，GEM 會取消；放開之後畫面回到按下之前。
	clickHold(t, machine, false, false, 300_000)
	if _, _, _, _, count := changedBox(onIcon, emuTOSFrame(t, machine.Memory)); count != 0 {
		t.Errorf("長按放開之後還有 %d 個像素沒回到原狀", count)
	}

	// 短按一下才是點選：圖示維持反白。
	clickHold(t, machine, true, false, 40_000)
	clickHold(t, machine, false, false, 400_000)
	selected := emuTOSFrame(t, machine.Memory)
	x0, y0, x1, y1, count = changedBox(onIcon, selected)
	if count == 0 || x0 != 0 || y0 != 17 || x1 != 71 || y1 != 50 {
		t.Errorf("左鍵點選：變動框 (%d,%d)-(%d,%d) count=%d，應該是 (0,17)-(71,50) 且非空",
			x0, y0, x1, y1, count)
	}

	// 右鍵是負對照：同一個位置按下與放開，桌面完全沒反應。
	clickHold(t, machine, false, true, 300_000)
	if _, _, _, _, count := changedBox(selected, emuTOSFrame(t, machine.Memory)); count != 0 {
		t.Errorf("右鍵按下竟然讓 %d 個像素變了", count)
	}
	clickHold(t, machine, false, false, 300_000)
	if _, _, _, _, count := changedBox(selected, emuTOSFrame(t, machine.Memory)); count != 0 {
		t.Errorf("右鍵放開竟然讓 %d 個像素變了", count)
	}
}
