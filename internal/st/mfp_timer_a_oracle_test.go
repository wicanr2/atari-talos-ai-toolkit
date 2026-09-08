package st

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"
	"os"
	"path/filepath"
	"testing"
)

// 自製 68000 微型程式：比較硬體契約，不是原版遊戲完成證據。
func TestTimerAOracleProbe(t *testing.T) {
	const base = uint32(0x10000)
	const receipt = uint32(0x11000)
	code := []byte{}
	words := func(ws ...uint16) {
		for _, w := range ws {
			code = binary.BigEndian.AppendUint16(code, w)
		}
	}
	write := func(a uint32, v byte) { words(0x13fc, uint16(v), uint16(a>>16), uint16(a)) }
	read := func(a uint32, index uint32) {
		words(0x13f9, uint16(a>>16), uint16(a), uint16((receipt+index)>>16), uint16(receipt+index))
	}
	delay := func() { words(0x303c, 200, 0x4e71, 0x51c8, 0xfffc) }
	write(MFPTACR, 0)
	write(MFPIERA, 0x20)
	write(MFPIMRA, 0)
	write(MFPTADR, 3)
	read(MFPTADR, 0)
	write(MFPTACR, 7)
	write(MFPTADR, 7)
	read(MFPTADR, 1)
	delay()
	write(MFPTACR, 0)
	read(MFPTADR, 2)
	delay()
	read(MFPTADR, 3)
	read(MFPIPRA, 4)
	write(MFPIERA, 0)
	read(MFPIPRA, 5)
	end := base + uint32(len(code))
	words(0x4e71, 0x4e71, 0x4e71)
	m, err := NewMachine(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	m.Memory.mmuConfig = 0x05
	for i, b := range code {
		if err := m.Memory.WriteByteFC(base+uint32(i), b, 5); err != nil {
			t.Fatal(err)
		}
	}
	m.CPU.State = m68k.State{PC: base + 4, SR: 0x2700, SSP: 0x8000, Prefetch: [2]uint16{binary.BigEndian.Uint16(code), binary.BigEndian.Uint16(code[2:])}}
	m.nextVBLClock = 1_000_000
	m.vblFrameClocks = colorST50HzFrameClocks
	for n := 0; m.CPU.State.PC != end+4; n++ {
		if n >= 10000 {
			t.Fatal("probe did not terminate")
		}
		if _, err := m.Step(); err != nil {
			t.Fatal(err)
		}
	}
	got := make([]byte, 6)
	for i := range got {
		got[i], err = m.Memory.ReadByteFC(receipt+uint32(i), 5)
		if err != nil {
			t.Fatal(err)
		}
	}
	checkTimerAReceipt(t, got)
	t.Logf("自製 probe sha=%x clocks=%d receipt=%v", sha256.Sum256(code), m.Clocks, got)
	if out := os.Getenv("TALOS_TIMERA_ORACLE_OUT"); out != "" {
		for name, data := range map[string][]byte{
			"timera-probe.bin": code, "timera-talos.bin": got,
			"timera-init.ini":       []byte("b VBL=10 :once :file /out/timera-oracle.ini\nc\n"),
			"timera-oracle.ini":     []byte(fmt.Sprintf("loadbin /out/timera-probe.bin $10000\nr sr=$2700\nr pc=$10000\nb pc=$%x :once :file /out/timera-oracle-end.ini\nc\n", end)),
			"timera-oracle-end.ini": []byte("savebin /out/timera-hatari.bin $11000 6\nquit\n"),
		} {
			if err := os.WriteFile(filepath.Join(out, name), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	if path := os.Getenv("TALOS_TIMERA_HATARI_RECEIPT"); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		checkTimerAReceipt(t, raw)
		t.Logf("Hatari receipt=%v；counter 相位不宣稱逐週期一致", raw)
	}
}

func checkTimerAReceipt(t *testing.T, b []byte) {
	t.Helper()
	if len(b) != 6 || b[0] != 3 || b[1] != 3 || b[2] < 1 || b[2] > 7 || b[2] != b[3] || b[4]&0x20 == 0 || b[5]&0x20 != 0 {
		t.Fatalf("Timer A receipt 不符停止載入／延後重載／停止保存／pending／disable 契約: %v", b)
	}
}
