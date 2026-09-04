package m68k

import "testing"

func TestNOPPipelineAndFunctionCode(t *testing.T) {
	memory := SparseMemory{0x1004: 0x12, 0x1005: 0x34}
	cpu := CPU{Bus: memory, State: State{
		D:  [8]uint32{1, 2, 3, 4, 5, 6, 7, 8},
		A:  [7]uint32{9, 10, 11, 12, 13, 14, 15},
		SR: 0x2000, PC: 0x1004, Prefetch: [2]uint16{0x4e71, 0xabcd},
	}}
	wantD, wantA := cpu.State.D, cpu.State.A
	result, err := cpu.Step()
	if err != nil {
		t.Fatal(err)
	}
	if cpu.State.D != wantD || cpu.State.A != wantA {
		t.Fatal("NOP changed a data or address register")
	}
	if cpu.State.PC != 0x1006 || cpu.State.Prefetch != [2]uint16{0xabcd, 0x1234} {
		t.Fatalf("unexpected pipeline state: %#v", cpu.State)
	}
	if result.Clocks != 4 || len(result.Transactions) != 1 {
		t.Fatalf("unexpected step result: %#v", result)
	}
	want := Transaction{Kind: "r", Cycle: 4, FC: 6, Address: 0x1004, Size: 2, Data: 0x1234, UDS: true, LDS: true}
	if result.Transactions[0] != want {
		t.Fatalf("transaction = %#v, want %#v", result.Transactions[0], want)
	}
}

func TestStepFailsClosed(t *testing.T) {
	for _, test := range []CPU{
		{State: State{Prefetch: [2]uint16{0x4e71}}},
		{Bus: SparseMemory{}, State: State{Prefetch: [2]uint16{0x4e70}}},
		{Bus: SparseMemory{}, State: State{Prefetch: [2]uint16{0x4ec0}}},
	} {
		if _, err := test.Step(); err == nil {
			t.Fatal("Step unexpectedly succeeded")
		}
	}
}

func TestOddBranchTargetEntersAddressError(t *testing.T) {
	memory := SparseMemory{
		0x1002: 0xab, 0x1003: 0xcd,
		0x000c: 0x00, 0x000d: 0x00, 0x000e: 0x20, 0x000f: 0x00,
		0x2000: 0x4e, 0x2001: 0x71, 0x2002: 0x70, 0x2003: 0x01,
	}
	cpu := CPU{Bus: memory, State: State{
		D: [8]uint32{1, 2, 3, 4, 5, 6, 7, 8},
		A: [7]uint32{9, 10, 11, 12, 13, 14, 15},
		USP: 0x8000, SSP: 0x9000,
		SR: 0x8000, PC: 0x1004, Prefetch: [2]uint16{0x6001, 0xabcd},
	}}
	result, err := cpu.Step()
	if err != nil {
		t.Fatal(err)
	}
	if result.Clocks != 60 || len(result.Transactions) != 12 {
		t.Fatalf("unexpected address-error result: %#v", result)
	}
	if cpu.State.SSP != 0x8ff2 || cpu.State.USP != 0x8000 || cpu.State.SR != 0x2000 {
		t.Fatalf("unexpected exception state: %#v", cpu.State)
	}
	if cpu.State.PC != 0x2004 || cpu.State.Prefetch != [2]uint16{0x4e71, 0x7001} {
		t.Fatalf("unexpected handler pipeline: %#v", cpu.State)
	}
	frame := []uint16{0x6012, 0x0000, 0x1003, 0x6001, 0x8000, 0x0000, 0x1002}
	for i, want := range frame {
		got, err := memory.ReadWord(0x8ff2+uint32(i*2), 5)
		if err != nil || got != want {
			t.Fatalf("frame word %d = 0x%04x, %v; want 0x%04x", i, got, err, want)
		}
	}
}
