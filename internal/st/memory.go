package st

import (
	"fmt"

	"github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"
)

const (
	AddressMask   = 0x00ff_ffff
	RAM512K       = 512 * 1024
	RAM1M         = 1024 * 1024
	TOSROMSize    = 192 * 1024
	TOSROMBase    = 0x00fc_0000
	TOSROMEnd     = 0x00fe_ffff
	CartridgeBase = 0x00fa_0000
	CartridgeEnd  = 0x00fb_ffff
	CartridgeSize = 128 * 1024
	IOBase        = 0x00ff_0000
	MMUConfig     = 0x00ff_8001
)

type FaultReason string

const (
	FaultProtected      FaultReason = "protected"
	FaultReadOnly       FaultReason = "read_only"
	FaultUnmapped       FaultReason = "unmapped"
	FaultReservedIO     FaultReason = "reserved_io"
	FaultFunctionCode   FaultReason = "unsupported_function_code"
	FaultOddWordAddress FaultReason = "odd_word_address"
)

type BusFault struct {
	Address      uint32
	FunctionCode uint8
	Write        bool
	Size         uint8
	Reason       FaultReason
}

func (f *BusFault) Error() string {
	direction := "read"
	if f.Write {
		direction = "write"
	}
	return fmt.Sprintf("st: %s %d-byte bus fault at 0x%06x fc=%d: %s",
		direction, f.Size, f.Address, f.FunctionCode, f.Reason)
}

func (f *BusFault) M68KBusFault() (uint32, uint8, bool, uint8) {
	return f.Address, f.FunctionCode, f.Write, f.Size
}

type Memory struct {
	ram       []byte
	rom       []byte
	mmuConfig byte
}

func NewMemory(ramSize int, tosROM []byte) (*Memory, error) {
	if ramSize != RAM512K && ramSize != RAM1M {
		return nil, fmt.Errorf("st: RAM size %d is not 512 KiB or 1 MiB", ramSize)
	}
	if len(tosROM) != TOSROMSize {
		return nil, fmt.Errorf("st: TOS ROM size %d, want %d", len(tosROM), TOSROMSize)
	}
	return &Memory{ram: make([]byte, ramSize), rom: append([]byte(nil), tosROM...)}, nil
}

func (m *Memory) ReadByte(address uint32, functionCode uint8) (byte, error) {
	address &= AddressMask
	if fault := m.validateAccess(address, functionCode, false, 1); fault != nil {
		return 0, fault
	}
	switch {
	case address == MMUConfig:
		return m.mmuConfig, nil
	case address < 8:
		return m.rom[address], nil
	case address < 0x0040_0000:
		if physical, ok := m.ramAddress(address); ok {
			return m.ram[physical], nil
		}
		return 0, m.fault(address, functionCode, false, 1, FaultUnmapped)
	case address >= CartridgeBase && address <= CartridgeEnd:
		return 0xff, nil
	case address >= TOSROMBase && address <= TOSROMEnd:
		return m.rom[address-TOSROMBase], nil
	default:
		return 0, m.fault(address, functionCode, false, 1, m.unmappedReason(address))
	}
}

func (m *Memory) ReadByteAt(address uint32, access m68k.BusAccess) (byte, uint32, error) {
	wait, err := busSlotWait(access.Clock)
	if err != nil {
		return 0, 0, err
	}
	value, err := m.ReadByte(address, access.FunctionCode)
	return value, wait, err
}

func (m *Memory) ReadWord(address uint32, functionCode uint8) (uint16, error) {
	address &= AddressMask
	if address&1 != 0 {
		return 0, m.fault(address, functionCode, false, 2, FaultOddWordAddress)
	}
	hi, err := m.ReadByte(address, functionCode)
	if err != nil {
		return 0, resizeFault(err, address, 2)
	}
	lo, err := m.ReadByte(address+1, functionCode)
	if err != nil {
		return 0, resizeFault(err, address, 2)
	}
	return uint16(hi)<<8 | uint16(lo), nil
}

func (m *Memory) ReadWordAt(address uint32, access m68k.BusAccess) (uint16, uint32, error) {
	wait, err := busSlotWait(access.Clock)
	if err != nil {
		return 0, 0, err
	}
	value, err := m.ReadWord(address, access.FunctionCode)
	return value, wait, err
}

func (m *Memory) WriteByte(address uint32, value byte, functionCode uint8) error {
	address &= AddressMask
	if fault := m.validateAccess(address, functionCode, true, 1); fault != nil {
		return fault
	}
	if address == MMUConfig {
		m.mmuConfig = value
		return nil
	}
	if address < 8 || address >= CartridgeBase && address <= CartridgeEnd ||
		address >= TOSROMBase && address <= TOSROMEnd {
		return m.fault(address, functionCode, true, 1, FaultReadOnly)
	}
	physical, ok := m.ramAddress(address)
	if !ok {
		return m.fault(address, functionCode, true, 1, m.unmappedReason(address))
	}
	m.ram[physical] = value
	return nil
}

func (m *Memory) WriteByteAt(address uint32, value byte, access m68k.BusAccess) (uint32, error) {
	wait, err := busSlotWait(access.Clock)
	if err != nil {
		return 0, err
	}
	return wait, m.WriteByte(address, value, access.FunctionCode)
}

func (m *Memory) WriteWord(address uint32, value uint16, functionCode uint8) error {
	address &= AddressMask
	if address&1 != 0 {
		return m.fault(address, functionCode, true, 2, FaultOddWordAddress)
	}
	hi, fault := m.writableRAMAddress(address, functionCode, 2)
	if fault != nil {
		return fault
	}
	lo, fault := m.writableRAMAddress((address+1)&AddressMask, functionCode, 2)
	if fault != nil {
		fault.Address = address
		return fault
	}
	m.ram[hi] = byte(value >> 8)
	m.ram[lo] = byte(value)
	return nil
}

func (m *Memory) WriteWordAt(address uint32, value uint16, access m68k.BusAccess) (uint32, error) {
	wait, err := busSlotWait(access.Clock)
	if err != nil {
		return 0, err
	}
	return wait, m.WriteWord(address, value, access.FunctionCode)
}

func busSlotWait(clock uint64) (uint32, error) {
	phase := clock & 3
	if phase&1 != 0 {
		return 0, fmt.Errorf("st: odd CPU bus clock %d (phase %d)", clock, phase)
	}
	if phase == 2 {
		return 2, nil
	}
	return 0, nil
}

func (m *Memory) writableRAMAddress(address uint32, functionCode uint8, size uint8) (uint32, *BusFault) {
	if fault := m.validateAccess(address, functionCode, true, size); fault != nil {
		return 0, fault
	}
	if address < 8 || address >= CartridgeBase && address <= CartridgeEnd ||
		address >= TOSROMBase && address <= TOSROMEnd {
		return 0, m.fault(address, functionCode, true, size, FaultReadOnly)
	}
	physical, ok := m.ramAddress(address)
	if !ok {
		return 0, m.fault(address, functionCode, true, size, m.unmappedReason(address))
	}
	return physical, nil
}

func (m *Memory) ColdReset() {
	m.mmuConfig = 0
}

func (m *Memory) M68KReset() error {
	m.ColdReset()
	return nil
}

func (m *Memory) ramAddress(address uint32) (uint32, bool) {
	logicalBank0 := mmuBankSize(m.mmuConfig >> 2 & 3)
	logicalBank1 := mmuBankSize(m.mmuConfig & 3)
	physicalBank0 := uint32(RAM512K)
	physicalBank1 := uint32(len(m.ram) - RAM512K)

	if address < logicalBank0 {
		return translate512KBank(address, logicalBank0), true
	}
	if address < logicalBank0+logicalBank1 && physicalBank1 != 0 {
		return physicalBank0 + translate512KBank(address, logicalBank1), true
	}
	return 0, false
}

func mmuBankSize(code byte) uint32 {
	switch code {
	case 0:
		return 128 * 1024
	case 1:
		return 512 * 1024
	case 2:
		return 2 * 1024 * 1024
	default:
		return 0
	}
}

func translate512KBank(address, logicalBankSize uint32) uint32 {
	switch logicalBankSize {
	case 2 * 1024 * 1024:
		address = (address&0x0ff800)>>1 | address&0x03ff
	case 128 * 1024:
		address = (address&0x03fe00)<<1 | address&0x03ff
	}
	return address & (RAM512K - 1)
}

func (m *Memory) validateAccess(address uint32, functionCode uint8, write bool, size uint8) *BusFault {
	if functionCode != 1 && functionCode != 2 && functionCode != 5 && functionCode != 6 {
		return m.fault(address, functionCode, write, size, FaultFunctionCode)
	}
	user := functionCode == 1 || functionCode == 2
	if user && (address < 0x800 || address >= IOBase) {
		return m.fault(address, functionCode, write, size, FaultProtected)
	}
	return nil
}

func (m *Memory) unmappedReason(address uint32) FaultReason {
	if address >= IOBase {
		return FaultReservedIO
	}
	return FaultUnmapped
}

func (m *Memory) fault(address uint32, functionCode uint8, write bool, size uint8, reason FaultReason) *BusFault {
	return &BusFault{Address: address & AddressMask, FunctionCode: functionCode,
		Write: write, Size: size, Reason: reason}
}

func resizeFault(err error, address uint32, size uint8) error {
	fault, ok := err.(*BusFault)
	if !ok {
		return err
	}
	copy := *fault
	copy.Address = address & AddressMask
	copy.Size = size
	return &copy
}
