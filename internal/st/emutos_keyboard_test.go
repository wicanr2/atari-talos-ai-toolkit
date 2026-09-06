package st

import "testing"

func pressKey(t *testing.T, machine *Machine, scanCode byte, settle int) {
	t.Helper()
	if err := machine.QueueKey(scanCode, true); err != nil {
		t.Fatalf("按下 %#02x：%v", scanCode, err)
	}
	runSteps(t, machine, 100_000)
	if err := machine.QueueKey(scanCode, false); err != nil {
		t.Fatalf("放開 %#02x：%v", scanCode, err)
	}
	runSteps(t, machine, settle)
}

// TestEmuTOSClosesADialogWithReturn is the fixed-ROM receipt for spec 145 (and
// for spec 144, which it needs — every keypress makes EmuTOS click the PSG).
// The mouse opens Desk → Desktop Info..., then Return closes it. Pressing "1"
// first is the negative control: a scan code the dialog does not act on leaves
// the screen alone.
func TestEmuTOSClosesADialogWithReturn(t *testing.T) {
	machine := emuTOSMachine(t)
	runSteps(t, machine, 14_000_000)

	// 游標從 (159,99) 移到選單列的 Desk 上，選單會自己下拉。
	moveMouse(t, machine, -100, -97)
	moveMouse(t, machine, -39, 0)
	runSteps(t, machine, 600_000)
	menu := emuTOSFrame(t, machine.Memory)

	// 往下移到「Desktop Info...」那一列再點下去。
	moveMouse(t, machine, 10, 12)
	runSteps(t, machine, 300_000)
	clickHold(t, machine, true, false, 40_000)
	clickHold(t, machine, false, false, 1_500_000)
	dialog := emuTOSFrame(t, machine.Memory)
	if _, _, _, _, count := changedBox(menu, dialog); count != 29830 {
		t.Fatalf("對話框開啟的變動像素是 %d，應該是 29830", count)
	}

	// 負對照：`1` 鍵（$02）對這個對話框沒有作用。
	pressKey(t, machine, 0x02, 800_000)
	if _, _, _, _, count := changedBox(dialog, emuTOSFrame(t, machine.Memory)); count != 0 {
		t.Errorf("按 `1` 竟然讓 %d 個像素變了", count)
	}

	// Return（$1C）關掉對話框。
	pressKey(t, machine, 0x1c, 1_500_000)
	if _, _, _, _, count := changedBox(dialog, emuTOSFrame(t, machine.Memory)); count != 29256 {
		t.Errorf("按 Return 的變動像素是 %d，應該是 29256", count)
	}
}

// TestEmuTOSTellsAClickFromADoubleClick is the fixed-ROM receipt for spec 146:
// the IKBD has no notion of a double click, so all the packet stream does is
// put four button packets on the wire close together. GEM does the timing —
// and its two outcomes differ.
func TestEmuTOSTellsAClickFromADoubleClick(t *testing.T) {
	machine := emuTOSMachine(t)
	runSteps(t, machine, 14_000_000)
	moveMouse(t, machine, -127, -67)
	onIcon := emuTOSFrame(t, machine.Memory)
	receiptsBefore := machine.Memory.floppyMediaReceipts.Total

	// 雙擊：兩次快速點擊。桌面走的是開啟那條路，沒有留下反白。
	for i := 0; i < 2; i++ {
		clickHold(t, machine, true, false, 30_000)
		clickHold(t, machine, false, false, 30_000)
	}
	runSteps(t, machine, 2_000_000)
	if _, _, _, _, count := changedBox(onIcon, emuTOSFrame(t, machine.Memory)); count != 0 {
		t.Errorf("雙擊之後還有 %d 個像素和點擊前不同，應該沒有留下反白", count)
	}
	// 沒有磁片可開，所以桌面根本沒去碰磁碟機。
	if got := machine.Memory.floppyMediaReceipts.Total; got != receiptsBefore {
		t.Errorf("雙擊之後媒體收據從 %d 變成 %d，桌面不該碰磁碟機", receiptsBefore, got)
	}

	// 單擊：同一個位置，結束狀態不一樣——圖示留著反白。
	clickHold(t, machine, true, false, 40_000)
	clickHold(t, machine, false, false, 600_000)
	x0, y0, x1, y1, count := changedBox(onIcon, emuTOSFrame(t, machine.Memory))
	if count != 1140 || x0 != 0 || y0 != 17 || x1 != 71 || y1 != 50 {
		t.Errorf("單擊：變動框 (%d,%d)-(%d,%d) count=%d，應該是 (0,17)-(71,50) 的 1140 個像素",
			x0, y0, x1, y1, count)
	}
}
