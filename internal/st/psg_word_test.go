package st

import (
	"github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"
	"testing"
)

func TestPSGWordHighLane(t *testing.T) {
	m, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	m.psgRegisters[7] = 0xc0
	if err := m.WriteWord(PSGRegisterSelect, 0x03ff, 5); err != nil {
		t.Fatal(err)
	}
	if err := m.WriteWord(PSGRegisterData, 0x05cc, 5); err != nil {
		t.Fatal(err)
	}
	if m.psgRegisterSelect != 3 || m.psgRegisters[3] != 5 {
		t.Fatal("wrong word lane")
	}
	wait, err := m.WriteWordAt(PSGRegisterData, 0x0700, m68k.BusAccess{Clock: 100, FunctionCode: 5})
	if err != nil || wait != 4 || m.psgRegisters[3] != 7 {
		t.Fatalf("wait=%d err=%v", wait, err)
	}
	if err := m.WriteWord(PSGRegisterData, 0x0900, 1); err == nil {
		t.Fatal("user access accepted")
	}
	if m.psgRegisters[3] != 7 {
		t.Fatal("failed access mutated PSG")
	}
}
