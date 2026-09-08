package m68k

// MOVEP uses alternate byte lanes; all phases are bus cycles (spec 150).
func (c *CPU) stepMOVEP(op uint16) (StepResult, error) {
	result := StepResult{}
	fc := uint8(1)
	if c.State.SR&supervisor != 0 {
		fc = 5
	}
	programFC := c.programFunctionCode()
	timed, _ := c.Bus.(TimedBus)
	appendPhase := func(tx Transaction, wait uint32) {
		if wait != 0 {
			result.Timeline = append(result.Timeline, BusPhase{Offset: result.Clocks, Cycles: wait})
			result.Clocks += wait
		}
		result.Transactions = append(result.Transactions, tx)
		result.Timeline = append(result.Timeline, BusPhase{Offset: result.Clocks, Cycles: 4, Transaction: &tx})
		result.Clocks += 4
	}
	prefetch := func() error {
		addr := c.State.PC & addressMask
		var word uint16
		var wait uint32
		var err error
		if timed != nil {
			word, wait, err = timed.ReadWordAt(addr, BusAccess{Clock: c.epoch + uint64(result.Clocks), FunctionCode: programFC})
		} else {
			word, err = c.Bus.ReadWord(addr, programFC)
		}
		if err != nil {
			return err
		}
		appendPhase(readTransaction(addr, programFC, word), wait)
		c.State.Prefetch[0], c.State.Prefetch[1] = c.State.Prefetch[1], word
		c.State.PC += 2
		return nil
	}
	disp := int16(c.State.Prefetch[1])
	address := c.addressRegister(uint8(op&7)) + uint32(int32(disp))
	reg := op >> 9 & 7
	n := 2
	if op&0x40 != 0 {
		n = 4
	}
	store := op&0x80 != 0
	value := c.State.D[reg]
	if err := prefetch(); err != nil {
		return result, err
	}
	var loaded uint32
	for i := 0; i < n; i++ {
		addr := (address + uint32(i*2)) & addressMask
		var b byte
		var wait uint32
		var err error
		access := BusAccess{Clock: c.epoch + uint64(result.Clocks), FunctionCode: fc}
		if store {
			b = byte(value >> uint((n-1-i)*8))
			if timed != nil {
				wait, err = timed.WriteByteAt(addr, b, access)
			} else {
				err = c.Bus.WriteByteFC(addr, b, fc)
			}
			if err == nil {
				appendPhase(writeByteTransaction(addr, fc, b), wait)
			}
		} else {
			if timed != nil {
				b, wait, err = timed.ReadByteAt(addr, access)
			} else {
				b, err = c.Bus.ReadByteFC(addr, fc)
			}
			if err == nil {
				appendPhase(readByteTransaction(addr, fc, b), wait)
				loaded = loaded<<8 | uint32(b)
			}
		}
		if err != nil {
			return result, err
		}
	}
	if !store {
		if n == 2 {
			loaded |= value & 0xffff0000
		}
		c.State.D[reg] = loaded
	}
	if err := prefetch(); err != nil {
		return result, err
	}
	return result, nil
}
