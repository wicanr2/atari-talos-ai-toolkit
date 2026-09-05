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
	MFPGPIP       = 0x00ff_fa01
	MFPAER        = 0x00ff_fa03
	MFPDDR        = 0x00ff_fa05
	MFPIERA       = 0x00ff_fa07
	MFPIERB       = 0x00ff_fa09
	MFPIPRA       = 0x00ff_fa0b
	MFPIPRB       = 0x00ff_fa0d
	MFPISRA       = 0x00ff_fa0f
	MFPISRB       = 0x00ff_fa11
	MFPIMRA       = 0x00ff_fa13
	MFPIMRB       = 0x00ff_fa15
	MFPVR         = 0x00ff_fa17
	MFPTACR       = 0x00ff_fa19
	MFPTBCR       = 0x00ff_fa1b
	MFPTCDCR      = 0x00ff_fa1d
	STVoidDMAByte = 0x00ff_860f
	STVoidRTCBase = 0x00ff_fc21
	STVoidRTCEnd  = 0x00ff_fc3f
)

type FaultReason string

const (
	FaultProtected              FaultReason = "protected"
	FaultReadOnly               FaultReason = "read_only"
	FaultUnmapped               FaultReason = "unmapped"
	FaultReservedIO             FaultReason = "reserved_io"
	FaultFunctionCode           FaultReason = "unsupported_function_code"
	FaultOddWordAddress         FaultReason = "odd_word_address"
	FaultUnsupportedDeviceState FaultReason = "unsupported_device_state"
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
	mfpGPIP   byte
	mfpAER    byte
	mfpDDR    byte
	mfpIERA   byte
	mfpIERB   byte
	mfpIPRA   byte
	mfpIPRB   byte
	mfpISRA   byte
	mfpISRB   byte
	mfpIMRA   byte
	mfpIMRB   byte
	mfpVR     byte
	mfpTACR   byte
	mfpTBCR   byte
	mfpTCDCR  byte
}

func (m *Memory) HasExactByteWriteTiming(address uint32) bool {
	return m.isModeledMFPByte(address)
}

func (m *Memory) isModeledMFPByte(address uint32) bool {
	address &= AddressMask
	return address == MFPGPIP || address == MFPAER || address == MFPDDR ||
		address == MFPIERA || address == MFPIERB || address == MFPIPRA || address == MFPIPRB ||
		address == MFPISRA || address == MFPISRB || address == MFPIMRA || address == MFPIMRB ||
		address == MFPVR || address == MFPTACR || address == MFPTBCR || address == MFPTCDCR
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
	case address == MFPGPIP:
		return m.mfpGPIP, nil
	case address == MFPAER:
		return m.mfpAER, nil
	case address == MFPDDR:
		return m.mfpDDR, nil
	case address == MFPIERA:
		return m.mfpIERA, nil
	case address == MFPIERB:
		return m.mfpIERB, nil
	case address == MFPIPRA:
		return m.mfpIPRA, nil
	case address == MFPIPRB:
		return m.mfpIPRB, nil
	case address == MFPISRA:
		return m.mfpISRA, nil
	case address == MFPISRB:
		return m.mfpISRB, nil
	case address == MFPIMRA:
		return m.mfpIMRA, nil
	case address == MFPIMRB:
		return m.mfpIMRB, nil
	case address == MFPVR:
		return m.mfpVR & 0xf8, nil
	case address == MFPTACR:
		return m.mfpTACR, nil
	case address == MFPTBCR:
		return m.mfpTBCR, nil
	case address == MFPTCDCR:
		return m.mfpTCDCR, nil
	case address == STVoidDMAByte:
		return 0xff, nil
	case address >= STVoidRTCBase && address <= STVoidRTCEnd:
		return 0xff, nil
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
	if m.isModeledMFPByte(address) {
		value, err := m.ReadByte(address, access.FunctionCode)
		return value, 4, err
	}
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
	if address <= STVoidRTCEnd && address+1 >= STVoidRTCBase {
		return 0, m.fault(address, functionCode, false, 2, m.unmappedReason(address))
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
	if address == MFPGPIP {
		m.mfpGPIP = m.mfpGPIP&^m.mfpDDR | value&m.mfpDDR
		return nil
	}
	if address == MFPAER {
		if m.mfpAER != 0 || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		return nil
	}
	if address == MFPDDR {
		if m.mfpDDR != 0 || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		return nil
	}
	if address == MFPIERA {
		if m.mfpIERA != 0 || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		return nil
	}
	if address == MFPIERB {
		if m.mfpIERB != 0 || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		return nil
	}
	if address == MFPIPRA {
		m.mfpIPRA &= value
		return nil
	}
	if address == MFPIPRB {
		m.mfpIPRB &= value
		return nil
	}
	if address == MFPISRA {
		m.mfpISRA &= value
		return nil
	}
	if address == MFPISRB {
		m.mfpISRB &= value
		return nil
	}
	if address == MFPIMRA {
		if m.mfpIPRA != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		m.mfpIMRA = value
		return nil
	}
	if address == MFPIMRB {
		if m.mfpIPRB != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		m.mfpIMRB = value
		return nil
	}
	if address == MFPVR {
		newVR := value & 0xf8
		if newVR&0x08 == 0 && (m.mfpIPRA != 0 || m.mfpIPRB != 0) {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		m.mfpVR = newVR
		if newVR&0x08 == 0 {
			m.mfpISRA = 0
			m.mfpISRB = 0
		}
		return nil
	}
	if address == MFPTACR {
		if m.mfpTACR != 0 || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		return nil
	}
	if address == MFPTBCR {
		if m.mfpTBCR != 0 || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		return nil
	}
	if address == MFPTCDCR {
		if m.mfpTCDCR != 0 || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		return nil
	}
	if address >= STVoidRTCBase && address <= STVoidRTCEnd {
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
	if m.isModeledMFPByte(address) {
		return 4, m.WriteByte(address, value, access.FunctionCode)
	}
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
	m.mfpGPIP = 0
	m.mfpAER = 0
	m.mfpDDR = 0
	m.mfpIERA = 0
	m.mfpIERB = 0
	m.mfpIPRA = 0
	m.mfpIPRB = 0
	m.mfpISRA = 0
	m.mfpISRB = 0
	m.mfpIMRA = 0
	m.mfpIMRB = 0
	m.mfpVR = 0
	m.mfpTACR = 0
	m.mfpTBCR = 0
	m.mfpTCDCR = 0
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
