package st

import (
	"fmt"

	"github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"
)

const (
	AddressMask        = 0x00ff_ffff
	RAM512K            = 512 * 1024
	RAM1M              = 1024 * 1024
	TOSROMSize         = 192 * 1024
	TOSROMBase         = 0x00fc_0000
	TOSROMEnd          = 0x00fe_ffff
	CartridgeBase      = 0x00fa_0000
	CartridgeEnd       = 0x00fb_ffff
	CartridgeSize      = 128 * 1024
	IOBase             = 0x00ff_0000
	MMUConfig          = 0x00ff_8001
	VideoBaseHigh      = 0x00ff_8201
	VideoBaseMiddle    = 0x00ff_8203
	VideoSyncMode      = 0x00ff_820a
	ShifterPaletteBase = 0x00ff_8240
	ShifterPaletteEnd  = 0x00ff_825e
	ShifterResolution  = 0x00ff_8260
	PSGRegisterSelect  = 0x00ff_8800
	PSGRegisterData    = 0x00ff_8802
	IKBDACIAControl    = 0x00ff_fc00
	IKBDACIAData       = 0x00ff_fc02
	MIDIACIAControl    = 0x00ff_fc04
	MIDIACIAData       = 0x00ff_fc06
	MFPGPIP            = 0x00ff_fa01
	MFPAER             = 0x00ff_fa03
	MFPDDR             = 0x00ff_fa05
	MFPIERA            = 0x00ff_fa07
	MFPIERB            = 0x00ff_fa09
	MFPIPRA            = 0x00ff_fa0b
	MFPIPRB            = 0x00ff_fa0d
	MFPISRA            = 0x00ff_fa0f
	MFPISRB            = 0x00ff_fa11
	MFPIMRA            = 0x00ff_fa13
	MFPIMRB            = 0x00ff_fa15
	MFPVR              = 0x00ff_fa17
	MFPTACR            = 0x00ff_fa19
	MFPTBCR            = 0x00ff_fa1b
	MFPTCDCR           = 0x00ff_fa1d
	MFPTADR            = 0x00ff_fa1f
	MFPTBDR            = 0x00ff_fa21
	MFPTCDR            = 0x00ff_fa23
	MFPTDDR            = 0x00ff_fa25
	MFPSCR             = 0x00ff_fa27
	MFPUCR             = 0x00ff_fa29
	MFPRSR             = 0x00ff_fa2b
	MFPTSR             = 0x00ff_fa2d
	MFPUDR             = 0x00ff_fa2f
	STVoidDMAByte      = 0x00ff_860f
	STVoidRTCBase      = 0x00ff_fc21
	STVoidRTCEnd       = 0x00ff_fc3f
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
	ram                     []byte
	rom                     []byte
	mmuConfig               byte
	videoBaseHigh           byte
	videoBaseMiddle         byte
	activeVideoBase         uint32
	videoSyncMode           byte
	videoSync50Transition   bool
	shifterPalette          [16]uint16
	shifterResolution       byte
	psgRegisterSelect       byte
	psgRegisters            [16]byte
	ikbdACIAControl         byte
	ikbdACIAStatus          byte
	ikbdACIAConfigured      bool
	ikbdACIATDR             byte
	ikbdACIATXPending       bool
	ikbdACIATXShiftTicks    uint8
	ikbdResetCommandDone    bool
	ikbdResetCommandHandled bool
	ikbdACIARDR             byte
	ikbdResetResponseRead   bool
	ikbdStaleRDRReads       uint8
	midiACIAControl         byte
	midiACIAStatus          byte
	midiACIAConfigured      bool
	mfpACIAEnableStage      uint8
	mfpTimerDSystemStage    uint8
	mfpTimerDStopStage      uint8
	mfpGPIP                 byte
	mfpGPIPIn               byte
	mfpAER                  byte
	mfpDDR                  byte
	mfpIERA                 byte
	mfpIERB                 byte
	mfpIPRA                 byte
	mfpIPRB                 byte
	mfpISRA                 byte
	mfpISRB                 byte
	mfpIMRA                 byte
	mfpIMRB                 byte
	mfpVR                   byte
	mfpTACR                 byte
	mfpTBCR                 byte
	mfpTCDCR                byte
	mfpTimerCStart          bool
	mfpTimerCStartClock     uint64
	mfpTimerDStart          bool
	mfpTimerDStartClock     uint64
	mfpTADR                 byte
	mfpTBDR                 byte
	mfpTCDR                 byte
	mfpTDDR                 byte
	mfpTAMain               byte
	mfpTBMain               byte
	mfpTCMain               byte
	mfpTDMain               byte
	mfpSCR                  byte
	mfpUCR                  byte
	mfpRSR                  byte
	mfpTSR                  byte
	mfpTSRSet               bool
}

func (m *Memory) HasExactByteWriteTiming(address uint32) bool {
	return m.isModeledMFPByte(address) || m.isModeledPSGByte(address) || m.isModeledACIAByte(address)
}

func (m *Memory) isModeledPSGByte(address uint32) bool {
	address &= AddressMask
	return address == PSGRegisterSelect || address == PSGRegisterData
}

func (m *Memory) isModeledIKBDACIAByte(address uint32) bool {
	address &= AddressMask
	return address == IKBDACIAControl || address == IKBDACIAData
}

func (m *Memory) isModeledACIAByte(address uint32) bool {
	address &= AddressMask
	return m.isModeledIKBDACIAByte(address) || address == MIDIACIAControl || address == MIDIACIAData
}

func (m *Memory) isModeledMFPByte(address uint32) bool {
	address &= AddressMask
	return address == MFPGPIP || address == MFPAER || address == MFPDDR ||
		address == MFPIERA || address == MFPIERB || address == MFPIPRA || address == MFPIPRB ||
		address == MFPISRA || address == MFPISRB || address == MFPIMRA || address == MFPIMRB ||
		address == MFPVR || address == MFPTACR || address == MFPTBCR || address == MFPTCDCR ||
		address == MFPTADR || address == MFPTBDR || address == MFPTCDR || address == MFPTDDR ||
		address == MFPSCR || address == MFPUCR || address == MFPRSR || address == MFPTSR
}

func NewMemory(ramSize int, tosROM []byte) (*Memory, error) {
	if ramSize != RAM512K && ramSize != RAM1M {
		return nil, fmt.Errorf("st: RAM size %d is not 512 KiB or 1 MiB", ramSize)
	}
	if len(tosROM) != TOSROMSize {
		return nil, fmt.Errorf("st: TOS ROM size %d, want %d", len(tosROM), TOSROMSize)
	}
	return &Memory{
		ram:       make([]byte, ramSize),
		rom:       append([]byte(nil), tosROM...),
		mfpGPIPIn: 0xa1,
	}, nil
}

func (m *Memory) ReadByte(address uint32, functionCode uint8) (byte, error) {
	address &= AddressMask
	if fault := m.validateAccess(address, functionCode, false, 1); fault != nil {
		return 0, fault
	}
	switch {
	case address == MMUConfig:
		return m.mmuConfig, nil
	case address == VideoBaseHigh:
		return m.videoBaseHigh, nil
	case address == VideoBaseMiddle:
		return m.videoBaseMiddle, nil
	case address == VideoSyncMode:
		return m.videoSyncMode | 0xfc, nil
	case address == ShifterResolution:
		return m.shifterResolution | 0xfc, nil
	case m.isModeledPSGByte(address):
		return 0, m.fault(address, functionCode, false, 1, FaultUnsupportedDeviceState)
	case address == IKBDACIAControl:
		if !m.ikbdACIAConfigured {
			return 0, m.fault(address, functionCode, false, 1, FaultUnsupportedDeviceState)
		}
		return m.ikbdACIAStatus, nil
	case address == IKBDACIAData:
		if m.ikbdACIAConfigured && m.ikbdACIAStatus&1 != 0 {
			value := m.ikbdACIARDR
			m.ikbdACIAStatus &^= 0x81
			m.ikbdResetResponseRead = true
			m.ikbdStaleRDRReads = 1
			return value, nil
		}
		if m.ikbdACIAConfigured && m.ikbdStaleRDRReads != 0 {
			m.ikbdStaleRDRReads--
			return m.ikbdACIARDR, nil
		}
		return 0, m.fault(address, functionCode, false, 1, FaultUnsupportedDeviceState)
	case address == MIDIACIAControl, address == MIDIACIAData:
		return 0, m.fault(address, functionCode, false, 1, FaultUnsupportedDeviceState)
	case address == MFPGPIP:
		m.mfpGPIP = m.mfpGPIP&m.mfpDDR | m.mfpGPIPIn&^m.mfpDDR
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
	case address == MFPTADR:
		if m.mfpTACR != 0 {
			return 0, m.fault(address, functionCode, false, 1, FaultUnsupportedDeviceState)
		}
		return m.mfpTAMain, nil
	case address == MFPTBDR:
		if m.mfpTBCR != 0 {
			return 0, m.fault(address, functionCode, false, 1, FaultUnsupportedDeviceState)
		}
		return m.mfpTBMain, nil
	case address == MFPTCDR:
		if m.mfpTCDCR&0x70 != 0 {
			return 0, m.fault(address, functionCode, false, 1, FaultUnsupportedDeviceState)
		}
		return m.mfpTCMain, nil
	case address == MFPTDDR:
		if m.mfpTCDCR&0x07 != 0 {
			return 0, m.fault(address, functionCode, false, 1, FaultUnsupportedDeviceState)
		}
		return m.mfpTDMain, nil
	case address == MFPSCR:
		return m.mfpSCR, nil
	case address == MFPUCR:
		return m.mfpUCR, nil
	case address == MFPRSR:
		return m.mfpRSR, nil
	case address == MFPTSR:
		if !m.mfpTSRSet {
			return 0, m.fault(address, functionCode, false, 1, FaultUnsupportedDeviceState)
		}
		return m.mfpTSR, nil
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
	if m.isModeledMFPByte(address) || m.isModeledPSGByte(address) || m.isModeledACIAByte(address) {
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
	if address >= ShifterPaletteBase && address <= ShifterPaletteEnd {
		if fault := m.validateAccess(address, functionCode, false, 2); fault != nil {
			return 0, fault
		}
		return m.shifterPalette[(address-ShifterPaletteBase)/2], nil
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
	if address == VideoBaseHigh {
		m.videoBaseHigh = value & 0x3f
		return nil
	}
	if address == VideoBaseMiddle {
		m.videoBaseMiddle = value
		return nil
	}
	if address == VideoSyncMode {
		if value&^byte(2) != 0 || m.videoSyncMode == 2 && value == 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		if m.videoSyncMode == 0 && value == 2 {
			m.videoSyncMode = 2
			m.videoSync50Transition = true
		}
		return nil
	}
	if address == ShifterResolution {
		if m.shifterResolution != 0 || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		return nil
	}
	if address == PSGRegisterSelect {
		if m.psgRegisterSelect == 0 && m.psgRegisters[7] == 0 && m.psgRegisters[14] == 0 && value == 7 ||
			m.psgRegisterSelect == 7 && m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 0 && value == 14 {
			m.psgRegisterSelect = value
			return nil
		}
		return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
	}
	if address == PSGRegisterData {
		if m.psgRegisterSelect == 7 && m.psgRegisters[7] == 0 && value == 0xc0 ||
			m.psgRegisterSelect == 14 && m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 0 && value == 7 {
			m.psgRegisters[m.psgRegisterSelect] = value
			return nil
		}
		return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
	}
	if address == IKBDACIAControl {
		if m.ikbdACIAControl == 0 && !m.ikbdACIAConfigured && value == 3 {
			m.ikbdACIAControl = value
			m.ikbdACIAStatus = 2
			return nil
		}
		if m.ikbdACIAControl == 3 && !m.ikbdACIAConfigured && value == 0x96 {
			m.ikbdACIAControl = value
			m.ikbdACIAConfigured = true
			return nil
		}
		return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
	}
	if address == IKBDACIAData {
		validFirst := value == 0x80 && m.ikbdACIATDR == 0 && m.ikbdACIATXShiftTicks == 0
		validSecond := value == 1 && m.ikbdACIATDR == 0x80 && m.ikbdACIATXShiftTicks != 0
		if m.ikbdACIAConfigured && m.ikbdACIAStatus&2 != 0 && !m.ikbdACIATXPending &&
			(validFirst || validSecond) {
			m.ikbdACIATDR = value
			m.ikbdACIATXPending = true
			m.ikbdACIAStatus &^= 2
			return nil
		}
		return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
	}
	if address == MIDIACIAControl {
		if m.midiACIAControl == 0 && !m.midiACIAConfigured && value == 3 {
			m.midiACIAControl = value
			m.midiACIAStatus = 2
			return nil
		}
		if m.midiACIAControl == 3 && !m.midiACIAConfigured && value == 0x95 {
			m.midiACIAControl = value
			m.midiACIAConfigured = true
			return nil
		}
		return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
	}
	if address == MIDIACIAData {
		return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
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
		serialReady := m.mfpUCR == 0x88 && m.mfpRSR == 1 && m.mfpTSR == 1 && m.mfpIPRA == 0
		if serialReady && (m.mfpIERA == 0 && value == 0x10 ||
			m.mfpIERA == 0x10 && (value == 0x10 || value == 0x14)) {
			m.mfpIERA = value
			return nil
		}
		if m.mfpIERA != 0 || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		return nil
	}
	if address == MFPIERB {
		if m.mfpIERB == 0 && value == 0x20 && m.mfpIPRB == 0 && m.mfpTCDCR&0x70 == 0x50 {
			m.mfpIERB = value
			return nil
		}
		if m.midiACIAConfigured && m.mfpIERB == 0x20 && m.mfpIMRB == 0x20 &&
			m.mfpACIAEnableStage == 0 && value == 0x20 {
			m.mfpACIAEnableStage = 1
			return nil
		}
		if m.mfpACIAEnableStage == 3 && m.mfpIERB == 0x20 && value == 0x60 &&
			m.mfpIPRB&0x40 == 0 && m.mfpISRB&0x40 == 0 {
			m.mfpIERB = value
			m.mfpACIAEnableStage = 4
			return nil
		}
		if m.mfpACIAEnableStage == 5 && m.mfpTimerDSystemStage == 0 &&
			m.mfpIERB == 0x60 && m.mfpIMRB == 0x60 && value == 0x60 {
			m.mfpTimerDSystemStage = 1
			return nil
		}
		if m.mfpTimerDSystemStage == 5 && m.mfpIERB == 0x60 && value == 0x70 &&
			m.mfpIPRB&0x10 == 0 && m.mfpISRB&0x10 == 0 {
			m.mfpIERB = value
			m.mfpTimerDSystemStage = 6
			return nil
		}
		if m.mfpTimerDSystemStage == 8 && m.mfpTimerDStopStage == 0 &&
			m.mfpIERB == 0x70 && m.mfpIMRB == 0x70 && value == 0x60 &&
			m.mfpIPRB&0x10 == 0 && m.mfpISRB&0x10 == 0 {
			m.mfpIERB = value
			m.mfpTimerDStopStage = 1
			return nil
		}
		if m.mfpTimerDStopStage == 4 && m.mfpIERB == 0x60 && value == 0x60 {
			m.mfpTimerDStopStage = 5
			return nil
		}
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
		if m.mfpACIAEnableStage == 1 && value == 0xbf {
			m.mfpACIAEnableStage = 2
		}
		if m.mfpTimerDSystemStage == 1 && value == 0xef {
			m.mfpTimerDSystemStage = 2
		}
		if m.mfpTimerDStopStage == 5 && value == 0xef {
			m.mfpTimerDStopStage = 6
		}
		return nil
	}
	if address == MFPISRA {
		m.mfpISRA &= value
		return nil
	}
	if address == MFPISRB {
		m.mfpISRB &= value
		if m.mfpACIAEnableStage == 2 && value == 0xbf {
			m.mfpACIAEnableStage = 3
		}
		if m.mfpTimerDSystemStage == 2 && value == 0xef {
			m.mfpTimerDSystemStage = 3
		}
		if m.mfpTimerDStopStage == 6 && value == 0xef {
			m.mfpTimerDStopStage = 7
		}
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
		if m.mfpTimerDStopStage == 3 && m.mfpIMRB == 0x60 && value == 0x60 {
			m.mfpTimerDStopStage = 4
			return nil
		}
		if m.mfpIPRB != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		if m.mfpTimerDStopStage == 1 && m.mfpIMRB == 0x70 && value == 0x60 {
			m.mfpIMRB = value
			m.mfpTimerDStopStage = 2
			return nil
		}
		m.mfpIMRB = value
		if m.mfpACIAEnableStage == 4 && value == 0x60 {
			m.mfpACIAEnableStage = 5
		}
		if m.mfpTimerDSystemStage == 6 && value == 0x70 {
			m.mfpTimerDSystemStage = 7
		}
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
		if m.mfpTCDCR == 0 && value == 0x50 && m.mfpTCDR == 0xc0 && m.mfpTCMain == 0xc0 {
			m.mfpTCDCR = value
			m.mfpTimerCStart = true
			return nil
		}
		if m.mfpTCDCR == 0x50 && value == 0x50 {
			return nil
		}
		if m.mfpTCDCR == 0x50 && value == 0x51 && m.mfpTDDR == 2 && m.mfpTDMain == 2 {
			m.mfpTCDCR = value
			m.mfpTimerDStart = true
			return nil
		}
		if m.mfpTimerDSystemStage == 3 && m.mfpTCDCR == 0x51 && value == 0x50 &&
			m.mfpTDDR == 2 && m.mfpTDMain == 2 {
			m.mfpTCDCR = value
			m.mfpTimerDStart = false
			m.mfpTimerDSystemStage = 4
			return nil
		}
		if m.mfpTimerDSystemStage == 7 && m.mfpTCDCR == 0x50 && value == 0x52 &&
			m.mfpTDDR == 0 && m.mfpTDMain == 0 {
			m.mfpTCDCR = value
			m.mfpTimerDStart = true
			m.mfpTimerDSystemStage = 8
			return nil
		}
		if m.mfpTimerDStopStage == 2 && m.mfpTCDCR == 0x52 && value == 0x50 &&
			m.mfpIERB == 0x60 && m.mfpIMRB == 0x60 {
			m.mfpTCDCR = value
			m.mfpTimerDStart = false
			m.mfpTimerDStartClock = 0
			m.mfpTimerDStopStage = 3
			return nil
		}
		if m.mfpTCDCR != 0 || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		return nil
	}
	if address == MFPTADR {
		if m.mfpTACR != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		m.mfpTADR, m.mfpTAMain = value, value
		return nil
	}
	if address == MFPTBDR {
		if m.mfpTBCR != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		m.mfpTBDR, m.mfpTBMain = value, value
		return nil
	}
	if address == MFPTCDR {
		if m.mfpTCDCR&0x70 != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		m.mfpTCDR, m.mfpTCMain = value, value
		return nil
	}
	if address == MFPTDDR {
		if m.mfpTCDCR&0x07 != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		m.mfpTDDR, m.mfpTDMain = value, value
		if m.mfpTimerDSystemStage == 4 && value == 0 {
			m.mfpTimerDSystemStage = 5
		}
		return nil
	}
	if address == MFPSCR {
		if m.mfpSCR != 0 || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		return nil
	}
	if address == MFPUCR {
		if m.mfpUCR == 0 && value == 0x88 && m.mfpTCDCR == 0x51 &&
			m.mfpTDDR == 2 && m.mfpTDMain == 2 {
			m.mfpUCR = value
			return nil
		}
		if m.mfpUCR != 0 || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		return nil
	}
	if address == MFPRSR {
		if m.mfpRSR == 0 && value == 1 && m.mfpUCR == 0x88 {
			m.mfpRSR = value
			return nil
		}
		if m.mfpRSR != 0 || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		return nil
	}
	if address == MFPTSR {
		if m.mfpTSRSet && m.mfpTSR == 0 && value == 1 && m.mfpUCR == 0x88 && m.mfpRSR == 1 {
			m.mfpTSR = value
			return nil
		}
		if (m.mfpTSRSet && m.mfpTSR != 0) || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		m.mfpTSR = 0
		m.mfpTSRSet = true
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
	if m.isModeledMFPByte(address) || m.isModeledPSGByte(address) || m.isModeledACIAByte(address) {
		wasTimerC := m.mfpTimerCStart
		wasSystemTimerD := m.mfpTimerDSystemStage == 8 && m.mfpTimerDStart
		err := m.WriteByte(address, value, access.FunctionCode)
		if err == nil && !wasTimerC && m.mfpTimerCStart {
			m.mfpTimerCStartClock = access.Clock
		}
		if err == nil && !wasSystemTimerD && m.mfpTimerDSystemStage == 8 && m.mfpTimerDStart {
			m.mfpTimerDStartClock = access.Clock
		}
		return 4, err
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
	if address >= ShifterPaletteBase && address <= ShifterPaletteEnd {
		if fault := m.validateAccess(address, functionCode, true, 2); fault != nil {
			return fault
		}
		m.shifterPalette[(address-ShifterPaletteBase)/2] = value & 0x0777
		return nil
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
	m.videoBaseHigh = 0
	m.videoBaseMiddle = 0
	m.activeVideoBase = 0
	m.videoSyncMode = 0
	m.videoSync50Transition = false
	m.shifterPalette = [16]uint16{}
	m.shifterResolution = 0
	m.psgRegisterSelect = 0
	m.psgRegisters = [16]byte{}
	m.ikbdACIAControl = 0
	m.ikbdACIAStatus = 0
	m.ikbdACIAConfigured = false
	m.ikbdACIATDR = 0
	m.ikbdACIATXPending = false
	m.ikbdACIATXShiftTicks = 0
	m.ikbdResetCommandDone = false
	m.ikbdResetCommandHandled = false
	m.ikbdACIARDR = 0
	m.ikbdResetResponseRead = false
	m.ikbdStaleRDRReads = 0
	m.midiACIAControl = 0
	m.midiACIAStatus = 0
	m.midiACIAConfigured = false
	m.mfpACIAEnableStage = 0
	m.mfpTimerDSystemStage = 0
	m.mfpTimerDStopStage = 0
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
	m.mfpTimerCStart = false
	m.mfpTimerCStartClock = 0
	m.mfpTimerDStart = false
	m.mfpTimerDStartClock = 0
	m.mfpTADR = 0
	m.mfpTBDR = 0
	m.mfpTCDR = 0
	m.mfpTDDR = 0
	m.mfpTAMain = 0
	m.mfpTBMain = 0
	m.mfpTCMain = 0
	m.mfpTDMain = 0
	m.mfpSCR = 0
	m.mfpUCR = 0
	m.mfpRSR = 0
	m.mfpTSR = 0
	m.mfpTSRSet = false
}

func (m *Memory) mfpVector(channel uint8) uint8 {
	return m.mfpVR&0xf0 | channel
}

func (m *Memory) acknowledgeMFPB(channel uint8) {
	bit := byte(1 << channel)
	m.mfpIPRB &^= bit
	if m.mfpVR&0x08 != 0 {
		m.mfpISRB |= bit
	}
}

func (m *Memory) advanceIKBDACIAClock() {
	if m.ikbdACIATXShiftTicks != 0 {
		m.ikbdACIATXShiftTicks--
	}
	if m.ikbdACIATXShiftTicks == 0 && m.ikbdACIATXPending {
		m.ikbdACIATXPending = false
		m.ikbdACIAStatus |= 2
		m.ikbdACIATXShiftTicks = 10
	} else if m.ikbdACIATXShiftTicks == 0 && m.ikbdACIATDR == 1 && !m.ikbdResetCommandHandled {
		m.ikbdResetCommandDone = true
		m.ikbdResetCommandHandled = true
	}
}

func (m *Memory) deliverIKBDResetResponse() {
	if m.ikbdACIAConfigured && m.ikbdACIAStatus&1 == 0 {
		m.ikbdACIARDR = 0xf1
		m.ikbdACIAStatus |= 0x81
	}
}

// ProgrammedVideoBase returns the address selected for the next Shifter base reload.
// It is not the base currently used by an active scanout.
func (m *Memory) ProgrammedVideoBase() uint32 {
	return uint32(m.videoBaseHigh)<<16 | uint32(m.videoBaseMiddle)<<8
}

// ActiveVideoBase returns the base committed by the latest modeled VBL reload.
func (m *Memory) ActiveVideoBase() uint32 {
	return m.activeVideoBase
}

func (m *Memory) reloadVideoBaseOnVBL() {
	m.activeVideoBase = m.ProgrammedVideoBase()
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
