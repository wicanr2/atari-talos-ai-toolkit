package st

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// Diagnostic boot from reset. No guest code or guest state is patched.
func TestPrivateDiskBoot(t *testing.T) {
	path := os.Getenv("TALOS_BOOT_DISK")
	if path == "" {
		t.Skip("設定 TALOS_BOOT_DISK 與 TALOS_TOS_ROM")
	}
	rom, err := os.ReadFile(os.Getenv("TALOS_TOS_ROM"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewMachine(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	var f *RawFloppy
	geometry := os.Getenv("TALOS_BOOT_GEOMETRY")
	if geometry == "" {
		f, err = NewRawFloppy(data)
	} else {
		var tracks, sides, sectors uint16
		if n, e := fmt.Sscanf(geometry, "%d/%d/%d", &tracks, &sides, &sectors); n != 3 || e != nil {
			t.Fatal("invalid TALOS_BOOT_GEOMETRY")
		}
		f, err = NewRawFloppyGeometry(data, tracks, sides, sectors)
	}
	if err != nil {
		t.Fatal(err)
	}
	m.Memory.attachFloppyA(f)
	t.Logf("rom=%x disk=%x explicit_geometry=%q", sha256.Sum256(rom), sha256.Sum256(data), geometry)
	if err := m.Reset(); err != nil {
		t.Fatal(err)
	}
	steps := 14_000_000
	if raw := os.Getenv("TALOS_BOOT_STEPS"); raw != "" {
		n, e := strconv.Atoi(raw)
		if e != nil || n < 1 || n > 100_000_000 {
			t.Fatal("TALOS_BOOT_STEPS must be 1..100000000")
		}
		steps = n
	}
	for i := 0; i < steps; i++ {
		if _, err := m.Step(); err != nil {
			privateBootFrame(t, m)
			t.Fatalf("step=%d instructions=%d clocks=%d PC=%08x state=%+v gate=%v", i, m.Instructions, m.Clocks, m.CPU.State.PC, m.CPU.State, err)
		}
		if i%1_000_000 == 0 {
			t.Logf("step=%d PC=%08x clocks=%d track=%d", i, m.CPU.State.PC, m.Clocks, m.Memory.fdcHeadTrack)
		}
	}
	privateBootFrame(t, m)
	if os.Getenv("TALOS_BOOT_ENTER") == "1" {
		// DM 入口的正常游標起點即 ENTER；只送 IKBD 按下／放開，不改遊戲狀態。
		for phase, pressed := range []bool{true, false} {
			if err := m.QueueMouseMotion(0, 0, pressed, false); err != nil {
				t.Fatal(err)
			}
			limit := 10000
			if !pressed {
				limit = 1_000_000
				if raw := os.Getenv("TALOS_BOOT_ENTER_STEPS"); raw != "" {
					n, err := strconv.Atoi(raw)
					if err != nil || n < 1 || n > 100_000_000 {
						t.Fatal("TALOS_BOOT_ENTER_STEPS must be 1..100000000")
					}
					limit = n
				}
			}
			for i := 0; i < limit; i++ {
				if _, err := m.Step(); err != nil {
					privateBootFrame(t, m, "talos-after-enter.png")
					t.Fatalf("ENTER phase=%d step=%d PC=%08x state=%+v: %v", phase, i, m.CPU.State.PC, m.CPU.State, err)
				}
				if i%5_000_000 == 0 {
					t.Logf("ENTER phase=%d step=%d PC=%08x clocks=%d track=%d", phase, i, m.CPU.State.PC, m.Clocks, m.Memory.fdcHeadTrack)
				}
			}
		}
		privateBootFrame(t, m, "talos-after-enter.png")
	}
	if path := os.Getenv("TALOS_BOOT_ACTIONS"); path != "" {
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		var actions []struct {
			Key         byte   `json:"key"`
			DX          int    `json:"dx"`
			DY          int    `json:"dy"`
			Left        bool   `json:"left"`
			Right       bool   `json:"right"`
			Clocks      uint64 `json:"clocks"`
			FrameSHA256 string `json:"frame_sha256"`
		}
		decoder := json.NewDecoder(file)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&actions); err != nil {
			t.Fatal(err)
		}
		if len(actions) > 200 {
			t.Fatal("too many actions")
		}
		for index, a := range actions {
			if a.Clocks < 1 || a.Clocks > 20_000_000 {
				t.Fatal("action clocks must be 1..20000000")
			}
			if a.Key != 0 {
				if err := m.QueueKey(a.Key, true); err != nil {
					t.Fatal(err)
				}
			} else if err := m.QueueMouseMotion(a.DX, a.DY, a.Left, a.Right); err != nil {
				t.Fatal(err)
			}
			deadline := m.Clocks + a.Clocks
			release := m.Clocks + 160000
			for m.Clocks < deadline {
				if a.Key != 0 && m.Clocks >= release {
					if err := m.QueueKey(a.Key, false); err != nil {
						t.Fatal(err)
					}
					a.Key = 0
				}
				if _, err := m.Step(); err != nil {
					privateBootFrame(t, m, fmt.Sprintf("talos-action-%03d.png", index))
					t.Fatalf("action=%d PC=%08x: %v", index, m.CPU.State.PC, err)
				}
			}
			if a.Key != 0 {
				if err := m.QueueKey(a.Key, false); err != nil {
					t.Fatal(err)
				}
			}
			t.Logf("action=%d clocks=%d TimerB events=%d iack=%d", index, m.Clocks, m.Memory.mfpTimerBEvents, m.Memory.mfpTimerBAcknowledged)
			privateBootFrame(t, m, fmt.Sprintf("talos-action-%03d.png", index))
			if a.FrameSHA256 != "" {
				frame, _, _, err := m.Framebuffer()
				if err != nil {
					t.Fatal(err)
				}
				if got := fmt.Sprintf("%x", sha256.Sum256(frame)); got != a.FrameSHA256 {
					t.Fatalf("action=%d framebuffer=%s want=%s", index, got, a.FrameSHA256)
				}
			}
		}
	}
	t.Log("步數上限已到；需目視與玩家操作驗證，不能僅以未觸發 gate 判定遊戲啟動")
}

func privateBootFrame(t *testing.T, m *Machine, names ...string) {
	t.Helper()
	t.Logf("Timer B: mode=%d data=%d main=%d events=%d iack=%d next=%d", m.Memory.mfpTBCR, m.Memory.mfpTBDR, m.Memory.mfpTBMain, m.Memory.mfpTimerBEvents, m.Memory.mfpTimerBAcknowledged, m.Memory.mfpTimerBNextEdge)
	t.Logf("Timer A: mode=%d data=%d main=%d timeouts=%d iack=%d enable=%02x mask=%02x pending=%02x service=%02x", m.Memory.mfpTACR, m.Memory.mfpTADR, m.Memory.mfpTAMain, m.Memory.mfpTimerATimeouts, m.Memory.mfpTimerAAcknowledged, m.Memory.mfpIERA, m.Memory.mfpIMRA, m.Memory.mfpIPRA, m.Memory.mfpISRA)
	frame, base, res, err := m.Framebuffer()
	if err != nil {
		t.Logf("framebuffer error: %v", err)
		return
	}
	t.Logf("frame base=%08x res=%d sha256=%x palette=%v", base, res, sha256.Sum256(frame), m.Memory.shifterPalette)
	out := os.Getenv("TALOS_BOOT_OUTPUT")
	if out == "" || res != 0 || len(frame) != 32000 {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, 320, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			idx := 0
			off := y*160 + (x/16)*8
			for plane := 0; plane < 4; plane++ {
				idx |= int((binary.BigEndian.Uint16(frame[off+plane*2:])>>uint(15-x%16))&1) << plane
			}
			v := m.Memory.shifterPalette[idx]
			img.SetRGBA(x, y, color.RGBA{uint8(((v >> 8) & 7) * 255 / 7), uint8(((v >> 4) & 7) * 255 / 7), uint8((v & 7) * 255 / 7), 255})
		}
	}
	name := "talos-boot.png"
	if len(names) > 0 {
		name = names[0]
	}
	file, err := os.Create(filepath.Join(out, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
