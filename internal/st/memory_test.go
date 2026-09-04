package st

import (
	"errors"
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
