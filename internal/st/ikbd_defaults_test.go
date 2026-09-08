package st

import "testing"

func TestIKBDPowerupMousePacketWithoutInitmous(t *testing.T) {
	m := ikbdReadyForCommands(t)
	m.ColdReset()
	m.ikbdACIAConfigured = true
	m.ikbdACIAStatus = 2
	if err := m.QueueMouseMotion(2, 3, true, false); err != nil {
		t.Fatal(err)
	}
	ikbdDrain(t, m, 0xfa, 2, 3)
}

func TestIKBDControllerResetRestoresMouseDefaults(t *testing.T) {
	m := ikbdReadyForCommands(t)
	m.ikbdMouseDisabled = true
	m.ikbdRelativeMouse, m.ikbdYAxisUp = false, false
	m.ikbdMouseThreshold = [2]byte{3, 4}
	m.ikbdMouseButtonAction = 4
	m.ikbdACIATXShift, m.ikbdACIATXShiftTicks = 1, 1
	m.advanceIKBDACIAClock()
	if !m.ikbdResetCommandDone || m.ikbdMouseDisabled || !m.ikbdRelativeMouse ||
		!m.ikbdYAxisUp || m.ikbdMouseThreshold != [2]byte{1, 1} || m.ikbdMouseButtonAction != 0 {
		t.Fatal("控制器重設沒有恢復滑鼠預設")
	}
}
