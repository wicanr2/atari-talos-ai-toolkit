package st

import "testing"

func TestMFPDisableChannelsClearsPendingOnly(t *testing.T) {
	m, e := NewMemory(RAM1M, testROM())
	if e != nil {
		t.Fatal(e)
	}
	m.mfpIERA = 0xff
	m.mfpIPRA = 0xff
	m.mfpISRA = 0xff
	m.mfpIERB = 0xff
	m.mfpIPRB = 0xff
	m.mfpISRB = 0xff
	for _, addr := range []uint32{MFPIERA, MFPIERB} {
		if e := m.WriteByteFC(addr, 0x40, 5); e != nil {
			t.Fatal(e)
		}
		if e := m.WriteByteFC(addr, 0x40, 5); e != nil {
			t.Fatal(e)
		}
	}
	if m.mfpIERA != 0x40 || m.mfpIERB != 0x40 || m.mfpIPRA != 0x40 || m.mfpIPRB != 0x40 || m.mfpISRA != 0xff || m.mfpISRB != 0xff {
		t.Fatal("disable changed unrelated state")
	}
	if e := m.WriteByteFC(MFPIERB, 0x80, 5); e == nil {
		t.Fatal("new unmodeled channel enabled")
	}
}
