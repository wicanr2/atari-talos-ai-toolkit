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
			return StepResult{}, fmt.Errorf("m68k: branch to odd address 0x%08x requires address error", target)
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
