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
