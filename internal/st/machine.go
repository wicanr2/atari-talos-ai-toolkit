package st

import "github.com/wicanr2/atari-talos-ai-toolkit/internal/m68k"

type Machine struct {
	CPU          m68k.CPU
	Memory       *Memory
	Instructions uint64
	Clocks       uint64
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
	if err := m.CPU.Reset(); err != nil {
		return err
	}
	m.Instructions = 0
	m.Clocks = 0
	return nil
}

func (m *Machine) Step() (m68k.StepResult, error) {
	result, err := m.CPU.Step()
	if err != nil {
		return result, err
	}
	m.Instructions++
	m.Clocks += uint64(result.Clocks)
	return result, nil
}
