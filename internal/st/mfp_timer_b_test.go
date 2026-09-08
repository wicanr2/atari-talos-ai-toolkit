package st

import (
	"github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"
	"testing"
)

func timerBMemory(t *testing.T, data byte) *Memory {
	t.Helper()
	m, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	m.ColdReset()
	m.videoSyncMode = 2
	if err = m.WriteByteFC(MFPTBDR, data, 5); err != nil {
		t.Fatal(err)
	}
	if err = m.WriteByteFC(MFPTBCR, 8, 5); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestTimerBDisplayEdgesAndReload(t *testing.T) {
	m := timerBMemory(t, 0)
	first := uint64(63*512 + 400)
	if got := m.timerBDeadline(); got != colorST50HzFrameClocks+118*512+400 {
		t.Fatalf("跨空白期到期時點 %d", got)
	}
	m.advanceTimerB(first - 1)
	if m.mfpTBMain != 0 || m.mfpTimerBEvents != 0 {
		t.Fatal("邊框不應計數")
	}
	m.advanceTimerB(first)
	if m.mfpTBMain != 255 {
		t.Fatal("第一下降緣未計數")
	}
	m.advanceTimerB(colorST50HzFrameClocks - 1)
	if m.mfpTimerBEvents != 200 || m.mfpTBMain != 56 {
		t.Fatal("應有 200 個顯示事件")
	}
	m.advanceTimerB(2*colorST50HzFrameClocks - 1)
	if m.mfpTimerBEvents != 400 || m.mfpTBMain != 112 {
		t.Fatal("跨幀／零表示 256 錯誤")
	}
}

func TestTimerBRejectsUnsupportedVideoWithoutMutation(t *testing.T) {
	m := timerBMemory(t, 7)
	if err := m.WriteByteFC(MFPTBCR, 0, 5); err != nil {
		t.Fatal(err)
	}
	m.videoSyncMode = 0
	if err := m.WriteByteFC(MFPTBCR, 8, 5); err == nil || m.mfpTBCR != 0 || m.mfpTBMain != 7 || m.mfpTimerBNextEdge != 0 {
		t.Fatal("未建模的顯示模式不可啟動 Timer B")
	}
}

func TestTimerBActiveDataStopAndIRQ(t *testing.T) {
	m := timerBMemory(t, 2)
	m.mfpIERA = 1
	m.mfpVR = 0x48
	first := m.mfpTimerBNextEdge
	m.advanceTimerB(first)
	if err := m.WriteByteFC(MFPTBDR, 7, 5); err != nil {
		t.Fatal(err)
	}
	if m.mfpTBMain != 1 {
		t.Fatal("執行中提前重載")
	}
	m.advanceTimerB(first + 512)
	if m.mfpTBMain != 7 || m.mfpIPRA != 1 {
		t.Fatal("遮罩期間應保存 pending")
	}
	if err := m.WriteByteFC(MFPIMRA, 1, 5); err != nil {
		t.Fatal(err)
	}
	machine := Machine{Memory: m}
	if ch, ok := machine.mfpInterruptChannel(); !ok || ch != 8 {
		t.Fatal("Timer B 請求未送出")
	}
	m.acknowledgeMFP(8)
	if m.mfpISRA != 1 || m.mfpIPRA != 0 {
		t.Fatal("software EOI")
	}
	m.mfpIPRA = 1
	if _, ok := machine.mfpInterruptChannel(); ok {
		t.Fatal("ISR 未阻擋同級")
	}
	m.mfpIERA |= 0x20
	m.mfpIMRA |= 0x20
	m.mfpIPRA |= 0x20
	if ch, ok := machine.mfpInterruptChannel(); !ok || ch != 13 {
		t.Fatal("Timer A 應優先")
	}
	if err := m.WriteByteFC(MFPTBCR, 0, 5); err != nil {
		t.Fatal(err)
	}
	m.advanceTimerB(first + 20000)
	if m.mfpTBMain != 7 {
		t.Fatal("停止未保留計數")
	}
	if err := m.WriteByteFC(MFPTBCR, 0x18, 5); err != nil {
		t.Fatal(err)
	}
	edge := m.mfpTimerBNextEdge
	if err := m.WriteByteFC(MFPTBCR, 8, 5); err != nil {
		t.Fatal(err)
	}
	if edge != m.mfpTimerBNextEdge {
		t.Fatal("同值重入不應重排")
	}
	if err := m.WriteByteFC(MFPTBCR, 1, 5); err == nil || m.mfpTBCR != 8 {
		t.Fatal("未知模式需原子拒絕")
	}
	m.ColdReset()
	if m.mfpTimerBEvents != 0 || m.mfpTimerBNextEdge != 0 || m.mfpTBCR != 0 {
		t.Fatal("cold reset")
	}
}

func TestTimerBWakesStoppedCPU(t *testing.T) {
	mem := timerBMemory(t, 139)
	mem.mmuConfig = 5
	for _, w := range []struct {
		a uint32
		v uint16
	}{{72 * 4, 0}, {72*4 + 2, 0x2000}, {0x2000, 0x4e71}, {0x2002, 0x4e71}, {0x2004, 0x4e71}} {
		if err := mem.WriteWord(w.a, w.v, 5); err != nil {
			t.Fatal(err)
		}
	}
	m := Machine{Memory: mem, nextVBLClock: 160000, vblFrameClocks: colorST50HzFrameClocks}
	m.CPU.Bus = mem
	m.CPU.State = m68k.State{PC: 0x1004, SR: 0x2300, SSP: 0x8000, Prefetch: [2]uint16{0x4e72, 0x2300}}
	mem.mfpVR = 0x48
	mem.mfpIERA = 1
	mem.mfpIMRA = 1
	if _, err := m.Step(); err != nil || !m.CPU.IsStopped() {
		t.Fatalf("STOP %v", err)
	}
	if _, err := m.Step(); err != nil {
		t.Fatal(err)
	}
	if m.CPU.IsStopped() || mem.mfpISRA != 1 || mem.mfpTimerBAcknowledged != 1 || m.CPU.State.PC != 0x2004 {
		t.Fatal("未由 B8 喚醒")
	}
}
