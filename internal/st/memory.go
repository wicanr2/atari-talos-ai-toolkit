package st

import "fmt"

const (
	AddressMask = 0x00ff_ffff
	RAM512K     = 512 * 1024
	RAM1M       = 1024 * 1024
	TOSROMSize  = 192 * 1024
	TOSROMBase  = 0x00fc_0000
	TOSROMEnd   = 0x00fe_ffff
	IOBase      = 0x00ff_0000
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
	ram []byte
	rom []byte
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
	case address < 8:
		return m.rom[address], nil
	case address < uint32(len(m.ram)):
		return m.ram[address], nil
	case address >= TOSROMBase && address <= TOSROMEnd:
		return m.rom[address-TOSROMBase], nil
	default:
		return 0, m.fault(address, functionCode, false, 1, m.unmappedReason(address))
	}
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

func (m *Memory) WriteByte(address uint32, value byte, functionCode uint8) error {
	address &= AddressMask
	if fault := m.validateAccess(address, functionCode, true, 1); fault != nil {
		return fault
	}
	if address < 8 || address >= TOSROMBase && address <= TOSROMEnd {
		return m.fault(address, functionCode, true, 1, FaultReadOnly)
	}
	if address >= uint32(len(m.ram)) {
		return m.fault(address, functionCode, true, 1, m.unmappedReason(address))
	}
	m.ram[address] = value
	return nil
}

func (m *Memory) WriteWord(address uint32, value uint16, functionCode uint8) error {
	address &= AddressMask
	if address&1 != 0 {
		return m.fault(address, functionCode, true, 2, FaultOddWordAddress)
	}
	if fault := m.validateWritableByte(address, functionCode, 2); fault != nil {
		return fault
	}
	if fault := m.validateWritableByte((address+1)&AddressMask, functionCode, 2); fault != nil {
		fault.Address = address
		return fault
	}
	m.ram[address] = byte(value >> 8)
	m.ram[address+1] = byte(value)
	return nil
}

func (m *Memory) validateWritableByte(address uint32, functionCode uint8, size uint8) *BusFault {
	if fault := m.validateAccess(address, functionCode, true, size); fault != nil {
		return fault
	}
	if address < 8 || address >= TOSROMBase && address <= TOSROMEnd {
		return m.fault(address, functionCode, true, size, FaultReadOnly)
	}
	if address >= uint32(len(m.ram)) {
		return m.fault(address, functionCode, true, size, m.unmappedReason(address))
	}
	return nil
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
