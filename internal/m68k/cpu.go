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
	ReadWord(address uint32, functionCode uint8) (uint16, error)
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

type controlEA struct {
	target       uint32
	returnPC     uint32
	extraClocks  uint32
	transactions []Transaction
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
	case 6:
		index, err := c.briefIndex(c.State.Prefetch[1])
		if err != nil {
			return controlEA{}, err
		}
		result.target = c.addressRegister(reg) + index + uint32(int32(int8(c.State.Prefetch[1])))
		result.returnPC = c.State.PC
		result.extraClocks = 6
	case 7:
		switch reg {
		case 0:
			result.target = uint32(int32(int16(c.State.Prefetch[1])))
			result.returnPC = c.State.PC
			result.extraClocks = 2
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
			result.transactions = []Transaction{readTransaction(address, fc, low)}
		case 2:
			result.target = basePC + uint32(int32(int16(c.State.Prefetch[1])))
			result.returnPC = c.State.PC
			result.extraClocks = 2
		case 3:
			index, err := c.briefIndex(c.State.Prefetch[1])
			if err != nil {
				return controlEA{}, err
			}
			result.target = basePC + index + uint32(int32(int8(c.State.Prefetch[1])))
			result.returnPC = c.State.PC
			result.extraClocks = 6
		default:
			return controlEA{}, fmt.Errorf("m68k: invalid control addressing mode %d:%d", mode, reg)
		}
	default:
		return controlEA{}, fmt.Errorf("m68k: invalid control addressing mode %d:%d", mode, reg)
	}
	return result, nil
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
	originalSR := c.State.SR
	originalFC := c.programFunctionCode()
	faultBusAddress := target & addressMask &^ 1

	newSP := c.State.SSP - 14
	ssw := opcode&0xffe0 | 0x0010 | uint16(originalFC)
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
		Kind: "re", Cycle: 4, FC: originalFC, Address: faultBusAddress,
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

func (c *CPU) setLogicalFlags(value uint32, bits uint8) {
	c.State.SR &^= 0x000f
	mask := uint32(0xffff_ffff)
	negative := uint32(0x8000_0000)
	if bits == 16 {
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
