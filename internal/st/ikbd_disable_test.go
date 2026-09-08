package st

import "testing"

func TestIKBDDisableAndResumeMouse(t *testing.T) {
	m := ikbdReadyForCommands(t)
	for _, b := range []byte{8, 0x10, 0x0b, 1, 1, 0x12, 0x1a} {
		if err := sendIKBDByte(t, m, b); err != nil {
			t.Fatal(err)
		}
	}
	if !m.ikbdMouseDisabled || !m.ikbdJoystickDisabled {
		t.Fatal("disable state missing")
	}
	if err := m.QueueMouseMotion(20, 10, true, false); err != nil {
		t.Fatal(err)
	}
	if m.ikbdUplinkCount != 0 {
		t.Fatal("disabled mouse generated packet")
	}
	if err := sendIKBDByte(t, m, 8); err != nil {
		t.Fatal(err)
	}
	if err := m.QueueMouseMotion(20, 10, true, false); err != nil {
		t.Fatal(err)
	}
	if m.ikbdUplinkCount != 3 {
		t.Fatal("resumed mouse missing packet")
	}
	m.ColdReset()
	if m.ikbdMouseDisabled || m.ikbdJoystickDisabled {
		t.Fatal("reset retained disable state")
	}
}
