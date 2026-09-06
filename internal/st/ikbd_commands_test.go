package st

import "testing"

// ikbdReadyForCommands puts the ACIA in the state the host writes commands
// from: configured, TDRE set, nothing in flight. That is the "jump straight to
// the event" shortcut — the boot path takes 6.7 million instructions to get
// here and none of them are what this slice is about.
func ikbdReadyForCommands(t *testing.T) *Memory {
	t.Helper()
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.ikbdACIAConfigured = true
	memory.ikbdACIAStatus = 2
	return memory
}

// sendIKBDByte writes one byte and clears the transmit latch, standing in for
// the serial shift the host waits out between bytes.
func sendIKBDByte(t *testing.T, m *Memory, value byte) error {
	t.Helper()
	if err := m.WriteByte(IKBDACIAData, value, 5); err != nil {
		return err
	}
	m.ikbdACIATXPending = false
	m.ikbdACIAStatus |= 2
	return nil
}

// TestIKBDInitmousCommandSequence covers spec 138: the seven bytes EmuTOS sends
// after it has read the clock, in the order Hatari's own trace decodes them.
func TestIKBDInitmousCommandSequence(t *testing.T) {
	memory := ikbdReadyForCommands(t)
	for _, value := range []byte{0x08, 0x0b, 0x01, 0x01, 0x10, 0x07, 0x00} {
		if err := sendIKBDByte(t, memory, value); err != nil {
			t.Fatalf("送 %02x：%v", value, err)
		}
	}
	if !memory.ikbdRelativeMouse {
		t.Error("$08 沒有設成相對滑鼠回報")
	}
	if memory.ikbdMouseThreshold != [2]byte{1, 1} {
		t.Errorf("$0B 的門檻是 %v，應該是 1,1", memory.ikbdMouseThreshold)
	}
	if !memory.ikbdYAxisUp {
		t.Error("$10 沒有把 Y 軸原點設在上")
	}
	if !memory.ikbdMouseButtonActionSet || memory.ikbdMouseButtonAction != 0 {
		t.Errorf("$07 的 button action 是 %02x/%v", memory.ikbdMouseButtonAction, memory.ikbdMouseButtonActionSet)
	}
	if memory.ikbdCommandRemaining != 0 {
		t.Errorf("組裝器沒有收乾淨：還等 %d 個參數", memory.ikbdCommandRemaining)
	}
}

// A command only takes effect once its parameters are complete, and the bytes
// in between are parameters — not commands, even when they happen to be valid
// command codes.
func TestIKBDParametersAreNotReadAsCommands(t *testing.T) {
	memory := ikbdReadyForCommands(t)
	if err := sendIKBDByte(t, memory, 0x0b); err != nil {
		t.Fatal(err)
	}
	if memory.ikbdMouseThreshold != [2]byte{} {
		t.Error("$0B 在參數收滿之前就生效了")
	}
	// $08 and $10 are real command codes; here they are threshold values.
	for _, value := range []byte{0x08, 0x10} {
		if err := sendIKBDByte(t, memory, value); err != nil {
			t.Fatalf("送參數 %02x：%v", value, err)
		}
	}
	if memory.ikbdMouseThreshold != [2]byte{0x08, 0x10} {
		t.Errorf("門檻是 %v，應該是 08,10", memory.ikbdMouseThreshold)
	}
	if memory.ikbdRelativeMouse || memory.ikbdYAxisUp {
		t.Error("參數被當成命令執行了")
	}
}

// Unknown command codes fail closed rather than being swallowed as
// parameterless commands.
func TestIKBDUnknownCommandFailsClosed(t *testing.T) {
	for _, opcode := range []byte{0x09, 0x0c, 0x0d, 0x0e, 0x14, 0x20, 0xff} {
		memory := ikbdReadyForCommands(t)
		if err := memory.WriteByte(IKBDACIAData, opcode, 5); err == nil {
			t.Errorf("命令 %02x 被接受了", opcode)
		}
		if memory.ikbdCommandRemaining != 0 || memory.ikbdACIATXPending {
			t.Errorf("命令 %02x 被拒絕後仍留下狀態", opcode)
		}
	}
}

// The transmit latch still gates every byte: writing while the previous one is
// in flight, or with TDRE clear, is refused.
func TestIKBDCommandStillWaitsForTDRE(t *testing.T) {
	memory := ikbdReadyForCommands(t)
	if err := memory.WriteByte(IKBDACIAData, 0x08, 5); err != nil {
		t.Fatal(err)
	}
	if memory.ikbdACIAStatus&2 != 0 || !memory.ikbdACIATXPending {
		t.Fatalf("第一個位元組沒有佔住 latch：status=%02x pending=%v",
			memory.ikbdACIAStatus, memory.ikbdACIATXPending)
	}
	if err := memory.WriteByte(IKBDACIAData, 0x10, 5); err == nil {
		t.Error("latch 還沒空就收下第二個位元組")
	}
}

// Cold reset clears the assembler and everything the four commands set.
func TestIKBDCommandStateClearsOnColdReset(t *testing.T) {
	memory := ikbdReadyForCommands(t)
	for _, value := range []byte{0x08, 0x0b, 0x03, 0x04, 0x10, 0x07, 0x02} {
		if err := sendIKBDByte(t, memory, value); err != nil {
			t.Fatal(err)
		}
	}
	memory.ColdReset()
	if memory.ikbdRelativeMouse || memory.ikbdYAxisUp || memory.ikbdMouseButtonActionSet ||
		memory.ikbdMouseThreshold != [2]byte{} || memory.ikbdMouseButtonAction != 0 ||
		memory.ikbdCommandRemaining != 0 || memory.ikbdCommandOpcode != 0 ||
		memory.ikbdCommandParamCount != 0 || memory.ikbdCommandParams != [2]byte{} {
		t.Fatal("cold reset 之後 IKBD 命令狀態沒清乾淨")
	}
}

// TestEmuTOSSendsInitmousCommands is the fixed-ROM receipt: the boot path used
// to stop on the first of these four writes (6,779,282 instructions). It now
// runs through all seven bytes and keeps going — since spec 140 there is no
// gate left on the boot path at all, so this checks the four settings are in
// place rather than where the machine stops.
//
// This is the regression anchor, not the verification: what the four commands
// do is pinned by the synthetic tests above.
func TestEmuTOSSendsInitmousCommands(t *testing.T) {
	machine := emuTOSMachine(t)
	for steps := 0; steps < 9_000_000; steps++ {
		if _, gate := machine.Step(); gate != nil {
			t.Fatalf("開機路徑在第 %d 條停住：%v", steps, gate)
		}
	}
	m := machine.Memory
	if !m.ikbdRelativeMouse || m.ikbdMouseThreshold != [2]byte{1, 1} || !m.ikbdYAxisUp ||
		!m.ikbdMouseButtonActionSet || m.ikbdMouseButtonAction != 0 || m.ikbdCommandRemaining != 0 {
		t.Errorf("四條命令的結果不對：relative=%v threshold=%v yUp=%v action=%02x/%v remaining=%d",
			m.ikbdRelativeMouse, m.ikbdMouseThreshold, m.ikbdYAxisUp,
			m.ikbdMouseButtonAction, m.ikbdMouseButtonActionSet, m.ikbdCommandRemaining)
	}
}
