package st

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"
)

var _ m68k.Bus = (*Memory)(nil)
var _ m68k.TimedBus = (*Memory)(nil)

func TestBusSlotWaitRejectsOddClockDomain(t *testing.T) {
	for _, test := range []struct {
		clock uint64
		wait  uint32
		ok    bool
	}{
		{clock: 0, wait: 0, ok: true},
		{clock: 2, wait: 2, ok: true},
		{clock: 4, wait: 0, ok: true},
		{clock: 6, wait: 2, ok: true},
		{clock: 1},
		{clock: 3},
	} {
		wait, err := busSlotWait(test.clock)
		if test.ok && (err != nil || wait != test.wait) {
			t.Fatalf("clock %d: wait=%d err=%v want %d", test.clock, wait, err, test.wait)
		}
		if !test.ok && err == nil {
			t.Fatalf("clock %d: odd phase unexpectedly accepted", test.clock)
		}
	}
}

func testROM() []byte {
	rom := make([]byte, TOSROMSize)
	for i := range rom {
		rom[i] = byte(i ^ i>>8)
	}
	return rom
}

func TestNewMemoryValidatesAndCopiesInputs(t *testing.T) {
	if _, err := NewMemory(256*1024, make([]byte, TOSROMSize)); err == nil {
		t.Fatal("unsupported RAM size unexpectedly accepted")
	}
	if _, err := NewMemory(RAM512K, make([]byte, TOSROMSize-1)); err == nil {
		t.Fatal("short TOS ROM unexpectedly accepted")
	}
	rom := testROM()
	memory, err := NewMemory(RAM1M, rom)
	if err != nil {
		t.Fatal(err)
	}
	want := rom[0]
	rom[0] ^= 0xff
	got, err := memory.ReadByte(TOSROMBase, 6)
	if err != nil || got != want {
		t.Fatalf("ROM input was not copied: got=%02x err=%v want=%02x", got, err, want)
	}
}

func TestMemoryResetShadowAndROMMirror(t *testing.T) {
	rom := testROM()
	memory, err := NewMemory(RAM512K, rom)
	if err != nil {
		t.Fatal(err)
	}
	for address := uint32(0); address < 8; address++ {
		shadow, shadowErr := memory.ReadByte(address, 6)
		mapped, mappedErr := memory.ReadByte(TOSROMBase+address, 6)
		if shadowErr != nil || mappedErr != nil || shadow != rom[address] || mapped != rom[address] {
			t.Fatalf("ROM mapping at %d: shadow=%02x/%v mapped=%02x/%v", address,
				shadow, shadowErr, mapped, mappedErr)
		}
	}
}

func TestShifterResolutionResetLowMode(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	for _, address := range []uint32{ShifterResolution, 0xffff8260} {
		if got, err := memory.ReadByte(address, 5); err != nil || got != 0xfc {
			t.Fatalf("resolution %08x=%02x/%v want fc", address, got, err)
		}
		if err := memory.WriteByte(address, 0, 5); err != nil {
			t.Fatalf("resolution zero write %08x: %v", address, err)
		}
	}
	for _, value := range []byte{1, 2, 3, 0xff} {
		err := memory.WriteByte(ShifterResolution, value, 5)
		var fault *BusFault
		if !errors.As(err, &fault) || fault.Reason != FaultUnsupportedDeviceState {
			t.Fatalf("resolution value %02x err=%v", value, err)
		}
		if got, readErr := memory.ReadByte(ShifterResolution, 5); readErr != nil || got != 0xfc {
			t.Fatalf("failed write committed value: got=%02x err=%v", got, readErr)
		}
	}
	if _, err := memory.ReadByte(ShifterResolution, 1); err == nil {
		t.Fatal("user resolution read unexpectedly succeeded")
	}
	if err := memory.WriteByte(ShifterResolution, 0, 1); err == nil {
		t.Fatal("user resolution write unexpectedly succeeded")
	}
	for _, address := range []uint32{ShifterResolution - 1, ShifterResolution + 1} {
		if _, err := memory.ReadByte(address, 5); err == nil {
			t.Fatalf("adjacent resolution byte %06x unexpectedly mapped", address)
		}
	}
	if _, err := memory.ReadWord(ShifterResolution, 5); err == nil {
		t.Fatal("resolution word read unexpectedly succeeded")
	}
	if err := memory.WriteWord(ShifterResolution, 0, 5); err == nil {
		t.Fatal("resolution word write unexpectedly succeeded")
	}
	if wait, err := memory.WriteByteAt(ShifterResolution, 0, m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 2 {
		t.Fatalf("timed resolution write wait=%d err=%v want 2", wait, err)
	}
	memory.shifterResolution = 2
	memory.ColdReset()
	if got, err := memory.ReadByte(ShifterResolution, 5); err != nil || got != 0xfc {
		t.Fatalf("reset resolution=%02x/%v want fc", got, err)
	}
}

func TestVideoSyncResetAndFixed50HzTransition(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	for _, address := range []uint32{VideoSyncMode, 0xffff820a} {
		if got, err := memory.ReadByte(address, 5); err != nil || got != 0xfc {
			t.Fatalf("sync %08x=%02x/%v want fc", address, got, err)
		}
	}
	if err := memory.WriteByte(VideoSyncMode, 0, 5); err != nil || memory.videoSync50Transition {
		t.Fatalf("sync zero write err=%v transition=%v", err, memory.videoSync50Transition)
	}
	if err := memory.WriteByte(VideoSyncMode, 2, 5); err != nil || !memory.videoSync50Transition {
		t.Fatalf("sync 0->2 err=%v transition=%v", err, memory.videoSync50Transition)
	}
	if got, err := memory.ReadByte(VideoSyncMode, 5); err != nil || got != 0xfe {
		t.Fatalf("50 Hz sync=%02x/%v want fe", got, err)
	}
	memory.videoSync50Transition = false
	if err := memory.WriteByte(VideoSyncMode, 2, 5); err != nil || memory.videoSync50Transition {
		t.Fatalf("sync 2->2 err=%v transition=%v", err, memory.videoSync50Transition)
	}
	for _, value := range []byte{0, 1, 3, 0xff} {
		err := memory.WriteByte(VideoSyncMode, value, 5)
		var fault *BusFault
		if !errors.As(err, &fault) || fault.Reason != FaultUnsupportedDeviceState {
			t.Fatalf("sync value %02x err=%v", value, err)
		}
		if got, readErr := memory.ReadByte(VideoSyncMode, 5); readErr != nil || got != 0xfe {
			t.Fatalf("failed sync write committed: got=%02x err=%v", got, readErr)
		}
	}
	if _, err := memory.ReadByte(VideoSyncMode, 1); err == nil {
		t.Fatal("user sync read unexpectedly succeeded")
	}
	if _, err := memory.ReadWord(VideoSyncMode, 5); err == nil {
		t.Fatal("sync word read unexpectedly succeeded")
	}
	memory.ColdReset()
	if got, err := memory.ReadByte(VideoSyncMode, 5); err != nil || got != 0xfc || memory.videoSync50Transition {
		t.Fatalf("reset sync=%02x/%v transition=%v", got, err, memory.videoSync50Transition)
	}
}

func TestProgrammedVideoBaseRegisters(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	for _, address := range []uint32{VideoBaseHigh, VideoBaseMiddle, 0xffff8201, 0xffff8203} {
		if got, err := memory.ReadByte(address, 5); err != nil || got != 0 {
			t.Fatalf("video base %08x reset=%02x/%v", address, got, err)
		}
	}
	if err := memory.WriteByte(VideoBaseHigh, 0xcf, 5); err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteByte(VideoBaseMiddle, 0x80, 5); err != nil {
		t.Fatal(err)
	}
	if got := memory.ProgrammedVideoBase(); got != 0x0f8000 {
		t.Fatalf("programmed video base=%06x want 0f8000", got)
	}
	if got := memory.ActiveVideoBase(); got != 0 {
		t.Fatalf("active video base changed before VBL: %06x", got)
	}
	memory.reloadVideoBaseOnVBL()
	if got := memory.ActiveVideoBase(); got != 0x0f8000 {
		t.Fatalf("active video base after VBL=%06x want 0f8000", got)
	}
	if err := memory.WriteByte(VideoBaseMiddle, 0x40, 5); err != nil {
		t.Fatal(err)
	}
	if got := memory.ActiveVideoBase(); got != 0x0f8000 {
		t.Fatalf("active video base followed programmed write: %06x", got)
	}
	if got, err := memory.ReadByte(VideoBaseHigh, 5); err != nil || got != 0x0f {
		t.Fatalf("high=%02x/%v want 0f", got, err)
	}
	if got, err := memory.ReadByte(VideoBaseMiddle, 5); err != nil || got != 0x40 {
		t.Fatalf("middle=%02x/%v want 40", got, err)
	}
	for _, address := range []uint32{VideoBaseHigh, VideoBaseMiddle} {
		if _, err := memory.ReadByte(address, 1); err == nil {
			t.Fatalf("user read %06x unexpectedly succeeded", address)
		}
		if err := memory.WriteByte(address, 0, 1); err == nil {
			t.Fatalf("user write %06x unexpectedly succeeded", address)
		}
		if _, err := memory.ReadWord(address, 5); err == nil {
			t.Fatalf("word read %06x unexpectedly succeeded", address)
		}
		if err := memory.WriteWord(address, 0, 5); err == nil {
			t.Fatalf("word write %06x unexpectedly succeeded", address)
		}
	}
	for _, address := range []uint32{VideoBaseHigh - 1, VideoBaseHigh + 1, VideoBaseMiddle + 1} {
		if _, err := memory.ReadByte(address, 5); err == nil {
			t.Fatalf("adjacent byte %06x unexpectedly mapped", address)
		}
	}
	if wait, err := memory.WriteByteAt(VideoBaseHigh, 0x12, m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 2 {
		t.Fatalf("timed video-base write wait=%d err=%v want 2", wait, err)
	}
	memory.ColdReset()
	if programmed, active := memory.ProgrammedVideoBase(), memory.ActiveVideoBase(); programmed != 0 || active != 0 {
		t.Fatalf("reset video bases programmed=%06x active=%06x", programmed, active)
	}
}

func TestShifterPaletteWordBank(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	for index := range 16 {
		address := ShifterPaletteBase + uint32(index*2)
		if got, err := memory.ReadWord(address, 5); err != nil || got != 0 {
			t.Fatalf("palette[%d] reset=%04x/%v", index, got, err)
		}
		value := uint16(0xf888 | index*0x111)
		if err := memory.WriteWord(address, value, 5); err != nil {
			t.Fatalf("palette[%d] write: %v", index, err)
		}
		if got, err := memory.ReadWord(address, 5); err != nil || got != value&0x0777 {
			t.Fatalf("palette[%d]=%04x/%v want %04x", index, got, err, value&0x0777)
		}
	}
	if got, err := memory.ReadWord(0xffff8240, 5); err != nil || got != memory.shifterPalette[0] {
		t.Fatalf("palette alias=%04x/%v", got, err)
	}
	if _, err := memory.ReadWord(ShifterPaletteBase, 1); err == nil {
		t.Fatal("user palette read unexpectedly succeeded")
	}
	if err := memory.WriteWord(ShifterPaletteBase, 0, 1); err == nil {
		t.Fatal("user palette write unexpectedly succeeded")
	}
	if _, err := memory.ReadWord(ShifterPaletteBase+1, 5); err == nil {
		t.Fatal("odd palette word read unexpectedly succeeded")
	}
	if _, err := memory.ReadByte(ShifterPaletteBase, 5); err == nil {
		t.Fatal("palette byte read unexpectedly succeeded")
	}
	if err := memory.WriteByte(ShifterPaletteBase, 0, 5); err == nil {
		t.Fatal("palette byte write unexpectedly succeeded")
	}
	for _, address := range []uint32{ShifterPaletteBase - 2, ShifterPaletteEnd + 2} {
		if _, err := memory.ReadWord(address, 5); err == nil {
			t.Fatalf("adjacent palette word %06x unexpectedly mapped", address)
		}
	}
	if wait, err := memory.WriteWordAt(ShifterPaletteBase, 0x0777, m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 2 {
		t.Fatalf("timed palette write wait=%d err=%v want 2", wait, err)
	}
	memory.ColdReset()
	for index, value := range memory.shifterPalette {
		if value != 0 {
			t.Fatalf("palette[%d] after reset=%04x", index, value)
		}
	}
}

func TestMemoryRAMBoundsAndAddressMask(t *testing.T) {
	for _, size := range []int{RAM512K, RAM1M} {
		memory, err := NewMemory(size, testROM())
		if err != nil {
			t.Fatal(err)
		}
		config := byte(0x04)
		if size == RAM1M {
			config = 0x05
		}
		if err := memory.WriteByte(MMUConfig, config, 5); err != nil {
			t.Fatal(err)
		}
		for _, address := range []uint32{8, 0x800, uint32(size - 1)} {
			if err := memory.WriteByte(address, 0xa5, 5); err != nil {
				t.Fatalf("RAM size %d write 0x%x: %v", size, address, err)
			}
			got, err := memory.ReadByte(address, 5)
			if err != nil || got != 0xa5 {
				t.Fatalf("RAM size %d read 0x%x: got=%02x err=%v", size, address, got, err)
			}
		}
		if err := memory.WriteByte(0x0100_0800, 0x3c, 5); err != nil {
			t.Fatal(err)
		}
		got, err := memory.ReadByte(0x800, 5)
		if err != nil || got != 0x3c {
			t.Fatalf("24-bit mask: got=%02x err=%v", got, err)
		}
	}
}

func TestMMUConfigurationRegisterAnd512KBankTranslation(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if got, err := memory.ReadByte(MMUConfig, 5); err != nil || got != 0 {
		t.Fatalf("cold MMU config=%02x err=%v", got, err)
	}
	if err := memory.WriteByte(MMUConfig, 0xfa, 6); err != nil {
		t.Fatal(err)
	}
	if got, err := memory.ReadByte(MMUConfig, 5); err != nil || got != 0xfa {
		t.Fatalf("preserved MMU config=%02x err=%v", got, err)
	}
	for _, test := range []struct {
		logical  uint32
		physical uint32
	}{
		{0x000800, 0x000400},
		{0x080800, 0x040400},
		{0x200800, 0x080400},
		{0x280800, 0x0c0400},
	} {
		if err := memory.WriteByte(test.logical, byte(test.logical>>19)+1, 5); err != nil {
			t.Fatalf("write logical %06x: %v", test.logical, err)
		}
		if got := memory.ram[test.physical]; got != byte(test.logical>>19)+1 {
			t.Fatalf("logical %06x mapped physical %06x=%02x", test.logical, test.physical, got)
		}
	}
	if err := memory.WriteByte(MMUConfig, 0x05, 5); err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteByte(0x080000, 0xa5, 5); err != nil || memory.ram[0x080000] != 0xa5 {
		t.Fatalf("identity bank1 write err=%v physical=%02x", err, memory.ram[0x080000])
	}
	if _, err := memory.ReadByte(MMUConfig, 1); err == nil {
		t.Fatal("user MMU read unexpectedly succeeded")
	}
	memory.ColdReset()
	if got, err := memory.ReadByte(MMUConfig, 5); err != nil || got != 0 {
		t.Fatalf("reset MMU config=%02x err=%v", got, err)
	}
}

func TestM68KResetClearsMMUConfigurationWithoutClearingRAM(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteByte(MMUConfig, 0x05, 5); err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteWord(0x0100, 0x1234, 5); err != nil {
		t.Fatal(err)
	}
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if got, err := memory.ReadByte(MMUConfig, 5); err != nil || got != 0 {
		t.Fatalf("MMU config after M68K reset=%02x err=%v", got, err)
	}
	if got, err := memory.ReadWord(0x0100, 5); err != nil || got != 0x1234 {
		t.Fatalf("RAM after M68K reset=%04x err=%v", got, err)
	}
}

func TestMFPGPIPResetStateByteAccess(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.mfpGPIP = 0xa5
	memory.mfpDDR = 0x0f
	if err := memory.WriteByte(MFPGPIP, 0x3c, 5); err != nil {
		t.Fatal(err)
	}
	if got, err := memory.ReadByte(0xfffffa01, 5); err != nil || got != 0xac {
		t.Fatalf("masked GPIP=%02x/%v want ac", got, err)
	}
	memory.ColdReset()
	if got, err := memory.ReadByte(MFPGPIP, 5); err != nil || got != 0xa1 {
		t.Fatalf("cold GPIP=%02x/%v", got, err)
	}
	if err := memory.WriteByte(MFPGPIP, 0xff, 5); err != nil {
		t.Fatal(err)
	}
	if got, _ := memory.ReadByte(MFPGPIP, 5); got != 0xa1 {
		t.Fatalf("DDR=0 write changed GPIP to %02x", got)
	}
	if wait, err := memory.WriteByteAt(MFPGPIP, 0, m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
		t.Fatalf("timed GPIP write wait=%d err=%v", wait, err)
	}
	if _, err := memory.ReadByte(MFPGPIP, 1); err == nil {
		t.Fatal("user GPIP read unexpectedly succeeded")
	}
	if err := memory.WriteByte(MFPGPIP, 0, 1); err == nil {
		t.Fatal("user GPIP write unexpectedly succeeded")
	}
	if _, err := memory.ReadWord(MFPGPIP, 5); err == nil {
		t.Fatal("odd GPIP word read unexpectedly succeeded")
	}
}

func TestMFPDDRResetStateZeroWrite(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if got, err := memory.ReadByte(MFPDDR, 5); err != nil || got != 0 {
		t.Fatalf("cold DDR=%02x/%v", got, err)
	}
	if wait, err := memory.WriteByteAt(0xfffffa05, 0, m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
		t.Fatalf("timed DDR zero write wait=%d err=%v", wait, err)
	}
	if err := memory.WriteByte(MFPDDR, 1, 5); err == nil {
		t.Fatal("nonzero DDR write unexpectedly accepted")
	} else {
		var fault *BusFault
		if !errors.As(err, &fault) || fault.Reason != FaultUnsupportedDeviceState {
			t.Fatalf("nonzero DDR fault=%#v/%v", fault, err)
		}
	}
	if _, err := memory.ReadByte(MFPDDR, 1); err == nil {
		t.Fatal("user DDR read unexpectedly succeeded")
	}
	if err := memory.WriteByte(MFPDDR, 0, 1); err == nil {
		t.Fatal("user DDR write unexpectedly succeeded")
	}
	if _, err := memory.ReadWord(MFPDDR, 5); err == nil {
		t.Fatal("odd DDR word read unexpectedly succeeded")
	}
	memory.mfpDDR = 0xff
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if got, err := memory.ReadByte(MFPDDR, 5); err != nil || got != 0 {
		t.Fatalf("DDR after M68K reset=%02x/%v", got, err)
	}
}

func TestMFPIERResetStateZeroWrites(t *testing.T) {
	for _, test := range []struct {
		name    string
		address uint32
		set     func(*Memory, byte)
	}{
		{name: "IERA", address: MFPIERA, set: func(m *Memory, value byte) { m.mfpIERA = value }},
		{name: "IERB", address: MFPIERB, set: func(m *Memory, value byte) { m.mfpIERB = value }},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory, err := NewMemory(RAM1M, testROM())
			if err != nil {
				t.Fatal(err)
			}
			if got, err := memory.ReadByte(test.address, 5); err != nil || got != 0 {
				t.Fatalf("cold %s=%02x/%v", test.name, got, err)
			}
			if wait, err := memory.WriteByteAt(test.address|0xff000000, 0,
				m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
				t.Fatalf("timed %s zero write wait=%d err=%v", test.name, wait, err)
			}
			if err := memory.WriteByte(test.address, 1, 5); err == nil {
				t.Fatalf("nonzero %s write unexpectedly accepted", test.name)
			} else {
				var fault *BusFault
				if !errors.As(err, &fault) || fault.Reason != FaultUnsupportedDeviceState {
					t.Fatalf("nonzero %s fault=%#v/%v", test.name, fault, err)
				}
			}
			if _, err := memory.ReadByte(test.address, 1); err == nil {
				t.Fatalf("user %s read unexpectedly succeeded", test.name)
			}
			if err := memory.WriteByte(test.address, 0, 1); err == nil {
				t.Fatalf("user %s write unexpectedly succeeded", test.name)
			}
			if _, err := memory.ReadWord(test.address, 5); err == nil {
				t.Fatalf("odd %s word read unexpectedly succeeded", test.name)
			}
			test.set(memory, 0xff)
			if err := memory.M68KReset(); err != nil {
				t.Fatal(err)
			}
			if got, err := memory.ReadByte(test.address, 5); err != nil || got != 0 {
				t.Fatalf("%s after M68K reset=%02x/%v", test.name, got, err)
			}
		})
	}
}

func TestMFPIPRWriteZeroToClear(t *testing.T) {
	for _, test := range []struct {
		name    string
		address uint32
		set     func(*Memory, byte)
	}{
		{name: "IPRA", address: MFPIPRA, set: func(m *Memory, value byte) { m.mfpIPRA = value }},
		{name: "IPRB", address: MFPIPRB, set: func(m *Memory, value byte) { m.mfpIPRB = value }},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory, err := NewMemory(RAM1M, testROM())
			if err != nil {
				t.Fatal(err)
			}
			test.set(memory, 0xa5)
			if wait, err := memory.WriteByteAt(test.address|0xff000000, 0x3c,
				m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
				t.Fatalf("timed %s clear wait=%d err=%v", test.name, wait, err)
			}
			if got, err := memory.ReadByte(test.address, 5); err != nil || got != 0x24 {
				t.Fatalf("masked %s=%02x/%v want 24", test.name, got, err)
			}
			test.set(memory, 0)
			if err := memory.WriteByte(test.address, 0xff, 5); err != nil {
				t.Fatal(err)
			}
			if got, _ := memory.ReadByte(test.address, 5); got != 0 {
				t.Fatalf("software set %s to %02x", test.name, got)
			}
			if _, err := memory.ReadByte(test.address, 1); err == nil {
				t.Fatalf("user %s read unexpectedly succeeded", test.name)
			}
			if err := memory.WriteByte(test.address, 0, 1); err == nil {
				t.Fatalf("user %s write unexpectedly succeeded", test.name)
			}
			if _, err := memory.ReadWord(test.address, 5); err == nil {
				t.Fatalf("odd %s word read unexpectedly succeeded", test.name)
			}
			test.set(memory, 0xff)
			if err := memory.M68KReset(); err != nil {
				t.Fatal(err)
			}
			if got, err := memory.ReadByte(test.address, 5); err != nil || got != 0 {
				t.Fatalf("%s after M68K reset=%02x/%v", test.name, got, err)
			}
		})
	}
}

func TestMFPISRWriteZeroToClear(t *testing.T) {
	for _, test := range []struct {
		name    string
		address uint32
		set     func(*Memory, byte)
	}{
		{name: "ISRA", address: MFPISRA, set: func(m *Memory, value byte) { m.mfpISRA = value }},
		{name: "ISRB", address: MFPISRB, set: func(m *Memory, value byte) { m.mfpISRB = value }},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory, err := NewMemory(RAM1M, testROM())
			if err != nil {
				t.Fatal(err)
			}
			test.set(memory, 0xa5)
			if wait, err := memory.WriteByteAt(test.address|0xff000000, 0x3c,
				m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
				t.Fatalf("timed %s clear wait=%d err=%v", test.name, wait, err)
			}
			if got, err := memory.ReadByte(test.address, 5); err != nil || got != 0x24 {
				t.Fatalf("masked %s=%02x/%v want 24", test.name, got, err)
			}
			test.set(memory, 0)
			if err := memory.WriteByte(test.address, 0xff, 5); err != nil {
				t.Fatal(err)
			}
			if got, _ := memory.ReadByte(test.address, 5); got != 0 {
				t.Fatalf("software set %s to %02x", test.name, got)
			}
			if _, err := memory.ReadByte(test.address, 1); err == nil {
				t.Fatalf("user %s read unexpectedly succeeded", test.name)
			}
			if err := memory.WriteByte(test.address, 0, 1); err == nil {
				t.Fatalf("user %s write unexpectedly succeeded", test.name)
			}
			if _, err := memory.ReadWord(test.address, 5); err == nil {
				t.Fatalf("odd %s word read unexpectedly succeeded", test.name)
			}
			test.set(memory, 0xff)
			if err := memory.M68KReset(); err != nil {
				t.Fatal(err)
			}
			if got, err := memory.ReadByte(test.address, 5); err != nil || got != 0 {
				t.Fatalf("%s after M68K reset=%02x/%v want 00", test.name, got, err)
			}
		})
	}
}

func TestMFPIMRMaskLatchWithoutPending(t *testing.T) {
	for _, test := range []struct {
		name        string
		address     uint32
		pending     func(*Memory, byte)
		readMask    func(*Memory) byte
		readPending func(*Memory) byte
	}{
		{name: "IMRA", address: MFPIMRA,
			pending:  func(m *Memory, value byte) { m.mfpIPRA = value },
			readMask: func(m *Memory) byte { return m.mfpIMRA }, readPending: func(m *Memory) byte { return m.mfpIPRA }},
		{name: "IMRB", address: MFPIMRB,
			pending:  func(m *Memory, value byte) { m.mfpIPRB = value },
			readMask: func(m *Memory) byte { return m.mfpIMRB }, readPending: func(m *Memory) byte { return m.mfpIPRB }},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory, err := NewMemory(RAM1M, testROM())
			if err != nil {
				t.Fatal(err)
			}
			for _, value := range []byte{0xa5, 0x3c, 0xff, 0x00} {
				if wait, err := memory.WriteByteAt(test.address|0xff000000, value,
					m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
					t.Fatalf("timed %s write %02x wait=%d err=%v", test.name, value, wait, err)
				}
				if got, err := memory.ReadByte(test.address, 5); err != nil || got != value {
					t.Fatalf("%s=%02x/%v want %02x", test.name, got, err, value)
				}
			}
			test.pending(memory, 1)
			beforeMask, beforePending := test.readMask(memory), test.readPending(memory)
			if err := memory.WriteByte(test.address, 0xff, 5); err == nil {
				t.Fatalf("%s write with pending unexpectedly succeeded", test.name)
			}
			if test.readMask(memory) != beforeMask || test.readPending(memory) != beforePending {
				t.Fatalf("%s failed write changed mask/pending", test.name)
			}
			if _, err := memory.ReadByte(test.address, 1); err == nil {
				t.Fatalf("user %s read unexpectedly succeeded", test.name)
			}
			if err := memory.WriteByte(test.address, 0, 1); err == nil {
				t.Fatalf("user %s write unexpectedly succeeded", test.name)
			}
			if _, err := memory.ReadWord(test.address, 5); err == nil {
				t.Fatalf("odd %s word read unexpectedly succeeded", test.name)
			}
			if err := memory.M68KReset(); err != nil {
				t.Fatal(err)
			}
			if got, err := memory.ReadByte(test.address, 5); err != nil || got != 0 {
				t.Fatalf("%s after M68K reset=%02x/%v want 00", test.name, got, err)
			}
		})
	}
}

func TestMFPVectorRegister(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if got, err := memory.ReadByte(MFPVR, 5); err != nil || got != 0 {
		t.Fatalf("reset VR=%02x/%v want 00", got, err)
	}
	if wait, err := memory.WriteByteAt(MFPVR|0xff000000, 0xaf,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
		t.Fatalf("timed VR write wait=%d err=%v", wait, err)
	}
	if got, err := memory.ReadByte(MFPVR, 5); err != nil || got != 0xa8 {
		t.Fatalf("software EOI VR=%02x/%v want a8", got, err)
	}
	memory.mfpISRA, memory.mfpISRB = 0xa5, 0x5a
	if err := memory.WriteByte(MFPVR, 0xa7, 5); err != nil {
		t.Fatal(err)
	}
	if got, _ := memory.ReadByte(MFPVR, 5); got != 0xa0 {
		t.Fatalf("automatic EOI VR=%02x want a0", got)
	}
	if memory.mfpISRA != 0 || memory.mfpISRB != 0 {
		t.Fatalf("automatic EOI left ISR=%02x/%02x", memory.mfpISRA, memory.mfpISRB)
	}
	if err := memory.WriteByte(MFPVR, 0x07, 5); err != nil {
		t.Fatal(err)
	}
	if got, _ := memory.ReadByte(MFPVR, 5); got != 0 {
		t.Fatalf("unused VR bits read %02x want 00", got)
	}
	if err := memory.WriteByte(MFPVR, 0x58, 5); err != nil {
		t.Fatal(err)
	}
	memory.mfpIPRA, memory.mfpIPRB = 1, 2
	memory.mfpISRA, memory.mfpISRB = 4, 8
	if err := memory.WriteByte(MFPVR, 0x50, 5); err == nil {
		t.Fatal("automatic EOI with pending unexpectedly succeeded")
	}
	if memory.mfpVR != 0x58 || memory.mfpIPRA != 1 || memory.mfpIPRB != 2 ||
		memory.mfpISRA != 4 || memory.mfpISRB != 8 {
		t.Fatalf("failed VR write changed VR/IPR/ISR=%02x/%02x/%02x/%02x/%02x",
			memory.mfpVR, memory.mfpIPRA, memory.mfpIPRB, memory.mfpISRA, memory.mfpISRB)
	}
	if _, err := memory.ReadByte(MFPVR, 1); err == nil {
		t.Fatal("user VR read unexpectedly succeeded")
	}
	if err := memory.WriteByte(MFPVR, 0, 1); err == nil {
		t.Fatal("user VR write unexpectedly succeeded")
	}
	if _, err := memory.ReadWord(MFPVR, 5); err == nil {
		t.Fatal("odd VR word read unexpectedly succeeded")
	}
	memory.mfpIPRA, memory.mfpIPRB = 0, 0
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if got, err := memory.ReadByte(MFPVR, 5); err != nil || got != 0 ||
		memory.mfpISRA != 0 || memory.mfpISRB != 0 {
		t.Fatalf("VR/ISR after M68K reset=%02x/%02x/%02x err=%v",
			got, memory.mfpISRA, memory.mfpISRB, err)
	}
}

func TestMFPTimerControlResetStopWrites(t *testing.T) {
	for _, test := range []struct {
		name    string
		address uint32
		set     func(*Memory, byte)
	}{
		{name: "TACR", address: MFPTACR, set: func(m *Memory, value byte) { m.mfpTACR = value }},
		{name: "TBCR", address: MFPTBCR, set: func(m *Memory, value byte) { m.mfpTBCR = value }},
		{name: "TCDCR", address: MFPTCDCR, set: func(m *Memory, value byte) { m.mfpTCDCR = value }},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory, err := NewMemory(RAM1M, testROM())
			if err != nil {
				t.Fatal(err)
			}
			if got, err := memory.ReadByte(test.address, 5); err != nil || got != 0 {
				t.Fatalf("reset %s=%02x/%v want 00", test.name, got, err)
			}
			if wait, err := memory.WriteByteAt(test.address|0xff000000, 0,
				m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
				t.Fatalf("timed %s zero write wait=%d err=%v", test.name, wait, err)
			}
			if err := memory.WriteByte(test.address, 1, 5); err == nil {
				t.Fatalf("nonzero %s write unexpectedly succeeded", test.name)
			}
			if got, _ := memory.ReadByte(test.address, 5); got != 0 {
				t.Fatalf("failed %s write changed value to %02x", test.name, got)
			}
			test.set(memory, 1)
			if err := memory.WriteByte(test.address, 0, 5); err == nil {
				t.Fatalf("active %s stop unexpectedly succeeded", test.name)
			}
			if got, _ := memory.ReadByte(test.address, 5); got != 1 {
				t.Fatalf("failed active %s stop changed value to %02x", test.name, got)
			}
			if _, err := memory.ReadByte(test.address, 1); err == nil {
				t.Fatalf("user %s read unexpectedly succeeded", test.name)
			}
			if err := memory.WriteByte(test.address, 0, 1); err == nil {
				t.Fatalf("user %s write unexpectedly succeeded", test.name)
			}
			if _, err := memory.ReadWord(test.address, 5); err == nil {
				t.Fatalf("odd %s word read unexpectedly succeeded", test.name)
			}
			if err := memory.M68KReset(); err != nil {
				t.Fatal(err)
			}
			if got, err := memory.ReadByte(test.address, 5); err != nil || got != 0 {
				t.Fatalf("%s after M68K reset=%02x/%v want 00", test.name, got, err)
			}
		})
	}
}

func TestMFPTimerCDelayStartTransition(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.mfpTCDR, memory.mfpTCMain = 0xc0, 0xc0
	if wait, err := memory.WriteByteAt(0xfffffa1d, 0x50,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
		t.Fatalf("Timer C start wait=%d err=%v", wait, err)
	}
	if got, err := memory.ReadByte(MFPTCDCR, 5); err != nil || got != 0x50 || !memory.mfpTimerCStart {
		t.Fatalf("Timer C start control=%02x transition=%v err=%v", got, memory.mfpTimerCStart, err)
	}
	for _, value := range []byte{0, 0x10, 0x51, 0x60, 0xff} {
		before := memory.mfpTCDCR
		if err := memory.WriteByte(MFPTCDCR, value, 5); err == nil {
			t.Fatalf("active Timer C value %02x unexpectedly accepted", value)
		}
		if memory.mfpTCDCR != before || !memory.mfpTimerCStart {
			t.Fatalf("failed value %02x changed control/transition=%02x/%v", value, memory.mfpTCDCR, memory.mfpTimerCStart)
		}
	}
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if memory.mfpTCDCR != 0 || memory.mfpTimerCStart {
		t.Fatalf("reset control/transition=%02x/%v", memory.mfpTCDCR, memory.mfpTimerCStart)
	}
	for _, data := range []byte{0, 0xbf, 0xc1, 0xff} {
		memory.mfpTCDR, memory.mfpTCMain = data, data
		if err := memory.WriteByte(MFPTCDCR, 0x50, 5); err == nil {
			t.Fatalf("Timer C start with data %02x unexpectedly accepted", data)
		}
		if memory.mfpTCDCR != 0 || memory.mfpTimerCStart {
			t.Fatalf("failed data %02x changed control/transition", data)
		}
	}
}

func TestMFPTimerCInterruptEnable(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	for _, setup := range []func(*Memory){
		func(m *Memory) {},
		func(m *Memory) { m.mfpTCDCR = 0x50; m.mfpIPRB = 0x20 },
	} {
		setup(memory)
		if err := memory.WriteByte(MFPIERB, 0x20, 5); err == nil {
			t.Fatal("unsafe Timer C IERB enable unexpectedly accepted")
		}
		memory.mfpTCDCR, memory.mfpIPRB = 0, 0
	}
	memory.mfpTCDCR = 0x50
	if err := memory.WriteByte(MFPTCDCR, 0x50, 5); err != nil || memory.mfpTimerDStart {
		t.Fatalf("Timer D stopped same-value write err=%v transition=%v", err, memory.mfpTimerDStart)
	}
	if wait, err := memory.WriteByteAt(0xfffffa09, 0x20,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
		t.Fatalf("Timer C IERB enable wait=%d err=%v", wait, err)
	}
	if memory.mfpIERB != 0x20 || memory.mfpIPRB != 0 {
		t.Fatalf("Timer C IERB/IPRB=%02x/%02x", memory.mfpIERB, memory.mfpIPRB)
	}
	for _, value := range []byte{0x20, 0x21, 0x40, 0xff} {
		if err := memory.WriteByte(MFPIERB, value, 5); err == nil {
			t.Fatalf("active IERB value %02x unexpectedly accepted", value)
		}
		if memory.mfpIERB != 0x20 || memory.mfpIPRB != 0 {
			t.Fatalf("failed value %02x changed IERB/IPRB", value)
		}
	}
}

func TestMFPTimerDDelayStartTransition(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.mfpTCDCR = 0x50
	for _, data := range []byte{0, 1, 3, 0xff} {
		memory.mfpTDDR, memory.mfpTDMain = data, data
		if err := memory.WriteByte(MFPTCDCR, 0x51, 5); err == nil {
			t.Fatalf("Timer D start with data %02x unexpectedly accepted", data)
		}
		if memory.mfpTCDCR != 0x50 || memory.mfpTimerDStart {
			t.Fatalf("failed data %02x changed control/transition", data)
		}
	}
	memory.mfpTDDR, memory.mfpTDMain = 2, 2
	if wait, err := memory.WriteByteAt(0xfffffa1d, 0x51,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
		t.Fatalf("Timer D start wait=%d err=%v", wait, err)
	}
	if memory.mfpTCDCR != 0x51 || !memory.mfpTimerDStart || memory.mfpTCDR != 0 || memory.mfpTCMain != 0 {
		t.Fatalf("Timer D start control/transition/C=%02x/%v/%02x/%02x",
			memory.mfpTCDCR, memory.mfpTimerDStart, memory.mfpTCDR, memory.mfpTCMain)
	}
	for _, value := range []byte{0, 0x50, 0x51, 0x52, 0x61, 0xff} {
		if err := memory.WriteByte(MFPTCDCR, value, 5); err == nil {
			t.Fatalf("active Timer D value %02x unexpectedly accepted", value)
		}
		if memory.mfpTCDCR != 0x51 || !memory.mfpTimerDStart {
			t.Fatalf("failed value %02x changed control/transition", value)
		}
	}
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if memory.mfpTCDCR != 0 || memory.mfpTimerCStart || memory.mfpTimerDStart {
		t.Fatalf("reset control/transitions=%02x/%v/%v", memory.mfpTCDCR, memory.mfpTimerCStart, memory.mfpTimerDStart)
	}
}

func TestMFPUSARTFixedSerialEnable(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteByte(MFPUCR, 0x88, 5); err == nil {
		t.Fatal("UCR enable before Timer D unexpectedly accepted")
	}
	memory.mfpTCDCR, memory.mfpTDDR, memory.mfpTDMain = 0x51, 2, 2
	if wait, err := memory.WriteByteAt(0xfffffa29, 0x88,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
		t.Fatalf("UCR enable wait=%d err=%v", wait, err)
	}
	if err := memory.WriteByte(MFPTSR, 1, 5); err == nil {
		t.Fatal("TSR enable before RSR unexpectedly accepted")
	}
	if err := memory.WriteByte(MFPRSR, 1, 5); err != nil {
		t.Fatalf("RSR enable: %v", err)
	}
	if err := memory.WriteByte(MFPTSR, 1, 5); err == nil {
		t.Fatal("TSR enable before software-known reset unexpectedly accepted")
	}
	if err := memory.WriteByte(MFPTSR, 0, 5); err != nil {
		t.Fatalf("TSR software reset: %v", err)
	}
	if wait, err := memory.WriteByteAt(MFPTSR, 1,
		m68k.BusAccess{Clock: 0, FunctionCode: 5}); err != nil || wait != 4 {
		t.Fatalf("TSR enable wait=%d err=%v", wait, err)
	}
	if memory.mfpUCR != 0x88 || memory.mfpRSR != 1 || memory.mfpTSR != 1 || !memory.mfpTSRSet {
		t.Fatalf("USART state UCR/RSR/TSR/set=%02x/%02x/%02x/%v",
			memory.mfpUCR, memory.mfpRSR, memory.mfpTSR, memory.mfpTSRSet)
	}
	for address, value := range map[uint32]byte{MFPUCR: 0x88, MFPRSR: 1, MFPTSR: 0x81} {
		if got, err := memory.ReadByte(address, 5); err != nil || got != value {
			t.Fatalf("USART read %06x=%02x/%v want %02x", address, got, err, value)
		}
	}
	for address, value := range map[uint32]byte{MFPUCR: 0x88, MFPRSR: 1, MFPTSR: 1} {
		if err := memory.WriteByte(address, value, 5); err == nil {
			t.Fatalf("repeat USART write %06x unexpectedly accepted", address)
		}
	}
}

func TestMFPUSARTReconfigureAfterTimerDStop(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.mfpTimerDStopStage = 7
	memory.mfpIERB, memory.mfpIMRB, memory.mfpTCDCR = 0x60, 0x60, 0x50
	memory.mfpUCR, memory.mfpRSR = 0x88, 1
	memory.mfpTSR, memory.mfpTSRSet = 1, true

	if err := memory.WriteByte(MFPTDDR, 2, 5); err == nil || memory.mfpUSARTReconfigStage != 0 ||
		memory.mfpTDDR != 0 || memory.mfpTDMain != 0 {
		t.Fatalf("out-of-order TDDR err/stage/data/main=%v/%d/%02x/%02x", err,
			memory.mfpUSARTReconfigStage, memory.mfpTDDR, memory.mfpTDMain)
	}
	steps := []struct {
		address uint32
		value   byte
		stage   uint8
	}{
		{MFPTCDCR, 0x50, 1},
		{MFPTDDR, 2, 2},
		{MFPTCDCR, 0x51, 3},
		{MFPUCR, 0x88, 4},
		{MFPRSR, 1, 5},
		{MFPTSR, 1, 6},
		{MFPSCR, 0, 7},
	}
	if err := memory.WriteByte(steps[0].address, steps[0].value, 5); err != nil ||
		memory.mfpUSARTReconfigStage != 1 {
		t.Fatalf("stage 1 start err/stage=%v/%d", err, memory.mfpUSARTReconfigStage)
	}
	if err := memory.WriteByte(MFPTDDR, 3, 5); err == nil || memory.mfpUSARTReconfigStage != 1 ||
		memory.mfpTDDR != 0 || memory.mfpTDMain != 0 {
		t.Fatalf("wrong TDDR err/stage/data/main=%v/%d/%02x/%02x", err,
			memory.mfpUSARTReconfigStage, memory.mfpTDDR, memory.mfpTDMain)
	}
	for _, step := range steps[1:] {
		if err := memory.WriteByte(step.address, step.value, 5); err != nil {
			t.Fatalf("stage %d write %06x=%02x: %v", step.stage, step.address, step.value, err)
		}
		if memory.mfpUSARTReconfigStage != step.stage {
			t.Fatalf("write %06x=%02x stage=%d want %d", step.address, step.value,
				memory.mfpUSARTReconfigStage, step.stage)
		}
	}
	if got, err := memory.ReadByte(MFPTSR, 5); err != nil || got != 0x81 {
		t.Fatalf("enabled TSR=%02x/%v want 81", got, err)
	}
	if !memory.mfpTimerDStart || memory.mfpTCDCR != 0x51 || memory.mfpTDDR != 2 {
		t.Fatalf("baud Timer D start/control/data=%v/%02x/%02x",
			memory.mfpTimerDStart, memory.mfpTCDCR, memory.mfpTDDR)
	}
}

func TestMFPUSARTInterruptEnableSequence(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteByte(MFPIERA, 0x10, 5); err == nil {
		t.Fatal("USART IERA before serial init unexpectedly accepted")
	}
	memory.mfpUCR, memory.mfpRSR, memory.mfpTSR, memory.mfpTSRSet = 0x88, 1, 1, true
	for _, value := range []byte{0x10, 0x10, 0x14} {
		if wait, err := memory.WriteByteAt(0xfffffa07, value,
			m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
			t.Fatalf("IERA transition to %02x wait=%d err=%v", value, wait, err)
		}
	}
	if memory.mfpIERA != 0x14 || memory.mfpIPRA != 0 {
		t.Fatalf("USART IERA/IPRA=%02x/%02x", memory.mfpIERA, memory.mfpIPRA)
	}
	for _, value := range []byte{0, 0x10, 0x14, 0x15, 0xff} {
		if err := memory.WriteByte(MFPIERA, value, 5); err == nil {
			t.Fatalf("final IERA value %02x unexpectedly accepted", value)
		}
		if memory.mfpIERA != 0x14 || memory.mfpIPRA != 0 {
			t.Fatalf("failed value %02x changed IERA/IPRA", value)
		}
	}
}

func TestPSGFixedBootPortWrites(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	for _, write := range []struct {
		address uint32
		value   byte
	}{
		{PSGRegisterSelect, 7},
		{PSGRegisterData, 0xc0},
		{PSGRegisterSelect, 14},
		{PSGRegisterData, 7},
	} {
		if wait, err := memory.WriteByteAt(write.address|0xff00_0000, write.value,
			m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
			t.Fatalf("PSG write %06x=%02x wait=%d err=%v", write.address, write.value, wait, err)
		}
	}
	if memory.psgRegisterSelect != 14 || memory.psgRegisters[7] != 0xc0 || memory.psgRegisters[14] != 7 {
		t.Fatalf("PSG selected/R7/R14=%02x/%02x/%02x",
			memory.psgRegisterSelect, memory.psgRegisters[7], memory.psgRegisters[14])
	}
	for _, address := range []uint32{PSGRegisterSelect, PSGRegisterData} {
		if _, err := memory.ReadByte(address, 5); err == nil {
			t.Fatalf("unmodeled PSG read %06x unexpectedly accepted", address)
		}
		if err := memory.WriteByte(address, 0, 5); err == nil {
			t.Fatalf("invalid PSG write %06x unexpectedly accepted", address)
		}
		if err := memory.WriteByte(address, 0, 1); err == nil {
			t.Fatalf("user PSG write %06x unexpectedly accepted", address)
		}
		if err := memory.WriteWord(address, 0, 5); err == nil {
			t.Fatalf("PSG word write %06x unexpectedly accepted", address)
		}
	}
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if memory.psgRegisterSelect != 0 || memory.psgRegisters != [16]byte{} {
		t.Fatalf("PSG reset selected/registers=%02x/%v", memory.psgRegisterSelect, memory.psgRegisters)
	}
}

func TestPSGFirstDriveSelectUpdate(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.psgRegisterSelect = 14
	memory.psgRegisters[7], memory.psgRegisters[14] = 0xc0, 7
	if _, err := memory.ReadByte(PSGRegisterSelect, 5); err == nil || memory.psgDriveStage != 0 {
		t.Fatal("out-of-order PSG read unexpectedly accepted")
	}
	if err := memory.WriteByte(PSGRegisterData, 5, 5); err == nil ||
		memory.psgDriveStage != 0 || memory.psgRegisters[14] != 7 {
		t.Fatal("out-of-order PSG data write unexpectedly accepted")
	}
	if wait, err := memory.WriteByteAt(PSGRegisterSelect, 14,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 || memory.psgDriveStage != 1 {
		t.Fatalf("PSG reselect wait/err/stage=%d/%v/%d", wait, err, memory.psgDriveStage)
	}
	if got, wait, err := memory.ReadByteAt(PSGRegisterSelect,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 || got != 7 ||
		memory.psgDriveStage != 2 {
		t.Fatalf("PSG port read value/wait/err/stage=%02x/%d/%v/%d", got, wait, err, memory.psgDriveStage)
	}
	if err := memory.WriteByte(PSGRegisterData, 3, 5); err == nil ||
		memory.psgDriveStage != 2 || memory.psgRegisters[14] != 7 {
		t.Fatal("wrong PSG port value unexpectedly accepted")
	}
	if wait, err := memory.WriteByteAt(PSGRegisterData, 5,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.psgDriveStage != 3 || memory.psgRegisters[14] != 5 {
		t.Fatalf("PSG port update wait/err/stage/R14=%d/%v/%d/%02x", wait, err,
			memory.psgDriveStage, memory.psgRegisters[14])
	}
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if memory.psgDriveStage != 0 || memory.psgRegisterSelect != 0 || memory.psgRegisters != [16]byte{} {
		t.Fatalf("PSG reset stage/select/registers=%d/%02x/%v", memory.psgDriveStage,
			memory.psgRegisterSelect, memory.psgRegisters)
	}
}

func TestPSGSelectsDriveOne(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.psgDriveStage = 3
	memory.psgRegisterSelect = 14
	memory.psgRegisters[7], memory.psgRegisters[14] = 0xc0, 5
	memory.fdcInitStage = 14
	if _, err := memory.ReadByte(PSGRegisterSelect, 5); err == nil || memory.psgDriveStage != 3 {
		t.Fatal("drive-one read before reselect unexpectedly accepted")
	}
	if err := memory.WriteByte(PSGRegisterData, 3, 5); err == nil ||
		memory.psgDriveStage != 3 || memory.psgRegisters[14] != 5 {
		t.Fatal("drive-one data before reselect unexpectedly accepted")
	}
	if wait, err := memory.WriteByteAt(PSGRegisterSelect, 14,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.psgDriveStage != 4 {
		t.Fatalf("drive-one reselect wait/err/stage=%d/%v/%d", wait, err, memory.psgDriveStage)
	}
	if got, wait, err := memory.ReadByteAt(PSGRegisterSelect,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 || got != 5 ||
		memory.psgDriveStage != 5 {
		t.Fatalf("drive-one read value/wait/err/stage=%02x/%d/%v/%d", got, wait, err,
			memory.psgDriveStage)
	}
	if err := memory.WriteByte(PSGRegisterData, 1, 5); err == nil ||
		memory.psgDriveStage != 5 || memory.psgRegisters[14] != 5 {
		t.Fatal("wrong drive-one port value unexpectedly accepted")
	}
	if wait, err := memory.WriteByteAt(PSGRegisterData, 3,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.psgDriveStage != 6 || memory.psgRegisters[14] != 3 {
		t.Fatalf("drive-one update wait/err/stage/R14=%d/%v/%d/%02x", wait, err,
			memory.psgDriveStage, memory.psgRegisters[14])
	}
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if memory.psgDriveStage != 0 || memory.psgRegisterSelect != 0 || memory.psgRegisters != [16]byte{} {
		t.Fatalf("drive-one reset stage/select/registers=%d/%02x/%v", memory.psgDriveStage,
			memory.psgRegisterSelect, memory.psgRegisters)
	}
}

func TestFlopVBLChecksDriveZeroAndRestoresPortA(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.psgDriveStage = 9
	memory.psgRegisterSelect = 14
	memory.psgRegisters[7], memory.psgRegisters[14] = 0xc0, 0x23
	memory.fdcInitStage = 14
	memory.fdcProbeDrive = 1
	memory.acsiStage = 5
	memory.dmaMode = 0x0080
	memory.fdcStatus = 0xe4
	memory.fdcStatusTypeI = true
	memory.ikbdClockReadbackComplete = true

	if err := memory.WriteByte(PSGRegisterData, 0x25, 5); err == nil {
		t.Fatal("media-check data write before register select unexpectedly accepted")
	}
	if err := memory.WriteByte(PSGRegisterSelect, 14, 5); err != nil || memory.flopVBLMediaStage != 1 {
		t.Fatalf("media-check select stage=%d err=%v", memory.flopVBLMediaStage, err)
	}
	if got, err := memory.ReadByte(PSGRegisterSelect, 5); err != nil || got != 0x23 ||
		memory.flopVBLMediaStage != 2 {
		t.Fatalf("media-check old port/stage=%02x/%d err=%v", got, memory.flopVBLMediaStage, err)
	}
	if err := memory.WriteByte(PSGRegisterData, 0x25, 5); err != nil ||
		memory.psgRegisters[14] != 0x25 || memory.flopVBLMediaStage != 3 {
		t.Fatalf("media-check drive-0 port/stage=%02x/%d err=%v",
			memory.psgRegisters[14], memory.flopVBLMediaStage, err)
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0080,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 6 ||
		memory.flopVBLMediaStage != 4 {
		t.Fatalf("media-check DMA wait/stage=%d/%d err=%v", wait, memory.flopVBLMediaStage, err)
	}
	if got, wait, err := memory.ReadWordAt(STDiskController,
		m68k.BusAccess{Clock: 100, FunctionCode: 5}); err != nil || wait != 4 || got != 0x00e4 ||
		memory.flopVBLMediaStage != 5 || memory.flopVBLStatusReadClock != 100 ||
		memory.fdcIRQ || memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("media-check status/wait/stage/clock/IRQ/GPIP=%04x/%d/%d/%d/%v/%02x err=%v",
			got, wait, memory.flopVBLMediaStage, memory.flopVBLStatusReadClock,
			memory.fdcIRQ, memory.mfpGPIPIn, err)
	}
	if err := memory.WriteByte(PSGRegisterSelect, 14, 5); err != nil || memory.flopVBLMediaStage != 6 {
		t.Fatalf("media-check restore select stage=%d err=%v", memory.flopVBLMediaStage, err)
	}
	if got, err := memory.ReadByte(PSGRegisterSelect, 5); err != nil || got != 0x25 ||
		memory.flopVBLMediaStage != 7 {
		t.Fatalf("media-check restore old port/stage=%02x/%d err=%v", got,
			memory.flopVBLMediaStage, err)
	}
	if err := memory.WriteByte(PSGRegisterData, 0x23, 5); err != nil ||
		memory.psgRegisters[14] != 0x23 || memory.flopVBLMediaStage != 8 ||
		!memory.flopVBLMediaComplete || memory.flopVBLMediaChecks != 1 || memory.flopVBLMediaDrive != 0 ||
		memory.fdcInitStage != 14 || memory.fdcProbeDrive != 1 {
		t.Fatalf("media-check completion port/stage/complete/checks/media-drive/FDC/probe-drive=%02x/%d/%v/%d/%d/%d/%d err=%v",
			memory.psgRegisters[14], memory.flopVBLMediaStage, memory.flopVBLMediaComplete,
			memory.flopVBLMediaChecks, memory.flopVBLMediaDrive, memory.fdcInitStage,
			memory.fdcProbeDrive, err)
	}
	memory.ColdReset()
	if memory.flopVBLMediaStage != 0 || memory.flopVBLStatusReadClock != 0 ||
		memory.flopVBLMediaComplete || memory.flopVBLMediaChecks != 0 || memory.flopVBLMediaDrive != -1 {
		t.Fatal("cold reset retained flopvbl media-check state")
	}
}

func TestFlopVBLAlternatesDriveChecks(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.psgDriveStage = 9
	memory.psgRegisterSelect = 14
	memory.psgRegisters[7], memory.psgRegisters[14] = 0xc0, 0x23
	memory.fdcInitStage = 14
	memory.fdcProbeDrive = 1
	memory.acsiStage = 5
	memory.dmaMode = 0x0080
	memory.fdcStatus = 0xe4
	memory.fdcStatusTypeI = true
	memory.ikbdClockReadbackComplete = true

	targets := [4]byte{0x25, 0x23, 0x25, 0x23}
	for cycle, target := range targets {
		if err := memory.WriteByte(PSGRegisterSelect, 14, 5); err != nil {
			t.Fatalf("cycle %d select: %v", cycle, err)
		}
		if got, err := memory.ReadByte(PSGRegisterSelect, 5); err != nil || got != 0x23 {
			t.Fatalf("cycle %d old port=%02x err=%v", cycle, got, err)
		}
		if err := memory.WriteByte(PSGRegisterData, target, 5); err != nil {
			t.Fatalf("cycle %d target %02x: %v", cycle, target, err)
		}
		if err := memory.WriteWord(STDMAControl, 0x0080, 5); err != nil {
			t.Fatalf("cycle %d DMA mode: %v", cycle, err)
		}
		clock := uint64(100 + 2*cycle)
		if got, _, err := memory.ReadWordAt(STDiskController,
			m68k.BusAccess{Clock: clock, FunctionCode: 5}); err != nil || got != 0xe4 {
			t.Fatalf("cycle %d status=%04x err=%v", cycle, got, err)
		}
		if err := memory.WriteByte(PSGRegisterSelect, 14, 5); err != nil {
			t.Fatalf("cycle %d restore select: %v", cycle, err)
		}
		if got, err := memory.ReadByte(PSGRegisterSelect, 5); err != nil || got != target {
			t.Fatalf("cycle %d selected port=%02x err=%v want %02x", cycle, got, err, target)
		}
		if err := memory.WriteByte(PSGRegisterData, 0x23, 5); err != nil ||
			memory.flopVBLMediaChecks != uint32(cycle+1) ||
			memory.flopVBLMediaDrive != int8(cycle&1) || memory.flopVBLStatusReadClock != clock ||
			memory.flopVBLMediaStage != 8 || memory.psgRegisters[14] != 0x23 {
			t.Fatalf("cycle %d completion checks/drive/clock/stage/port=%d/%d/%d/%d/%02x err=%v",
				cycle, memory.flopVBLMediaChecks, memory.flopVBLMediaDrive,
				memory.flopVBLStatusReadClock, memory.flopVBLMediaStage,
				memory.psgRegisters[14], err)
		}
	}
}

func TestFloppyMediaReadLocksDriveZeroAtTrackZero(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.psgDriveStage = 9
	memory.psgRegisterSelect = 14
	memory.psgRegisters[7], memory.psgRegisters[14] = 0xc0, 0x23
	memory.flopVBLMediaStage = 8
	memory.flopVBLMediaComplete = true
	memory.flopVBLMediaChecks = 73
	memory.fdcInitStage = 14
	memory.fdcProbeDrive = 1
	memory.acsiStage = 5
	memory.dmaMode = 0x0080

	if err := memory.WriteWord(STDiskController, 0, 5); err == nil {
		t.Fatal("track data before selector unexpectedly accepted")
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0082,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 6 ||
		memory.floppyReadStage != 1 || memory.dmaMode != 0x0082 {
		t.Fatalf("track selector wait/stage/mode=%d/%d/%04x err=%v",
			wait, memory.floppyReadStage, memory.dmaMode, err)
	}
	if wait, err := memory.WriteWordAt(STDiskController, 0,
		m68k.BusAccess{Clock: 100, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 2 || memory.floppyReadTrack != 0 ||
		memory.floppyReadTrackWriteClock != 100 {
		t.Fatalf("track write wait/stage/track/clock=%d/%d/%02x/%d err=%v",
			wait, memory.floppyReadStage, memory.floppyReadTrack,
			memory.floppyReadTrackWriteClock, err)
	}
	if err := memory.WriteByte(PSGRegisterData, 0x25, 5); err == nil {
		t.Fatal("drive data before select/read unexpectedly accepted")
	}
	if err := memory.WriteByte(PSGRegisterSelect, 14, 5); err != nil || memory.floppyReadStage != 3 {
		t.Fatalf("drive select stage=%d err=%v", memory.floppyReadStage, err)
	}
	if got, err := memory.ReadByte(PSGRegisterSelect, 5); err != nil || got != 0x23 ||
		memory.floppyReadStage != 4 {
		t.Fatalf("drive old port/stage=%02x/%d err=%v", got, memory.floppyReadStage, err)
	}
	if err := memory.WriteByte(PSGRegisterData, 0x25, 5); err != nil ||
		memory.floppyReadStage != 5 || memory.floppyReadDrive != 0 ||
		memory.psgRegisters[14] != 0x25 || memory.flopVBLMediaChecks != 73 {
		t.Fatalf("drive completion stage/drive/port/checks=%d/%d/%02x/%d err=%v",
			memory.floppyReadStage, memory.floppyReadDrive, memory.psgRegisters[14],
			memory.flopVBLMediaChecks, err)
	}
	if err := memory.WriteWord(STDiskController, 1, 5); err == nil {
		t.Fatal("sector data before selector unexpectedly accepted")
	}
	if err := memory.WriteWord(STDMAControl, 0x0084, 5); err != nil || memory.floppyReadStage != 6 {
		t.Fatalf("sector selector stage=%d err=%v", memory.floppyReadStage, err)
	}
	if err := memory.WriteWord(STDiskController, 1, 5); err != nil ||
		memory.floppyReadStage != 7 || memory.floppyReadSector != 1 {
		t.Fatalf("sector data stage/sector=%d/%02x err=%v",
			memory.floppyReadStage, memory.floppyReadSector, err)
	}
	if err := memory.WriteByte(STDMAAddressMiddle, 0x10, 5); err != nil || memory.floppyReadStage != 7 {
		t.Fatalf("out-of-order DMA middle stage=%d err=%v", memory.floppyReadStage, err)
	}
	for index, write := range []struct {
		address uint32
		value   byte
		stage   uint8
	}{{STDMAAddressLow, 0x04, 8}, {STDMAAddressMiddle, 0x10, 9}, {STDMAAddressHigh, 0, 10}} {
		if err := memory.WriteByte(write.address, write.value, 5); err != nil ||
			memory.floppyReadStage != write.stage || memory.floppyReadDMAAddressStage != uint8(index+1) {
			t.Fatalf("DMA address %d stage/address-stage=%d/%d err=%v", index,
				memory.floppyReadStage, memory.floppyReadDMAAddressStage, err)
		}
	}
	if memory.dmaAddress != 0x001004 {
		t.Fatalf("DMA address=%06x", memory.dmaAddress)
	}
	if err := memory.WriteWord(STDMAControl, 0x0190, 5); err != nil ||
		memory.floppyReadStage != 11 || memory.floppyReadDMAResetCount != 1 {
		t.Fatalf("DMA reset one stage/count=%d/%d err=%v",
			memory.floppyReadStage, memory.floppyReadDMAResetCount, err)
	}
	if err := memory.WriteWord(STDMAControl, 0x0090, 5); err != nil ||
		memory.floppyReadStage != 12 || memory.floppyReadDMAResetCount != 2 {
		t.Fatalf("DMA reset two stage/count=%d/%d err=%v",
			memory.floppyReadStage, memory.floppyReadDMAResetCount, err)
	}
	if err := memory.WriteWord(STDiskController, 1, 5); err != nil ||
		memory.floppyReadStage != 13 || memory.dmaSectorCount != 1 {
		t.Fatalf("DMA count stage/count=%d/%d err=%v",
			memory.floppyReadStage, memory.dmaSectorCount, err)
	}
	if err := memory.WriteWord(STDMAControl, 0x0080, 5); err != nil || memory.floppyReadStage != 14 {
		t.Fatalf("command selector stage=%d err=%v", memory.floppyReadStage, err)
	}
	before := append([]byte(nil), memory.ram[0x1004:0x1204]...)
	if wait, err := memory.WriteWordAt(STDiskController, 0x0080,
		m68k.BusAccess{Clock: 200, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 15 || memory.floppyReadCommand != 0x80 ||
		memory.floppyReadCommandClock != 200 || memory.fdcCommand != 0x80 ||
		memory.fdcIRQ || memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("read command wait/stage/command/clock/FDC/IRQ/GPIP=%d/%d/%02x/%d/%02x/%v/%02x err=%v",
			wait, memory.floppyReadStage, memory.floppyReadCommand,
			memory.floppyReadCommandClock, memory.fdcCommand, memory.fdcIRQ,
			memory.mfpGPIPIn, err)
	}
	if !bytes.Equal(before, memory.ram[0x1004:0x1204]) {
		t.Fatal("no-disk command modified DMA buffer before successful transfer")
	}
	if err := memory.WriteWord(STDiskController, 0x00d0, 5); err == nil {
		t.Fatal("force interrupt before timeout selector unexpectedly accepted")
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0080,
		m68k.BusAccess{Clock: 300, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 16 || memory.floppyReadTimeoutSelectorClock != 300 {
		t.Fatalf("timeout selector wait/stage/clock=%d/%d/%d err=%v", wait,
			memory.floppyReadStage, memory.floppyReadTimeoutSelectorClock, err)
	}
	if err := memory.WriteWord(STDiskController, 0x00d8, 5); err == nil ||
		memory.floppyReadStage != 16 || memory.fdcCommand != 0x80 || memory.fdcStatus != 0x81 {
		t.Fatalf("wrong force interrupt mutated stage/command/status=%d/%02x/%02x err=%v",
			memory.floppyReadStage, memory.fdcCommand, memory.fdcStatus, err)
	}
	if wait, err := memory.WriteWordAt(STDiskController, 0x00d0,
		m68k.BusAccess{Clock: 400, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 17 || memory.floppyReadForceInterrupt != 0xd0 ||
		memory.floppyReadForceInterruptClock != 400 || memory.fdcCommand != 0xd0 ||
		memory.fdcStatus != 0x80 || memory.fdcStatusTypeI || memory.fdcIRQ ||
		memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("force interrupt wait/stage/receipt/clock/FDC/status/type/IRQ/GPIP=%d/%d/%02x/%d/%02x/%02x/%v/%v/%02x err=%v",
			wait, memory.floppyReadStage, memory.floppyReadForceInterrupt,
			memory.floppyReadForceInterruptClock, memory.fdcCommand, memory.fdcStatus,
			memory.fdcStatusTypeI, memory.fdcIRQ, memory.mfpGPIPIn, err)
	}
	if !bytes.Equal(before, memory.ram[0x1004:0x1204]) {
		t.Fatal("force interrupt modified DMA buffer")
	}
	if err := memory.WriteWord(STDiskController, 0, 5); err == nil {
		t.Fatal("retry data before selector unexpectedly accepted")
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0086,
		m68k.BusAccess{Clock: 500, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 18 || memory.dmaMode != 0x0086 {
		t.Fatalf("retry data selector wait/stage/mode=%d/%d/%04x err=%v",
			wait, memory.floppyReadStage, memory.dmaMode, err)
	}
	if wait, err := memory.WriteWordAt(STDiskController, 0,
		m68k.BusAccess{Clock: 600, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 19 || memory.floppyReadRetryData != 0 || memory.fdcData != 0 {
		t.Fatalf("retry data wait/stage/data/FDC=%d/%d/%02x/%02x err=%v", wait,
			memory.floppyReadStage, memory.floppyReadRetryData, memory.fdcData, err)
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0080,
		m68k.BusAccess{Clock: 700, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 20 || memory.dmaMode != 0x0080 {
		t.Fatalf("retry command selector wait/stage/mode=%d/%d/%04x err=%v",
			wait, memory.floppyReadStage, memory.dmaMode, err)
	}
	if err := memory.WriteWord(STDiskController, 0x0012, 5); err == nil ||
		memory.floppyReadStage != 20 || memory.fdcSeekPending {
		t.Fatal("wrong retry seek command unexpectedly accepted")
	}
	if wait, err := memory.WriteWordAt(STDiskController, 0x0013,
		m68k.BusAccess{Clock: 800, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 21 || memory.floppyReadRetrySeekCommand != 0x13 ||
		memory.floppyReadRetrySeekStartClock != 800 || memory.fdcSeekStartClock != 800 ||
		!memory.fdcSeekPending || memory.fdcStatus != 0xe5 || !memory.fdcStatusTypeI ||
		memory.fdcIRQ || memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("retry seek wait/stage/command/clocks/pending/status/type/IRQ/GPIP=%d/%d/%02x/%d/%d/%v/%02x/%v/%v/%02x err=%v",
			wait, memory.floppyReadStage, memory.floppyReadRetrySeekCommand,
			memory.floppyReadRetrySeekStartClock, memory.fdcSeekStartClock,
			memory.fdcSeekPending, memory.fdcStatus, memory.fdcStatusTypeI,
			memory.fdcIRQ, memory.mfpGPIPIn, err)
	}
	for index := 0; index < 9; index++ {
		if got, err := memory.ReadByte(MFPGPIP, 5); err != nil || got&0x20 == 0 {
			t.Fatalf("retry inactive poll %d=%02x err=%v", index, got, err)
		}
	}
	if memory.floppyReadRetryInactivePolls != 9 || !memory.fdcSeekPending {
		t.Fatalf("retry polls/pending=%d/%v", memory.floppyReadRetryInactivePolls,
			memory.fdcSeekPending)
	}
	machine := &Machine{Memory: memory, Clocks: 1528}
	machine.CPU.Bus = memory
	machine.advanceClockedDevices()
	if !machine.fdcSeekClockStarted || machine.nextFDCSeekClock != 1529 ||
		memory.floppyReadStage != 21 || !memory.fdcSeekPending {
		t.Fatalf("retry early scheduler/next/stage/pending=%v/%d/%d/%v",
			machine.fdcSeekClockStarted, machine.nextFDCSeekClock,
			memory.floppyReadStage, memory.fdcSeekPending)
	}
	machine.Clocks = 1529
	machine.advanceClockedDevices()
	if machine.fdcSeekClockStarted || machine.nextFDCSeekClock != 0 ||
		memory.floppyReadStage != 22 || memory.fdcSeekPending || memory.fdcStatus != 0xe4 ||
		!memory.fdcIRQ || memory.mfpGPIPIn&0x20 != 0 {
		t.Fatalf("retry complete scheduler/next/stage/pending/status/IRQ/GPIP=%v/%d/%d/%v/%02x/%v/%02x",
			machine.fdcSeekClockStarted, machine.nextFDCSeekClock, memory.floppyReadStage,
			memory.fdcSeekPending, memory.fdcStatus, memory.fdcIRQ, memory.mfpGPIPIn)
	}
	if got, err := memory.ReadByte(MFPGPIP, 5); err != nil || got&0x20 != 0 ||
		!memory.floppyReadRetryIRQObserved {
		t.Fatalf("retry IRQ poll=%02x observed=%v err=%v", got,
			memory.floppyReadRetryIRQObserved, err)
	}
	if _, err := memory.ReadWord(STDiskController, 5); err == nil {
		t.Fatal("retry status before selector unexpectedly accepted")
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0080,
		m68k.BusAccess{Clock: 1600, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 23 {
		t.Fatalf("retry status selector wait/stage=%d/%d err=%v", wait,
			memory.floppyReadStage, err)
	}
	if value, wait, err := memory.ReadWordAt(STDiskController,
		m68k.BusAccess{Clock: 1700, FunctionCode: 5}); err != nil || wait != 4 ||
		value != 0x00e4 || memory.floppyReadStage != 24 ||
		memory.floppyReadRetryStatusReadClock != 1700 || memory.fdcIRQ ||
		memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("retry status value/wait/stage/clock/IRQ/GPIP=%04x/%d/%d/%d/%v/%02x err=%v",
			value, wait, memory.floppyReadStage, memory.floppyReadRetryStatusReadClock,
			memory.fdcIRQ, memory.mfpGPIPIn, err)
	}
	if !bytes.Equal(before, memory.ram[0x1004:0x1204]) {
		t.Fatal("retry dummy seek modified DMA buffer")
	}
	mediaChecks := memory.flopVBLMediaChecks
	if _, _, err := memory.ReadByteAt(PSGRegisterSelect,
		m68k.BusAccess{Clock: 1800, FunctionCode: 5}); err == nil {
		t.Fatal("retry drive port read before selector unexpectedly accepted")
	}
	if wait, err := memory.WriteByteAt(PSGRegisterSelect, 14,
		m68k.BusAccess{Clock: 1900, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 25 || memory.psgRegisterSelect != 14 {
		t.Fatalf("retry drive selector wait/stage/select=%d/%d/%02x err=%v", wait,
			memory.floppyReadStage, memory.psgRegisterSelect, err)
	}
	if value, wait, err := memory.ReadByteAt(PSGRegisterSelect,
		m68k.BusAccess{Clock: 2000, FunctionCode: 5}); err != nil || wait != 4 ||
		value != 0x25 || memory.floppyReadStage != 26 || memory.psgRegisters[14] != 0x25 {
		t.Fatalf("retry drive read value/wait/stage/port=%02x/%d/%d/%02x err=%v",
			value, wait, memory.floppyReadStage, memory.psgRegisters[14], err)
	}
	if err := memory.WriteByte(PSGRegisterData, 0x23, 5); err == nil ||
		memory.floppyReadStage != 26 || memory.psgRegisters[14] != 0x25 {
		t.Fatalf("wrong retry drive value mutated stage/port=%d/%02x err=%v",
			memory.floppyReadStage, memory.psgRegisters[14], err)
	}
	if wait, err := memory.WriteByteAt(PSGRegisterData, 0x25,
		m68k.BusAccess{Clock: 2100, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 27 || memory.floppyReadRetryDrivePort != 0x25 ||
		memory.floppyReadRetryDriveWriteClock != 2100 || memory.psgRegisters[14] != 0x25 ||
		memory.flopVBLMediaChecks != mediaChecks {
		t.Fatalf("retry drive write wait/stage/receipt/clock/port/checks=%d/%d/%02x/%d/%02x/%d err=%v",
			wait, memory.floppyReadStage, memory.floppyReadRetryDrivePort,
			memory.floppyReadRetryDriveWriteClock, memory.psgRegisters[14],
			memory.flopVBLMediaChecks, err)
	}
	if err := memory.WriteWord(STDiskController, 1, 5); err == nil ||
		memory.floppyReadStage != 27 || memory.floppyReadRetrySector != 0 {
		t.Fatalf("retry sector before selector mutated stage/sector=%d/%02x err=%v",
			memory.floppyReadStage, memory.floppyReadRetrySector, err)
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0084,
		m68k.BusAccess{Clock: 2202, FunctionCode: 5}); err != nil || wait != 6 ||
		memory.floppyReadStage != 28 || memory.dmaMode != 0x0084 {
		t.Fatalf("retry sector selector wait/stage/mode=%d/%d/%04x err=%v",
			wait, memory.floppyReadStage, memory.dmaMode, err)
	}
	retryAddressBefore := memory.dmaAddress
	if err := memory.WriteByte(STDMAAddressLow, 0x04, 5); err == nil ||
		memory.floppyReadStage != 28 || memory.dmaAddress != retryAddressBefore {
		t.Fatalf("retry DMA before sector mutated stage/address=%d/%06x err=%v",
			memory.floppyReadStage, memory.dmaAddress, err)
	}
	if wait, err := memory.WriteWordAt(STDiskController, 1,
		m68k.BusAccess{Clock: 2300, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 29 || memory.floppyReadRetrySector != 1 {
		t.Fatalf("retry sector wait/stage/sector=%d/%d/%02x err=%v",
			wait, memory.floppyReadStage, memory.floppyReadRetrySector, err)
	}
	if err := memory.WriteByte(STDMAAddressMiddle, 0x10, 5); err == nil ||
		memory.floppyReadStage != 29 || memory.floppyReadRetryDMAAddressStage != 0 {
		t.Fatalf("retry DMA out of order mutated stage/address-stage=%d/%d err=%v",
			memory.floppyReadStage, memory.floppyReadRetryDMAAddressStage, err)
	}
	for index, write := range []struct {
		address uint32
		value   byte
		stage   uint8
	}{
		{STDMAAddressLow, 0x04, 30},
		{STDMAAddressMiddle, 0x10, 31},
		{STDMAAddressHigh, 0x00, 32},
	} {
		if wait, err := memory.WriteByteAt(write.address, write.value,
			m68k.BusAccess{Clock: uint64(2400 + index*100), FunctionCode: 5}); err != nil || wait != 0 ||
			memory.floppyReadStage != write.stage ||
			memory.floppyReadRetryDMAAddressStage != uint8(index+1) {
			t.Fatalf("retry DMA address[%d] wait/stage/address-stage=%d/%d/%d err=%v", index,
				wait, memory.floppyReadStage, memory.floppyReadRetryDMAAddressStage, err)
		}
	}
	if memory.dmaAddress != 0x001004 || memory.floppyReadDMAAddressStage != 3 {
		t.Fatalf("retry DMA address/first receipt=%06x/%d", memory.dmaAddress,
			memory.floppyReadDMAAddressStage)
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0190,
		m68k.BusAccess{Clock: 2700, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 33 || memory.floppyReadRetryDMAResetCount != 1 {
		t.Fatalf("retry DMA first reset wait/stage/count=%d/%d/%d err=%v", wait,
			memory.floppyReadStage, memory.floppyReadRetryDMAResetCount, err)
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0090,
		m68k.BusAccess{Clock: 2800, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 34 || memory.floppyReadRetryDMAResetCount != 2 {
		t.Fatalf("retry DMA second reset wait/stage/count=%d/%d/%d err=%v", wait,
			memory.floppyReadStage, memory.floppyReadRetryDMAResetCount, err)
	}
	if wait, err := memory.WriteWordAt(STDiskController, 1,
		m68k.BusAccess{Clock: 2900, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 35 || memory.dmaSectorCount != 1 {
		t.Fatalf("retry DMA count wait/stage/count=%d/%d/%d err=%v", wait,
			memory.floppyReadStage, memory.dmaSectorCount, err)
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0080,
		m68k.BusAccess{Clock: 3000, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 36 || memory.dmaMode != 0x0080 {
		t.Fatalf("retry command selector wait/stage/mode=%d/%d/%04x err=%v", wait,
			memory.floppyReadStage, memory.dmaMode, err)
	}
	if wait, err := memory.WriteWordAt(STDiskController, 0x0080,
		m68k.BusAccess{Clock: 3100, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 37 || memory.floppyReadRetryCommand != 0x80 ||
		memory.floppyReadRetryCommandClock != 3100 || memory.fdcCommand != 0x80 ||
		memory.fdcStatus != 0x81 || memory.fdcStatusTypeI || memory.fdcIRQ ||
		memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("retry command wait/stage/receipt/clock/FDC/status/type/IRQ/GPIP=%d/%d/%02x/%d/%02x/%02x/%v/%v/%02x err=%v",
			wait, memory.floppyReadStage, memory.floppyReadRetryCommand,
			memory.floppyReadRetryCommandClock, memory.fdcCommand, memory.fdcStatus,
			memory.fdcStatusTypeI, memory.fdcIRQ, memory.mfpGPIPIn, err)
	}
	if !bytes.Equal(before, memory.ram[0x1004:0x1204]) ||
		memory.floppyReadSector != 1 || memory.floppyReadDMAResetCount != 2 ||
		memory.floppyReadCommand != 0x80 || memory.floppyReadCommandClock != 200 {
		t.Fatal("retry command modified DMA buffer or first transaction receipts")
	}
	if err := memory.WriteWord(STDiskController, 0x00d0, 5); err == nil ||
		memory.floppyReadStage != 37 || memory.floppyRetryForceInterrupt != 0 ||
		memory.fdcCommand != 0x80 || memory.fdcStatus != 0x81 {
		t.Fatalf("retry force interrupt before selector mutated stage/receipt/FDC/status=%d/%02x/%02x/%02x err=%v",
			memory.floppyReadStage, memory.floppyRetryForceInterrupt,
			memory.fdcCommand, memory.fdcStatus, err)
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0080,
		m68k.BusAccess{Clock: 3202, FunctionCode: 5}); err != nil || wait != 6 ||
		memory.floppyReadStage != 38 || memory.floppyRetryTimeoutSelectorClock != 3202 {
		t.Fatalf("retry timeout selector wait/stage/clock=%d/%d/%d err=%v", wait,
			memory.floppyReadStage, memory.floppyRetryTimeoutSelectorClock, err)
	}
	if err := memory.WriteWord(STDiskController, 0x00d8, 5); err == nil ||
		memory.floppyReadStage != 38 || memory.floppyRetryForceInterrupt != 0 ||
		memory.fdcCommand != 0x80 || memory.fdcStatus != 0x81 {
		t.Fatalf("wrong retry force interrupt mutated stage/receipt/FDC/status=%d/%02x/%02x/%02x err=%v",
			memory.floppyReadStage, memory.floppyRetryForceInterrupt,
			memory.fdcCommand, memory.fdcStatus, err)
	}
	if wait, err := memory.WriteWordAt(STDiskController, 0x00d0,
		m68k.BusAccess{Clock: 3300, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 39 || memory.floppyRetryForceInterrupt != 0xd0 ||
		memory.floppyRetryForceInterruptClock != 3300 || memory.fdcCommand != 0xd0 ||
		memory.fdcStatus != 0x80 || memory.fdcStatusTypeI || memory.fdcIRQ ||
		memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("retry force interrupt wait/stage/receipt/clock/FDC/status/type/IRQ/GPIP=%d/%d/%02x/%d/%02x/%02x/%v/%v/%02x err=%v",
			wait, memory.floppyReadStage, memory.floppyRetryForceInterrupt,
			memory.floppyRetryForceInterruptClock, memory.fdcCommand, memory.fdcStatus,
			memory.fdcStatusTypeI, memory.fdcIRQ, memory.mfpGPIPIn, err)
	}
	if !bytes.Equal(before, memory.ram[0x1004:0x1204]) ||
		memory.floppyReadTimeoutSelectorClock != 300 || memory.floppyReadForceInterrupt != 0xd0 ||
		memory.floppyReadForceInterruptClock != 400 {
		t.Fatal("retry timeout modified DMA buffer or first timeout receipts")
	}
	if err := memory.WriteWord(STDiskController, 0, 5); err == nil ||
		memory.floppyReadStage != 39 || memory.floppyRetry2Data != 0 {
		t.Fatalf("second dummy data before selector mutated stage/data=%d/%02x err=%v",
			memory.floppyReadStage, memory.floppyRetry2Data, err)
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0086,
		m68k.BusAccess{Clock: 3400, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 40 || memory.dmaMode != 0x0086 {
		t.Fatalf("second dummy data selector wait/stage/mode=%d/%d/%04x err=%v",
			wait, memory.floppyReadStage, memory.dmaMode, err)
	}
	if wait, err := memory.WriteWordAt(STDiskController, 0,
		m68k.BusAccess{Clock: 3500, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 41 || memory.floppyRetry2Data != 0 || memory.fdcData != 0 {
		t.Fatalf("second dummy data wait/stage/data/FDC=%d/%d/%02x/%02x err=%v", wait,
			memory.floppyReadStage, memory.floppyRetry2Data, memory.fdcData, err)
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0080,
		m68k.BusAccess{Clock: 3600, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 42 || memory.dmaMode != 0x0080 {
		t.Fatalf("second dummy command selector wait/stage/mode=%d/%d/%04x err=%v",
			wait, memory.floppyReadStage, memory.dmaMode, err)
	}
	if err := memory.WriteWord(STDiskController, 0x0012, 5); err == nil ||
		memory.floppyReadStage != 42 || memory.fdcSeekPending {
		t.Fatal("wrong second dummy seek command unexpectedly accepted")
	}
	if wait, err := memory.WriteWordAt(STDiskController, 0x0013,
		m68k.BusAccess{Clock: 3700, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 43 || memory.floppyRetry2SeekCommand != 0x13 ||
		memory.floppyRetry2SeekStartClock != 3700 || memory.fdcSeekStartClock != 3700 ||
		!memory.fdcSeekPending || memory.fdcStatus != 0xe5 || !memory.fdcStatusTypeI ||
		memory.fdcIRQ || memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("second dummy seek wait/stage/command/clocks/pending/status/type/IRQ/GPIP=%d/%d/%02x/%d/%d/%v/%02x/%v/%v/%02x err=%v",
			wait, memory.floppyReadStage, memory.floppyRetry2SeekCommand,
			memory.floppyRetry2SeekStartClock, memory.fdcSeekStartClock,
			memory.fdcSeekPending, memory.fdcStatus, memory.fdcStatusTypeI,
			memory.fdcIRQ, memory.mfpGPIPIn, err)
	}
	for index := 0; index < 9; index++ {
		if got, err := memory.ReadByte(MFPGPIP, 5); err != nil || got&0x20 == 0 {
			t.Fatalf("second dummy inactive poll %d=%02x err=%v", index, got, err)
		}
	}
	if memory.floppyRetry2InactivePolls != 9 || !memory.fdcSeekPending {
		t.Fatalf("second dummy polls/pending=%d/%v", memory.floppyRetry2InactivePolls,
			memory.fdcSeekPending)
	}
	machine.Clocks = 4428
	machine.advanceClockedDevices()
	if !machine.fdcSeekClockStarted || machine.nextFDCSeekClock != 4429 ||
		memory.floppyReadStage != 43 || !memory.fdcSeekPending {
		t.Fatalf("second dummy early scheduler/next/stage/pending=%v/%d/%d/%v",
			machine.fdcSeekClockStarted, machine.nextFDCSeekClock,
			memory.floppyReadStage, memory.fdcSeekPending)
	}
	machine.Clocks = 4429
	machine.advanceClockedDevices()
	if machine.fdcSeekClockStarted || machine.nextFDCSeekClock != 0 ||
		memory.floppyReadStage != 44 || memory.fdcSeekPending || memory.fdcStatus != 0xe4 ||
		!memory.fdcIRQ || memory.mfpGPIPIn&0x20 != 0 {
		t.Fatalf("second dummy complete scheduler/next/stage/pending/status/IRQ/GPIP=%v/%d/%d/%v/%02x/%v/%02x",
			machine.fdcSeekClockStarted, machine.nextFDCSeekClock, memory.floppyReadStage,
			memory.fdcSeekPending, memory.fdcStatus, memory.fdcIRQ, memory.mfpGPIPIn)
	}
	if got, err := memory.ReadByte(MFPGPIP, 5); err != nil || got&0x20 != 0 ||
		!memory.floppyRetry2IRQObserved {
		t.Fatalf("second dummy IRQ poll=%02x observed=%v err=%v", got,
			memory.floppyRetry2IRQObserved, err)
	}
	if _, err := memory.ReadWord(STDiskController, 5); err == nil {
		t.Fatal("second dummy status before selector unexpectedly accepted")
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0080,
		m68k.BusAccess{Clock: 4500, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.floppyReadStage != 45 {
		t.Fatalf("second dummy status selector wait/stage=%d/%d err=%v", wait,
			memory.floppyReadStage, err)
	}
	if value, wait, err := memory.ReadWordAt(STDiskController,
		m68k.BusAccess{Clock: 4600, FunctionCode: 5}); err != nil || wait != 4 ||
		value != 0x00e4 || memory.floppyReadStage != 46 ||
		memory.floppyRetry2StatusReadClock != 4600 || memory.fdcIRQ ||
		memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("second dummy status value/wait/stage/clock/IRQ/GPIP=%04x/%d/%d/%d/%v/%02x err=%v",
			value, wait, memory.floppyReadStage, memory.floppyRetry2StatusReadClock,
			memory.fdcIRQ, memory.mfpGPIPIn, err)
	}
	if !bytes.Equal(before, memory.ram[0x1004:0x1204]) ||
		memory.floppyReadRetrySeekStartClock != 800 || memory.floppyReadRetryInactivePolls != 9 ||
		!memory.floppyReadRetryIRQObserved || memory.floppyReadRetryStatusReadClock != 1700 {
		t.Fatal("second dummy seek modified DMA buffer or first dummy-seek receipts")
	}
	memory.ColdReset()
	if memory.floppyReadStage != 0 || memory.floppyReadTrack != 0 || memory.floppyReadDrive != -1 ||
		memory.floppyReadTrackWriteClock != 0 || memory.floppyReadSector != 0 ||
		memory.floppyReadDMAAddressStage != 0 || memory.floppyReadDMAResetCount != 0 ||
		memory.floppyReadCommand != 0 || memory.floppyReadCommandClock != 0 ||
		memory.floppyReadTimeoutSelectorClock != 0 || memory.floppyReadForceInterrupt != 0 ||
		memory.floppyReadForceInterruptClock != 0 || memory.floppyReadRetryData != 0 ||
		memory.floppyReadRetrySeekCommand != 0 || memory.floppyReadRetrySeekStartClock != 0 ||
		memory.floppyReadRetryInactivePolls != 0 || memory.floppyReadRetryIRQObserved ||
		memory.floppyReadRetryStatusReadClock != 0 || memory.floppyReadRetryDrivePort != 0 ||
		memory.floppyReadRetryDriveWriteClock != 0 || memory.floppyReadRetrySector != 0 ||
		memory.floppyReadRetryDMAAddressStage != 0 || memory.floppyReadRetryDMAResetCount != 0 ||
		memory.floppyReadRetryCommand != 0 || memory.floppyReadRetryCommandClock != 0 ||
		memory.floppyRetryTimeoutSelectorClock != 0 ||
		memory.floppyRetryForceInterrupt != 0 || memory.floppyRetryForceInterruptClock != 0 ||
		memory.floppyRetry2Data != 0 || memory.floppyRetry2SeekCommand != 0 ||
		memory.floppyRetry2SeekStartClock != 0 || memory.floppyRetry2InactivePolls != 0 ||
		memory.floppyRetry2IRQObserved || memory.floppyRetry2StatusReadClock != 0 {
		t.Fatal("cold reset retained floppy read-lock state")
	}
}

func TestSTFDCForceInterruptInit(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.psgDriveStage = 3
	memory.psgRegisterSelect = 14
	memory.psgRegisters[7], memory.psgRegisters[14] = 0xc0, 5
	if err := memory.WriteWord(STDiskController, 0x00d0, 5); err == nil || memory.fdcInitStage != 0 {
		t.Fatal("FDC command before DMA mode unexpectedly accepted")
	}
	if wait, err := memory.WriteWordAt(0xffff8606, 0x0080,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 6 ||
		memory.dmaMode != 0x0080 || memory.fdcInitStage != 1 || memory.fdcProbeDrive != 0 {
		t.Fatalf("DMA mode wait/err/mode/stage/drive=%d/%v/%04x/%d/%d", wait, err,
			memory.dmaMode, memory.fdcInitStage, memory.fdcProbeDrive)
	}
	if err := memory.WriteWord(STDiskController, 0x000b, 5); err == nil ||
		memory.fdcInitStage != 1 || memory.fdcCommand != 0 {
		t.Fatal("wrong FDC command unexpectedly accepted")
	}
	if wait, err := memory.WriteWordAt(STDiskController, 0x00d0,
		m68k.BusAccess{Clock: 0, FunctionCode: 5}); err != nil || wait != 4 {
		t.Fatalf("FDC command wait/err=%d/%v", wait, err)
	}
	if memory.fdcInitStage != 2 || memory.fdcCommand != 0xd0 || memory.fdcStatus != 0x80 ||
		!memory.fdcStatusTypeI || memory.fdcIRQ || memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("FDC stage/command/status/typeI/IRQ/GPIP=%d/%02x/%02x/%v/%v/%02x",
			memory.fdcInitStage, memory.fdcCommand, memory.fdcStatus,
			memory.fdcStatusTypeI, memory.fdcIRQ, memory.mfpGPIPIn)
	}
	if err := memory.WriteWord(STDMAControl, 0x0080, 1); err == nil {
		t.Fatal("user DMA mode write unexpectedly accepted")
	}
	if err := memory.WriteByte(STDMAControl, 0x80, 5); err == nil {
		t.Fatal("byte DMA mode write unexpectedly accepted")
	}
	if _, err := memory.ReadWord(STDMAControl, 5); err == nil {
		t.Fatal("unmodeled DMA status read unexpectedly accepted")
	}
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if memory.dmaMode != 0 || memory.fdcCommand != 0 || memory.fdcStatus != 0 ||
		memory.fdcStatusTypeI || memory.fdcIRQ || memory.fdcInitStage != 0 || memory.mfpGPIPIn&0x20 == 0 {
		t.Fatalf("FDC reset mode/command/status/typeI/IRQ/stage/GPIP=%04x/%02x/%02x/%v/%v/%d/%02x",
			memory.dmaMode, memory.fdcCommand, memory.fdcStatus, memory.fdcStatusTypeI,
			memory.fdcIRQ, memory.fdcInitStage, memory.mfpGPIPIn)
	}
	if memory.fdcProbeDrive != -1 {
		t.Fatalf("FDC reset probe drive=%d want -1", memory.fdcProbeDrive)
	}
}

func TestSTYM2149ParallelPortStrobeInit(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.psgDriveStage = 6
	memory.psgRegisterSelect = 14
	memory.psgRegisters[7] = 0xc0
	memory.psgRegisters[14] = 3
	memory.fdcInitStage = 14
	memory.acsiStage = 5

	if _, err := memory.ReadByte(PSGRegisterSelect, 5); err == nil ||
		memory.psgDriveStage != 6 || memory.psgRegisters[14] != 3 {
		t.Fatal("strobe read before select unexpectedly accepted or mutated state")
	}
	if err := memory.WriteByte(PSGRegisterSelect, 13, 5); err == nil ||
		memory.psgDriveStage != 6 || memory.psgRegisterSelect != 14 {
		t.Fatal("wrong strobe register unexpectedly accepted or mutated state")
	}
	if wait, err := memory.WriteByteAt(PSGRegisterSelect, 14,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.psgDriveStage != 7 || memory.psgRegisters[14] != 3 {
		t.Fatalf("strobe select wait/err/stage/R14=%d/%v/%d/%02x", wait, err,
			memory.psgDriveStage, memory.psgRegisters[14])
	}
	if err := memory.WriteByte(PSGRegisterData, 0x23, 5); err == nil ||
		memory.psgDriveStage != 7 || memory.psgRegisters[14] != 3 {
		t.Fatal("strobe write before read unexpectedly accepted or mutated state")
	}
	if got, wait, err := memory.ReadByteAt(PSGRegisterSelect,
		m68k.BusAccess{Clock: 0, FunctionCode: 5}); err != nil || got != 3 || wait != 4 ||
		memory.psgDriveStage != 8 || memory.psgRegisters[14] != 3 {
		t.Fatalf("strobe read value/wait/err/stage/R14=%02x/%d/%v/%d/%02x", got, wait,
			err, memory.psgDriveStage, memory.psgRegisters[14])
	}
	if err := memory.WriteByte(PSGRegisterData, 0x22, 5); err == nil ||
		memory.psgDriveStage != 8 || memory.psgRegisters[14] != 3 {
		t.Fatal("wrong strobe data unexpectedly accepted or mutated state")
	}
	if wait, err := memory.WriteByteAt(PSGRegisterData, 0x23,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.psgDriveStage != 9 || memory.psgRegisters[14] != 0x23 ||
		memory.psgRegisters[14]&7 != 3 {
		t.Fatalf("strobe write wait/err/stage/R14=%d/%v/%d/%02x", wait, err,
			memory.psgDriveStage, memory.psgRegisters[14])
	}
	if err := memory.WriteByte(PSGRegisterSelect, 14, 1); err == nil {
		t.Fatal("user strobe register write unexpectedly accepted")
	}
	if err := memory.WriteWord(PSGRegisterSelect, 14, 5); err == nil {
		t.Fatal("word strobe register write unexpectedly accepted")
	}
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if memory.psgDriveStage != 0 || memory.psgRegisterSelect != 0 ||
		memory.psgRegisters != [16]byte{} {
		t.Fatalf("strobe reset stage/select/registers=%d/%02x/%v", memory.psgDriveStage,
			memory.psgRegisterSelect, memory.psgRegisters)
	}
}

func TestFDCDriveOneRestartsProbeWithoutDriveZeroReceipts(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.psgDriveStage = 6
	memory.psgRegisterSelect = 14
	memory.psgRegisters[14] = 3
	memory.fdcInitStage = 14
	memory.fdcProbeDrive = 0
	memory.fdcRestoreStartClock = 11
	memory.fdcRestoreInactivePolls = 9
	memory.fdcRestoreIRQObserved = true
	memory.fdcStatusReadClock = 22
	memory.fdcData = 7
	memory.fdcSeekStartClock = 33
	memory.fdcSeekInactivePolls = 9
	memory.fdcSeekIRQObserved = true
	memory.fdcSeekStatusReadClock = 44

	if wait, err := memory.WriteWordAt(STDMAControl, 0x0080,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 6 {
		t.Fatalf("drive-one restart wait/err=%d/%v", wait, err)
	}
	if memory.fdcProbeDrive != 1 || memory.fdcInitStage != 1 || memory.dmaMode != 0x0080 ||
		memory.fdcRestoreStartClock != 0 || memory.fdcRestoreInactivePolls != 0 ||
		memory.fdcRestoreIRQObserved || memory.fdcStatusReadClock != 0 || memory.fdcData != 0 ||
		memory.fdcSeekStartClock != 0 || memory.fdcSeekInactivePolls != 0 ||
		memory.fdcSeekIRQObserved || memory.fdcSeekStatusReadClock != 0 {
		t.Fatalf("drive-one restart state=%+v", memory)
	}
	if err := memory.WriteWord(STDiskController, 0x00d0, 5); err != nil ||
		memory.fdcInitStage != 2 || memory.fdcCommand != 0xd0 || memory.fdcStatus != 0x80 {
		t.Fatalf("drive-one force err/stage/command/status=%v/%d/%02x/%02x", err,
			memory.fdcInitStage, memory.fdcCommand, memory.fdcStatus)
	}
}

func TestSTFloppyDMAAddressRegisters(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	for _, address := range []uint32{STDMAAddressHigh, STDMAAddressMiddle, STDMAAddressLow} {
		if got, err := memory.ReadByte(address, 5); err != nil || got != 0 {
			t.Fatalf("reset DMA byte %06x=%02x err=%v", address, got, err)
		}
	}
	if err := memory.WriteByte(STDMAAddressLow, 0xff, 5); err != nil || memory.dmaAddress != 0xfe {
		t.Fatalf("low alignment err/address=%v/%06x", err, memory.dmaAddress)
	}
	if err := memory.WriteByte(STDMAAddressMiddle, 0x34, 5); err != nil || memory.dmaAddress != 0x34fe {
		t.Fatalf("middle write err/address=%v/%06x", err, memory.dmaAddress)
	}
	if err := memory.WriteByte(STDMAAddressHigh, 0xff, 5); err != nil || memory.dmaAddress != 0x3f34fe {
		t.Fatalf("high mask err/address=%v/%06x", err, memory.dmaAddress)
	}
	for address, want := range map[uint32]byte{
		STDMAAddressHigh: 0x3f, STDMAAddressMiddle: 0x34, STDMAAddressLow: 0xfe,
	} {
		if got, err := memory.ReadByte(address, 5); err != nil || got != want {
			t.Fatalf("DMA byte %06x=%02x err=%v want %02x", address, got, err, want)
		}
	}
	if err := memory.WriteByte(STDMAAddressLow, 0, 1); err == nil || memory.dmaAddress != 0x3f34fe {
		t.Fatalf("user write err/address=%v/%06x", err, memory.dmaAddress)
	}
	if _, err := memory.ReadByte(STDMAAddressLow, 1); err == nil {
		t.Fatal("user DMA address read unexpectedly accepted")
	}
	if err := memory.WriteWord(0x00ff8608, 0, 5); err == nil || memory.dmaAddress != 0x3f34fe {
		t.Fatalf("word write err/address=%v/%06x", err, memory.dmaAddress)
	}

	lowRipple, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if err := lowRipple.WriteByte(STDMAAddressLow, 0x80, 5); err != nil {
		t.Fatal(err)
	}
	if err := lowRipple.WriteByte(STDMAAddressLow, 0, 5); err != nil || lowRipple.dmaAddress != 0x100 {
		t.Fatalf("low ripple err/address=%v/%06x", err, lowRipple.dmaAddress)
	}

	middleRipple, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if err := middleRipple.WriteByte(STDMAAddressMiddle, 0x80, 5); err != nil {
		t.Fatal(err)
	}
	if err := middleRipple.WriteByte(STDMAAddressMiddle, 0, 5); err != nil || middleRipple.dmaAddress != 0x10000 {
		t.Fatalf("middle ripple err/address=%v/%06x", err, middleRipple.dmaAddress)
	}

	memory.fdcProbeDrive = 1
	memory.fdcInitStage = 14
	memory.fdcCommand = 0x13
	memory.fdcStatus = 0xe4
	memory.dmaMode = 0x0080
	if err := memory.WriteByte(STDMAAddressLow, 4, 5); err != nil || memory.dmaAddressWriteStage != 1 {
		t.Fatalf("boot low err/stage=%v/%d", err, memory.dmaAddressWriteStage)
	}
	if err := memory.WriteByte(STDMAAddressMiddle, 0x10, 5); err != nil || memory.dmaAddressWriteStage != 2 {
		t.Fatalf("boot middle err/stage=%v/%d", err, memory.dmaAddressWriteStage)
	}
	if err := memory.WriteByte(STDMAAddressHigh, 0, 5); err != nil ||
		memory.dmaAddressWriteStage != 3 || memory.dmaAddress != 0x1004 ||
		memory.fdcCommand != 0x13 || memory.fdcStatus != 0xe4 || memory.dmaMode != 0x0080 {
		t.Fatalf("boot high err/stage/address/FDC=%v/%d/%06x/%02x/%02x/%04x", err,
			memory.dmaAddressWriteStage, memory.dmaAddress, memory.fdcCommand,
			memory.fdcStatus, memory.dmaMode)
	}
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if memory.dmaAddress != 0 || memory.dmaAddressWriteStage != 0 {
		t.Fatalf("reset address/stage=%06x/%d", memory.dmaAddress, memory.dmaAddressWriteStage)
	}
}

func TestSTDMAResetToggleAndZeroSectorCount(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.fdcProbeDrive = 1
	memory.fdcInitStage = 14
	memory.dmaAddressWriteStage = 3
	memory.dmaAddress = 0x1004
	memory.dmaMode = 0x0080
	memory.dmaSectorCount = 7
	if err := memory.WriteWord(STDiskController, 0, 5); err == nil ||
		memory.dmaMode != 0x0080 || memory.dmaSectorCount != 7 || memory.dmaInitStage != 0 {
		t.Fatal("sector count write before DMA reset unexpectedly accepted or mutated state")
	}
	if err := memory.WriteWord(STDMAControl, 0x0090, 5); err == nil ||
		memory.dmaMode != 0x0080 || memory.dmaSectorCount != 7 || memory.dmaResetCount != 0 {
		t.Fatal("wrong first DMA mode unexpectedly accepted or mutated state")
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0190,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 6 ||
		memory.dmaMode != 0x0190 || memory.dmaSectorCount != 0 ||
		memory.dmaResetCount != 1 || memory.dmaInitStage != 1 {
		t.Fatalf("first reset wait/err/mode/count/resets/stage=%d/%v/%04x/%d/%d/%d", wait, err,
			memory.dmaMode, memory.dmaSectorCount, memory.dmaResetCount, memory.dmaInitStage)
	}
	memory.dmaSectorCount = 9
	if err := memory.WriteWord(STDMAControl, 0x0080, 5); err == nil ||
		memory.dmaMode != 0x0190 || memory.dmaSectorCount != 9 ||
		memory.dmaResetCount != 1 || memory.dmaInitStage != 1 {
		t.Fatal("wrong second DMA mode unexpectedly accepted or mutated state")
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0090,
		m68k.BusAccess{Clock: 0, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.dmaMode != 0x0090 || memory.dmaSectorCount != 0 ||
		memory.dmaResetCount != 2 || memory.dmaInitStage != 2 {
		t.Fatalf("second reset wait/err/mode/count/resets/stage=%d/%v/%04x/%d/%d/%d", wait, err,
			memory.dmaMode, memory.dmaSectorCount, memory.dmaResetCount, memory.dmaInitStage)
	}
	if err := memory.WriteWord(STDiskController, 1, 5); err == nil ||
		memory.dmaSectorCount != 0 || memory.dmaInitStage != 2 {
		t.Fatal("wrong sector count unexpectedly accepted or mutated state")
	}
	if wait, err := memory.WriteWordAt(STDiskController, 0,
		m68k.BusAccess{Clock: 0, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.dmaSectorCount != 0 || memory.dmaResetCount != 2 || memory.dmaInitStage != 3 {
		t.Fatalf("sector count wait/err/count/resets/stage=%d/%v/%d/%d/%d", wait, err,
			memory.dmaSectorCount, memory.dmaResetCount, memory.dmaInitStage)
	}
	if err := memory.WriteWord(STDMAControl, 0x0190, 1); err == nil {
		t.Fatal("user DMA reset unexpectedly accepted")
	}
	if err := memory.WriteByte(STDMAControl, 0x90, 5); err == nil {
		t.Fatal("byte DMA reset unexpectedly accepted")
	}
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if memory.dmaMode != 0 || memory.dmaSectorCount != 0 || memory.dmaResetCount != 0 ||
		memory.dmaInitStage != 0 {
		t.Fatalf("reset mode/count/resets/stage=%04x/%d/%d/%d", memory.dmaMode,
			memory.dmaSectorCount, memory.dmaResetCount, memory.dmaInitStage)
	}
}

func TestEmptyACSITargetZeroCommandStart(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.fdcProbeDrive = 1
	memory.fdcInitStage = 14
	memory.dmaAddressWriteStage = 3
	memory.dmaInitStage = 3
	memory.dmaMode = 0x0090
	memory.dmaResetCount = 2
	memory.mfpGPIPIn |= 0x20
	if err := memory.WriteWord(STDiskController, 0, 5); err == nil ||
		memory.acsiStage != 0 || memory.acsiTarget != -1 || memory.dmaMode != 0x0090 {
		t.Fatal("ACSI data before mode unexpectedly accepted or mutated state")
	}
	if err := memory.WriteWord(STDMAControl, 0x008a, 5); err == nil ||
		memory.acsiStage != 0 || memory.acsiTarget != -1 || memory.dmaMode != 0x0090 {
		t.Fatal("wrong ACSI first mode unexpectedly accepted or mutated state")
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0088,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 6 ||
		memory.dmaMode != 0x0088 || memory.acsiStage != 1 || memory.acsiTarget != 0 ||
		memory.dmaResetCount != 2 {
		t.Fatalf("ACSI select wait/err/mode/stage/target/resets=%d/%v/%04x/%d/%d/%d", wait,
			err, memory.dmaMode, memory.acsiStage, memory.acsiTarget, memory.dmaResetCount)
	}
	if err := memory.WriteWord(STDiskController, 1, 5); err == nil ||
		memory.acsiStage != 1 || memory.acsiCommand != 0 || memory.mfpGPIPIn&0x20 == 0 {
		t.Fatal("wrong ACSI command unexpectedly accepted or raised IRQ")
	}
	if wait, err := memory.WriteWordAt(STDiskController, 0,
		m68k.BusAccess{Clock: 0, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.acsiStage != 2 || memory.acsiCommand != 0 || memory.mfpGPIPIn&0x20 == 0 ||
		memory.fdcIRQ {
		t.Fatalf("ACSI command wait/err/stage/command/GPIP/IRQ=%d/%v/%d/%02x/%02x/%v", wait,
			err, memory.acsiStage, memory.acsiCommand, memory.mfpGPIPIn, memory.fdcIRQ)
	}
	if err := memory.WriteWord(STDMAControl, 0x0088, 5); err == nil ||
		memory.acsiStage != 2 || memory.dmaMode != 0x0088 {
		t.Fatal("wrong ACSI next mode unexpectedly accepted or mutated state")
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x008a,
		m68k.BusAccess{Clock: 0, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.acsiStage != 3 || memory.dmaMode != 0x008a || memory.mfpGPIPIn&0x20 == 0 ||
		memory.fdcIRQ || memory.dmaResetCount != 2 {
		t.Fatalf("ACSI next wait/err/stage/mode/GPIP/IRQ/resets=%d/%v/%d/%04x/%02x/%v/%d", wait,
			err, memory.acsiStage, memory.dmaMode, memory.mfpGPIPIn, memory.fdcIRQ,
			memory.dmaResetCount)
	}
	if err := memory.WriteWord(STDMAControl, 0x0090, 5); err == nil ||
		memory.acsiStage != 3 || memory.dmaMode != 0x008a {
		t.Fatal("wrong ACSI timeout return unexpectedly accepted or mutated state")
	}
	if wait, err := memory.WriteWordAt(STDMAControl, 0x0080,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 6 ||
		memory.acsiStage != 4 || memory.dmaMode != 0x0080 || memory.dmaInitStage != 0 ||
		memory.dmaResetCount != 0 || memory.fdcInitStage != 14 || memory.fdcProbeDrive != 1 ||
		memory.mfpGPIPIn&0x20 == 0 || memory.fdcIRQ || memory.acsiTimeoutReturnClock != 2 ||
		memory.acsiAttemptMask != 1 || memory.acsiCommandReceipts[0] != 0 ||
		memory.acsiTimeoutReturnClocks[0] != 2 {
		t.Fatalf("ACSI return wait/err/stage/mode/init/resets/FDC/drive/GPIP/IRQ/clock=%d/%v/%d/%04x/%d/%d/%d/%d/%02x/%v/%d",
			wait, err, memory.acsiStage, memory.dmaMode, memory.dmaInitStage,
			memory.dmaResetCount, memory.fdcInitStage, memory.fdcProbeDrive,
			memory.mfpGPIPIn, memory.fdcIRQ, memory.acsiTimeoutReturnClock)
	}
	if err := memory.WriteWord(STDMAControl, 0x0080, 1); err == nil {
		t.Fatal("user ACSI mode unexpectedly accepted")
	}
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if memory.acsiStage != 0 || memory.acsiTarget != -1 || memory.acsiCommand != 0 ||
		memory.acsiAttemptMask != 0 || memory.acsiCommandReceipts != [8]byte{} ||
		memory.acsiTimeoutReturnClock != 0 || memory.acsiTimeoutReturnClocks != [8]uint64{} {
		t.Fatalf("reset ACSI stage/target/command/clock=%d/%d/%02x/%d", memory.acsiStage,
			memory.acsiTarget, memory.acsiCommand, memory.acsiTimeoutReturnClock)
	}
}

func TestEmptyACSITargetScan(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.fdcProbeDrive = 1
	memory.fdcInitStage = 14
	memory.dmaAddressWriteStage = 3
	memory.dmaMode = 0x0080
	memory.acsiStage = 4
	memory.acsiTarget = 0
	memory.acsiAttemptMask = 1
	memory.mfpGPIPIn |= 0x20

	for target := int8(1); target <= 7; target++ {
		for _, write := range []struct {
			address uint32
			value   uint16
		}{
			{STDMAControl, 0x0190},
			{STDMAControl, 0x0090},
			{STDiskController, 0},
			{STDMAControl, 0x0088},
		} {
			if err := memory.WriteWord(write.address, write.value, 5); err != nil {
				t.Fatalf("target %d setup %06x=%04x: %v", target, write.address, write.value, err)
			}
		}
		command := uint16(uint8(target) << 5)
		if err := memory.WriteWord(STDiskController, command|1, 5); err == nil ||
			memory.acsiStage != 1 || memory.acsiAttemptMask != byte((1<<uint8(target))-1) {
			t.Fatalf("target %d wrong command mutated stage/mask=%d/%02x", target,
				memory.acsiStage, memory.acsiAttemptMask)
		}
		if err := memory.WriteWord(STDiskController, command, 5); err != nil {
			t.Fatalf("target %d command %02x: %v", target, command, err)
		}
		if err := memory.WriteWord(STDMAControl, 0x008a, 5); err != nil {
			t.Fatalf("target %d next mode: %v", target, err)
		}
		clock := uint64(target) * 100
		if _, err := memory.WriteWordAt(STDMAControl, 0x0080,
			m68k.BusAccess{Clock: clock, FunctionCode: 5}); err != nil {
			t.Fatalf("target %d timeout return: %v", target, err)
		}
		wantStage := uint8(4)
		if target == 7 {
			wantStage = 5
		}
		if memory.acsiStage != wantStage || memory.acsiTarget != target ||
			memory.acsiCommandReceipts[target] != byte(command) ||
			memory.acsiTimeoutReturnClocks[target] != clock || memory.dmaMode != 0x0080 ||
			memory.dmaInitStage != 0 || memory.mfpGPIPIn&0x20 == 0 || memory.fdcIRQ {
			t.Fatalf("target %d stage/command/clock/mode/init/GPIP/IRQ=%d/%02x/%d/%04x/%d/%02x/%v",
				target, memory.acsiStage, memory.acsiCommandReceipts[target],
				memory.acsiTimeoutReturnClocks[target], memory.dmaMode, memory.dmaInitStage,
				memory.mfpGPIPIn, memory.fdcIRQ)
		}
	}

	wantCommands := [8]byte{0x00, 0x20, 0x40, 0x60, 0x80, 0xa0, 0xc0, 0xe0}
	if memory.acsiAttemptMask != 0xff || memory.acsiCommandReceipts != wantCommands ||
		memory.acsiTimeoutReturnClock != 700 {
		t.Fatalf("final mask/commands/latest clock=%02x/%v/%d", memory.acsiAttemptMask,
			memory.acsiCommandReceipts, memory.acsiTimeoutReturnClock)
	}
	if err := memory.WriteWord(STDMAControl, 0x0088, 5); err == nil || memory.acsiStage != 5 {
		t.Fatal("ninth ACSI target unexpectedly accepted or mutated completed scan")
	}
}

func TestIKBDACIAControlInit(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := memory.ReadByte(IKBDACIAControl, 5); err == nil {
		t.Fatal("unconfigured ACIA status read unexpectedly accepted")
	}
	if err := memory.WriteByte(IKBDACIAControl, 0x96, 5); err == nil {
		t.Fatal("ACIA config before reset unexpectedly accepted")
	}
	for _, value := range []byte{3, 0x96} {
		if wait, err := memory.WriteByteAt(IKBDACIAControl|0xff00_0000, value,
			m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
			t.Fatalf("ACIA control %02x wait=%d err=%v", value, wait, err)
		}
	}
	if got, err := memory.ReadByte(0xfffffc00, 5); err != nil || got != 2 ||
		memory.ikbdACIAControl != 0x96 || !memory.ikbdACIAConfigured {
		t.Fatalf("ACIA control/status/configured=%02x/%02x/%v err=%v",
			memory.ikbdACIAControl, got, memory.ikbdACIAConfigured, err)
	}
	for _, value := range []byte{0, 3, 0x96, 0xff} {
		if err := memory.WriteByte(IKBDACIAControl, value, 5); err == nil {
			t.Fatalf("configured ACIA control %02x unexpectedly accepted", value)
		}
	}
	if _, err := memory.ReadByte(IKBDACIAData, 5); err == nil {
		t.Fatal("ACIA data read unexpectedly accepted")
	}
	if err := memory.WriteByte(IKBDACIAData, 0, 5); err == nil {
		t.Fatal("ACIA data write unexpectedly accepted")
	}
	if _, err := memory.ReadByte(IKBDACIAControl, 1); err == nil {
		t.Fatal("user ACIA status read unexpectedly accepted")
	}
	if err := memory.WriteWord(IKBDACIAControl, 0, 5); err == nil {
		t.Fatal("ACIA word write unexpectedly accepted")
	}
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if memory.ikbdACIAControl != 0 || memory.ikbdACIAStatus != 0 || memory.ikbdACIAConfigured {
		t.Fatalf("ACIA reset control/status/configured=%02x/%02x/%v",
			memory.ikbdACIAControl, memory.ikbdACIAStatus, memory.ikbdACIAConfigured)
	}
}

func TestMIDIACIAControlInit(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteByte(MIDIACIAControl, 0x95, 5); err == nil {
		t.Fatal("MIDI ACIA config before reset unexpectedly accepted")
	}
	for _, value := range []byte{3, 0x95} {
		if wait, err := memory.WriteByteAt(MIDIACIAControl|0xff00_0000, value,
			m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
			t.Fatalf("MIDI ACIA control %02x wait=%d err=%v", value, wait, err)
		}
	}
	if memory.midiACIAControl != 0x95 || memory.midiACIAStatus != 2 || !memory.midiACIAConfigured {
		t.Fatalf("MIDI ACIA control/status/configured=%02x/%02x/%v", memory.midiACIAControl,
			memory.midiACIAStatus, memory.midiACIAConfigured)
	}
	for _, value := range []byte{0, 3, 0x95, 0xff} {
		if err := memory.WriteByte(MIDIACIAControl, value, 5); err == nil {
			t.Fatalf("configured MIDI ACIA control %02x unexpectedly accepted", value)
		}
	}
	if got, err := memory.ReadByte(MIDIACIAControl, 5); err != nil || got != 2 {
		t.Fatalf("configured MIDI ACIA status=%02x err=%v", got, err)
	}
	if err := memory.WriteByte(MIDIACIAData, 0, 5); err == nil {
		t.Fatal("MIDI ACIA data write unexpectedly accepted")
	}
	if err := memory.WriteByte(MIDIACIAControl, 0, 1); err == nil {
		t.Fatal("user MIDI ACIA write unexpectedly accepted")
	}
	if err := memory.WriteWord(MIDIACIAControl, 0, 5); err == nil {
		t.Fatal("MIDI ACIA word write unexpectedly accepted")
	}
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if memory.midiACIAControl != 0 || memory.midiACIAStatus != 0 || memory.midiACIAConfigured {
		t.Fatalf("MIDI ACIA reset control/status/configured=%02x/%02x/%v", memory.midiACIAControl,
			memory.midiACIAStatus, memory.midiACIAConfigured)
	}
}

func TestMFPACIAInterruptChannelEnable(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.midiACIAConfigured = true
	memory.mfpIERB = 0x20
	memory.mfpIMRB = 0x20
	for _, step := range []struct {
		address uint32
		value   byte
		stage   uint8
	}{
		{MFPIERB, 0x20, 1},
		{MFPIPRB, 0xbf, 2},
		{MFPISRB, 0xbf, 3},
		{MFPIERB, 0x60, 4},
		{MFPIMRB, 0x60, 5},
	} {
		if err := memory.WriteByte(step.address, step.value, 5); err != nil {
			t.Fatalf("write %06x=%02x: %v", step.address, step.value, err)
		}
		if memory.mfpACIAEnableStage != step.stage {
			t.Fatalf("write %06x=%02x stage=%d want %d", step.address, step.value,
				memory.mfpACIAEnableStage, step.stage)
		}
	}
	if memory.mfpIERB != 0x60 || memory.mfpIMRB != 0x60 || memory.mfpIPRB != 0 || memory.mfpISRB != 0 {
		t.Fatalf("IERB/IMRB/IPRB/ISRB=%02x/%02x/%02x/%02x", memory.mfpIERB, memory.mfpIMRB,
			memory.mfpIPRB, memory.mfpISRB)
	}

	invalid, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	invalid.midiACIAConfigured = true
	invalid.mfpIERB = 0x20
	invalid.mfpIMRB = 0x20
	if err := invalid.WriteByte(MFPIERB, 0x60, 5); err == nil || invalid.mfpIERB != 0x20 ||
		invalid.mfpACIAEnableStage != 0 {
		t.Fatalf("skipped clear err/state=%v %02x/%d", err, invalid.mfpIERB, invalid.mfpACIAEnableStage)
	}
}

func TestMFPTimerDSystemClockStart(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.mfpACIAEnableStage = 5
	memory.mfpIERB = 0x60
	memory.mfpIMRB = 0x60
	memory.mfpTCDCR = 0x51
	memory.mfpTDDR = 2
	memory.mfpTDMain = 2
	memory.mfpTimerDStart = true
	for _, step := range []struct {
		address uint32
		value   byte
		stage   uint8
	}{
		{MFPIERB, 0x60, 1},
		{MFPIPRB, 0xef, 2},
		{MFPISRB, 0xef, 3},
		{MFPTCDCR, 0x50, 4},
		{MFPTDDR, 0, 5},
		{MFPIERB, 0x70, 6},
		{MFPIMRB, 0x70, 7},
		{MFPTCDCR, 0x52, 8},
	} {
		if err := memory.WriteByte(step.address, step.value, 5); err != nil {
			t.Fatalf("write %06x=%02x: %v", step.address, step.value, err)
		}
		if memory.mfpTimerDSystemStage != step.stage {
			t.Fatalf("write %06x=%02x stage=%d want %d", step.address, step.value,
				memory.mfpTimerDSystemStage, step.stage)
		}
	}
	if memory.mfpIERB != 0x70 || memory.mfpIMRB != 0x70 || memory.mfpTCDCR != 0x52 ||
		memory.mfpTDDR != 0 || memory.mfpTDMain != 0 || !memory.mfpTimerDStart {
		t.Fatalf("IERB/IMRB/TCDCR/TDDR/main/start=%02x/%02x/%02x/%02x/%02x/%v",
			memory.mfpIERB, memory.mfpIMRB, memory.mfpTCDCR, memory.mfpTDDR,
			memory.mfpTDMain, memory.mfpTimerDStart)
	}

	invalid, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	invalid.mfpACIAEnableStage = 5
	invalid.mfpIERB = 0x60
	invalid.mfpIMRB = 0x60
	invalid.mfpTCDCR = 0x51
	invalid.mfpTDDR = 2
	invalid.mfpTDMain = 2
	if err := invalid.WriteByte(MFPTCDCR, 0x50, 5); err == nil || invalid.mfpTCDCR != 0x51 ||
		invalid.mfpTimerDSystemStage != 0 {
		t.Fatalf("skipped clears err/control/stage=%v %02x/%d", err, invalid.mfpTCDCR,
			invalid.mfpTimerDSystemStage)
	}
}

func TestMFPBAcknowledgeUsesVectorAndSoftwareEOI(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.mfpVR = 0x48
	memory.mfpIPRB = 0x10
	if got := memory.mfpVector(4); got != 68 {
		t.Fatalf("vector=%d want 68", got)
	}
	memory.acknowledgeMFPB(4)
	if memory.mfpIPRB != 0 || memory.mfpISRB != 0x10 {
		t.Fatalf("pending/in-service=%02x/%02x want 00/10", memory.mfpIPRB, memory.mfpISRB)
	}
	if err := memory.WriteByte(MFPISRB, 0xef, 5); err != nil {
		t.Fatal(err)
	}
	if memory.mfpISRB != 0 {
		t.Fatalf("in-service=%02x want 00", memory.mfpISRB)
	}
}

func TestMFPTimerDStopAndChannelClear(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.mfpTimerDSystemStage = 8
	memory.mfpIERB, memory.mfpIMRB = 0x70, 0x70
	memory.mfpTCDCR, memory.mfpTimerDStart = 0x52, true
	for _, step := range []struct {
		address uint32
		value   byte
		stage   uint8
	}{
		{MFPIERB, 0x60, 1},
		{MFPIMRB, 0x60, 2},
		{MFPTCDCR, 0x50, 3},
		{MFPIMRB, 0x60, 4},
		{MFPIERB, 0x60, 5},
		{MFPIPRB, 0xef, 6},
		{MFPISRB, 0xef, 7},
	} {
		if err := memory.WriteByte(step.address, step.value, 5); err != nil {
			t.Fatalf("write %06x=%02x: %v", step.address, step.value, err)
		}
		if memory.mfpTimerDStopStage != step.stage {
			t.Fatalf("write %06x=%02x stage=%d want %d", step.address, step.value,
				memory.mfpTimerDStopStage, step.stage)
		}
	}
	if memory.mfpIERB != 0x60 || memory.mfpIMRB != 0x60 || memory.mfpTCDCR != 0x50 ||
		memory.mfpTimerDStart {
		t.Fatalf("IERB/IMRB/TCDCR/start=%02x/%02x/%02x/%v",
			memory.mfpIERB, memory.mfpIMRB, memory.mfpTCDCR, memory.mfpTimerDStart)
	}
}

func TestIKBDACIAFirstTransmitData(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []byte{3, 0x96} {
		if err := memory.WriteByte(IKBDACIAControl, value, 5); err != nil {
			t.Fatal(err)
		}
	}
	if err := memory.WriteByte(IKBDACIAData, 0x80, 5); err != nil {
		t.Fatal(err)
	}
	if memory.ikbdACIATDR != 0x80 || !memory.ikbdACIATXPending || memory.ikbdACIAStatus != 0 {
		t.Fatalf("TDR/pending/status=%02x/%v/%02x", memory.ikbdACIATDR,
			memory.ikbdACIATXPending, memory.ikbdACIAStatus)
	}
	if err := memory.WriteByte(IKBDACIAData, 0x80, 5); err == nil {
		t.Fatal("pending ACIA data write unexpectedly accepted")
	}
	memory.advanceIKBDACIAClock()
	if memory.ikbdACIATXPending || memory.ikbdACIAStatus != 2 || memory.ikbdACIATXShiftTicks != 10 {
		t.Fatalf("advanced pending/status/shift=%v/%02x/%d", memory.ikbdACIATXPending,
			memory.ikbdACIAStatus, memory.ikbdACIATXShiftTicks)
	}
	if err := memory.WriteByte(IKBDACIAData, 1, 5); err != nil {
		t.Fatal(err)
	}
	for tick := 9; tick > 0; tick-- {
		memory.advanceIKBDACIAClock()
		if !memory.ikbdACIATXPending || memory.ikbdACIAStatus != 0 ||
			memory.ikbdACIATXShiftTicks != uint8(tick) {
			t.Fatalf("tick %d pending/status/shift=%v/%02x/%d", tick, memory.ikbdACIATXPending,
				memory.ikbdACIAStatus, memory.ikbdACIATXShiftTicks)
		}
	}
	memory.advanceIKBDACIAClock()
	if memory.ikbdACIATXPending || memory.ikbdACIAStatus != 2 || memory.ikbdACIATXShiftTicks != 10 ||
		memory.ikbdACIATDR != 1 {
		t.Fatalf("second transfer TDR/pending/status/shift=%02x/%v/%02x/%d", memory.ikbdACIATDR,
			memory.ikbdACIATXPending, memory.ikbdACIAStatus, memory.ikbdACIATXShiftTicks)
	}
	for tick := 9; tick > 0; tick-- {
		memory.advanceIKBDACIAClock()
	}
	if memory.ikbdResetCommandDone {
		t.Fatal("IKBD reset command completed before stop tick")
	}
	memory.advanceIKBDACIAClock()
	if !memory.ikbdResetCommandDone || !memory.ikbdResetCommandHandled || memory.ikbdACIATXShiftTicks != 0 {
		t.Fatalf("reset command done/handled/shift=%v/%v/%d", memory.ikbdResetCommandDone,
			memory.ikbdResetCommandHandled, memory.ikbdACIATXShiftTicks)
	}
	memory.deliverIKBDResetResponse()
	if memory.ikbdACIARDR != 0xf1 || memory.ikbdACIAStatus != 0x83 {
		t.Fatalf("RDR/status=%02x/%02x", memory.ikbdACIARDR, memory.ikbdACIAStatus)
	}
	if got, err := memory.ReadByte(IKBDACIAData, 5); err != nil || got != 0xf1 ||
		memory.ikbdACIAStatus != 2 || memory.ikbdStaleRDRReads != 1 || memory.mfpGPIPIn&0x10 == 0 {
		t.Fatalf("RDR read=%02x status=%02x err=%v", got, memory.ikbdACIAStatus, err)
	}
	if got, err := memory.ReadByte(IKBDACIAData, 5); err != nil || got != 0xf1 ||
		memory.ikbdACIAStatus != 2 || memory.ikbdStaleRDRReads != 0 {
		t.Fatalf("stale RDR read=%02x status/allowance=%02x/%d err=%v", got,
			memory.ikbdACIAStatus, memory.ikbdStaleRDRReads, err)
	}
	if _, err := memory.ReadByte(IKBDACIAData, 5); err == nil {
		t.Fatal("exhausted stale RDR read unexpectedly accepted")
	}
}

func TestIKBDACIAClockRequestTransmit(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.ikbdACIAControl = 0x96
	memory.ikbdACIAConfigured = true
	memory.ikbdACIAStatus = 2
	memory.ikbdACIATDR = 1
	memory.ikbdResetCommandDone = true
	memory.ikbdResetCommandHandled = true
	memory.ikbdResetResponseRead = true
	memory.psgDriveStage = 9
	memory.acsiStage = 5

	if err := memory.WriteByte(IKBDACIAData, 0x1b, 5); err == nil ||
		memory.ikbdACIATDR != 1 || memory.ikbdACIAStatus != 2 || memory.ikbdACIATXPending {
		t.Fatal("wrong clock request unexpectedly accepted or mutated ACIA")
	}
	if wait, err := memory.WriteByteAt(IKBDACIAData, 0x1c,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 ||
		memory.ikbdACIATDR != 0x1c || memory.ikbdACIAStatus != 0 ||
		!memory.ikbdACIATXPending || memory.ikbdClockRequestDone ||
		memory.ikbdClockRequestHandled {
		t.Fatalf("clock request wait/err/TDR/status/pending/done/handled=%d/%v/%02x/%02x/%v/%v/%v",
			wait, err, memory.ikbdACIATDR, memory.ikbdACIAStatus,
			memory.ikbdACIATXPending, memory.ikbdClockRequestDone,
			memory.ikbdClockRequestHandled)
	}
	memory.advanceIKBDACIAClock()
	if memory.ikbdACIATXPending || memory.ikbdACIAStatus != 2 ||
		memory.ikbdACIATXShiftTicks != 10 {
		t.Fatalf("clock request shift pending/status/ticks=%v/%02x/%d",
			memory.ikbdACIATXPending, memory.ikbdACIAStatus, memory.ikbdACIATXShiftTicks)
	}
	for tick := 9; tick > 0; tick-- {
		memory.advanceIKBDACIAClock()
		if memory.ikbdClockRequestDone || memory.ikbdACIATXShiftTicks != uint8(tick) {
			t.Fatalf("clock request tick %d done/remaining=%v/%d", tick,
				memory.ikbdClockRequestDone, memory.ikbdACIATXShiftTicks)
		}
	}
	memory.advanceIKBDACIAClock()
	if !memory.ikbdClockRequestDone || !memory.ikbdClockRequestHandled ||
		memory.ikbdACIATXShiftTicks != 0 || memory.ikbdACIAStatus != 2 {
		t.Fatalf("clock request completion done/handled/ticks/status=%v/%v/%d/%02x",
			memory.ikbdClockRequestDone, memory.ikbdClockRequestHandled,
			memory.ikbdACIATXShiftTicks, memory.ikbdACIAStatus)
	}
	memory.advanceIKBDACIAClock()
	if !memory.ikbdClockRequestDone || !memory.ikbdClockRequestHandled {
		t.Fatal("clock request receipt changed after completion")
	}
	if err := memory.WriteByte(IKBDACIAData, 0x1c, 5); err == nil {
		t.Fatal("duplicate clock request unexpectedly accepted")
	}
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if memory.ikbdClockRequestDone || memory.ikbdClockRequestHandled {
		t.Fatalf("clock request reset done/handled=%v/%v", memory.ikbdClockRequestDone,
			memory.ikbdClockRequestHandled)
	}
}

func TestIKBDACIAClockResponseUsesMFPChannelSix(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.ikbdACIAConfigured = true
	memory.ikbdACIAStatus = 2
	memory.ikbdClockRequestHandled = true
	memory.mfpGPIPIn = 0xa1
	memory.mfpIERB = 0x60
	memory.mfpIMRB = 0x60
	memory.mfpVR = 0x48

	for index, want := range ikbdClockResponse {
		if !memory.deliverIKBDClockResponse(1, uint8(index), want) {
			t.Fatalf("response byte %d was not delivered", index)
		}
		if memory.ikbdACIAStatus != 0x83 || memory.mfpGPIPIn&0x10 != 0 || memory.mfpIPRB&0x40 == 0 {
			t.Fatalf("byte %d status/GPIP/IPRB=%02x/%02x/%02x", index,
				memory.ikbdACIAStatus, memory.mfpGPIPIn, memory.mfpIPRB)
		}
		if memory.deliverIKBDClockResponse(1, uint8(index+1), 0xee) {
			t.Fatalf("byte %d overwrote unread RDR", index+1)
		}
		memory.acknowledgeMFPB(6)
		if memory.mfpIPRB&0x40 != 0 || memory.mfpISRB&0x40 == 0 {
			t.Fatalf("byte %d acknowledge IPRB/ISRB=%02x/%02x", index, memory.mfpIPRB, memory.mfpISRB)
		}
		got, wait, err := memory.ReadByteAt(IKBDACIAData,
			m68k.BusAccess{Clock: uint64(100 + index), FunctionCode: 5})
		if err != nil || wait != 4 || got != want || memory.ikbdACIAStatus != 2 ||
			memory.mfpGPIPIn&0x10 == 0 || memory.ikbdStaleRDRReads != 0 {
			t.Fatalf("byte %d read/wait/status/GPIP/stale=%02x/%d/%02x/%02x/%d err=%v",
				index, got, wait, memory.ikbdACIAStatus, memory.mfpGPIPIn,
				memory.ikbdStaleRDRReads, err)
		}
		if err := memory.WriteByte(MFPISRB, 0xbf, 5); err != nil || memory.mfpISRB&0x40 != 0 {
			t.Fatalf("byte %d end-of-interrupt ISRB=%02x err=%v", index, memory.mfpISRB, err)
		}
	}
	if !memory.ikbdClockResponseComplete || memory.ikbdClockResponseActive ||
		memory.ikbdClockResponseReads != ikbdClockResponse ||
		memory.ikbdClockResponseReadClocks != [7]uint64{100, 101, 102, 103, 104, 105, 106} {
		t.Fatalf("response completion/active/reads/clocks=%v/%v/%v/%v",
			memory.ikbdClockResponseComplete, memory.ikbdClockResponseActive,
			memory.ikbdClockResponseReads, memory.ikbdClockResponseReadClocks)
	}
}

func TestIKBDACIASetClockBuffersSevenFrames(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.ikbdACIAConfigured = true
	memory.ikbdACIAStatus = 2
	memory.ikbdClockResponseComplete = true
	memory.ikbdClockRequestHandled = true
	clock := uint64(1000)
	if err := memory.WriteByte(IKBDACIAData, 0x1a, 5); err == nil {
		t.Fatal("wrong set-clock command unexpectedly accepted")
	}
	for index, value := range ikbdSetClockPacket {
		if wait, err := memory.WriteByteAt(IKBDACIAData, value,
			m68k.BusAccess{Clock: clock, FunctionCode: 5}); err != nil || wait != 4 {
			t.Fatalf("write %d value=%02x wait=%d err=%v", index, value, wait, err)
		}
		if memory.ikbdSetClockWriteCount != uint8(index+1) || memory.ikbdACIAStatus&2 != 0 {
			t.Fatalf("write %d count/status=%d/%02x", index,
				memory.ikbdSetClockWriteCount, memory.ikbdACIAStatus)
		}
		if index == 0 {
			clock += 1024
			memory.advanceIKBDACIAClock(clock)
			if memory.ikbdACIATXShift != value || memory.ikbdACIATXShiftTicks != 10 ||
				memory.ikbdACIAStatus&2 == 0 {
				t.Fatalf("first shift byte/ticks/status=%02x/%d/%02x",
					memory.ikbdACIATXShift, memory.ikbdACIATXShiftTicks, memory.ikbdACIAStatus)
			}
			continue
		}
		for tick := 0; tick < 10; tick++ {
			clock += 1024
			memory.advanceIKBDACIAClock(clock)
		}
	}
	for tick := 0; tick < 10; tick++ {
		clock += 1024
		memory.advanceIKBDACIAClock(clock)
	}
	if !memory.ikbdSetClockComplete || memory.ikbdSetClockCompleteCount != 7 ||
		memory.ikbdSetClockWrites != ikbdSetClockPacket ||
		memory.ikbdSetClockCompletions != ikbdSetClockPacket ||
		memory.ikbdSetClockCompletionClocks != [7]uint64{12264, 22504, 32744, 42984, 53224, 63464, 73704} {
		t.Fatalf("set-clock complete/count/writes/completions/clocks=%v/%d/%v/%v/%v",
			memory.ikbdSetClockComplete, memory.ikbdSetClockCompleteCount,
			memory.ikbdSetClockWrites, memory.ikbdSetClockCompletions,
			memory.ikbdSetClockCompletionClocks)
	}
	if err := memory.WriteByte(IKBDACIAData, 0, 5); err == nil {
		t.Fatal("eighth set-clock byte unexpectedly accepted")
	}
	memory.ColdReset()
	if memory.ikbdACIATXShift != 0 || memory.ikbdSetClockWriteCount != 0 ||
		memory.ikbdSetClockCompleteCount != 0 || memory.ikbdSetClockComplete ||
		memory.ikbdSetClockWrites != [7]byte{} || memory.ikbdSetClockCompletions != [7]byte{} ||
		memory.ikbdSetClockCompletionClocks != [7]uint64{} {
		t.Fatal("cold reset retained set-clock state")
	}
}

func TestIKBDACIAReadbackRequestBuffersAcrossPackets(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.ikbdACIAConfigured = true
	memory.ikbdACIAStatus = 2
	memory.ikbdClockResponseComplete = true
	memory.ikbdClockRequestHandled = true
	memory.ikbdSetClockWrites = ikbdSetClockPacket
	memory.ikbdSetClockWriteCount = 7
	memory.ikbdSetClockCompletions = ikbdSetClockPacket
	memory.ikbdSetClockCompleteCount = 6
	memory.ikbdACIATXShift = 0
	memory.ikbdACIATXShiftTicks = 1

	if wait, err := memory.WriteByteAt(IKBDACIAData, 0x1c,
		m68k.BusAccess{Clock: 1000, FunctionCode: 5}); err != nil || wait != 4 ||
		!memory.ikbdClockReadbackRequestWritten || !memory.ikbdACIATXPending ||
		memory.ikbdACIAStatus&2 != 0 {
		t.Fatalf("readback buffer wait/written/pending/status=%d/%v/%v/%02x err=%v",
			wait, memory.ikbdClockReadbackRequestWritten, memory.ikbdACIATXPending,
			memory.ikbdACIAStatus, err)
	}
	memory.advanceIKBDACIAClock(2000)
	if !memory.ikbdSetClockComplete || memory.ikbdSetClockCompleteCount != 7 ||
		memory.ikbdSetClockCompletionClocks[6] != 2000 || memory.ikbdACIATXShift != 0x1c ||
		memory.ikbdACIATXShiftTicks != 10 || memory.ikbdACIATXPending || memory.ikbdACIAStatus&2 == 0 {
		t.Fatalf("packet boundary set-complete/count/clock/shift/ticks/pending/status=%v/%d/%d/%02x/%d/%v/%02x",
			memory.ikbdSetClockComplete, memory.ikbdSetClockCompleteCount,
			memory.ikbdSetClockCompletionClocks[6], memory.ikbdACIATXShift,
			memory.ikbdACIATXShiftTicks, memory.ikbdACIATXPending, memory.ikbdACIAStatus)
	}
	for tick := 1; tick <= 10; tick++ {
		memory.advanceIKBDACIAClock(uint64(2000 + tick*1024))
	}
	if !memory.ikbdClockReadbackRequestDone || !memory.ikbdClockReadbackRequestHandled ||
		memory.ikbdACIATXShiftTicks != 0 {
		t.Fatalf("readback request done/handled/ticks=%v/%v/%d",
			memory.ikbdClockReadbackRequestDone, memory.ikbdClockReadbackRequestHandled,
			memory.ikbdACIATXShiftTicks)
	}
	if err := memory.WriteByte(IKBDACIAData, 0x1c, 5); err == nil {
		t.Fatal("duplicate readback request unexpectedly accepted")
	}
}

func TestIKBDACIAClockReadbackResponseHasIndependentReceipts(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.ikbdACIAConfigured = true
	memory.ikbdACIAStatus = 2
	memory.ikbdClockReadbackRequestHandled = true
	memory.mfpGPIPIn = 0xa1
	for index, want := range ikbdClockReadback {
		if !memory.deliverIKBDClockResponse(2, uint8(index), want) {
			t.Fatalf("readback byte %d was not delivered", index)
		}
		memory.recordIKBDClockResponseDeliveryClock(2, uint8(index), uint64(200+index))
		got, wait, err := memory.ReadByteAt(IKBDACIAData,
			m68k.BusAccess{Clock: uint64(300 + index), FunctionCode: 5})
		if err != nil || wait != 4 || got != want || memory.ikbdACIAStatus != 2 ||
			memory.mfpGPIPIn&0x10 == 0 {
			t.Fatalf("readback byte %d got/wait/status/GPIP=%02x/%d/%02x/%02x err=%v",
				index, got, wait, memory.ikbdACIAStatus, memory.mfpGPIPIn, err)
		}
	}
	if !memory.ikbdClockReadbackComplete || memory.ikbdClockReadbackActive ||
		memory.ikbdClockReadbackReads != ikbdClockReadback ||
		memory.ikbdClockReadbackDeliveryClocks != [7]uint64{200, 201, 202, 203, 204, 205, 206} ||
		memory.ikbdClockReadbackReadClocks != [7]uint64{300, 301, 302, 303, 304, 305, 306} ||
		memory.ikbdClockResponseDelivered != 0 || memory.ikbdClockResponseReadCount != 0 {
		t.Fatalf("readback complete/active/reads/delivery/read/first-round=%v/%v/%v/%v/%v/%d/%d",
			memory.ikbdClockReadbackComplete, memory.ikbdClockReadbackActive,
			memory.ikbdClockReadbackReads, memory.ikbdClockReadbackDeliveryClocks,
			memory.ikbdClockReadbackReadClocks, memory.ikbdClockResponseDelivered,
			memory.ikbdClockResponseReadCount)
	}
	memory.ColdReset()
	if memory.ikbdClockReadbackRequestWritten || memory.ikbdClockReadbackRequestDone ||
		memory.ikbdClockReadbackRequestHandled || memory.ikbdClockReadbackActive ||
		memory.ikbdClockReadbackDelivered != 0 || memory.ikbdClockReadbackReadCount != 0 ||
		memory.ikbdClockReadbackReads != [7]byte{} ||
		memory.ikbdClockReadbackDeliveryClocks != [7]uint64{} ||
		memory.ikbdClockReadbackReadClocks != [7]uint64{} || memory.ikbdClockReadbackComplete {
		t.Fatal("cold reset retained clock-readback state")
	}
}

func TestIKBDACIARecurringClockPollsResetPerCycle(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	memory.ikbdACIAConfigured = true
	memory.ikbdACIAStatus = 2
	memory.ikbdClockRequestHandled = true
	memory.ikbdSetClockComplete = true
	memory.ikbdClockReadbackRequestHandled = true
	memory.ikbdClockReadbackComplete = true
	memory.flopVBLMediaComplete = true

	for cycle := uint32(1); cycle <= 2; cycle++ {
		if err := memory.WriteByte(IKBDACIAData, 0x1b, 5); err == nil {
			t.Fatalf("cycle %d wrong command unexpectedly accepted", cycle)
		}
		if err := memory.WriteByte(IKBDACIAData, 0x1c, 5); err != nil ||
			!memory.ikbdClockPollRequestWritten || !memory.ikbdACIATXPending {
			t.Fatalf("cycle %d request written/pending=%v/%v err=%v", cycle,
				memory.ikbdClockPollRequestWritten, memory.ikbdACIATXPending, err)
		}
		if err := memory.WriteByte(IKBDACIAData, 0x1c, 5); err == nil {
			t.Fatalf("cycle %d overlapping request unexpectedly accepted", cycle)
		}
		memory.advanceIKBDACIAClock(1024)
		for tick := 1; tick <= 10; tick++ {
			memory.advanceIKBDACIAClock(uint64((tick + 1) * 1024))
		}
		if memory.ikbdClockPollRequestWritten ||
			memory.ikbdClockPollRequestCount != cycle {
			t.Fatalf("cycle %d request written/completions=%v/%d", cycle,
				memory.ikbdClockPollRequestWritten, memory.ikbdClockPollRequestCount)
		}
		for index, want := range ikbdClockReadback {
			if !memory.deliverIKBDClockResponse(3, uint8(index), want) {
				t.Fatalf("cycle %d response %d was not delivered", cycle, index)
			}
			memory.recordIKBDClockResponseDeliveryClock(3, uint8(index), uint64(cycle*1000)+uint64(index))
			got, _, err := memory.ReadByteAt(IKBDACIAData,
				m68k.BusAccess{Clock: uint64(cycle*2000) + uint64(index), FunctionCode: 5})
			if err != nil || got != want {
				t.Fatalf("cycle %d response %d=%02x err=%v want %02x", cycle, index, got, err, want)
			}
		}
		if memory.ikbdClockPollCompleteCount != cycle ||
			memory.ikbdClockPollResponseActive ||
			memory.ikbdClockPollResponseReads != ikbdClockReadback {
			t.Fatalf("cycle %d complete/active/reads=%d/%v/%v", cycle,
				memory.ikbdClockPollCompleteCount,
				memory.ikbdClockPollResponseActive, memory.ikbdClockPollResponseReads)
		}
	}
	memory.ColdReset()
	if memory.ikbdClockPollRequestWritten || memory.ikbdClockPollRequestCount != 0 ||
		memory.ikbdClockPollResponseActive || memory.ikbdClockPollResponseDelivered != 0 ||
		memory.ikbdClockPollResponseReadCount != 0 || memory.ikbdClockPollResponseReads != [7]byte{} ||
		memory.ikbdClockPollDeliveryClocks != [7]uint64{} || memory.ikbdClockPollReadClocks != [7]uint64{} ||
		memory.ikbdClockPollCompleteCount != 0 {
		t.Fatal("cold reset retained recurring clock-poll state")
	}
}

func TestMFPTimerDataStoppedLoad(t *testing.T) {
	for _, test := range []struct {
		name     string
		address  uint32
		activate func(*Memory)
		data     func(*Memory) byte
		main     func(*Memory) byte
	}{
		{name: "TADR", address: MFPTADR, activate: func(m *Memory) { m.mfpTACR = 1 },
			data: func(m *Memory) byte { return m.mfpTADR }, main: func(m *Memory) byte { return m.mfpTAMain }},
		{name: "TBDR", address: MFPTBDR, activate: func(m *Memory) { m.mfpTBCR = 1 },
			data: func(m *Memory) byte { return m.mfpTBDR }, main: func(m *Memory) byte { return m.mfpTBMain }},
		{name: "TCDR", address: MFPTCDR, activate: func(m *Memory) { m.mfpTCDCR = 0x10 },
			data: func(m *Memory) byte { return m.mfpTCDR }, main: func(m *Memory) byte { return m.mfpTCMain }},
		{name: "TDDR", address: MFPTDDR, activate: func(m *Memory) { m.mfpTCDCR = 0x01 },
			data: func(m *Memory) byte { return m.mfpTDDR }, main: func(m *Memory) byte { return m.mfpTDMain }},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory, err := NewMemory(RAM1M, testROM())
			if err != nil {
				t.Fatal(err)
			}
			for _, value := range []byte{0xa5, 0x3c, 0x00} {
				if wait, err := memory.WriteByteAt(test.address|0xff000000, value,
					m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
					t.Fatalf("timed %s write %02x wait=%d err=%v", test.name, value, wait, err)
				}
				if got, err := memory.ReadByte(test.address, 5); err != nil || got != value ||
					test.data(memory) != value || test.main(memory) != value {
					t.Fatalf("%s load got=%02x/%02x/%02x err=%v want %02x",
						test.name, got, test.data(memory), test.main(memory), err, value)
				}
			}
			test.activate(memory)
			beforeData, beforeMain := test.data(memory), test.main(memory)
			if err := memory.WriteByte(test.address, 0x5a, 5); err == nil {
				t.Fatalf("active %s write unexpectedly succeeded", test.name)
			}
			if _, err := memory.ReadByte(test.address, 5); err == nil {
				t.Fatalf("active %s read unexpectedly succeeded", test.name)
			}
			if test.data(memory) != beforeData || test.main(memory) != beforeMain {
				t.Fatalf("failed active %s access changed data/main", test.name)
			}
			if _, err := memory.ReadByte(test.address, 1); err == nil {
				t.Fatalf("user %s read unexpectedly succeeded", test.name)
			}
			if err := memory.WriteByte(test.address, 0, 1); err == nil {
				t.Fatalf("user %s write unexpectedly succeeded", test.name)
			}
			if _, err := memory.ReadWord(test.address, 5); err == nil {
				t.Fatalf("odd %s word read unexpectedly succeeded", test.name)
			}
			if err := memory.M68KReset(); err != nil {
				t.Fatal(err)
			}
			if got, err := memory.ReadByte(test.address, 5); err != nil || got != 0 ||
				test.data(memory) != 0 || test.main(memory) != 0 {
				t.Fatalf("%s after reset=%02x/%02x/%02x err=%v",
					test.name, got, test.data(memory), test.main(memory), err)
			}
		})
	}
	if memory, err := NewMemory(RAM1M, testROM()); err != nil {
		t.Fatal(err)
	} else if _, err := memory.ReadByte(MFPSCR, 5); err != nil {
		t.Fatalf("neighboring SCR reset read: %v", err)
	}
}

func TestMFPUSARTResetZeroWrites(t *testing.T) {
	tests := []struct {
		name    string
		address uint32
		field   func(*Memory) byte
		set     func(*Memory, byte)
	}{
		{name: "SCR", address: MFPSCR, field: func(m *Memory) byte { return m.mfpSCR }, set: func(m *Memory, v byte) { m.mfpSCR = v }},
		{name: "UCR", address: MFPUCR, field: func(m *Memory) byte { return m.mfpUCR }, set: func(m *Memory, v byte) { m.mfpUCR = v }},
		{name: "RSR", address: MFPRSR, field: func(m *Memory) byte { return m.mfpRSR }, set: func(m *Memory, v byte) { m.mfpRSR = v }},
		{name: "TSR", address: MFPTSR, field: func(m *Memory) byte { return m.mfpTSR }, set: func(m *Memory, v byte) { m.mfpTSR, m.mfpTSRSet = v, true }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			memory, err := NewMemory(RAM1M, testROM())
			if err != nil {
				t.Fatal(err)
			}
			if test.address == MFPTSR {
				if _, err := memory.ReadByte(test.address, 5); err == nil {
					t.Fatal("hardware-reset TSR unexpectedly treated as known")
				}
			} else if got, err := memory.ReadByte(test.address, 5); err != nil || got != 0 {
				t.Fatalf("cold %s=%02x/%v want 00", test.name, got, err)
			}
			if wait, err := memory.WriteByteAt(test.address|0xff000000, 0,
				m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
				t.Fatalf("timed %s zero write wait=%d err=%v", test.name, wait, err)
			}
			want := byte(0)
			if test.address == MFPTSR {
				want = 0x80
			}
			if got, err := memory.ReadByte(test.address, 5); err != nil || got != want {
				t.Fatalf("initialized %s=%02x/%v want %02x", test.name, got, err, want)
			}
			test.set(memory, 0x5a)
			if err := memory.WriteByte(test.address, 0, 5); err == nil {
				t.Fatalf("non-reset %s state unexpectedly accepted", test.name)
			}
			if test.field(memory) != 0x5a {
				t.Fatalf("failed %s write changed state", test.name)
			}
			test.set(memory, 0)
			if err := memory.WriteByte(test.address, 1, 5); err == nil {
				t.Fatalf("nonzero %s write unexpectedly accepted", test.name)
			}
			if _, err := memory.ReadByte(test.address, 1); err == nil {
				t.Fatalf("user %s read unexpectedly succeeded", test.name)
			}
			if err := memory.WriteByte(test.address, 0, 1); err == nil {
				t.Fatalf("user %s write unexpectedly succeeded", test.name)
			}
			if _, err := memory.ReadWord(test.address, 5); err == nil {
				t.Fatalf("odd %s word read unexpectedly succeeded", test.name)
			}
		})
	}
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := memory.ReadByte(MFPUDR, 5); err == nil {
		t.Fatal("UDR unexpectedly mapped")
	}
	memory.mfpSCR, memory.mfpUCR, memory.mfpRSR = 1, 2, 3
	memory.mfpTSR, memory.mfpTSRSet = 4, true
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	for _, address := range []uint32{MFPSCR, MFPUCR, MFPRSR} {
		if got, err := memory.ReadByte(address, 5); err != nil || got != 0 {
			t.Fatalf("reset USART register %06x=%02x/%v want 00", address, got, err)
		}
	}
	if _, err := memory.ReadByte(MFPTSR, 5); err == nil {
		t.Fatal("hardware-reset TSR unexpectedly remained known")
	}
}

func TestMFPAERResetStateZeroWrite(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if got, err := memory.ReadByte(MFPAER, 5); err != nil || got != 0 {
		t.Fatalf("cold AER=%02x/%v", got, err)
	}
	if wait, err := memory.WriteByteAt(0xfffffa03, 0, m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 4 {
		t.Fatalf("timed AER zero write wait=%d err=%v", wait, err)
	}
	if err := memory.WriteByte(MFPAER, 1, 5); err == nil {
		t.Fatal("nonzero AER write unexpectedly accepted")
	} else {
		var fault *BusFault
		if !errors.As(err, &fault) || fault.Reason != FaultUnsupportedDeviceState {
			t.Fatalf("nonzero AER fault=%#v/%v", fault, err)
		}
	}
	if _, err := memory.ReadByte(MFPAER, 1); err == nil {
		t.Fatal("user AER read unexpectedly succeeded")
	}
	if err := memory.WriteByte(MFPAER, 0, 1); err == nil {
		t.Fatal("user AER write unexpectedly succeeded")
	}
	if _, err := memory.ReadWord(MFPAER, 5); err == nil {
		t.Fatal("odd AER word read unexpectedly succeeded")
	}
	memory.mfpAER = 0xff
	if err := memory.M68KReset(); err != nil {
		t.Fatal(err)
	}
	if got, err := memory.ReadByte(MFPAER, 5); err != nil || got != 0 {
		t.Fatalf("AER after M68K reset=%02x/%v", got, err)
	}
}

func TestEmptyCartridgeWindowReadsFFAndRejectsWrites(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if CartridgeEnd-CartridgeBase+1 != CartridgeSize {
		t.Fatalf("cartridge window size=%x want %x", CartridgeEnd-CartridgeBase+1, CartridgeSize)
	}
	for _, test := range []struct {
		address uint32
		fc      uint8
	}{
		{CartridgeBase, 1},
		{CartridgeBase + CartridgeSize/2, 2},
		{CartridgeEnd, 5},
		{CartridgeEnd, 6},
	} {
		got, readErr := memory.ReadByte(test.address, test.fc)
		if readErr != nil || got != 0xff {
			t.Fatalf("ReadByte(%06x,fc=%d)=%02x/%v", test.address, test.fc, got, readErr)
		}
	}
	for _, address := range []uint32{CartridgeBase, CartridgeEnd - 1} {
		got, readErr := memory.ReadWord(address, 5)
		if readErr != nil || got != 0xffff {
			t.Fatalf("ReadWord(%06x)=%04x/%v", address, got, readErr)
		}
	}
	if err := memory.WriteByte(MMUConfig, 0x05, 5); err != nil {
		t.Fatal(err)
	}
	if got, err := memory.ReadWord(CartridgeBase, 5); err != nil || got != 0xffff {
		t.Fatalf("cartridge after MMU change=%04x/%v", got, err)
	}
	for _, write := range []struct {
		address uint32
		word    bool
	}{
		{CartridgeBase, false},
		{CartridgeEnd - 1, true},
	} {
		var writeErr error
		if write.word {
			writeErr = memory.WriteWord(write.address, 0x1234, 5)
		} else {
			writeErr = memory.WriteByte(write.address, 0x12, 5)
		}
		var fault *BusFault
		if !errors.As(writeErr, &fault) || fault.Reason != FaultReadOnly ||
			fault.Address != write.address || !fault.Write {
			t.Fatalf("write fault at %06x=%#v/%v", write.address, fault, writeErr)
		}
	}
	if _, err := memory.ReadWord(CartridgeEnd, 5); err == nil {
		t.Fatal("odd cartridge-end word read unexpectedly succeeded")
	} else {
		var fault *BusFault
		if !errors.As(err, &fault) || fault.Reason != FaultOddWordAddress {
			t.Fatalf("odd cartridge-end fault=%#v/%v", fault, err)
		}
	}
	if _, err := memory.ReadByte(CartridgeBase-1, 5); err == nil {
		t.Fatal("pre-cartridge gap unexpectedly mapped")
	}
}

func TestSTRicohVoidDMAByteRead(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	for _, address := range []uint32{STVoidDMAByte, 0xffff860f} {
		got, readErr := memory.ReadByte(address, 5)
		if readErr != nil || got != 0xff {
			t.Fatalf("ReadByte(%08x)=%02x/%v want ff", address, got, readErr)
		}
		for _, value := range []byte{0, 0x12, 0xff} {
			if err := memory.WriteByte(address, value, 5); err != nil {
				t.Fatalf("WriteByte(%08x,%02x): %v", address, value, err)
			}
			if got, readErr := memory.ReadByte(address, 5); readErr != nil || got != 0xff {
				t.Fatalf("void state after write %08x=%02x/%v want ff", address, got, readErr)
			}
		}
	}
	if wait, err := memory.WriteByteAt(STVoidDMAByte, 0x5a,
		m68k.BusAccess{Clock: 2, FunctionCode: 5}); err != nil || wait != 2 {
		t.Fatalf("timed void write wait=%d err=%v want 2/<nil>", wait, err)
	}
	if err := memory.WriteWord(STVoidDMAByte-1, 0, 5); err == nil {
		t.Fatal("word access spanning void byte unexpectedly succeeded")
	}
	for _, test := range []struct {
		name    string
		address uint32
		fc      uint8
		write   bool
		reason  FaultReason
	}{
		{name: "user protection", address: STVoidDMAByte, fc: 1, reason: FaultProtected},
		{name: "preceding reserved byte", address: STVoidDMAByte - 1, fc: 5, reason: FaultReservedIO},
		{name: "following reserved byte", address: STVoidDMAByte + 1, fc: 5, reason: FaultReservedIO},
		{name: "user write protection", address: STVoidDMAByte, fc: 1, write: true, reason: FaultProtected},
	} {
		t.Run(test.name, func(t *testing.T) {
			var accessErr error
			if test.write {
				accessErr = memory.WriteByte(test.address, 0x12, test.fc)
			} else {
				_, accessErr = memory.ReadByte(test.address, test.fc)
			}
			var fault *BusFault
			if !errors.As(accessErr, &fault) || fault.Reason != test.reason ||
				fault.Address != test.address&AddressMask || fault.FunctionCode != test.fc ||
				fault.Write != test.write || fault.Size != 1 {
				t.Fatalf("fault=%#v err=%v", fault, accessErr)
			}
		})
	}
}

func TestSTWithoutMegaRTCUsesVoidByteRange(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	for _, address := range []uint32{STVoidRTCBase, STVoidRTCBase + 0x0f, STVoidRTCEnd, 0xfffffc21} {
		got, readErr := memory.ReadByte(address, 5)
		if readErr != nil || got != 0xff {
			t.Fatalf("ReadByte(%08x)=%02x/%v want ff", address, got, readErr)
		}
		if writeErr := memory.WriteByte(address, 0x05, 5); writeErr != nil {
			t.Fatalf("WriteByte(%08x): %v", address, writeErr)
		}
		got, readErr = memory.ReadByte(address, 5)
		if readErr != nil || got != 0xff {
			t.Fatalf("ReadByte(%08x) after discarded write=%02x/%v want ff", address, got, readErr)
		}
	}
	for _, test := range []struct {
		name    string
		address uint32
		fc      uint8
		write   bool
		reason  FaultReason
	}{
		{name: "user read protection", address: STVoidRTCBase, fc: 1, reason: FaultProtected},
		{name: "user write protection", address: STVoidRTCBase, fc: 1, write: true, reason: FaultProtected},
		{name: "preceding reserved byte", address: STVoidRTCBase - 1, fc: 5, reason: FaultReservedIO},
		{name: "following reserved byte", address: STVoidRTCEnd + 1, fc: 5, reason: FaultReservedIO},
	} {
		t.Run(test.name, func(t *testing.T) {
			var accessErr error
			if test.write {
				accessErr = memory.WriteByte(test.address, 0x12, test.fc)
			} else {
				_, accessErr = memory.ReadByte(test.address, test.fc)
			}
			var fault *BusFault
			if !errors.As(accessErr, &fault) || fault.Reason != test.reason ||
				fault.Address != test.address || fault.FunctionCode != test.fc ||
				fault.Write != test.write || fault.Size != 1 {
				t.Fatalf("fault=%#v err=%v", fault, accessErr)
			}
		})
	}
	if _, err := memory.ReadWord(STVoidRTCBase+1, 5); err == nil {
		t.Fatal("word access into void RTC range unexpectedly accepted")
	}
}

func TestMMUAbsentSecondPhysicalBankFaults(t *testing.T) {
	memory, err := NewMemory(RAM512K, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteByte(MMUConfig, 0x05, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := memory.ReadByte(0x080000, 5); err == nil {
		t.Fatal("absent physical bank1 unexpectedly readable")
	}
}

func TestMemoryProtectionReadOnlyAndUnmappedFaults(t *testing.T) {
	memory, err := NewMemory(RAM512K, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteByte(0x800, 0x5a, 1); err != nil {
		t.Fatalf("user RAM above protected range: %v", err)
	}
	if got, err := memory.ReadByte(0x800, 2); err != nil || got != 0x5a {
		t.Fatalf("user program read above protected range: got=%02x err=%v", got, err)
	}
	tests := []struct {
		name    string
		address uint32
		fc      uint8
		write   bool
		reason  FaultReason
	}{
		{"user low RAM read", 0x7ff, 1, false, FaultProtected},
		{"user low RAM write", 0x7ff, 1, true, FaultProtected},
		{"reset shadow write", 0, 5, true, FaultReadOnly},
		{"TOS ROM write", TOSROMBase, 5, true, FaultReadOnly},
		{"unmapped gap", RAM512K, 5, false, FaultUnmapped},
		{"reserved supervisor I/O", IOBase, 5, false, FaultReservedIO},
		{"protected user I/O", IOBase, 1, false, FaultProtected},
		{"unsupported FC", 0x800, 7, false, FaultFunctionCode},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var accessErr error
			if test.write {
				accessErr = memory.WriteByte(test.address, 1, test.fc)
			} else {
				_, accessErr = memory.ReadByte(test.address, test.fc)
			}
			var fault *BusFault
			if !errors.As(accessErr, &fault) || fault.Reason != test.reason ||
				fault.Address != test.address || fault.FunctionCode != test.fc ||
				fault.Write != test.write || fault.Size != 1 {
				t.Fatalf("fault=%#v err=%v", fault, accessErr)
			}
		})
	}
}

func TestMemoryBigEndianWordAndAtomicFailure(t *testing.T) {
	memory, err := NewMemory(RAM512K, testROM())
	if err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteByte(MMUConfig, 0x04, 5); err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteWord(0x800, 0x1234, 5); err != nil {
		t.Fatal(err)
	}
	got, err := memory.ReadWord(0x800, 5)
	if err != nil || got != 0x1234 {
		t.Fatalf("word=%04x err=%v", got, err)
	}
	for _, address := range []uint32{0x801, RAM512K - 1} {
		err := memory.WriteWord(address, 0xabcd, 5)
		var fault *BusFault
		if !errors.As(err, &fault) || fault.Size != 2 {
			t.Fatalf("word fault at 0x%x: %#v %v", address, fault, err)
		}
	}
	last, err := memory.ReadByte(RAM512K-1, 5)
	if err != nil || last != 0 {
		t.Fatalf("cross-boundary word partially wrote RAM: %02x %v", last, err)
	}
}

func TestMOVEWordProtectedReadEntersBusErrorVector2(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	const handler = uint32(0x1000)
	for address, value := range map[uint32]uint16{
		0x0008: 0x0000, 0x000a: uint16(handler),
		handler: 0x60fe, handler + 2: 0x60fe,
		0x2004: 0x0000,
	} {
		if err := memory.WriteWord(address, value, 5); err != nil {
			t.Fatalf("seed 0x%x: %v", address, err)
		}
	}
	cpu := m68k.CPU{Bus: memory, State: m68k.State{
		D: [8]uint32{0x00fc0370}, SSP: 0x2000, SR: 0x0300,
		PC: 0x2004, Prefetch: [2]uint16{0x3039, 0x0000},
	}}
	result, err := cpu.Step()
	if err != nil {
		t.Fatal(err)
	}
	if result.Clocks != 72 {
		t.Fatalf("clocks=%d want 72", result.Clocks)
	}
	if cpu.State.SSP != 0x1ff2 || cpu.State.SR != 0x2300 || cpu.State.PC != handler+4 ||
		cpu.State.Prefetch != [2]uint16{0x60fe, 0x60fe} || cpu.State.D[0] != 0x00fc0370 {
		t.Fatalf("state=%+v", cpu.State)
	}
	wantFrame := []uint16{0x3031, 0x0000, 0x0000, 0x3039, 0x0300, 0x0000, 0x2006}
	for index, want := range wantFrame {
		got, readErr := memory.ReadWord(cpu.State.SSP+uint32(index*2), 5)
		if readErr != nil || got != want {
			t.Fatalf("frame[%d]=%04x/%v want %04x", index, got, readErr, want)
		}
	}
}

func TestTSTByteReservedIOReadEntersBusErrorVector2(t *testing.T) {
	memory, err := NewMemory(RAM1M, testROM())
	if err != nil {
		t.Fatal(err)
	}
	const handler = uint32(0x1000)
	for address, value := range map[uint32]uint16{
		0x0008: 0x0000, 0x000a: uint16(handler),
		handler: 0x21c9, handler + 2: 0x0008,
	} {
		if err := memory.WriteWord(address, value, 5); err != nil {
			t.Fatalf("seed 0x%x: %v", address, err)
		}
	}
	cpu := m68k.CPU{Bus: memory, State: m68k.State{
		A: [7]uint32{0xffff8a3c}, SSP: 0x2000, SR: 0x2704,
		PC: 0x2004, Prefetch: [2]uint16{0x4a10, 0x4e71},
	}}
	result, err := cpu.Step()
	if err != nil {
		t.Fatal(err)
	}
	if result.Clocks != 64 || cpu.State.SSP != 0x1ff2 || cpu.State.SR != 0x2704 ||
		cpu.State.PC != handler+4 || cpu.State.Prefetch != [2]uint16{0x21c9, 0x0008} ||
		cpu.State.A[0] != 0xffff8a3c {
		t.Fatalf("result=%+v state=%+v", result, cpu.State)
	}
	if len(result.Transactions) != 12 {
		t.Fatalf("transactions=%d want 12: %+v", len(result.Transactions), result.Transactions)
	}
	fault := result.Transactions[0]
	if fault.Kind != "re" || fault.Address != 0xff8a3c || fault.Size != 1 || fault.FC != 5 ||
		!fault.UDS || fault.LDS {
		t.Fatalf("fault transaction=%+v", fault)
	}
	wantFrame := []uint16{0x4a15, 0xffff, 0x8a3c, 0x4a10, 0x2704, 0x0000, 0x2002}
	for index, want := range wantFrame {
		got, readErr := memory.ReadWord(cpu.State.SSP+uint32(index*2), 5)
		if readErr != nil || got != want {
			t.Fatalf("frame[%d]=%04x/%v want %04x", index, got, readErr, want)
		}
	}
}

func TestDivideByZeroEntersVector5(t *testing.T) {
	for _, opcode := range []uint16{0x80c1, 0x81c1} {
		t.Run(fmt.Sprintf("opcode_%04x", opcode), func(t *testing.T) {
			memory, err := NewMemory(RAM1M, testROM())
			if err != nil {
				t.Fatal(err)
			}
			const handler = uint32(0x1000)
			for address, value := range map[uint32]uint16{
				0x0014: 0x0000, 0x0016: uint16(handler),
				handler: 0x60fe, handler + 2: 0x0000,
			} {
				if err := memory.WriteWord(address, value, 5); err != nil {
					t.Fatalf("seed 0x%x: %v", address, err)
				}
			}
			cpu := m68k.CPU{Bus: memory, State: m68k.State{
				D: [8]uint32{1, 0}, SSP: 0x2000, SR: 0x0300,
				PC: 0x2004, Prefetch: [2]uint16{opcode, 0x60fe},
			}}
			result, err := cpu.Step()
			if err != nil {
				t.Fatal(err)
			}
			if result.Clocks != 40 || cpu.State.D[0] != 1 || cpu.State.SSP != 0x1ffa ||
				cpu.State.SR != 0x2304 || cpu.State.PC != handler+4 ||
				cpu.State.Prefetch != [2]uint16{0x60fe, 0x0000} {
				t.Fatalf("result=%+v state=%+v", result, cpu.State)
			}
			wantFrame := []uint16{0x0304, 0x0000, 0x2002}
			for index, want := range wantFrame {
				got, readErr := memory.ReadWord(cpu.State.SSP+uint32(index*2), 5)
				if readErr != nil || got != want {
					t.Fatalf("frame[%d]=%04x/%v want %04x", index, got, readErr, want)
				}
			}
		})
	}
}
