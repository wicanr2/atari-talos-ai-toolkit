package st

import (
	"errors"
	"fmt"
	"testing"

	"github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"
)

var _ m68k.Bus = (*Memory)(nil)

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
