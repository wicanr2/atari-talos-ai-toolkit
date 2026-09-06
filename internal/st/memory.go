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
	STDiskController   = 0x00ff_8604
	STDMAControl       = 0x00ff_8606
	STDMAAddressHigh   = 0x00ff_8609
	STDMAAddressMiddle = 0x00ff_860b
	STDMAAddressLow    = 0x00ff_860d
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
	ram                             []byte
	rom                             []byte
	mmuConfig                       byte
	videoBaseHigh                   byte
	videoBaseMiddle                 byte
	activeVideoBase                 uint32
	videoSyncMode                   byte
	videoSync50Transition           bool
	shifterPalette                  [16]uint16
	shifterResolution               byte
	psgRegisterSelect               byte
	psgRegisters                    [16]byte
	psgDriveStage                   uint8
	flopVBLMediaStage               uint8
	flopVBLStatusReadClock          uint64
	flopVBLMediaComplete            bool
	flopVBLMediaChecks              uint32
	flopVBLMediaDrive               int8
	floppyReadStage                 uint8
	floppyMediaLegacy               [3]floppyMediaReceipt
	dmaMode                         uint16
	dmaAddress                      uint32
	dmaAddressWriteStage            uint8
	dmaSectorCount                  uint8
	dmaResetCount                   uint8
	dmaInitStage                    uint8
	acsiStage                       uint8
	acsiTarget                      int8
	acsiCommand                     byte
	acsiAttemptMask                 byte
	acsiCommandReceipts             [8]byte
	acsiTimeoutReturnClock          uint64
	acsiTimeoutReturnClocks         [8]uint64
	fdcCommand                      byte
	fdcStatus                       byte
	fdcStatusTypeI                  bool
	fdcIRQ                          bool
	fdcInitStage                    uint8
	fdcProbeDrive                   int8
	fdcRestorePending               bool
	fdcRestoreStartClock            uint64
	fdcRestoreInactivePolls         uint8
	fdcRestoreIRQObserved           bool
	fdcStatusReadClock              uint64
	fdcData                         byte
	fdcSeekPending                  bool
	fdcSeekStartClock               uint64
	fdcSeekInactivePolls            uint8
	fdcSeekIRQObserved              bool
	fdcSeekStatusReadClock          uint64
	ikbdACIAControl                 byte
	ikbdACIAStatus                  byte
	ikbdACIAConfigured              bool
	ikbdACIATDR                     byte
	ikbdACIATXShift                 byte
	ikbdACIATXPending               bool
	ikbdACIATXShiftTicks            uint8
	ikbdResetCommandDone            bool
	ikbdResetCommandHandled         bool
	ikbdClockRequestDone            bool
	ikbdClockRequestHandled         bool
	ikbdACIARDR                     byte
	ikbdClockResponseActive         bool
	ikbdClockResponseDelivered      uint8
	ikbdClockResponseReadCount      uint8
	ikbdClockResponseReads          [7]byte
	ikbdClockResponseReadClocks     [7]uint64
	ikbdClockResponseComplete       bool
	ikbdSetClockWrites              [7]byte
	ikbdSetClockWriteCount          uint8
	ikbdSetClockCompletions         [7]byte
	ikbdSetClockCompleteCount       uint8
	ikbdSetClockCompletionClocks    [7]uint64
	ikbdSetClockComplete            bool
	ikbdClockReadbackRequestWritten bool
	ikbdClockReadbackRequestDone    bool
	ikbdClockReadbackRequestHandled bool
	ikbdClockReadbackActive         bool
	ikbdClockReadbackDelivered      uint8
	ikbdClockReadbackReadCount      uint8
	ikbdClockReadbackReads          [7]byte
	ikbdClockReadbackDeliveryClocks [7]uint64
	ikbdClockReadbackReadClocks     [7]uint64
	ikbdClockReadbackComplete       bool
	ikbdClockPollRequestWritten     bool
	ikbdClockPollRequestCount       uint32
	ikbdClockPollResponseActive     bool
	ikbdClockPollResponseDelivered  uint8
	ikbdClockPollResponseReadCount  uint8
	ikbdClockPollResponseReads      [7]byte
	ikbdClockPollDeliveryClocks     [7]uint64
	ikbdClockPollReadClocks         [7]uint64
	ikbdClockPollCompleteCount      uint32
	ikbdResetResponseRead           bool
	ikbdStaleRDRReads               uint8
	midiACIAControl                 byte
	midiACIAStatus                  byte
	midiACIAConfigured              bool
	mfpACIAEnableStage              uint8
	mfpTimerDSystemStage            uint8
	mfpTimerDStopStage              uint8
	mfpUSARTReconfigStage           uint8
	mfpGPIP                         byte
	mfpGPIPIn                       byte
	mfpAER                          byte
	mfpDDR                          byte
	mfpIERA                         byte
	mfpIERB                         byte
	mfpIPRA                         byte
	mfpIPRB                         byte
	mfpISRA                         byte
	mfpISRB                         byte
	mfpIMRA                         byte
	mfpIMRB                         byte
	mfpVR                           byte
	mfpTACR                         byte
	mfpTBCR                         byte
	mfpTCDCR                        byte
	mfpTimerCStart                  bool
	mfpTimerCStartClock             uint64
	mfpTimerDStart                  bool
	mfpTimerDStartClock             uint64
	mfpTADR                         byte
	mfpTBDR                         byte
	mfpTCDR                         byte
	mfpTDDR                         byte
	mfpTAMain                       byte
	mfpTBMain                       byte
	mfpTCMain                       byte
	mfpTDMain                       byte
	mfpSCR                          byte
	mfpUCR                          byte
	mfpRSR                          byte
	mfpTSR                          byte
	mfpTSRSet                       bool
}

func (m *Memory) flopVBLTargetPort() byte {
	if m.flopVBLMediaDrive == 0 {
		return 0x25
	}
	if m.flopVBLMediaDrive == 1 {
		return 0x23
	}
	return 0xff
}

func (m *Memory) HasExactByteWriteTiming(address uint32) bool {
	return m.isModeledMFPByte(address) || m.isModeledPSGByte(address) || m.isModeledACIAByte(address)
}

func (m *Memory) HasExactWordWriteTiming(address uint32) bool {
	address &= AddressMask
	return address == STDMAControl || address == STDiskController
}

func (m *Memory) HasExactWordReadTiming(address uint32) bool {
	return address&AddressMask == STDiskController
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

func isDMAAddressByte(address uint32) bool {
	address &= AddressMask
	return address == STDMAAddressHigh || address == STDMAAddressMiddle || address == STDMAAddressLow
}

func NewMemory(ramSize int, tosROM []byte) (*Memory, error) {
	if ramSize != RAM512K && ramSize != RAM1M {
		return nil, fmt.Errorf("st: RAM size %d is not 512 KiB or 1 MiB", ramSize)
	}
	if len(tosROM) != TOSROMSize {
		return nil, fmt.Errorf("st: TOS ROM size %d, want %d", len(tosROM), TOSROMSize)
	}
	return &Memory{
		ram:           make([]byte, ramSize),
		rom:           append([]byte(nil), tosROM...),
		mfpGPIPIn:     0xa1,
		fdcProbeDrive: -1,
		acsiTarget:    -1,
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
	case address == STDMAAddressHigh:
		return byte(m.dmaAddress >> 16), nil
	case address == STDMAAddressMiddle:
		return byte(m.dmaAddress >> 8), nil
	case address == STDMAAddressLow:
		return byte(m.dmaAddress), nil
	case address == PSGRegisterSelect && m.psgDriveStage == 1 &&
		m.psgRegisterSelect == 14 && m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 7:
		m.psgDriveStage = 2
		return m.psgRegisters[14], nil
	case address == PSGRegisterSelect && m.psgDriveStage == 4 && m.fdcInitStage == 14 &&
		m.psgRegisterSelect == 14 && m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 5:
		m.psgDriveStage = 5
		return m.psgRegisters[14], nil
	case address == PSGRegisterSelect && m.psgDriveStage == 7 && m.acsiStage == 5 &&
		m.fdcInitStage == 14 && m.psgRegisterSelect == 14 && m.psgRegisters[7] == 0xc0 &&
		m.psgRegisters[14] == 3:
		m.psgDriveStage = 8
		return m.psgRegisters[14], nil
	case address == PSGRegisterSelect && m.psgDriveStage == 9 && m.flopVBLMediaStage == 1 &&
		m.ikbdClockReadbackComplete && m.psgRegisterSelect == 14 && m.psgRegisters[14] == 0x23:
		m.flopVBLMediaStage = 2
		return m.psgRegisters[14], nil
	case address == PSGRegisterSelect && m.psgDriveStage == 9 && m.flopVBLMediaStage == 6 &&
		m.psgRegisterSelect == 14 && m.psgRegisters[14] == m.flopVBLTargetPort():
		m.flopVBLMediaStage = 7
		return m.psgRegisters[14], nil
	case address == PSGRegisterSelect && m.floppyReadStage == 3 && m.psgDriveStage == 9 &&
		m.psgRegisterSelect == 14 && m.psgRegisters[14] == 0x23:
		m.floppyReadStage = 4
		return m.psgRegisters[14], nil
	case address == PSGRegisterSelect && m.floppyReadStage == 25 && m.psgDriveStage == 9 &&
		m.psgRegisterSelect == 14 && m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 0x25:
		m.floppyReadStage = 26
		return m.psgRegisters[14], nil
	case address == PSGRegisterSelect && m.floppyReadStage == 47 && m.psgDriveStage == 9 &&
		m.psgRegisterSelect == 14 && m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 0x25:
		m.floppyReadStage = 48
		return m.psgRegisters[14], nil
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
			m.mfpGPIPIn |= 0x10
			if m.ikbdClockReadbackActive && m.ikbdClockReadbackReadCount < m.ikbdClockReadbackDelivered {
				m.ikbdClockReadbackReads[m.ikbdClockReadbackReadCount] = value
				m.ikbdClockReadbackReadCount++
				if m.ikbdClockReadbackReadCount == uint8(len(m.ikbdClockReadbackReads)) {
					m.ikbdClockReadbackActive = false
					m.ikbdClockReadbackComplete = true
				}
			} else if m.ikbdClockPollResponseActive &&
				m.ikbdClockPollResponseReadCount < m.ikbdClockPollResponseDelivered {
				m.ikbdClockPollResponseReads[m.ikbdClockPollResponseReadCount] = value
				m.ikbdClockPollResponseReadCount++
				if m.ikbdClockPollResponseReadCount == uint8(len(m.ikbdClockPollResponseReads)) {
					m.ikbdClockPollResponseActive = false
					m.ikbdClockPollCompleteCount++
				}
			} else if m.ikbdClockResponseActive && m.ikbdClockResponseReadCount < m.ikbdClockResponseDelivered {
				m.ikbdClockResponseReads[m.ikbdClockResponseReadCount] = value
				m.ikbdClockResponseReadCount++
				if m.ikbdClockResponseReadCount == uint8(len(m.ikbdClockResponseReads)) {
					m.ikbdClockResponseActive = false
					m.ikbdClockResponseComplete = true
				}
			} else {
				m.ikbdResetResponseRead = true
				m.ikbdStaleRDRReads = 1
			}
			return value, nil
		}
		if m.ikbdACIAConfigured && m.ikbdStaleRDRReads != 0 {
			m.ikbdStaleRDRReads--
			return m.ikbdACIARDR, nil
		}
		return 0, m.fault(address, functionCode, false, 1, FaultUnsupportedDeviceState)
	case address == MIDIACIAControl:
		if m.midiACIAConfigured {
			return m.midiACIAStatus, nil
		}
		return 0, m.fault(address, functionCode, false, 1, FaultUnsupportedDeviceState)
	case address == MIDIACIAData:
		return 0, m.fault(address, functionCode, false, 1, FaultUnsupportedDeviceState)
	case address == MFPGPIP:
		m.mfpGPIP = m.mfpGPIP&m.mfpDDR | m.mfpGPIPIn&^m.mfpDDR
		if m.fdcInitStage == 4 && m.fdcRestorePending && m.mfpGPIP&0x20 != 0 {
			m.fdcRestoreInactivePolls++
		}
		if m.fdcInitStage == 5 && m.fdcIRQ && m.mfpGPIP&0x20 == 0 {
			m.fdcRestoreIRQObserved = true
		}
		if m.fdcInitStage == 11 && m.fdcSeekPending && m.mfpGPIP&0x20 != 0 {
			m.fdcSeekInactivePolls++
		}
		if m.fdcInitStage == 12 && m.fdcIRQ && m.mfpGPIP&0x20 == 0 {
			m.fdcSeekIRQObserved = true
		}
		if m.floppyReadStage == 21 && m.fdcSeekPending && m.mfpGPIP&0x20 != 0 {
			m.floppyMediaLegacy[0].InactivePolls++
		}
		if m.floppyReadStage == 22 && m.fdcIRQ && m.mfpGPIP&0x20 == 0 {
			m.floppyMediaLegacy[0].IRQObserved = true
		}
		if m.floppyReadStage == 43 && m.fdcSeekPending && m.mfpGPIP&0x20 != 0 {
			m.floppyMediaLegacy[1].InactivePolls++
		}
		if m.floppyReadStage == 44 && m.fdcIRQ && m.mfpGPIP&0x20 == 0 {
			m.floppyMediaLegacy[1].IRQObserved = true
		}
		if m.floppyReadStage == 65 && m.fdcSeekPending && m.mfpGPIP&0x20 != 0 {
			m.floppyMediaLegacy[2].InactivePolls++
		}
		if m.floppyReadStage == 66 && m.fdcIRQ && m.mfpGPIP&0x20 == 0 {
			m.floppyMediaLegacy[2].IRQObserved = true
		}
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
		return m.mfpTSR | 0x80, nil
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
		clockReadCount := m.ikbdClockResponseReadCount
		readbackCount := m.ikbdClockReadbackReadCount
		pollCount := m.ikbdClockPollResponseReadCount
		value, err := m.ReadByte(address, access.FunctionCode)
		if err == nil && address&AddressMask == IKBDACIAData &&
			m.ikbdClockResponseReadCount == clockReadCount+1 {
			m.ikbdClockResponseReadClocks[clockReadCount] = access.Clock
		}
		if err == nil && address&AddressMask == IKBDACIAData &&
			m.ikbdClockReadbackReadCount == readbackCount+1 {
			m.ikbdClockReadbackReadClocks[readbackCount] = access.Clock
		}
		if err == nil && address&AddressMask == IKBDACIAData &&
			m.ikbdClockPollResponseReadCount == pollCount+1 {
			m.ikbdClockPollReadClocks[pollCount] = access.Clock
		}
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
	if address == STDiskController {
		if fault := m.validateAccess(address, functionCode, false, 2); fault != nil {
			return 0, fault
		}
		if m.flopVBLMediaStage == 4 && m.dmaMode == 0x0080 && m.fdcInitStage == 14 &&
			m.psgRegisters[14] == m.flopVBLTargetPort() {
			m.fdcStatus = 0xe4
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.flopVBLMediaStage = 5
			return uint16(m.fdcStatus), nil
		}
		if m.floppyReadStage == 23 && m.dmaMode == 0x0080 && m.fdcStatusTypeI &&
			m.fdcIRQ && m.mfpGPIPIn&0x20 == 0 {
			m.fdcStatus = 0xe4
			value := uint16(m.fdcStatus)
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.floppyReadStage = 24
			return value, nil
		}
		if m.floppyReadStage == 45 && m.dmaMode == 0x0080 && m.fdcStatusTypeI &&
			m.fdcIRQ && m.mfpGPIPIn&0x20 == 0 {
			m.fdcStatus = 0xe4
			value := uint16(m.fdcStatus)
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.floppyReadStage = 46
			return value, nil
		}
		if m.floppyReadStage == 67 && m.dmaMode == 0x0080 && m.fdcStatusTypeI &&
			m.fdcIRQ && m.mfpGPIPIn&0x20 == 0 {
			m.fdcStatus = 0xe4
			value := uint16(m.fdcStatus)
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.floppyReadStage = 68
			return value, nil
		}
		if (m.fdcInitStage != 6 && m.fdcInitStage != 13) || m.dmaMode != 0x0080 || !m.fdcStatusTypeI ||
			!m.fdcIRQ || m.mfpGPIPIn&0x20 != 0 {
			return 0, m.fault(address, functionCode, false, 2, FaultUnsupportedDeviceState)
		}
		m.fdcStatus = 0xe4
		value := uint16(m.fdcStatus)
		m.fdcIRQ = false
		m.mfpGPIPIn |= 0x20
		if m.fdcInitStage == 6 {
			m.fdcInitStage = 7
		} else {
			m.fdcInitStage = 14
		}
		return value, nil
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
	statusReadStage := m.fdcInitStage
	mediaReadStage := m.flopVBLMediaStage
	floppyReadStage := m.floppyReadStage
	wasStatusRead := address&AddressMask == STDiskController &&
		(statusReadStage == 6 || statusReadStage == 13)
	value, err := m.ReadWord(address, access.FunctionCode)
	if err == nil && address&AddressMask == STDiskController {
		wait += 4
		if wasStatusRead {
			if statusReadStage == 6 && m.fdcInitStage == 7 {
				m.fdcStatusReadClock = access.Clock
			}
			if statusReadStage == 13 && m.fdcInitStage == 14 {
				m.fdcSeekStatusReadClock = access.Clock
			}
		}
		if mediaReadStage == 4 && m.flopVBLMediaStage == 5 {
			m.flopVBLStatusReadClock = access.Clock
		}
		if floppyReadStage == 23 && m.floppyReadStage == 24 {
			m.floppyMediaLegacy[0].StatusReadClock = access.Clock
		}
		if floppyReadStage == 45 && m.floppyReadStage == 46 {
			m.floppyMediaLegacy[1].StatusReadClock = access.Clock
		}
		if floppyReadStage == 67 && m.floppyReadStage == 68 {
			m.floppyMediaLegacy[2].StatusReadClock = access.Clock
		}
	}
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
	if isDMAAddressByte(address) {
		validRetryAddressWrite := m.floppyReadStage == 29 && address == STDMAAddressLow && value == 0x04 ||
			m.floppyReadStage == 30 && address == STDMAAddressMiddle && value == 0x10 ||
			m.floppyReadStage == 31 && address == STDMAAddressHigh && value == 0
		validRetry3AddressWrite := m.floppyReadStage == 51 && address == STDMAAddressLow && value == 0x04 ||
			m.floppyReadStage == 52 && address == STDMAAddressMiddle && value == 0x10 ||
			m.floppyReadStage == 53 && address == STDMAAddressHigh && value == 0
		if m.floppyReadStage >= 27 && m.floppyReadStage <= 59 &&
			!validRetryAddressWrite && !validRetry3AddressWrite {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		candidate := m.dmaAddress
		switch address {
		case STDMAAddressHigh:
			candidate = candidate&0x00ffff | uint32(value)<<16
		case STDMAAddressMiddle:
			candidate = candidate&0xff00ff | uint32(value)<<8
		case STDMAAddressLow:
			candidate = candidate&0xffff00 | uint32(value)
		}
		if m.dmaAddress&0x80 != 0 && candidate&0x80 == 0 {
			candidate += 0x100
		} else if m.dmaAddress&0x8000 != 0 && candidate&0x8000 == 0 {
			candidate += 0x10000
		}
		m.dmaAddress = candidate & 0x003ffffe
		if m.floppyReadStage >= 7 && m.floppyReadStage <= 9 {
			switch {
			case m.floppyReadStage == 7 && address == STDMAAddressLow && value == 0x04:
				m.floppyReadStage = 8
				m.floppyMediaLegacy[0].DMAAddressStage = 1
			case m.floppyReadStage == 8 && address == STDMAAddressMiddle && value == 0x10:
				m.floppyReadStage = 9
				m.floppyMediaLegacy[0].DMAAddressStage = 2
			case m.floppyReadStage == 9 && address == STDMAAddressHigh && value == 0:
				m.floppyReadStage = 10
				m.floppyMediaLegacy[0].DMAAddressStage = 3
			}
		}
		if validRetryAddressWrite {
			m.floppyReadStage++
			m.floppyMediaLegacy[1].DMAAddressStage++
		}
		if validRetry3AddressWrite {
			m.floppyReadStage++
			m.floppyMediaLegacy[2].DMAAddressStage++
		}
		if m.fdcProbeDrive == 1 && m.fdcInitStage == 14 {
			switch {
			case m.dmaAddressWriteStage == 0 && address == STDMAAddressLow && value == 0x04:
				m.dmaAddressWriteStage = 1
			case m.dmaAddressWriteStage == 1 && address == STDMAAddressMiddle && value == 0x10:
				m.dmaAddressWriteStage = 2
			case m.dmaAddressWriteStage == 2 && address == STDMAAddressHigh && value == 0x00:
				m.dmaAddressWriteStage = 3
			}
		}
		return nil
	}
	if address == PSGRegisterSelect {
		if m.floppyReadStage == 2 && m.psgDriveStage == 9 && m.psgRegisterSelect == 14 &&
			m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 0x23 && value == 14 {
			m.floppyReadStage = 3
			return nil
		}
		if m.floppyReadStage == 24 && m.psgDriveStage == 9 && m.psgRegisterSelect == 14 &&
			m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 0x25 && value == 14 {
			m.floppyReadStage = 25
			return nil
		}
		if m.floppyReadStage == 46 && m.psgDriveStage == 9 && m.psgRegisterSelect == 14 &&
			m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 0x25 && value == 14 {
			m.floppyReadStage = 47
			return nil
		}
		if m.psgDriveStage == 9 && (m.flopVBLMediaStage == 0 || m.flopVBLMediaStage == 8) &&
			m.ikbdClockReadbackComplete &&
			m.fdcInitStage == 14 && m.acsiStage == 5 && m.psgRegisterSelect == 14 &&
			m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 0x23 && value == 14 {
			m.flopVBLMediaDrive = int8(m.flopVBLMediaChecks & 1)
			m.flopVBLMediaStage = 1
			return nil
		}
		if m.psgDriveStage == 9 && m.flopVBLMediaStage == 5 && m.psgRegisterSelect == 14 &&
			m.psgRegisters[14] == m.flopVBLTargetPort() && value == 14 {
			m.flopVBLMediaStage = 6
			return nil
		}
		if m.psgDriveStage == 6 && m.acsiStage == 5 && m.fdcInitStage == 14 &&
			m.psgRegisterSelect == 14 && m.psgRegisters[7] == 0xc0 &&
			m.psgRegisters[14] == 3 && value == 14 {
			m.psgDriveStage = 7
			return nil
		}
		if m.psgDriveStage == 3 && m.fdcInitStage == 14 && m.psgRegisterSelect == 14 &&
			m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 5 && value == 14 {
			m.psgDriveStage = 4
			return nil
		}
		if m.psgDriveStage == 0 && m.psgRegisterSelect == 14 &&
			m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 7 && value == 14 {
			m.psgDriveStage = 1
			return nil
		}
		if m.psgRegisterSelect == 0 && m.psgRegisters[7] == 0 && m.psgRegisters[14] == 0 && value == 7 ||
			m.psgRegisterSelect == 7 && m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 0 && value == 14 {
			m.psgRegisterSelect = value
			return nil
		}
		return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
	}
	if address == PSGRegisterData {
		if m.floppyReadStage == 4 && m.psgDriveStage == 9 && m.psgRegisterSelect == 14 &&
			m.psgRegisters[14] == 0x23 && value == 0x25 {
			m.psgRegisters[14] = value
			m.floppyMediaLegacy[0].Drive = 0
			m.floppyReadStage = 5
			return nil
		}
		if m.floppyReadStage == 26 && m.psgDriveStage == 9 && m.psgRegisterSelect == 14 &&
			m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 0x25 && value == 0x25 {
			m.psgRegisters[14] = value
			m.floppyMediaLegacy[1].DrivePort = value
			m.floppyReadStage = 27
			return nil
		}
		if m.floppyReadStage == 48 && m.psgDriveStage == 9 && m.psgRegisterSelect == 14 &&
			m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 0x25 && value == 0x25 {
			m.psgRegisters[14] = value
			m.floppyMediaLegacy[2].DrivePort = value
			m.floppyReadStage = 49
			return nil
		}
		if m.psgDriveStage == 9 && m.flopVBLMediaStage == 2 && m.psgRegisterSelect == 14 &&
			m.psgRegisters[14] == 0x23 && value == m.flopVBLTargetPort() {
			m.psgRegisters[14] = value
			m.flopVBLMediaStage = 3
			return nil
		}
		if m.psgDriveStage == 9 && m.flopVBLMediaStage == 7 && m.psgRegisterSelect == 14 &&
			m.psgRegisters[14] == m.flopVBLTargetPort() && value == 0x23 {
			m.psgRegisters[14] = value
			m.flopVBLMediaStage = 8
			m.flopVBLMediaComplete = true
			m.flopVBLMediaChecks++
			return nil
		}
		if m.psgDriveStage == 8 && m.acsiStage == 5 && m.fdcInitStage == 14 &&
			m.psgRegisterSelect == 14 && m.psgRegisters[7] == 0xc0 &&
			m.psgRegisters[14] == 3 && value == 0x23 {
			m.psgRegisters[14] = value
			m.psgDriveStage = 9
			return nil
		}
		if m.psgDriveStage == 5 && m.fdcInitStage == 14 && m.psgRegisterSelect == 14 &&
			m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 5 && value == 3 {
			m.psgRegisters[14] = value
			m.psgDriveStage = 6
			return nil
		}
		if m.psgDriveStage == 2 && m.psgRegisterSelect == 14 &&
			m.psgRegisters[7] == 0xc0 && m.psgRegisters[14] == 7 && value == 5 {
			m.psgRegisters[14] = value
			m.psgDriveStage = 3
			return nil
		}
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
		validClockRequest := value == 0x1c && m.ikbdACIATDR == 1 &&
			m.ikbdACIATXShiftTicks == 0 && m.psgDriveStage == 9 && m.acsiStage == 5 &&
			m.ikbdResetResponseRead && m.ikbdStaleRDRReads == 0 && !m.ikbdClockRequestHandled
		validSetClock := m.ikbdClockResponseComplete && !m.ikbdSetClockComplete &&
			int(m.ikbdSetClockWriteCount) < len(ikbdSetClockPacket) &&
			value == ikbdSetClockPacket[m.ikbdSetClockWriteCount]
		validClockReadback := value == 0x1c && m.ikbdSetClockWriteCount == 7 &&
			m.ikbdSetClockCompleteCount == 6 && !m.ikbdSetClockComplete &&
			m.ikbdACIATXShiftTicks != 0 && !m.ikbdClockReadbackRequestWritten
		validClockPoll := value == 0x1c && m.ikbdClockReadbackComplete && m.flopVBLMediaComplete &&
			m.ikbdACIATXShiftTicks == 0 && !m.ikbdClockPollRequestWritten &&
			!m.ikbdClockPollResponseActive &&
			m.ikbdClockPollRequestCount == m.ikbdClockPollCompleteCount
		if m.ikbdACIAConfigured && m.ikbdACIAStatus&2 != 0 && !m.ikbdACIATXPending &&
			(validFirst || validSecond || validClockRequest || validSetClock || validClockReadback || validClockPoll) {
			m.ikbdACIATDR = value
			m.ikbdACIATXPending = true
			m.ikbdACIAStatus &^= 2
			if validSetClock {
				m.ikbdSetClockWrites[m.ikbdSetClockWriteCount] = value
				m.ikbdSetClockWriteCount++
			} else if validClockReadback {
				m.ikbdClockReadbackRequestWritten = true
			} else if validClockPoll {
				m.ikbdClockPollRequestWritten = true
				m.ikbdClockPollResponseDelivered = 0
				m.ikbdClockPollResponseReadCount = 0
				m.ikbdClockPollResponseReads = [7]byte{}
				m.ikbdClockPollDeliveryClocks = [7]uint64{}
				m.ikbdClockPollReadClocks = [7]uint64{}
			}
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
		if m.mfpTimerDStopStage == 7 && m.mfpUSARTReconfigStage == 0 &&
			m.mfpTCDCR == 0x50 && m.mfpTDDR == 0 && m.mfpTDMain == 0 && value == 0x50 {
			m.mfpUSARTReconfigStage = 1
			return nil
		}
		if m.mfpUSARTReconfigStage == 2 && m.mfpTCDCR == 0x50 && value == 0x51 &&
			m.mfpTDDR == 2 && m.mfpTDMain == 2 {
			m.mfpTCDCR = value
			m.mfpTimerDStart = true
			m.mfpUSARTReconfigStage = 3
			return nil
		}
		if m.mfpTimerDStopStage == 7 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
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
		if m.mfpTimerDStopStage == 7 {
			if m.mfpUSARTReconfigStage != 1 || value != 2 {
				return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
			}
			m.mfpTDDR, m.mfpTDMain = value, value
			m.mfpUSARTReconfigStage = 2
			return nil
		}
		m.mfpTDDR, m.mfpTDMain = value, value
		if m.mfpTimerDSystemStage == 4 && value == 0 {
			m.mfpTimerDSystemStage = 5
		}
		return nil
	}
	if address == MFPSCR {
		if m.mfpUSARTReconfigStage == 6 && m.mfpSCR == 0 && value == 0 {
			m.mfpUSARTReconfigStage = 7
			return nil
		}
		if m.mfpTimerDStopStage == 7 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		if m.mfpSCR != 0 || value != 0 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
		return nil
	}
	if address == MFPUCR {
		if m.mfpUSARTReconfigStage == 3 && m.mfpUCR == 0x88 && value == 0x88 {
			m.mfpUSARTReconfigStage = 4
			return nil
		}
		if m.mfpTimerDStopStage == 7 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
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
		if m.mfpUSARTReconfigStage == 4 && m.mfpRSR == 1 && value == 1 {
			m.mfpUSARTReconfigStage = 5
			return nil
		}
		if m.mfpTimerDStopStage == 7 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
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
		if m.mfpUSARTReconfigStage == 5 && m.mfpTSRSet && m.mfpTSR == 1 && value == 1 {
			m.mfpUSARTReconfigStage = 6
			return nil
		}
		if m.mfpTimerDStopStage == 7 {
			return m.fault(address, functionCode, true, 1, FaultUnsupportedDeviceState)
		}
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
	if address == STVoidDMAByte {
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
		floppyReadStage := m.floppyReadStage
		err := m.WriteByte(address, value, access.FunctionCode)
		if err == nil && !wasTimerC && m.mfpTimerCStart {
			m.mfpTimerCStartClock = access.Clock
		}
		if err == nil && !wasSystemTimerD && m.mfpTimerDSystemStage == 8 && m.mfpTimerDStart &&
			m.mfpTCDCR&0x07 == 2 {
			m.mfpTimerDStartClock = access.Clock
		}
		if err == nil && floppyReadStage == 26 && m.floppyReadStage == 27 {
			m.floppyMediaLegacy[1].DriveWriteClock = access.Clock
		}
		if err == nil && floppyReadStage == 48 && m.floppyReadStage == 49 {
			m.floppyMediaLegacy[2].DriveWriteClock = access.Clock
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
	if address == STDMAControl || address == STDiskController {
		if fault := m.validateAccess(address, functionCode, true, 2); fault != nil {
			return fault
		}
		if address == STDMAControl && m.floppyReadStage == 0 && m.flopVBLMediaStage == 8 &&
			m.flopVBLMediaChecks != 0 && m.fdcInitStage == 14 && m.acsiStage == 5 &&
			m.psgDriveStage == 9 && m.dmaMode == 0x0080 && value == 0x0082 {
			m.dmaMode = value
			m.floppyReadStage = 1
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 1 && m.dmaMode == 0x0082 && value == 0 {
			m.floppyMediaLegacy[0].Track = 0
			m.floppyReadStage = 2
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 5 && m.dmaMode == 0x0082 && value == 0x0084 {
			m.dmaMode = value
			m.floppyReadStage = 6
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 6 && m.dmaMode == 0x0084 && value == 1 {
			m.floppyMediaLegacy[0].Sector = 1
			m.floppyReadStage = 7
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 10 &&
			m.floppyMediaLegacy[0].DMAAddressStage == 3 && value == 0x0190 {
			m.dmaMode = value
			m.dmaSectorCount = 0
			m.floppyMediaLegacy[0].DMAResetCount = 1
			m.floppyReadStage = 11
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 11 && m.dmaMode == 0x0190 && value == 0x0090 {
			m.dmaMode = value
			m.dmaSectorCount = 0
			m.floppyMediaLegacy[0].DMAResetCount = 2
			m.floppyReadStage = 12
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 12 && m.dmaMode == 0x0090 && value == 1 {
			m.dmaSectorCount = 1
			m.floppyReadStage = 13
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 13 && m.dmaMode == 0x0090 && value == 0x0080 {
			m.dmaMode = value
			m.floppyReadStage = 14
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 14 && m.dmaMode == 0x0080 && value == 0x0080 {
			m.floppyMediaLegacy[0].ReadCommand = 0x80
			m.fdcCommand = 0x80
			m.fdcStatus = 0x81
			m.fdcStatusTypeI = false
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.floppyReadStage = 15
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 15 && m.dmaMode == 0x0080 &&
			m.floppyMediaLegacy[0].ReadCommand == 0x80 && m.fdcCommand == 0x80 && m.fdcStatus == 0x81 &&
			!m.fdcStatusTypeI && !m.fdcIRQ && value == 0x0080 {
			m.floppyReadStage = 16
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 16 && m.dmaMode == 0x0080 &&
			m.floppyMediaLegacy[0].ReadCommand == 0x80 && m.fdcCommand == 0x80 && m.fdcStatus == 0x81 &&
			!m.fdcStatusTypeI && !m.fdcIRQ && value == 0x00d0 {
			m.floppyMediaLegacy[0].ForceInterrupt = 0xd0
			m.fdcCommand = 0xd0
			m.fdcStatus = 0x80
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.floppyReadStage = 17
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 17 && m.dmaMode == 0x0080 &&
			m.fdcCommand == 0xd0 && m.fdcStatus == 0x80 && !m.fdcStatusTypeI &&
			!m.fdcIRQ && value == 0x0086 {
			m.dmaMode = value
			m.floppyReadStage = 18
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 18 && m.dmaMode == 0x0086 &&
			value == 0 {
			m.floppyMediaLegacy[0].SeekData = 0
			m.fdcData = 0
			m.floppyReadStage = 19
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 19 && m.dmaMode == 0x0086 &&
			value == 0x0080 {
			m.dmaMode = value
			m.floppyReadStage = 20
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 20 && m.dmaMode == 0x0080 &&
			m.floppyMediaLegacy[0].SeekData == 0 && value == 0x0013 {
			m.floppyMediaLegacy[0].SeekCommand = 0x13
			m.fdcCommand = 0x13
			m.fdcStatus = 0xe5
			m.fdcStatusTypeI = true
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.fdcSeekPending = true
			m.floppyReadStage = 21
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 22 && m.dmaMode == 0x0080 &&
			m.fdcStatusTypeI && m.fdcIRQ && m.mfpGPIPIn&0x20 == 0 && value == 0x0080 {
			m.floppyReadStage = 23
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 27 && m.dmaMode == 0x0080 &&
			m.floppyMediaLegacy[1].DrivePort == 0x25 && value == 0x0084 {
			m.dmaMode = value
			m.floppyReadStage = 28
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 28 && m.dmaMode == 0x0084 && value == 1 {
			m.floppyMediaLegacy[1].Sector = 1
			m.floppyReadStage = 29
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 32 &&
			m.floppyMediaLegacy[1].DMAAddressStage == 3 && value == 0x0190 {
			m.dmaMode = value
			m.dmaSectorCount = 0
			m.floppyMediaLegacy[1].DMAResetCount = 1
			m.floppyReadStage = 33
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 33 && m.dmaMode == 0x0190 && value == 0x0090 {
			m.dmaMode = value
			m.dmaSectorCount = 0
			m.floppyMediaLegacy[1].DMAResetCount = 2
			m.floppyReadStage = 34
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 34 && m.dmaMode == 0x0090 && value == 1 {
			m.dmaSectorCount = 1
			m.floppyReadStage = 35
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 35 && m.dmaMode == 0x0090 && value == 0x0080 {
			m.dmaMode = value
			m.floppyReadStage = 36
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 36 && m.dmaMode == 0x0080 && value == 0x0080 {
			m.floppyMediaLegacy[1].ReadCommand = 0x80
			m.fdcCommand = 0x80
			m.fdcStatus = 0x81
			m.fdcStatusTypeI = false
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.floppyReadStage = 37
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 37 && m.dmaMode == 0x0080 &&
			m.floppyMediaLegacy[1].ReadCommand == 0x80 && m.fdcCommand == 0x80 && m.fdcStatus == 0x81 &&
			!m.fdcStatusTypeI && !m.fdcIRQ && value == 0x0080 {
			m.floppyReadStage = 38
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 38 && m.dmaMode == 0x0080 &&
			m.floppyMediaLegacy[1].ReadCommand == 0x80 && m.fdcCommand == 0x80 && m.fdcStatus == 0x81 &&
			!m.fdcStatusTypeI && !m.fdcIRQ && value == 0x00d0 {
			m.floppyMediaLegacy[1].ForceInterrupt = 0xd0
			m.fdcCommand = 0xd0
			m.fdcStatus = 0x80
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.floppyReadStage = 39
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 39 && m.dmaMode == 0x0080 &&
			m.fdcCommand == 0xd0 && m.fdcStatus == 0x80 && !m.fdcStatusTypeI &&
			!m.fdcIRQ && value == 0x0086 {
			m.dmaMode = value
			m.floppyReadStage = 40
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 40 && m.dmaMode == 0x0086 && value == 0 {
			m.floppyMediaLegacy[1].SeekData = 0
			m.fdcData = 0
			m.floppyReadStage = 41
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 41 && m.dmaMode == 0x0086 && value == 0x0080 {
			m.dmaMode = value
			m.floppyReadStage = 42
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 42 && m.dmaMode == 0x0080 &&
			m.floppyMediaLegacy[1].SeekData == 0 && value == 0x0013 {
			m.floppyMediaLegacy[1].SeekCommand = 0x13
			m.fdcCommand = 0x13
			m.fdcStatus = 0xe5
			m.fdcStatusTypeI = true
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.fdcSeekPending = true
			m.floppyReadStage = 43
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 44 && m.dmaMode == 0x0080 &&
			m.fdcStatusTypeI && m.fdcIRQ && m.mfpGPIPIn&0x20 == 0 && value == 0x0080 {
			m.floppyReadStage = 45
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 49 && m.dmaMode == 0x0080 &&
			m.floppyMediaLegacy[2].DrivePort == 0x25 && value == 0x0084 {
			m.dmaMode = value
			m.floppyReadStage = 50
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 50 && m.dmaMode == 0x0084 && value == 1 {
			m.floppyMediaLegacy[2].Sector = 1
			m.floppyReadStage = 51
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 54 &&
			m.floppyMediaLegacy[2].DMAAddressStage == 3 && value == 0x0190 {
			m.dmaMode = value
			m.dmaSectorCount = 0
			m.floppyMediaLegacy[2].DMAResetCount = 1
			m.floppyReadStage = 55
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 55 && m.dmaMode == 0x0190 && value == 0x0090 {
			m.dmaMode = value
			m.dmaSectorCount = 0
			m.floppyMediaLegacy[2].DMAResetCount = 2
			m.floppyReadStage = 56
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 56 && m.dmaMode == 0x0090 && value == 1 {
			m.dmaSectorCount = 1
			m.floppyReadStage = 57
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 57 && m.dmaMode == 0x0090 && value == 0x0080 {
			m.dmaMode = value
			m.floppyReadStage = 58
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 58 && m.dmaMode == 0x0080 && value == 0x0080 {
			m.floppyMediaLegacy[2].ReadCommand = 0x80
			m.fdcCommand = 0x80
			m.fdcStatus = 0x81
			m.fdcStatusTypeI = false
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.floppyReadStage = 59
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 59 && m.dmaMode == 0x0080 &&
			m.floppyMediaLegacy[2].ReadCommand == 0x80 && m.fdcCommand == 0x80 && m.fdcStatus == 0x81 &&
			!m.fdcStatusTypeI && !m.fdcIRQ && value == 0x0080 {
			m.floppyReadStage = 60
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 60 && m.dmaMode == 0x0080 &&
			m.floppyMediaLegacy[2].ReadCommand == 0x80 && m.fdcCommand == 0x80 && m.fdcStatus == 0x81 &&
			!m.fdcStatusTypeI && !m.fdcIRQ && value == 0x00d0 {
			m.floppyMediaLegacy[2].ForceInterrupt = 0xd0
			m.fdcCommand = 0xd0
			m.fdcStatus = 0x80
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.floppyReadStage = 61
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 61 && m.dmaMode == 0x0080 &&
			m.fdcCommand == 0xd0 && m.fdcStatus == 0x80 && !m.fdcStatusTypeI &&
			!m.fdcIRQ && value == 0x0086 {
			m.dmaMode = value
			m.floppyReadStage = 62
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 62 && m.dmaMode == 0x0086 && value == 0 {
			m.floppyMediaLegacy[2].SeekData = 0
			m.fdcData = 0
			m.floppyReadStage = 63
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 63 && m.dmaMode == 0x0086 && value == 0x0080 {
			m.dmaMode = value
			m.floppyReadStage = 64
			return nil
		}
		if address == STDiskController && m.floppyReadStage == 64 && m.dmaMode == 0x0080 &&
			m.floppyMediaLegacy[2].SeekData == 0 && value == 0x0013 {
			m.floppyMediaLegacy[2].SeekCommand = 0x13
			m.fdcCommand = 0x13
			m.fdcStatus = 0xe5
			m.fdcStatusTypeI = true
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.fdcSeekPending = true
			m.floppyReadStage = 65
			return nil
		}
		if address == STDMAControl && m.floppyReadStage == 66 && m.dmaMode == 0x0080 &&
			m.fdcStatusTypeI && m.fdcIRQ && m.mfpGPIPIn&0x20 == 0 && value == 0x0080 {
			m.floppyReadStage = 67
			return nil
		}
		if address == STDMAControl && m.flopVBLMediaStage == 3 && m.psgDriveStage == 9 &&
			m.fdcInitStage == 14 && m.psgRegisters[14] == m.flopVBLTargetPort() && value == 0x0080 {
			m.dmaMode = value
			m.flopVBLMediaStage = 4
			return nil
		}
		if address == STDMAControl && m.fdcProbeDrive == 1 && m.fdcInitStage == 14 &&
			m.dmaAddressWriteStage == 3 && m.dmaInitStage == 0 && m.dmaMode == 0x0080 && value == 0x0190 {
			m.dmaMode = value
			m.dmaSectorCount = 0
			m.dmaResetCount = 1
			m.dmaInitStage = 1
			return nil
		}
		if address == STDMAControl && m.fdcProbeDrive == 1 && m.fdcInitStage == 14 &&
			m.dmaAddressWriteStage == 3 && m.dmaInitStage == 1 && m.dmaMode == 0x0190 && value == 0x0090 {
			m.dmaMode = value
			m.dmaSectorCount = 0
			m.dmaResetCount = 2
			m.dmaInitStage = 2
			return nil
		}
		if address == STDiskController && m.fdcProbeDrive == 1 && m.fdcInitStage == 14 &&
			m.dmaAddressWriteStage == 3 && m.dmaInitStage == 2 && m.dmaMode&0x0010 != 0 && value == 0 {
			m.dmaSectorCount = byte(value)
			m.dmaInitStage = 3
			return nil
		}
		if address == STDMAControl && m.dmaInitStage == 3 && m.dmaMode == 0x0090 &&
			value == 0x0088 && ((m.acsiStage == 0 && m.acsiTarget == -1) ||
			(m.acsiStage == 4 && m.acsiTarget >= 0 && m.acsiTarget < 7)) {
			m.dmaMode = value
			m.acsiTarget++
			m.acsiStage = 1
			return nil
		}
		expectedACSICommand := uint16(uint8(m.acsiTarget) << 5)
		if address == STDiskController && m.acsiStage == 1 && m.acsiTarget >= 0 &&
			m.acsiTarget <= 7 && m.dmaMode == 0x0088 && value == expectedACSICommand {
			m.acsiCommand = byte(value)
			m.acsiAttemptMask |= 1 << uint8(m.acsiTarget)
			m.acsiCommandReceipts[m.acsiTarget] = byte(value)
			m.acsiStage = 2
			return nil
		}
		if address == STDMAControl && m.acsiStage == 2 && m.acsiTarget >= 0 &&
			m.acsiTarget <= 7 && uint16(m.acsiCommand) == expectedACSICommand &&
			m.dmaMode == 0x0088 && value == 0x008a {
			m.dmaMode = value
			m.acsiStage = 3
			return nil
		}
		if address == STDMAControl && m.acsiStage == 3 && m.acsiTarget >= 0 &&
			m.acsiTarget <= 7 && uint16(m.acsiCommand) == expectedACSICommand &&
			m.dmaMode == 0x008a && value == 0x0080 {
			m.dmaMode = value
			m.dmaResetCount = 0
			m.dmaInitStage = 0
			if m.acsiTarget == 7 {
				m.acsiStage = 5
			} else {
				m.acsiStage = 4
			}
			return nil
		}
		if address == STDMAControl && m.fdcInitStage == 2 && m.dmaMode == 0x0080 && value == 0x0080 {
			m.fdcInitStage = 3
			return nil
		}
		if address == STDMAControl && m.fdcInitStage == 5 && m.dmaMode == 0x0080 && value == 0x0080 {
			m.fdcInitStage = 6
			return nil
		}
		if address == STDMAControl && m.fdcInitStage == 7 && m.dmaMode == 0x0080 && value == 0x0086 {
			m.dmaMode = value
			m.fdcInitStage = 8
			return nil
		}
		if address == STDiskController && m.fdcInitStage == 8 && m.dmaMode == 0x0086 && value == 0x0000 {
			m.fdcData = byte(value)
			m.fdcInitStage = 9
			return nil
		}
		if address == STDMAControl && m.fdcInitStage == 9 && m.dmaMode == 0x0086 && value == 0x0080 {
			m.dmaMode = value
			m.fdcInitStage = 10
			return nil
		}
		if address == STDiskController && m.fdcInitStage == 10 && m.dmaMode == 0x0080 &&
			m.fdcData == 0 && value == 0x0013 {
			m.fdcCommand = byte(value)
			m.fdcStatus = 0xe5
			m.fdcStatusTypeI = true
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.fdcSeekPending = true
			m.fdcInitStage = 11
			return nil
		}
		if address == STDMAControl && m.fdcInitStage == 12 && m.dmaMode == 0x0080 && value == 0x0080 {
			m.fdcInitStage = 13
			return nil
		}
		if address == STDiskController && m.fdcInitStage == 3 && m.dmaMode == 0x0080 && value == 0x000b {
			m.fdcCommand = byte(value)
			m.fdcStatus = 0x81
			m.fdcStatusTypeI = true
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.fdcRestorePending = true
			m.fdcInitStage = 4
			return nil
		}
		if address == STDMAControl && m.fdcInitStage == 0 && m.psgDriveStage == 3 && value == 0x0080 {
			m.dmaMode = value
			m.fdcProbeDrive = 0
			m.fdcInitStage = 1
			return nil
		}
		if address == STDMAControl && m.fdcProbeDrive == 0 && m.fdcInitStage == 14 && m.psgDriveStage == 6 &&
			m.psgRegisterSelect == 14 && m.psgRegisters[14] == 3 && value == 0x0080 {
			m.dmaMode = value
			m.fdcProbeDrive = 1
			m.fdcRestorePending = false
			m.fdcRestoreStartClock = 0
			m.fdcRestoreInactivePolls = 0
			m.fdcRestoreIRQObserved = false
			m.fdcStatusReadClock = 0
			m.fdcData = 0
			m.fdcSeekPending = false
			m.fdcSeekStartClock = 0
			m.fdcSeekInactivePolls = 0
			m.fdcSeekIRQObserved = false
			m.fdcSeekStatusReadClock = 0
			m.fdcInitStage = 1
			return nil
		}
		if address == STDiskController && m.fdcInitStage == 1 && m.dmaMode == 0x0080 && value == 0x00d0 {
			m.fdcCommand = byte(value)
			m.fdcStatus = 0x80
			m.fdcStatusTypeI = true
			m.fdcIRQ = false
			m.mfpGPIPIn |= 0x20
			m.fdcInitStage = 2
			return nil
		}
		return m.fault(address, functionCode, true, 2, FaultUnsupportedDeviceState)
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
	wasRestorePending := m.fdcRestorePending
	wasSeekPending := m.fdcSeekPending
	acsiStage := m.acsiStage
	floppyReadStage := m.floppyReadStage
	err = m.WriteWord(address, value, access.FunctionCode)
	if err == nil {
		address &= AddressMask
		if address == STDMAControl || address == STDiskController {
			wait += 4
		}
		if !wasRestorePending && m.fdcRestorePending {
			m.fdcRestoreStartClock = access.Clock
		}
		if !wasSeekPending && m.fdcSeekPending {
			m.fdcSeekStartClock = access.Clock
		}
		if acsiStage == 3 && (m.acsiStage == 4 || m.acsiStage == 5) {
			m.acsiTimeoutReturnClock = access.Clock
			m.acsiTimeoutReturnClocks[m.acsiTarget] = access.Clock
		}
		if floppyReadStage == 1 && m.floppyReadStage == 2 {
			m.floppyMediaLegacy[0].TrackWriteClock = access.Clock
		}
		if floppyReadStage == 14 && m.floppyReadStage == 15 {
			m.floppyMediaLegacy[0].ReadCommandClock = access.Clock
		}
		if floppyReadStage == 15 && m.floppyReadStage == 16 {
			m.floppyMediaLegacy[0].TimeoutSelectorClock = access.Clock
		}
		if floppyReadStage == 16 && m.floppyReadStage == 17 {
			m.floppyMediaLegacy[0].ForceInterruptClock = access.Clock
		}
		if floppyReadStage == 20 && m.floppyReadStage == 21 {
			m.floppyMediaLegacy[0].SeekStartClock = access.Clock
		}
		if floppyReadStage == 36 && m.floppyReadStage == 37 {
			m.floppyMediaLegacy[1].ReadCommandClock = access.Clock
		}
		if floppyReadStage == 37 && m.floppyReadStage == 38 {
			m.floppyMediaLegacy[1].TimeoutSelectorClock = access.Clock
		}
		if floppyReadStage == 38 && m.floppyReadStage == 39 {
			m.floppyMediaLegacy[1].ForceInterruptClock = access.Clock
		}
		if floppyReadStage == 42 && m.floppyReadStage == 43 {
			m.floppyMediaLegacy[1].SeekStartClock = access.Clock
		}
		if floppyReadStage == 58 && m.floppyReadStage == 59 {
			m.floppyMediaLegacy[2].ReadCommandClock = access.Clock
		}
		if floppyReadStage == 59 && m.floppyReadStage == 60 {
			m.floppyMediaLegacy[2].TimeoutSelectorClock = access.Clock
		}
		if floppyReadStage == 60 && m.floppyReadStage == 61 {
			m.floppyMediaLegacy[2].ForceInterruptClock = access.Clock
		}
		if floppyReadStage == 64 && m.floppyReadStage == 65 {
			m.floppyMediaLegacy[2].SeekStartClock = access.Clock
		}
	}
	return wait, err
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
	m.psgDriveStage = 0
	m.flopVBLMediaStage = 0
	m.flopVBLStatusReadClock = 0
	m.flopVBLMediaComplete = false
	m.flopVBLMediaChecks = 0
	m.flopVBLMediaDrive = -1
	m.floppyReadStage = 0
	m.floppyMediaLegacy[0].Track = 0
	m.floppyMediaLegacy[0].Drive = -1
	m.floppyMediaLegacy[0].TrackWriteClock = 0
	m.floppyMediaLegacy[0].Sector = 0
	m.floppyMediaLegacy[0].DMAAddressStage = 0
	m.floppyMediaLegacy[0].DMAResetCount = 0
	m.floppyMediaLegacy[0].ReadCommand = 0
	m.floppyMediaLegacy[0].ReadCommandClock = 0
	m.floppyMediaLegacy[0].TimeoutSelectorClock = 0
	m.floppyMediaLegacy[0].ForceInterrupt = 0
	m.floppyMediaLegacy[0].ForceInterruptClock = 0
	m.floppyMediaLegacy[0].SeekData = 0
	m.floppyMediaLegacy[0].SeekCommand = 0
	m.floppyMediaLegacy[0].SeekStartClock = 0
	m.floppyMediaLegacy[0].InactivePolls = 0
	m.floppyMediaLegacy[0].IRQObserved = false
	m.floppyMediaLegacy[0].StatusReadClock = 0
	m.floppyMediaLegacy[1].DrivePort = 0
	m.floppyMediaLegacy[1].DriveWriteClock = 0
	m.floppyMediaLegacy[1].Sector = 0
	m.floppyMediaLegacy[1].DMAAddressStage = 0
	m.floppyMediaLegacy[1].DMAResetCount = 0
	m.floppyMediaLegacy[1].ReadCommand = 0
	m.floppyMediaLegacy[1].ReadCommandClock = 0
	m.floppyMediaLegacy[1].TimeoutSelectorClock = 0
	m.floppyMediaLegacy[1].ForceInterrupt = 0
	m.floppyMediaLegacy[1].ForceInterruptClock = 0
	m.floppyMediaLegacy[1].SeekData = 0
	m.floppyMediaLegacy[1].SeekCommand = 0
	m.floppyMediaLegacy[1].SeekStartClock = 0
	m.floppyMediaLegacy[1].InactivePolls = 0
	m.floppyMediaLegacy[1].IRQObserved = false
	m.floppyMediaLegacy[1].StatusReadClock = 0
	m.floppyMediaLegacy[2].DrivePort = 0
	m.floppyMediaLegacy[2].DriveWriteClock = 0
	m.floppyMediaLegacy[2].Sector = 0
	m.floppyMediaLegacy[2].DMAAddressStage = 0
	m.floppyMediaLegacy[2].DMAResetCount = 0
	m.floppyMediaLegacy[2].ReadCommand = 0
	m.floppyMediaLegacy[2].ReadCommandClock = 0
	m.floppyMediaLegacy[2].TimeoutSelectorClock = 0
	m.floppyMediaLegacy[2].ForceInterrupt = 0
	m.floppyMediaLegacy[2].ForceInterruptClock = 0
	m.floppyMediaLegacy[2].SeekData = 0
	m.floppyMediaLegacy[2].SeekCommand = 0
	m.floppyMediaLegacy[2].SeekStartClock = 0
	m.floppyMediaLegacy[2].InactivePolls = 0
	m.floppyMediaLegacy[2].IRQObserved = false
	m.floppyMediaLegacy[2].StatusReadClock = 0
	m.dmaMode = 0
	m.dmaAddress = 0
	m.dmaAddressWriteStage = 0
	m.dmaSectorCount = 0
	m.dmaResetCount = 0
	m.dmaInitStage = 0
	m.acsiStage = 0
	m.acsiTarget = -1
	m.acsiCommand = 0
	m.acsiAttemptMask = 0
	m.acsiCommandReceipts = [8]byte{}
	m.acsiTimeoutReturnClock = 0
	m.acsiTimeoutReturnClocks = [8]uint64{}
	m.fdcCommand = 0
	m.fdcStatus = 0
	m.fdcStatusTypeI = false
	m.fdcIRQ = false
	m.fdcInitStage = 0
	m.fdcProbeDrive = -1
	m.fdcRestorePending = false
	m.fdcRestoreStartClock = 0
	m.fdcRestoreInactivePolls = 0
	m.fdcRestoreIRQObserved = false
	m.fdcStatusReadClock = 0
	m.fdcData = 0
	m.fdcSeekPending = false
	m.fdcSeekStartClock = 0
	m.fdcSeekInactivePolls = 0
	m.fdcSeekIRQObserved = false
	m.fdcSeekStatusReadClock = 0
	m.mfpGPIPIn |= 0x20
	m.ikbdACIAControl = 0
	m.ikbdACIAStatus = 0
	m.ikbdACIAConfigured = false
	m.ikbdACIATDR = 0
	m.ikbdACIATXShift = 0
	m.ikbdACIATXPending = false
	m.ikbdACIATXShiftTicks = 0
	m.ikbdResetCommandDone = false
	m.ikbdResetCommandHandled = false
	m.ikbdClockRequestDone = false
	m.ikbdClockRequestHandled = false
	m.ikbdACIARDR = 0
	m.ikbdClockResponseActive = false
	m.ikbdClockResponseDelivered = 0
	m.ikbdClockResponseReadCount = 0
	m.ikbdClockResponseReads = [7]byte{}
	m.ikbdClockResponseReadClocks = [7]uint64{}
	m.ikbdClockResponseComplete = false
	m.ikbdSetClockWrites = [7]byte{}
	m.ikbdSetClockWriteCount = 0
	m.ikbdSetClockCompletions = [7]byte{}
	m.ikbdSetClockCompleteCount = 0
	m.ikbdSetClockCompletionClocks = [7]uint64{}
	m.ikbdSetClockComplete = false
	m.ikbdClockReadbackRequestWritten = false
	m.ikbdClockReadbackRequestDone = false
	m.ikbdClockReadbackRequestHandled = false
	m.ikbdClockReadbackActive = false
	m.ikbdClockReadbackDelivered = 0
	m.ikbdClockReadbackReadCount = 0
	m.ikbdClockReadbackReads = [7]byte{}
	m.ikbdClockReadbackDeliveryClocks = [7]uint64{}
	m.ikbdClockReadbackReadClocks = [7]uint64{}
	m.ikbdClockReadbackComplete = false
	m.ikbdClockPollRequestWritten = false
	m.ikbdClockPollRequestCount = 0
	m.ikbdClockPollResponseActive = false
	m.ikbdClockPollResponseDelivered = 0
	m.ikbdClockPollResponseReadCount = 0
	m.ikbdClockPollResponseReads = [7]byte{}
	m.ikbdClockPollDeliveryClocks = [7]uint64{}
	m.ikbdClockPollReadClocks = [7]uint64{}
	m.ikbdClockPollCompleteCount = 0
	m.ikbdResetResponseRead = false
	m.ikbdStaleRDRReads = 0
	m.midiACIAControl = 0
	m.midiACIAStatus = 0
	m.midiACIAConfigured = false
	m.mfpACIAEnableStage = 0
	m.mfpTimerDSystemStage = 0
	m.mfpTimerDStopStage = 0
	m.mfpUSARTReconfigStage = 0
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

func (m *Memory) completeFDCRestore() {
	if !m.fdcRestorePending {
		return
	}
	m.fdcStatus = 0x84
	m.fdcIRQ = true
	m.mfpGPIPIn &^= 0x20
	m.fdcRestorePending = false
	m.fdcInitStage = 5
}

func (m *Memory) completeFDCSeek() {
	if !m.fdcSeekPending {
		return
	}
	m.fdcStatus = 0xe4
	m.fdcIRQ = true
	m.mfpGPIPIn &^= 0x20
	m.fdcSeekPending = false
	if m.floppyReadStage == 21 {
		m.floppyReadStage = 22
	} else if m.floppyReadStage == 43 {
		m.floppyReadStage = 44
	} else if m.floppyReadStage == 65 {
		m.floppyReadStage = 66
	} else {
		m.fdcInitStage = 12
	}
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

func (m *Memory) advanceIKBDACIAClock(clocks ...uint64) {
	hadShift := m.ikbdACIATXShiftTicks != 0
	if hadShift {
		m.ikbdACIATXShiftTicks--
	}
	if hadShift && m.ikbdACIATXShiftTicks == 0 {
		completed := m.ikbdACIATXShift
		if completed == 1 && !m.ikbdResetCommandHandled {
			m.ikbdResetCommandDone = true
			m.ikbdResetCommandHandled = true
		} else if completed == 0x1c && !m.ikbdClockRequestHandled {
			m.ikbdClockRequestDone = true
			m.ikbdClockRequestHandled = true
		} else if completed == 0x1c && m.ikbdSetClockComplete && !m.ikbdClockReadbackRequestHandled {
			m.ikbdClockReadbackRequestDone = true
			m.ikbdClockReadbackRequestHandled = true
		} else if completed == 0x1c && m.ikbdClockPollRequestWritten &&
			m.ikbdClockReadbackComplete && m.flopVBLMediaComplete {
			m.ikbdClockPollRequestWritten = false
			m.ikbdClockPollRequestCount++
		} else if m.ikbdClockResponseComplete &&
			m.ikbdSetClockCompleteCount < m.ikbdSetClockWriteCount &&
			completed == m.ikbdSetClockWrites[m.ikbdSetClockCompleteCount] {
			index := m.ikbdSetClockCompleteCount
			m.ikbdSetClockCompletions[index] = completed
			if len(clocks) != 0 {
				m.ikbdSetClockCompletionClocks[index] = clocks[0]
			}
			m.ikbdSetClockCompleteCount++
			if m.ikbdSetClockCompleteCount == uint8(len(m.ikbdSetClockCompletions)) {
				m.ikbdSetClockComplete = true
			}
		}
	}
	if m.ikbdACIATXShiftTicks == 0 && m.ikbdACIATXPending {
		m.ikbdACIATXShift = m.ikbdACIATDR
		m.ikbdACIATXPending = false
		m.ikbdACIAStatus |= 2
		m.ikbdACIATXShiftTicks = 10
	}
}

func (m *Memory) deliverIKBDResetResponse() {
	if m.ikbdACIAConfigured && m.ikbdACIAStatus&1 == 0 {
		m.ikbdACIARDR = 0xf1
		m.ikbdACIAStatus |= 0x81
	}
}

func (m *Memory) nextIKBDClockResponse(round uint8) (int, byte) {
	if round == 1 && int(m.ikbdClockResponseDelivered) < len(ikbdClockResponse) {
		index := int(m.ikbdClockResponseDelivered)
		return index, ikbdClockResponse[index]
	}
	if round == 2 && int(m.ikbdClockReadbackDelivered) < len(ikbdClockReadback) {
		index := int(m.ikbdClockReadbackDelivered)
		return index, ikbdClockReadback[index]
	}
	if round == 3 && int(m.ikbdClockPollResponseDelivered) < len(ikbdClockReadback) {
		index := int(m.ikbdClockPollResponseDelivered)
		return index, ikbdClockReadback[index]
	}
	return -1, 0
}

func (m *Memory) deliverIKBDClockResponse(round, index uint8, value byte) bool {
	validRound := round == 1 && m.ikbdClockRequestHandled && index == m.ikbdClockResponseDelivered ||
		round == 2 && m.ikbdClockReadbackRequestHandled && index == m.ikbdClockReadbackDelivered ||
		round == 3 && m.ikbdClockPollRequestCount > m.ikbdClockPollCompleteCount &&
			index == m.ikbdClockPollResponseDelivered
	if !m.ikbdACIAConfigured || !validRound || int(index) >= len(ikbdClockResponse) ||
		m.ikbdACIAStatus&1 != 0 {
		return false
	}
	if round == 1 {
		m.ikbdClockResponseActive = true
		m.ikbdClockResponseDelivered++
	} else if round == 2 {
		m.ikbdClockReadbackActive = true
		m.ikbdClockReadbackDelivered++
	} else {
		m.ikbdClockPollResponseActive = true
		m.ikbdClockPollResponseDelivered++
	}
	m.ikbdACIARDR = value
	m.ikbdACIAStatus |= 0x81
	m.mfpGPIPIn &^= 0x10
	m.mfpIPRB |= 0x40
	return true
}

func (m *Memory) recordIKBDClockResponseDeliveryClock(round, index uint8, clock uint64) {
	if round == 2 {
		m.ikbdClockReadbackDeliveryClocks[index] = clock
	} else if round == 3 {
		m.ikbdClockPollDeliveryClocks[index] = clock
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
