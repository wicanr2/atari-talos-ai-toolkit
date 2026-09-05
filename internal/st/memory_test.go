package st

import (
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
	for address, value := range map[uint32]byte{MFPUCR: 0x88, MFPRSR: 1, MFPTSR: 1} {
		if err := memory.WriteByte(address, value, 5); err == nil {
			t.Fatalf("repeat USART write %06x unexpectedly accepted", address)
		}
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
		memory.ikbdACIAStatus != 2 {
		t.Fatalf("RDR read=%02x status=%02x err=%v", got, memory.ikbdACIAStatus, err)
	}
	if _, err := memory.ReadByte(IKBDACIAData, 5); err == nil {
		t.Fatal("empty RDR read unexpectedly accepted")
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
			if got, err := memory.ReadByte(test.address, 5); err != nil || got != 0 {
				t.Fatalf("initialized %s=%02x/%v want 00", test.name, got, err)
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
		{name: "write remains unsupported", address: STVoidDMAByte, fc: 5, write: true, reason: FaultReservedIO},
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
