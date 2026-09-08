package st

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 自製探針：CPU 寫入 Timer B，再持續讀回；不使用遊戲或外部模擬器程式碼。
func TestTimerBExportOracleProbe(t *testing.T) {
	out := os.Getenv("TALOS_TIMERB_ORACLE_OUT")
	if out == "" {
		t.Skip("設定 TALOS_TIMERB_ORACLE_OUT 匯出外部探針")
	}
	words := []uint16{0x13fc, 0, 0x00ff, 0xfa1b, 0x13fc, 255, 0x00ff, 0xfa21, 0x13fc, 8, 0x00ff, 0xfa1b, 0x1039, 0x00ff, 0xfa21, 0x60f8}
	code := make([]byte, len(words)*2)
	for i, v := range words {
		binary.BigEndian.PutUint16(code[i*2:], v)
	}
	for name, data := range map[string][]byte{
		"timerb-probe.bin": code,
		"timerb-init.ini":  []byte("b VBL=10 :once :file /out/timerb-start.ini\nc\n"),
		"timerb-start.ini": []byte("loadbin /out/timerb-probe.bin $10000\nr sr=$2700\nr pc=$10000\nc\n"),
	} {
		if err := os.WriteFile(filepath.Join(out, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTimerBHatariDisplayEdgeReceipt(t *testing.T) {
	path := os.Getenv("TALOS_TIMERB_HATARI_TRACE")
	if path == "" {
		t.Skip("設定 TALOS_TIMERB_HATARI_TRACE 回讀獨立收據")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	count := 0
	for i, line := range lines {
		if !strings.Contains(line, "mfp/video timer B new event count") {
			continue
		}
		var row, clock, phase, pending int
		if i == 0 {
			t.Fatal("missing edge receipt")
		}
		if n, err := fmt.Sscanf(lines[i-1], "EndLine TB %d video_cyc=%d line_cyc=%d pending_int_cnt=%d", &row, &clock, &phase, &pending); err != nil || n != 4 {
			t.Fatalf("bad receipt %s", lines[i-1])
		}
		if row != 63+count%200 || phase-pending != 400 {
			t.Fatalf("edge %d row=%d phase=%d pending=%d", count, row, phase, pending)
		}
		count++
	}
	if count != 400 {
		t.Fatalf("got %d edges, want 400", count)
	}
}
