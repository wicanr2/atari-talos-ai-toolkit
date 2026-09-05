package st

import "github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"

const firstColorSTVBLClock uint64 = 263*508 + 64

type Machine struct {
	CPU            m68k.CPU
	Memory         *Memory
	Instructions   uint64
	Interrupts     uint64
	Clocks         uint64
	firstVBLRaised bool
	vblPending     bool
}

func NewMachine(ramSize int, tosROM []byte) (*Machine, error) {
	memory, err := NewMemory(ramSize, tosROM)
	if err != nil {
		return nil, err
	}
	machine := &Machine{Memory: memory}
	machine.CPU.Bus = memory
	return machine, nil
}

func (m *Machine) Reset() error {
	m.Memory.ColdReset()
	if err := m.CPU.Reset(); err != nil {
		return err
	}
	m.Instructions = 0
	m.Interrupts = 0
	m.Clocks = 0
	m.firstVBLRaised = false
	m.vblPending = false
	return nil
}

func (m *Machine) Step() (m68k.StepResult, error) {
	if m.vblPending {
		result, accepted, err := m.CPU.AcceptAutovectorAt(4, m.Clocks)
		if err != nil {
			return result, err
		}
		if accepted {
			m.vblPending = false
			m.Interrupts++
			m.Clocks += uint64(result.Clocks)
			return result, nil
		}
	}
	result, err := m.CPU.StepAt(m.Clocks)
	if err != nil {
		return result, err
	}
	m.Instructions++
	m.Clocks += uint64(result.Clocks)
	if !m.firstVBLRaised && m.Clocks >= firstColorSTVBLClock {
		m.firstVBLRaised = true
		m.vblPending = true
	}
	return result, nil
}
