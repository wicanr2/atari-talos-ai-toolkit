package m68k

import "fmt"

const (
	addressMask = 0x00ff_ffff
	supervisor  = 0x2000
)

type State struct {
	D        [8]uint32
	A        [7]uint32
	USP      uint32
	SSP      uint32
	SR       uint16
	PC       uint32
	Prefetch [2]uint16
}

type Transaction struct {
	Kind    string
	Cycle   uint32
	FC      uint8
	Address uint32
	Size    uint8
	Data    uint16
	UDS     bool
	LDS     bool
}

type StepResult struct {
	Clocks       uint32
	Transactions []Transaction
}

type Bus interface {
	ReadByte(address uint32, functionCode uint8) (byte, error)
	ReadWord(address uint32, functionCode uint8) (uint16, error)
	WriteByte(address uint32, value byte, functionCode uint8) error
	WriteWord(address uint32, value uint16, functionCode uint8) error
}

type CPU struct {
	State State
	Bus   Bus
}

func (c *CPU) Step() (StepResult, error) {
	if c.Bus == nil {
		return StepResult{}, fmt.Errorf("m68k: nil bus")
	}
	opcode := c.State.Prefetch[0]
	switch {
	case opcode&0xf000 == 0x1000:
		return c.stepMOVEByte(opcode)
	case opcode&0xf000 == 0x2000:
		return c.stepMOVELong(opcode)
	case opcode&0xf000 == 0x3000:
		return c.stepMOVEWord(opcode)
	case opcode&0xf1c0 == 0x41c0 && isControlMode(opcode):
		return c.stepLEA(opcode)
	case opcode&0xffc0 == 0x4840 && isControlMode(opcode):
		return c.stepPEA(opcode)
	case opcode&0xffc0 == 0x4ec0:
		return c.stepJMP(opcode)
	case opcode&0xffc0 == 0x4e80:
		return c.stepJSR(opcode)
	case opcode == 0x4e75:
		return c.stepRTS(opcode)
	case opcode&0xff00 == 0x6100:
		return c.stepBSR(opcode)
	case opcode&0xf000 == 0x6000 && opcode&0x0f00 != 0x0100:
		return c.stepBranch(opcode)
	case opcode == 0x4e71:
		// NOP changes only the prefetch pipeline.
	case opcode&0xfff8 == 0x4840:
		reg := opcode & 7
		value := c.State.D[reg]
		value = value<<16 | value>>16
		c.State.D[reg] = value
		c.setLogicalFlags(value, 32)
	case opcode&0xfff8 == 0x4880:
		reg := opcode & 7
		value := c.State.D[reg]
		word := uint32(uint16(int16(int8(value))))
		c.State.D[reg] = value&0xffff_0000 | word
		c.setLogicalFlags(word, 16)
	case opcode&0xfff8 == 0x48c0:
		reg := opcode & 7
		value := uint32(int32(int16(c.State.D[reg])))
		c.State.D[reg] = value
		c.setLogicalFlags(value, 32)
	case opcode&0xf100 == 0x7000:
		reg := (opcode >> 9) & 7
		value := uint32(int32(int8(opcode)))
		c.State.D[reg] = value
		c.State.SR &^= 0x000f
		if value == 0 {
			c.State.SR |= 0x0004
		}
		if value&0x8000_0000 != 0 {
			c.State.SR |= 0x0008
		}
	default:
		return StepResult{}, fmt.Errorf("m68k: opcode 0x%04x is not implemented", c.State.Prefetch[0])
	}

	address := c.State.PC & addressMask
	fc := uint8(2)
	if c.State.SR&supervisor != 0 {
		fc = 6
	}
	word, err := c.Bus.ReadWord(address, fc)
	if err != nil {
		return StepResult{}, err
	}
	c.State.Prefetch[0] = c.State.Prefetch[1]
	c.State.Prefetch[1] = word
	c.State.PC += 2

	return StepResult{Clocks: 4, Transactions: []Transaction{{
		Kind: "r", Cycle: 4, FC: fc, Address: address, Size: 2,
		Data: word, UDS: true, LDS: true,
	}}}, nil
}

type moveByteStep struct {
	cpu          *CPU
	programFC    uint8
	dataFC       uint8
	transactions []Transaction
}

func (c *CPU) stepMOVEByte(opcode uint16) (StepResult, error) {
	step := moveByteStep{cpu: c, programFC: c.programFunctionCode(), dataFC: 1}
	if c.State.SR&supervisor != 0 {
		step.dataFC = 5
	}
	sourceMode := uint8(opcode >> 3 & 7)
	sourceReg := uint8(opcode & 7)
	destinationMode := uint8(opcode >> 6 & 7)
	destinationReg := uint8(opcode >> 9 & 7)
	if destinationMode == 1 || destinationMode == 7 && destinationReg > 1 {
		return StepResult{}, fmt.Errorf("m68k: invalid MOVE.B destination mode %d:%d", destinationMode, destinationReg)
	}

	value, sourceCost, sourceMemory, err := step.readSource(sourceMode, sourceReg)
	if err != nil {
		return StepResult{}, err
	}
	destinationCost, err := step.writeDestination(destinationMode, destinationReg, value, sourceMemory)
	if err != nil {
		return StepResult{}, err
	}
	c.setLogicalFlags(uint32(value), 8)
	return StepResult{Clocks: 4 + sourceCost + destinationCost, Transactions: step.transactions}, nil
}

func (s *moveByteStep) readSource(mode, reg uint8) (byte, uint32, bool, error) {
	c := s.cpu
	var address uint32
	readFC := s.dataFC
	var cost uint32
	if mode == 0 {
		return byte(c.State.D[reg]), 0, false, nil
	}
	if mode == 1 {
		return 0, 0, false, fmt.Errorf("m68k: MOVE.B address-register source is invalid")
	}
	switch mode {
	case 2:
		address, cost = c.addressRegister(reg), 4
	case 3:
		address, cost = c.addressRegister(reg), 4
	case 4:
		address = c.addressRegister(reg) - byteAddressDelta(reg)
		c.setAddressRegister(reg, address)
		cost = 6
	case 5:
		extension, err := s.consumeExtension()
		if err != nil {
			return 0, 0, false, err
		}
		address = c.addressRegister(reg) + uint32(int32(int16(extension)))
		cost = 8
	case 6:
		extension, err := s.consumeExtension()
		if err != nil {
			return 0, 0, false, err
		}
		index, err := c.briefIndex(extension)
		if err != nil {
			return 0, 0, false, err
		}
		address = c.addressRegister(reg) + index + uint32(int32(int8(extension)))
		cost = 10
	case 7:
		switch reg {
		case 0:
			extension, err := s.consumeExtension()
			if err != nil {
				return 0, 0, false, err
			}
			address, cost = uint32(int32(int16(extension))), 8
		case 1:
			high, err := s.consumeExtension()
			if err != nil {
				return 0, 0, false, err
			}
			low, err := s.consumeExtension()
			if err != nil {
				return 0, 0, false, err
			}
			address, cost = uint32(high)<<16|uint32(low), 12
		case 2:
			base := c.State.PC - 2
			extension, err := s.consumeExtension()
			if err != nil {
				return 0, 0, false, err
			}
			address = base + uint32(int32(int16(extension)))
			readFC, cost = s.programFC, 8
		case 3:
			base := c.State.PC - 2
			extension, err := s.consumeExtension()
			if err != nil {
				return 0, 0, false, err
			}
			index, err := c.briefIndex(extension)
			if err != nil {
				return 0, 0, false, err
			}
			address = base + index + uint32(int32(int8(extension)))
			readFC, cost = s.programFC, 10
		case 4:
			extension, err := s.consumeExtension()
			return byte(extension), 4, false, err
		default:
			return 0, 0, false, fmt.Errorf("m68k: invalid MOVE.B source mode %d:%d", mode, reg)
		}
	}
	value, err := c.Bus.ReadByte(address&addressMask, readFC)
	if err != nil {
		return 0, 0, true, err
	}
	s.transactions = append(s.transactions, readByteTransaction(address&addressMask, readFC, value))
	if mode == 3 {
		c.setAddressRegister(reg, c.addressRegister(reg)+byteAddressDelta(reg))
	}
	return value, cost, true, nil
}

func (s *moveByteStep) writeDestination(mode, reg uint8, value byte, sourceMemory bool) (uint32, error) {
	c := s.cpu
	if mode == 0 {
		c.State.D[reg] = c.State.D[reg]&0xffff_ff00 | uint32(value)
		return 0, s.refill()
	}
	var address uint32
	switch mode {
	case 2:
		address = c.addressRegister(reg)
		if err := s.writeByte(address, value); err != nil {
			return 0, err
		}
		return 4, s.refill()
	case 3:
		address = c.addressRegister(reg)
		if err := s.writeByte(address, value); err != nil {
			return 0, err
		}
		c.setAddressRegister(reg, c.addressRegister(reg)+byteAddressDelta(reg))
		return 4, s.refill()
	case 4:
		if err := s.refill(); err != nil {
			return 0, err
		}
		address = c.addressRegister(reg) - byteAddressDelta(reg)
		c.setAddressRegister(reg, address)
		return 4, s.writeByte(address, value)
	case 5:
		extension, err := s.consumeExtension()
		if err != nil {
			return 0, err
		}
		address = c.addressRegister(reg) + uint32(int32(int16(extension)))
		if err := s.writeByte(address, value); err != nil {
			return 0, err
		}
		return 8, s.refill()
	case 6:
		extension, err := s.consumeExtension()
		if err != nil {
			return 0, err
		}
		index, err := c.briefIndex(extension)
		if err != nil {
			return 0, err
		}
		address = c.addressRegister(reg) + index + uint32(int32(int8(extension)))
		if err := s.writeByte(address, value); err != nil {
			return 0, err
		}
		return 10, s.refill()
	case 7:
		if reg == 0 {
			extension, err := s.consumeExtension()
			if err != nil {
				return 0, err
			}
			address = uint32(int32(int16(extension)))
			if err := s.writeByte(address, value); err != nil {
				return 0, err
			}
			return 8, s.refill()
		}
		if reg != 1 {
			return 0, fmt.Errorf("m68k: invalid MOVE.B destination mode %d:%d", mode, reg)
		}
		if sourceMemory {
			high := c.State.Prefetch[1]
			lowAddress := c.State.PC & addressMask
			low, err := c.Bus.ReadWord(lowAddress, s.programFC)
			if err != nil {
				return 0, err
			}
			s.transactions = append(s.transactions, readTransaction(lowAddress, s.programFC, low))
			address = uint32(high)<<16 | uint32(low)
			if err := s.writeByte(address, value); err != nil {
				return 0, err
			}
			first, err := s.readProgramWord(c.State.PC + 2)
			if err != nil {
				return 0, err
			}
			second, err := s.readProgramWord(c.State.PC + 4)
			if err != nil {
				return 0, err
			}
			c.State.Prefetch = [2]uint16{first, second}
			c.State.PC += 6
			return 12, nil
		}
		high, err := s.consumeExtension()
		if err != nil {
			return 0, err
		}
		low, err := s.consumeExtension()
		if err != nil {
			return 0, err
		}
		address = uint32(high)<<16 | uint32(low)
		if err := s.writeByte(address, value); err != nil {
			return 0, err
		}
		return 12, s.refill()
	default:
		return 0, fmt.Errorf("m68k: invalid MOVE.B destination mode %d:%d", mode, reg)
	}
}

func (s *moveByteStep) consumeExtension() (uint16, error) {
	extension := s.cpu.State.Prefetch[1]
	word, err := s.readProgramWord(s.cpu.State.PC)
	if err != nil {
		return 0, err
	}
	s.cpu.State.Prefetch[0] = extension
	s.cpu.State.Prefetch[1] = word
	s.cpu.State.PC += 2
	return extension, nil
}

func (s *moveByteStep) refill() error {
	_, err := s.consumeExtension()
	return err
}

func (s *moveByteStep) readProgramWord(address uint32) (uint16, error) {
	word, err := s.cpu.Bus.ReadWord(address&addressMask, s.programFC)
	if err == nil {
		s.transactions = append(s.transactions, readTransaction(address&addressMask, s.programFC, word))
	}
	return word, err
}

func (s *moveByteStep) writeByte(address uint32, value byte) error {
	address &= addressMask
	if err := s.cpu.Bus.WriteByte(address, value, s.dataFC); err != nil {
		return err
	}
	s.transactions = append(s.transactions, writeByteTransaction(address, s.dataFC, value))
	return nil
}

func byteAddressDelta(reg uint8) uint32 {
	if reg == 7 {
		return 2
	}
	return 1
}

type moveWordStep struct {
	moveByteStep
	opcode    uint16
	initialPC uint32
}

func (c *CPU) stepMOVEWord(opcode uint16) (StepResult, error) {
	stream := moveByteStep{cpu: c, programFC: c.programFunctionCode(), dataFC: 1}
	if c.State.SR&supervisor != 0 {
		stream.dataFC = 5
	}
	step := moveWordStep{moveByteStep: stream, opcode: opcode, initialPC: c.State.PC}
	sourceMode := uint8(opcode >> 3 & 7)
	sourceReg := uint8(opcode & 7)
	destinationMode := uint8(opcode >> 6 & 7)
	destinationReg := uint8(opcode >> 9 & 7)
	if destinationMode == 1 || destinationMode == 7 && destinationReg > 1 {
		return StepResult{}, fmt.Errorf("m68k: invalid MOVE.W destination mode %d:%d", destinationMode, destinationReg)
	}

	value, sourceCost, sourceMemory, fault, err := step.readWordSource(sourceMode, sourceReg)
	if err != nil {
		return StepResult{}, err
	}
	if fault != nil {
		return *fault, nil
	}
	destinationCost, fault, err := step.writeWordDestination(destinationMode, destinationReg, value, sourceCost, sourceMemory)
	if err != nil {
		return StepResult{}, err
	}
	if fault != nil {
		return *fault, nil
	}
	c.setLogicalFlags(uint32(value), 16)
	return StepResult{Clocks: 4 + sourceCost + destinationCost, Transactions: step.transactions}, nil
}

func (s *moveWordStep) readWordSource(mode, reg uint8) (uint16, uint32, bool, *StepResult, error) {
	c := s.cpu
	if mode == 0 {
		return uint16(c.State.D[reg]), 0, false, nil, nil
	}
	if mode == 1 {
		return uint16(c.addressRegister(reg)), 0, false, nil, nil
	}

	var address uint32
	var cost uint32
	readFC := s.dataFC
	switch mode {
	case 2:
		address, cost = c.addressRegister(reg), 4
	case 3:
		address, cost = c.addressRegister(reg), 4
		c.setAddressRegister(reg, address+2)
	case 4:
		address = c.addressRegister(reg) - 2
		c.setAddressRegister(reg, address)
		cost = 6
	case 5:
		extension, err := s.consumeExtension()
		if err != nil {
			return 0, 0, false, nil, err
		}
		address, cost = c.addressRegister(reg)+uint32(int32(int16(extension))), 8
	case 6:
		extension, err := s.consumeExtension()
		if err != nil {
			return 0, 0, false, nil, err
		}
		index, err := c.briefIndex(extension)
		if err != nil {
			return 0, 0, false, nil, err
		}
		address, cost = c.addressRegister(reg)+index+uint32(int32(int8(extension))), 10
	case 7:
		switch reg {
		case 0:
			extension, err := s.consumeExtension()
			if err != nil {
				return 0, 0, false, nil, err
			}
			address, cost = uint32(int32(int16(extension))), 8
		case 1:
			high, err := s.consumeExtension()
			if err != nil {
				return 0, 0, false, nil, err
			}
			low, err := s.consumeExtension()
			if err != nil {
				return 0, 0, false, nil, err
			}
			address, cost = uint32(high)<<16|uint32(low), 12
		case 2, 3:
			base := c.State.PC - 2
			extension, err := s.consumeExtension()
			if err != nil {
				return 0, 0, false, nil, err
			}
			address = base + uint32(int32(int16(extension)))
			cost = 8
			if reg == 3 {
				index, err := c.briefIndex(extension)
				if err != nil {
					return 0, 0, false, nil, err
				}
				address = base + index + uint32(int32(int8(extension)))
				cost = 10
			}
			readFC = s.programFC
		case 4:
			extension, err := s.consumeExtension()
			return extension, 4, false, nil, err
		default:
			return 0, 0, false, nil, fmt.Errorf("m68k: invalid MOVE.W source mode %d:%d", mode, reg)
		}
	default:
		return 0, 0, false, nil, fmt.Errorf("m68k: invalid MOVE.W source mode %d:%d", mode, reg)
	}

	if address&1 != 0 {
		result, err := c.enterAddressError(s.opcode, address, s.sourceFaultPC(mode, reg), s.transactions, 54+cost, readFC, "re", true)
		return 0, cost, true, &result, err
	}
	value, err := c.Bus.ReadWord(address&addressMask, readFC)
	if err != nil {
		return 0, 0, true, nil, err
	}
	s.transactions = append(s.transactions, readTransaction(address&addressMask, readFC, value))
	return value, cost, true, nil, nil
}

func (s *moveWordStep) sourceFaultPC(mode, reg uint8) uint32 {
	switch mode {
	case 2, 3, 5, 6:
		return s.initialPC - 2
	case 4:
		return s.initialPC
	case 7:
		switch reg {
		case 0:
			return s.initialPC
		case 1:
			return s.initialPC + 2
		default:
			return s.initialPC - 2
		}
	}
	return s.initialPC
}

func (s *moveWordStep) writeWordDestination(mode, reg uint8, value uint16, sourceCost uint32, sourceMemory bool) (uint32, *StepResult, error) {
	c := s.cpu
	if mode == 0 {
		c.State.D[reg] = c.State.D[reg]&0xffff_0000 | uint32(value)
		return 0, nil, s.refill()
	}

	savedPC := c.State.PC
	var address uint32
	var cost, faultExtra uint32
	refillBeforeWrite := false
	switch mode {
	case 2:
		address, cost = c.addressRegister(reg), 4
	case 3:
		address, cost = c.addressRegister(reg), 4
	case 4:
		address, cost, faultExtra = c.addressRegister(reg)-2, 4, 4
		c.setAddressRegister(reg, address)
		refillBeforeWrite = true
	case 5:
		extension, err := s.consumeExtension()
		if err != nil {
			return 0, nil, err
		}
		address, cost, faultExtra = c.addressRegister(reg)+uint32(int32(int16(extension))), 8, 4
	case 6:
		extension, err := s.consumeExtension()
		if err != nil {
			return 0, nil, err
		}
		index, err := c.briefIndex(extension)
		if err != nil {
			return 0, nil, err
		}
		address, cost, faultExtra = c.addressRegister(reg)+index+uint32(int32(int8(extension))), 10, 6
	case 7:
		if reg == 0 {
			extension, err := s.consumeExtension()
			if err != nil {
				return 0, nil, err
			}
			address, cost, faultExtra = uint32(int32(int16(extension))), 8, 4
			break
		}
		if reg != 1 {
			return 0, nil, fmt.Errorf("m68k: invalid MOVE.W destination mode %d:%d", mode, reg)
		}
		cost, faultExtra = 12, 8
		if sourceMemory {
			faultExtra = 4
			high := c.State.Prefetch[1]
			lowAddress := c.State.PC & addressMask
			low, err := c.Bus.ReadWord(lowAddress, s.programFC)
			if err != nil {
				return 0, nil, err
			}
			s.transactions = append(s.transactions, readTransaction(lowAddress, s.programFC, low))
			address = uint32(high)<<16 | uint32(low)
			if address&1 != 0 {
				return s.wordWriteFault(s.opcode, address, savedPC, value, sourceCost, faultExtra)
			}
			if err := s.writeWord(address, value); err != nil {
				return 0, nil, err
			}
			first, err := s.readProgramWord(c.State.PC + 2)
			if err != nil {
				return 0, nil, err
			}
			second, err := s.readProgramWord(c.State.PC + 4)
			if err != nil {
				return 0, nil, err
			}
			c.State.Prefetch = [2]uint16{first, second}
			c.State.PC += 6
			return cost, nil, nil
		}
		high, err := s.consumeExtension()
		if err != nil {
			return 0, nil, err
		}
		low, err := s.consumeExtension()
		if err != nil {
			return 0, nil, err
		}
		address = uint32(high)<<16 | uint32(low)
		savedPC += 2
	default:
		return 0, nil, fmt.Errorf("m68k: invalid MOVE.W destination mode %d:%d", mode, reg)
	}

	if refillBeforeWrite {
		if err := s.refill(); err != nil {
			return 0, nil, err
		}
	}
	if address&1 != 0 {
		faultOpcode := s.opcode
		if refillBeforeWrite {
			faultOpcode = c.State.Prefetch[0]
		}
		return s.wordWriteFault(faultOpcode, address, savedPC, value, sourceCost, faultExtra)
	}
	if err := s.writeWord(address, value); err != nil {
		return 0, nil, err
	}
	if mode == 3 {
		c.setAddressRegister(reg, address+2)
	}
	if refillBeforeWrite {
		return cost, nil, nil
	}
	return cost, nil, s.refill()
}

func (s *moveWordStep) wordWriteFault(opcode uint16, address, savedPC uint32, value uint16, sourceCost, faultExtra uint32) (uint32, *StepResult, error) {
	s.cpu.setLogicalFlags(uint32(value), 16)
	result, err := s.cpu.enterAddressError(opcode, address, savedPC, s.transactions, 58+sourceCost+faultExtra, s.dataFC, "we", false)
	return 0, &result, err
}

func (s *moveWordStep) writeWord(address uint32, value uint16) error {
	address &= addressMask
	if err := s.cpu.Bus.WriteWord(address, value, s.dataFC); err != nil {
		return err
	}
	s.transactions = append(s.transactions, writeTransaction(address, s.dataFC, value))
	return nil
}

type moveLongStep struct {
	moveByteStep
	opcode    uint16
	initialPC uint32
}

func (c *CPU) stepMOVELong(opcode uint16) (StepResult, error) {
	stream := moveByteStep{cpu: c, programFC: c.programFunctionCode(), dataFC: 1}
	if c.State.SR&supervisor != 0 {
		stream.dataFC = 5
	}
	step := moveLongStep{moveByteStep: stream, opcode: opcode, initialPC: c.State.PC}
	sourceMode := uint8(opcode >> 3 & 7)
	sourceReg := uint8(opcode & 7)
	destinationMode := uint8(opcode >> 6 & 7)
	destinationReg := uint8(opcode >> 9 & 7)
	if destinationMode == 1 || destinationMode == 7 && destinationReg > 1 {
		return StepResult{}, fmt.Errorf("m68k: invalid MOVE.L destination mode %d:%d", destinationMode, destinationReg)
	}

	value, sourceCost, sourceMemory, fault, err := step.readLongSource(sourceMode, sourceReg)
	if err != nil {
		return StepResult{}, err
	}
	if fault != nil {
		return *fault, nil
	}
	destinationCost, fault, err := step.writeLongDestination(destinationMode, destinationReg, value, sourceCost, sourceMemory)
	if err != nil {
		return StepResult{}, err
	}
	if fault != nil {
		return *fault, nil
	}
	c.setLogicalFlags(value, 32)
	return StepResult{Clocks: 4 + sourceCost + destinationCost, Transactions: step.transactions}, nil
}

func (s *moveLongStep) readLongSource(mode, reg uint8) (uint32, uint32, bool, *StepResult, error) {
	c := s.cpu
	if mode == 0 {
		return c.State.D[reg], 0, false, nil, nil
	}
	if mode == 1 {
		return c.addressRegister(reg), 0, false, nil, nil
	}

	var address uint32
	var cost, faultCost uint32
	readFC := s.dataFC
	switch mode {
	case 2:
		address, cost, faultCost = c.addressRegister(reg), 8, 4
	case 3:
		address, cost, faultCost = c.addressRegister(reg), 8, 4
	case 4:
		address = c.addressRegister(reg) - 4
		c.setAddressRegister(reg, address)
		cost, faultCost = 10, 6
	case 5:
		extension, err := s.consumeExtension()
		if err != nil {
			return 0, 0, false, nil, err
		}
		address = c.addressRegister(reg) + uint32(int32(int16(extension)))
		cost, faultCost = 12, 8
	case 6:
		extension, err := s.consumeExtension()
		if err != nil {
			return 0, 0, false, nil, err
		}
		index, err := c.briefIndex(extension)
		if err != nil {
			return 0, 0, false, nil, err
		}
		address = c.addressRegister(reg) + index + uint32(int32(int8(extension)))
		cost, faultCost = 14, 10
	case 7:
		switch reg {
		case 0:
			extension, err := s.consumeExtension()
			if err != nil {
				return 0, 0, false, nil, err
			}
			address, cost, faultCost = uint32(int32(int16(extension))), 12, 8
		case 1:
			high, err := s.consumeExtension()
			if err != nil {
				return 0, 0, false, nil, err
			}
			low, err := s.consumeExtension()
			if err != nil {
				return 0, 0, false, nil, err
			}
			address, cost, faultCost = uint32(high)<<16|uint32(low), 16, 12
		case 2, 3:
			base := c.State.PC - 2
			extension, err := s.consumeExtension()
			if err != nil {
				return 0, 0, false, nil, err
			}
			address = base + uint32(int32(int16(extension)))
			cost, faultCost = 12, 8
			if reg == 3 {
				index, err := c.briefIndex(extension)
				if err != nil {
					return 0, 0, false, nil, err
				}
				address = base + index + uint32(int32(int8(extension)))
				cost, faultCost = 14, 10
			}
			readFC = s.programFC
		case 4:
			high, err := s.consumeExtension()
			if err != nil {
				return 0, 0, false, nil, err
			}
			low, err := s.consumeExtension()
			return uint32(high)<<16 | uint32(low), 8, false, nil, err
		default:
			return 0, 0, false, nil, fmt.Errorf("m68k: invalid MOVE.L source mode %d:%d", mode, reg)
		}
	default:
		return 0, 0, false, nil, fmt.Errorf("m68k: invalid MOVE.L source mode %d:%d", mode, reg)
	}

	if address&1 != 0 {
		result, err := c.enterAddressError(s.opcode, address, s.sourceFaultPC(mode, reg), s.transactions, 54+faultCost, readFC, "re", true)
		return 0, cost, true, &result, err
	}
	high, err := s.readLongWord(address, readFC)
	if err != nil {
		return 0, 0, true, nil, err
	}
	low, err := s.readLongWord(address+2, readFC)
	if err != nil {
		return 0, 0, true, nil, err
	}
	if mode == 3 {
		c.setAddressRegister(reg, address+4)
	}
	return uint32(high)<<16 | uint32(low), cost, true, nil, nil
}

func (s *moveLongStep) sourceFaultPC(mode, reg uint8) uint32 {
	switch mode {
	case 2, 3, 4, 5, 6:
		return s.initialPC - 2
	case 7:
		switch reg {
		case 0:
			return s.initialPC
		case 1:
			return s.initialPC + 2
		default:
			return s.initialPC - 2
		}
	}
	return s.initialPC
}

func (s *moveLongStep) readLongWord(address uint32, fc uint8) (uint16, error) {
	word, err := s.cpu.Bus.ReadWord(address&addressMask, fc)
	if err == nil {
		s.transactions = append(s.transactions, readTransaction(address&addressMask, fc, word))
	}
	return word, err
}

func (s *moveLongStep) writeLongDestination(mode, reg uint8, value uint32, sourceCost uint32, sourceMemory bool) (uint32, *StepResult, error) {
	c := s.cpu
	if mode == 0 {
		c.State.D[reg] = value
		return 0, nil, s.refill()
	}

	savedPC := c.State.PC
	var address uint32
	var cost, faultExtra uint32
	refillBeforeWrite := false
	switch mode {
	case 2:
		address, cost = c.addressRegister(reg), 8
	case 3:
		address, cost = c.addressRegister(reg), 8
	case 4:
		address, cost, faultExtra = c.addressRegister(reg)-2, 8, 4
		refillBeforeWrite = true
	case 5:
		extension, err := s.consumeExtension()
		if err != nil {
			return 0, nil, err
		}
		address, cost, faultExtra = c.addressRegister(reg)+uint32(int32(int16(extension))), 12, 4
	case 6:
		extension, err := s.consumeExtension()
		if err != nil {
			return 0, nil, err
		}
		index, err := c.briefIndex(extension)
		if err != nil {
			return 0, nil, err
		}
		address, cost, faultExtra = c.addressRegister(reg)+index+uint32(int32(int8(extension))), 14, 6
	case 7:
		if reg == 0 {
			extension, err := s.consumeExtension()
			if err != nil {
				return 0, nil, err
			}
			address, cost, faultExtra = uint32(int32(int16(extension))), 12, 4
			break
		}
		if reg != 1 {
			return 0, nil, fmt.Errorf("m68k: invalid MOVE.L destination mode %d:%d", mode, reg)
		}
		cost, faultExtra = 16, 8
		if sourceMemory {
			faultExtra = 4
			high := c.State.Prefetch[1]
			lowAddress := c.State.PC & addressMask
			low, err := c.Bus.ReadWord(lowAddress, s.programFC)
			if err != nil {
				return 0, nil, err
			}
			s.transactions = append(s.transactions, readTransaction(lowAddress, s.programFC, low))
			address = uint32(high)<<16 | uint32(low)
			if address&1 != 0 {
				return s.longWriteFault(mode, reg, address, savedPC, value, sourceCost, faultExtra, sourceMemory)
			}
			if err := s.writeLong(address, value); err != nil {
				return 0, nil, err
			}
			first, err := s.readProgramWord(c.State.PC + 2)
			if err != nil {
				return 0, nil, err
			}
			second, err := s.readProgramWord(c.State.PC + 4)
			if err != nil {
				return 0, nil, err
			}
			c.State.Prefetch = [2]uint16{first, second}
			c.State.PC += 6
			return cost, nil, nil
		}
		high, err := s.consumeExtension()
		if err != nil {
			return 0, nil, err
		}
		low, err := s.consumeExtension()
		if err != nil {
			return 0, nil, err
		}
		address = uint32(high)<<16 | uint32(low)
		savedPC += 2
	default:
		return 0, nil, fmt.Errorf("m68k: invalid MOVE.L destination mode %d:%d", mode, reg)
	}

	if refillBeforeWrite {
		if err := s.refill(); err != nil {
			return 0, nil, err
		}
	}
	if address&1 != 0 {
		return s.longWriteFault(mode, reg, address, savedPC, value, sourceCost, faultExtra, sourceMemory)
	}
	if mode == 4 {
		address -= 2
		c.setAddressRegister(reg, address)
	}
	var err error
	if mode == 4 {
		err = s.writeLongPredecrement(address, value)
	} else {
		err = s.writeLong(address, value)
	}
	if err != nil {
		return 0, nil, err
	}
	if mode == 3 {
		c.setAddressRegister(reg, address+4)
	}
	if refillBeforeWrite {
		return cost, nil, nil
	}
	return cost, nil, s.refill()
}

func (s *moveLongStep) longWriteFault(mode, reg uint8, address, savedPC uint32, value uint32, sourceCost, faultExtra uint32, sourceMemory bool) (uint32, *StepResult, error) {
	if sourceMemory {
		if mode == 2 || mode == 3 || mode == 7 && reg == 1 {
			s.cpu.setLogicalFlags(uint32(uint16(value)), 16)
		} else {
			s.cpu.setLogicalFlags(value, 32)
		}
	} else {
		switch mode {
		case 4, 7:
			s.cpu.setLogicalFlags(value, 32)
		case 5, 6:
			s.cpu.State.SR &^= 0x000c
			if value == 0 {
				s.cpu.State.SR |= 0x0004
			}
			if value&0x8000_0000 != 0 {
				s.cpu.State.SR |= 0x0008
			}
		}
	}
	result, err := s.cpu.enterAddressError(s.opcode, address, savedPC, s.transactions, 58+sourceCost+faultExtra, s.dataFC, "we", false)
	return 0, &result, err
}

func (s *moveLongStep) writeLong(address, value uint32) error {
	if err := s.writeLongWord(address, uint16(value>>16)); err != nil {
		return err
	}
	return s.writeLongWord(address+2, uint16(value))
}

func (s *moveLongStep) writeLongPredecrement(address, value uint32) error {
	if err := s.writeLongWord(address+2, uint16(value)); err != nil {
		return err
	}
	return s.writeLongWord(address, uint16(value>>16))
}

func (s *moveLongStep) writeLongWord(address uint32, value uint16) error {
	address &= addressMask
	if err := s.cpu.Bus.WriteWord(address, value, s.dataFC); err != nil {
		return err
	}
	s.transactions = append(s.transactions, writeTransaction(address, s.dataFC, value))
	return nil
}

type controlEA struct {
	target         uint32
	returnPC       uint32
	extraClocks    uint32
	extensionWords uint8
	indexed        bool
	transactions   []Transaction
}

func (c *CPU) decodeControlEA(opcode uint16) (controlEA, error) {
	mode := uint8(opcode >> 3 & 7)
	reg := uint8(opcode & 7)
	basePC := c.State.PC - 2
	result := controlEA{returnPC: basePC}
	switch mode {
	case 2:
		result.target = c.addressRegister(reg)
	case 5:
		result.target = c.addressRegister(reg) + uint32(int32(int16(c.State.Prefetch[1])))
		result.returnPC = c.State.PC
		result.extraClocks = 2
		result.extensionWords = 1
	case 6:
		index, err := c.briefIndex(c.State.Prefetch[1])
		if err != nil {
			return controlEA{}, err
		}
		result.target = c.addressRegister(reg) + index + uint32(int32(int8(c.State.Prefetch[1])))
		result.returnPC = c.State.PC
		result.extraClocks = 6
		result.extensionWords = 1
		result.indexed = true
	case 7:
		switch reg {
		case 0:
			result.target = uint32(int32(int16(c.State.Prefetch[1])))
			result.returnPC = c.State.PC
			result.extraClocks = 2
			result.extensionWords = 1
		case 1:
			address := c.State.PC & addressMask
			fc := c.programFunctionCode()
			low, err := c.Bus.ReadWord(address, fc)
			if err != nil {
				return controlEA{}, err
			}
			result.target = uint32(c.State.Prefetch[1])<<16 | uint32(low)
			result.returnPC = c.State.PC + 2
			result.extraClocks = 4
			result.extensionWords = 2
			result.transactions = []Transaction{readTransaction(address, fc, low)}
		case 2:
			result.target = basePC + uint32(int32(int16(c.State.Prefetch[1])))
			result.returnPC = c.State.PC
			result.extraClocks = 2
			result.extensionWords = 1
		case 3:
			index, err := c.briefIndex(c.State.Prefetch[1])
			if err != nil {
				return controlEA{}, err
			}
			result.target = basePC + index + uint32(int32(int8(c.State.Prefetch[1])))
			result.returnPC = c.State.PC
			result.extraClocks = 6
			result.extensionWords = 1
			result.indexed = true
		default:
			return controlEA{}, fmt.Errorf("m68k: invalid control addressing mode %d:%d", mode, reg)
		}
	default:
		return controlEA{}, fmt.Errorf("m68k: invalid control addressing mode %d:%d", mode, reg)
	}
	return result, nil
}

func isControlMode(opcode uint16) bool {
	mode := uint8(opcode >> 3 & 7)
	reg := uint8(opcode & 7)
	return mode == 2 || mode == 5 || mode == 6 || mode == 7 && reg <= 3
}

func (c *CPU) stepLEA(opcode uint16) (StepResult, error) {
	ea, err := c.decodeControlEA(opcode)
	if err != nil {
		return StepResult{}, err
	}
	destination := uint8(opcode >> 9 & 7)
	c.setAddressRegister(destination, ea.target)
	clocks := uint32(4 + 4*ea.extensionWords)
	if ea.indexed {
		clocks += 4
	}
	return c.refillSequential(ea, clocks)
}

func (c *CPU) stepPEA(opcode uint16) (StepResult, error) {
	ea, err := c.decodeControlEA(opcode)
	if err != nil {
		return StepResult{}, err
	}
	programFC := c.programFunctionCode()
	transactions := append([]Transaction{}, ea.transactions...)
	clocks := uint32(12 + 4*ea.extensionWords)
	if ea.indexed {
		clocks += 4
	}

	if ea.extensionWords == 0 {
		address := c.State.PC & addressMask
		word, err := c.Bus.ReadWord(address, programFC)
		if err != nil {
			return StepResult{}, err
		}
		transactions = append(transactions, readTransaction(address, programFC, word))
		stackTransactions, err := c.pushLong(ea.target)
		if err != nil {
			return StepResult{}, err
		}
		transactions = append(transactions, stackTransactions...)
		c.State.Prefetch[0] = c.State.Prefetch[1]
		c.State.Prefetch[1] = word
		c.State.PC += 2
		return StepResult{Clocks: clocks, Transactions: transactions}, nil
	}

	start := c.State.PC + uint32(ea.extensionWords-1)*2
	firstAddress := start & addressMask
	first, err := c.Bus.ReadWord(firstAddress, programFC)
	if err != nil {
		return StepResult{}, err
	}
	transactions = append(transactions, readTransaction(firstAddress, programFC, first))
	absolute := opcode>>3&7 == 7 && opcode&7 <= 1
	if absolute {
		stackTransactions, err := c.pushLong(ea.target)
		if err != nil {
			return StepResult{}, err
		}
		transactions = append(transactions, stackTransactions...)
	}
	secondAddress := (start + 2) & addressMask
	second, err := c.Bus.ReadWord(secondAddress, programFC)
	if err != nil {
		return StepResult{}, err
	}
	transactions = append(transactions, readTransaction(secondAddress, programFC, second))
	if !absolute {
		stackTransactions, err := c.pushLong(ea.target)
		if err != nil {
			return StepResult{}, err
		}
		transactions = append(transactions, stackTransactions...)
	}
	c.State.Prefetch = [2]uint16{first, second}
	c.State.PC = start + 4
	return StepResult{Clocks: clocks, Transactions: transactions}, nil
}

func (c *CPU) refillSequential(ea controlEA, clocks uint32) (StepResult, error) {
	fc := c.programFunctionCode()
	transactions := append([]Transaction{}, ea.transactions...)
	if ea.extensionWords == 0 {
		address := c.State.PC & addressMask
		word, err := c.Bus.ReadWord(address, fc)
		if err != nil {
			return StepResult{}, err
		}
		transactions = append(transactions, readTransaction(address, fc, word))
		c.State.Prefetch[0] = c.State.Prefetch[1]
		c.State.Prefetch[1] = word
		c.State.PC += 2
		return StepResult{Clocks: clocks, Transactions: transactions}, nil
	}
	start := c.State.PC + uint32(ea.extensionWords-1)*2
	firstAddress := start & addressMask
	first, err := c.Bus.ReadWord(firstAddress, fc)
	if err != nil {
		return StepResult{}, err
	}
	secondAddress := (start + 2) & addressMask
	second, err := c.Bus.ReadWord(secondAddress, fc)
	if err != nil {
		return StepResult{}, err
	}
	transactions = append(transactions,
		readTransaction(firstAddress, fc, first),
		readTransaction(secondAddress, fc, second))
	c.State.Prefetch = [2]uint16{first, second}
	c.State.PC = start + 4
	return StepResult{Clocks: clocks, Transactions: transactions}, nil
}

func (c *CPU) pushLong(value uint32) ([]Transaction, error) {
	stack := &c.State.USP
	dataFC := uint8(1)
	if c.State.SR&supervisor != 0 {
		stack = &c.State.SSP
		dataFC = 5
	}
	newSP := *stack - 4
	firstAddress := newSP & addressMask
	if err := c.Bus.WriteWord(firstAddress, uint16(value>>16), dataFC); err != nil {
		return nil, err
	}
	secondAddress := (newSP + 2) & addressMask
	if err := c.Bus.WriteWord(secondAddress, uint16(value), dataFC); err != nil {
		return nil, err
	}
	*stack = newSP
	return []Transaction{
		writeTransaction(firstAddress, dataFC, uint16(value>>16)),
		writeTransaction(secondAddress, dataFC, uint16(value)),
	}, nil
}

func (c *CPU) setAddressRegister(reg uint8, value uint32) {
	if reg < 7 {
		c.State.A[reg] = value
		return
	}
	if c.State.SR&supervisor != 0 {
		c.State.SSP = value
		return
	}
	c.State.USP = value
}

func (c *CPU) briefIndex(extension uint16) (uint32, error) {
	reg := uint8(extension >> 12 & 7)
	value := c.State.D[reg]
	if extension&0x8000 != 0 {
		value = c.addressRegister(reg)
	}
	if extension&0x0800 == 0 {
		value = uint32(int32(int16(value)))
	}
	return value, nil
}

func (c *CPU) addressRegister(reg uint8) uint32 {
	if reg < 7 {
		return c.State.A[reg]
	}
	if c.State.SR&supervisor != 0 {
		return c.State.SSP
	}
	return c.State.USP
}

func (c *CPU) stepJMP(opcode uint16) (StepResult, error) {
	ea, err := c.decodeControlEA(opcode)
	if err != nil {
		return StepResult{}, err
	}
	if ea.target&1 != 0 {
		return c.enterInstructionAddressError(
			opcode, ea.target, c.State.PC-2, ea.transactions, 58+ea.extraClocks,
		)
	}
	result, err := c.refillBranch(ea.target, 8+ea.extraClocks)
	if err != nil {
		return StepResult{}, err
	}
	result.Transactions = append(ea.transactions, result.Transactions...)
	return result, nil
}

func (c *CPU) stepJSR(opcode uint16) (StepResult, error) {
	ea, err := c.decodeControlEA(opcode)
	if err != nil {
		return StepResult{}, err
	}
	if ea.target&1 != 0 {
		return c.enterInstructionAddressError(
			opcode, ea.target, ea.returnPC, ea.transactions, 58+ea.extraClocks,
		)
	}

	programFC := c.programFunctionCode()
	targetAddress := ea.target & addressMask
	first, err := c.Bus.ReadWord(targetAddress, programFC)
	if err != nil {
		return StepResult{}, err
	}
	transactions := append(ea.transactions, readTransaction(targetAddress, programFC, first))
	stack := &c.State.USP
	dataFC := uint8(1)
	if c.State.SR&supervisor != 0 {
		stack = &c.State.SSP
		dataFC = 5
	}
	newSP := *stack - 4
	returnHigh := uint16(ea.returnPC >> 16)
	returnLow := uint16(ea.returnPC)
	stackAddress := newSP & addressMask
	if err := c.Bus.WriteWord(stackAddress, returnHigh, dataFC); err != nil {
		return StepResult{}, err
	}
	stackLowAddress := (newSP + 2) & addressMask
	if err := c.Bus.WriteWord(stackLowAddress, returnLow, dataFC); err != nil {
		return StepResult{}, err
	}
	*stack = newSP
	transactions = append(transactions,
		writeTransaction(stackAddress, dataFC, returnHigh),
		writeTransaction(stackLowAddress, dataFC, returnLow))
	secondAddress := (ea.target + 2) & addressMask
	second, err := c.Bus.ReadWord(secondAddress, programFC)
	if err != nil {
		return StepResult{}, err
	}
	transactions = append(transactions, readTransaction(secondAddress, programFC, second))
	c.State.Prefetch = [2]uint16{first, second}
	c.State.PC = ea.target + 4
	return StepResult{Clocks: 16 + ea.extraClocks, Transactions: transactions}, nil
}

func (c *CPU) stepRTS(opcode uint16) (StepResult, error) {
	stack := &c.State.USP
	dataFC := uint8(1)
	if c.State.SR&supervisor != 0 {
		stack = &c.State.SSP
		dataFC = 5
	}
	stackAddress := *stack & addressMask
	returnHigh, err := c.Bus.ReadWord(stackAddress, dataFC)
	if err != nil {
		return StepResult{}, err
	}
	secondAddress := (*stack + 2) & addressMask
	returnLow, err := c.Bus.ReadWord(secondAddress, dataFC)
	if err != nil {
		return StepResult{}, err
	}
	*stack += 4
	returnPC := uint32(returnHigh)<<16 | uint32(returnLow)
	prefix := []Transaction{
		readTransaction(stackAddress, dataFC, returnHigh),
		readTransaction(secondAddress, dataFC, returnLow),
	}
	if returnPC&1 != 0 {
		return c.enterInstructionAddressError(opcode, returnPC, c.State.PC-2, prefix, 66)
	}
	result, err := c.refillBranch(returnPC, 16)
	if err != nil {
		return StepResult{}, err
	}
	result.Transactions = append(prefix, result.Transactions...)
	return result, nil
}

func (c *CPU) stepBSR(opcode uint16) (StepResult, error) {
	displacement8 := uint8(opcode)
	base := c.State.PC - 2
	returnPC := base
	var displacement int32
	if displacement8 == 0 {
		displacement = int32(int16(c.State.Prefetch[1]))
		returnPC = c.State.PC
	} else {
		displacement = int32(int8(displacement8))
	}
	target := uint32(int32(base) + displacement)

	stack := &c.State.USP
	dataFC := uint8(1)
	if c.State.SR&supervisor != 0 {
		stack = &c.State.SSP
		dataFC = 5
	}
	newSP := *stack - 4
	returnHigh := uint16(returnPC >> 16)
	returnLow := uint16(returnPC)
	firstAddress := newSP & addressMask
	if err := c.Bus.WriteWord(firstAddress, returnHigh, dataFC); err != nil {
		return StepResult{}, err
	}
	secondAddress := (newSP + 2) & addressMask
	if err := c.Bus.WriteWord(secondAddress, returnLow, dataFC); err != nil {
		return StepResult{}, err
	}
	*stack = newSP
	prefix := []Transaction{
		writeTransaction(firstAddress, dataFC, returnHigh),
		writeTransaction(secondAddress, dataFC, returnLow),
	}

	if target&1 != 0 {
		return c.enterInstructionAddressError(opcode, target, target, prefix, 68)
	}
	result, err := c.refillBranch(target, 18)
	if err != nil {
		return StepResult{}, err
	}
	result.Transactions = append(prefix, result.Transactions...)
	return result, nil
}

func (c *CPU) stepBranch(opcode uint16) (StepResult, error) {
	condition := uint8(opcode >> 8 & 0x0f)
	taken := condition == 0 || branchCondition(condition, c.State.SR)
	displacement8 := uint8(opcode)
	base := c.State.PC - 2

	if taken {
		var displacement int32
		if displacement8 == 0 {
			displacement = int32(int16(c.State.Prefetch[1]))
		} else {
			displacement = int32(int8(displacement8))
		}
		target := uint32(int32(base) + displacement)
		if target&1 != 0 {
			return c.enterInstructionAddressError(opcode, target, base, nil, 60)
		}
		return c.refillBranch(target, 10)
	}

	if displacement8 != 0 {
		address := c.State.PC & addressMask
		fc := c.programFunctionCode()
		word, err := c.Bus.ReadWord(address, fc)
		if err != nil {
			return StepResult{}, err
		}
		c.State.Prefetch[0] = c.State.Prefetch[1]
		c.State.Prefetch[1] = word
		c.State.PC += 2
		return StepResult{Clocks: 8, Transactions: []Transaction{
			readTransaction(address, fc, word),
		}}, nil
	}

	return c.refillBranch(c.State.PC, 12)
}

func (c *CPU) enterInstructionAddressError(
	opcode uint16,
	target uint32,
	savedPC uint32,
	prefix []Transaction,
	clocks uint32,
) (StepResult, error) {
	return c.enterAddressError(opcode, target, savedPC, prefix, clocks, c.programFunctionCode(), "re", true)
}

func (c *CPU) enterAddressError(
	opcode uint16,
	target uint32,
	savedPC uint32,
	prefix []Transaction,
	clocks uint32,
	faultFC uint8,
	faultKind string,
	read bool,
) (StepResult, error) {
	originalSR := c.State.SR
	faultBusAddress := target & addressMask &^ 1

	newSP := c.State.SSP - 14
	ssw := opcode&0xffe0 | uint16(faultFC)
	if read {
		ssw |= 0x0010
	}
	writes := []struct {
		address uint32
		value   uint16
	}{
		{newSP + 12, uint16(savedPC)},
		{newSP + 8, originalSR},
		{newSP + 10, uint16(savedPC >> 16)},
		{newSP + 6, opcode},
		{newSP + 4, uint16(target)},
		{newSP, ssw},
		{newSP + 2, uint16(target >> 16)},
	}
	transactions := append(prefix, Transaction{
		Kind: faultKind, Cycle: 4, FC: faultFC, Address: faultBusAddress,
		Size: 2, Data: 0, UDS: true, LDS: true,
	})
	for _, write := range writes {
		address := write.address & addressMask
		if err := c.Bus.WriteWord(address, write.value, 5); err != nil {
			return StepResult{}, err
		}
		transactions = append(transactions, writeTransaction(address, 5, write.value))
	}

	vectorHigh, err := c.Bus.ReadWord(0x00000c, 5)
	if err != nil {
		return StepResult{}, err
	}
	vectorLow, err := c.Bus.ReadWord(0x00000e, 5)
	if err != nil {
		return StepResult{}, err
	}
	transactions = append(transactions,
		readTransaction(0x00000c, 5, vectorHigh),
		readTransaction(0x00000e, 5, vectorLow))
	handler := uint32(vectorHigh)<<16 | uint32(vectorLow)
	handlerAddress := handler & addressMask
	first, err := c.Bus.ReadWord(handlerAddress, 6)
	if err != nil {
		return StepResult{}, err
	}
	secondAddress := (handler + 2) & addressMask
	second, err := c.Bus.ReadWord(secondAddress, 6)
	if err != nil {
		return StepResult{}, err
	}
	transactions = append(transactions,
		readTransaction(handlerAddress, 6, first),
		readTransaction(secondAddress, 6, second))

	c.State.SSP = newSP
	c.State.SR = originalSR | supervisor
	c.State.SR &^= 0x8000
	c.State.Prefetch = [2]uint16{first, second}
	c.State.PC = handler + 4
	return StepResult{Clocks: clocks, Transactions: transactions}, nil
}

func (c *CPU) refillBranch(address uint32, clocks uint32) (StepResult, error) {
	fc := c.programFunctionCode()
	firstAddress := address & addressMask
	first, err := c.Bus.ReadWord(firstAddress, fc)
	if err != nil {
		return StepResult{}, err
	}
	secondAddress := (address + 2) & addressMask
	second, err := c.Bus.ReadWord(secondAddress, fc)
	if err != nil {
		return StepResult{}, err
	}
	c.State.Prefetch = [2]uint16{first, second}
	c.State.PC = address + 4
	return StepResult{Clocks: clocks, Transactions: []Transaction{
		readTransaction(firstAddress, fc, first),
		readTransaction(secondAddress, fc, second),
	}}, nil
}

func branchCondition(condition uint8, sr uint16) bool {
	c := sr&0x0001 != 0
	v := sr&0x0002 != 0
	z := sr&0x0004 != 0
	n := sr&0x0008 != 0
	switch condition {
	case 2:
		return !c && !z
	case 3:
		return c || z
	case 4:
		return !c
	case 5:
		return c
	case 6:
		return !z
	case 7:
		return z
	case 8:
		return !v
	case 9:
		return v
	case 10:
		return !n
	case 11:
		return n
	case 12:
		return n == v
	case 13:
		return n != v
	case 14:
		return !z && n == v
	case 15:
		return z || n != v
	default:
		return false
	}
}

func (c *CPU) programFunctionCode() uint8 {
	if c.State.SR&supervisor != 0 {
		return 6
	}
	return 2
}

func readTransaction(address uint32, fc uint8, data uint16) Transaction {
	return Transaction{Kind: "r", Cycle: 4, FC: fc, Address: address, Size: 2,
		Data: data, UDS: true, LDS: true}
}

func writeTransaction(address uint32, fc uint8, data uint16) Transaction {
	return Transaction{Kind: "w", Cycle: 4, FC: fc, Address: address, Size: 2,
		Data: data, UDS: true, LDS: true}
}

func readByteTransaction(address uint32, fc uint8, data byte) Transaction {
	return byteTransaction("r", address, fc, data)
}

func writeByteTransaction(address uint32, fc uint8, data byte) Transaction {
	return byteTransaction("w", address, fc, data)
}

func byteTransaction(kind string, address uint32, fc uint8, data byte) Transaction {
	transaction := Transaction{Kind: kind, Cycle: 4, FC: fc, Address: address &^ 1, Size: 1}
	if address&1 == 0 {
		transaction.Data = uint16(data) << 8
		transaction.UDS = true
	} else {
		transaction.Data = uint16(data)
		transaction.LDS = true
	}
	return transaction
}

func (c *CPU) setLogicalFlags(value uint32, bits uint8) {
	c.State.SR &^= 0x000f
	mask := uint32(0xffff_ffff)
	negative := uint32(0x8000_0000)
	if bits == 8 {
		mask = 0x0000_00ff
		negative = 0x0000_0080
	} else if bits == 16 {
		mask = 0x0000_ffff
		negative = 0x0000_8000
	}
	value &= mask
	if value == 0 {
		c.State.SR |= 0x0004
	}
	if value&negative != 0 {
		c.State.SR |= 0x0008
	}
}
