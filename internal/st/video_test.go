package st

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
)

func TestDecodeLowResolutionPlanarRejectsWrongSize(t *testing.T) {
	for _, size := range []int{0, lowResolutionFrameBytes - 1, lowResolutionFrameBytes + 1} {
		if got, err := DecodeLowResolutionPlanar(make([]byte, size)); err == nil || got != nil {
			t.Fatalf("size %d decoded as len=%d err=%v", size, len(got), err)
		}
	}
}

func TestDecodeLowResolutionPlanarPlaneBitsAndBounds(t *testing.T) {
	source := make([]byte, lowResolutionFrameBytes)
	for want := 0; want < 16; want++ {
		group := want * lowResolutionGroupBytes
		for plane := 0; plane < 4; plane++ {
			if want&(1<<plane) != 0 {
				source[group+plane*2] = 0x80
			}
		}
	}
	last := len(source) - lowResolutionGroupBytes
	source[last+0], source[last+1] = 0x00, 0x01
	source[last+6], source[last+7] = 0x00, 0x01
	source[lowResolutionLineBytes] = 0x80
	indices, err := DecodeLowResolutionPlanar(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(indices) != lowResolutionPixelCount {
		t.Fatalf("indices len=%d want %d", len(indices), lowResolutionPixelCount)
	}
	for want := 0; want < 16; want++ {
		pixel := want * lowResolutionGroupPixels
		if got := indices[pixel]; got != byte(want) {
			t.Fatalf("group %d first pixel=%d want %d", want, got, want)
		}
		if want != 0 && indices[pixel+1] != 0 {
			t.Fatalf("group %d second pixel=%d want 0", want, indices[pixel+1])
		}
	}
	if got := indices[len(indices)-1]; got != 9 {
		t.Fatalf("last pixel=%d want 9", got)
	}
	if indices[LowResolutionWidth-1] != 0 || indices[LowResolutionWidth] != 1 {
		t.Fatalf("line boundary pixels=%d/%d want 0/1", indices[LowResolutionWidth-1], indices[LowResolutionWidth])
	}
}

func TestMemoryLowResolutionFrameSnapshotsActiveRAMAndPalette(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteByte(VideoBaseMiddle, 0x10, 5); err != nil {
		t.Fatal(err)
	}
	memory.reloadVideoBaseOnVBL()
	for plane, value := range []byte{0x80, 0x00, 0x80, 0x00, 0x80, 0x00, 0x80, 0x00} {
		if err := memory.WriteByte(0x1000+uint32(plane), value, 5); err != nil {
			t.Fatal(err)
		}
	}
	if err := memory.WriteWord(ShifterPaletteEnd, 0x0777, 5); err != nil {
		t.Fatal(err)
	}
	frame, err := memory.LowResolutionFrame()
	if err != nil {
		t.Fatal(err)
	}
	if frame.Width != 320 || frame.Height != 200 || len(frame.Indices) != 64000 ||
		frame.Indices[0] != 15 || frame.Indices[1] != 0 || frame.Palette[15] != 0x0777 {
		t.Fatalf("frame=%+v first=%v", frame, frame.Indices[:2])
	}
	if err := memory.WriteByte(0x1000, 0, 5); err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteWord(ShifterPaletteEnd, 0, 5); err != nil {
		t.Fatal(err)
	}
	if frame.Indices[0] != 15 || frame.Palette[15] != 0x0777 {
		t.Fatalf("snapshot followed mutable memory: index=%d palette=%04x", frame.Indices[0], frame.Palette[15])
	}
}

func TestMemoryLowResolutionFrameFailsOnUnmappedActiveBase(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteByte(VideoBaseHigh, 0x1f, 5); err != nil {
		t.Fatal(err)
	}
	memory.reloadVideoBaseOnVBL()
	if _, err := memory.LowResolutionFrame(); err == nil {
		t.Fatal("unmapped active framebuffer unexpectedly decoded")
	}
}

func TestMemoryLowResolutionFrameUsesRAMAtZeroNotResetROMShadow(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	frame, err := memory.LowResolutionFrame()
	if err != nil {
		t.Fatal(err)
	}
	for offset, index := range frame.Indices {
		if index != 0 {
			t.Fatalf("base-zero RAM frame pixel %d=%d; likely read reset ROM shadow", offset, index)
		}
	}
	memory.shifterResolution = 1
	if _, err := memory.LowResolutionFrame(); err == nil {
		t.Fatal("medium-resolution state unexpectedly decoded as low-resolution")
	}
}

func TestHatariEmuTOSVBL7LowResolutionIndices(t *testing.T) {
	path := os.Getenv("TALOS_HATARI_FRAMEBUFFER")
	if path == "" {
		t.Skip("TALOS_HATARI_FRAMEBUFFER is not set")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(source)); got != "98dcbfd3bd49a1854a7544d349d1d6dee0a629f66ed976262bdfd9fd72a0570f" {
		t.Fatalf("Hatari framebuffer SHA-256=%s", got)
	}
	indices, err := DecodeLowResolutionPlanar(source)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(indices)); got != "6157070b2e1adde8ec0cf121ee72133824c34edc5b133ff0064632a73e910444" {
		t.Fatalf("decoded indices SHA-256=%s", got)
	}
	var counts [16]int
	first := -1
	for offset, index := range indices {
		counts[index]++
		if first < 0 && index != 0 {
			first = offset
		}
	}
	if counts != [16]int{63679, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 321} || first != 1 {
		t.Fatalf("histogram=%v first=%d", counts, first)
	}
}
