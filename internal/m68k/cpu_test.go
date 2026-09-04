package m68k

import (
	"reflect"
	"testing"
)

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

func TestUNLKRestoresFrameAndActiveStack(t *testing.T) {
	memory := SparseMemory{
		0x8000: 0x12, 0x8001: 0x34, 0x8002: 0x56, 0x8003: 0x78,
		0x1004: 0x4e, 0x1005: 0x71,
	}
	cpu := CPU{Bus: memory, State: State{
		A: [7]uint32{0x8000}, USP: 0x9000, SSP: 0xa000, SR: 0x001f,
		PC: 0x1004, Prefetch: [2]uint16{0x4e58, 0xabcd},
	}}
	result, err := cpu.Step()
	if err != nil {
		t.Fatal(err)
	}
	if cpu.State.A[0] != 0x12345678 || cpu.State.USP != 0x8004 || cpu.State.SSP != 0xa000 || cpu.State.SR != 0x001f {
		t.Fatalf("unexpected UNLK state: %#v", cpu.State)
	}
	want := []Transaction{
		{Kind: "r", Cycle: 4, FC: 1, Address: 0x8000, Size: 2, Data: 0x1234, UDS: true, LDS: true},
		{Kind: "r", Cycle: 4, FC: 1, Address: 0x8002, Size: 2, Data: 0x5678, UDS: true, LDS: true},
		{Kind: "r", Cycle: 4, FC: 2, Address: 0x1004, Size: 2, Data: 0x4e71, UDS: true, LDS: true},
	}
	if result.Clocks != 12 || !reflect.DeepEqual(result.Transactions, want) {
		t.Fatalf("unexpected UNLK bus result: %#v", result)
	}
}

func TestMOVEBytePostIncrementA7AndBusLane(t *testing.T) {
	memory := SparseMemory{
		0x1004: 0x12, 0x1005: 0x34,
		0x8001: 0x80,
	}
	cpu := CPU{Bus: memory, State: State{
		D: [8]uint32{0x12345678}, USP: 0x8001,
		SR: 0x0011, PC: 0x1004, Prefetch: [2]uint16{0x101f, 0xabcd},
	}}
	result, err := cpu.Step()
	if err != nil {
		t.Fatal(err)
	}
	if cpu.State.D[0] != 0x12345680 || cpu.State.USP != 0x8003 || cpu.State.SR != 0x0018 {
		t.Fatalf("unexpected MOVE.B state: %#v", cpu.State)
	}
	want := Transaction{Kind: "r", Cycle: 4, FC: 1, Address: 0x8000, Size: 1,
		Data: 0x0080, LDS: true}
	if result.Clocks != 8 || len(result.Transactions) != 2 || result.Transactions[0] != want {
		t.Fatalf("unexpected MOVE.B bus result: %#v", result)
	}
}

func TestMOVEBytePostIncrementSourceAndDestinationAlias(t *testing.T) {
	memory := SparseMemory{
		0x1004: 0x4e, 0x1005: 0x71,
		0x8000: 0x12, 0x8001: 0xff,
	}
	cpu := CPU{Bus: memory, State: State{
		A: [7]uint32{0x8000}, SR: 0x0011,
		PC: 0x1004, Prefetch: [2]uint16{0x10d8, 0xabcd},
	}}
	result, err := cpu.Step()
	if err != nil {
		t.Fatal(err)
	}
	if cpu.State.A[0] != 0x8002 || memory[0x8001] != 0x12 || cpu.State.SR != 0x0010 {
		t.Fatalf("unexpected aliased MOVE.B state: %#v RAM=%#v", cpu.State, memory)
	}
	want := []Transaction{
		{Kind: "r", Cycle: 4, FC: 1, Address: 0x8000, Size: 1, Data: 0x1200, UDS: true},
		{Kind: "w", Cycle: 4, FC: 1, Address: 0x8000, Size: 1, Data: 0x0012, LDS: true},
		{Kind: "r", Cycle: 4, FC: 2, Address: 0x1004, Size: 2, Data: 0x4e71, UDS: true, LDS: true},
	}
	if result.Clocks != 12 || !reflect.DeepEqual(result.Transactions, want) {
		t.Fatalf("unexpected aliased MOVE.B bus result: %#v", result)
	}
}

func TestADDAWordSignExtendsAndPreservesSR(t *testing.T) {
	memory := SparseMemory{
		0x1004: 0x4e, 0x1005: 0x71,
		0x1006: 0x70, 0x1007: 0x01,
	}
	cpu := CPU{Bus: memory, State: State{
		A: [7]uint32{0x100}, SR: 0x001f,
		PC: 0x1004, Prefetch: [2]uint16{0xd0fc, 0xffff},
	}}
	result, err := cpu.Step()
	if err != nil {
		t.Fatal(err)
	}
	if cpu.State.A[0] != 0xff || cpu.State.SR != 0x001f {
		t.Fatalf("unexpected ADDA.W state: %#v", cpu.State)
	}
	if result.Clocks != 12 || cpu.State.PC != 0x1008 || cpu.State.Prefetch != [2]uint16{0x4e71, 0x7001} {
		t.Fatalf("unexpected ADDA.W pipeline: state=%#v result=%#v", cpu.State, result)
	}
}

func TestADDALongWritesActiveStackAndPreservesSR(t *testing.T) {
	memory := SparseMemory{0x1004: 0x4e, 0x1005: 0x71}
	cpu := CPU{Bus: memory, State: State{
		D: [8]uint32{2}, USP: 0x8000, SSP: 0x9000, SR: 0x001f,
		PC: 0x1004, Prefetch: [2]uint16{0xdfc0, 0xabcd},
	}}
	result, err := cpu.Step()
	if err != nil {
		t.Fatal(err)
	}
	if cpu.State.USP != 0x8002 || cpu.State.SSP != 0x9000 || cpu.State.SR != 0x001f {
		t.Fatalf("unexpected ADDA.L state: %#v", cpu.State)
	}
	if result.Clocks != 8 {
		t.Fatalf("unexpected ADDA.L clocks: %#v", result)
	}
}

func TestANDByteDataRegistersUpdateLogicalFlags(t *testing.T) {
	memory := SparseMemory{0x1004: 0x4e, 0x1005: 0x71}
	cpu := CPU{Bus: memory, State: State{
		D: [8]uint32{0x0000000f, 0x123456f0}, SR: 0x0013,
		PC: 0x1004, Prefetch: [2]uint16{0xc200, 0xabcd},
	}}
	result, err := cpu.Step()
	if err != nil {
		t.Fatal(err)
	}
	if cpu.State.D[1] != 0x12345600 || cpu.State.SR != 0x0014 {
		t.Fatalf("unexpected AND.B state: %#v", cpu.State)
	}
	if result.Clocks != 4 {
		t.Fatalf("unexpected AND.B clocks: %#v", result)
	}
}

func TestANDLongMemoryUsesRMWBusOrder(t *testing.T) {
	memory := SparseMemory{
		0x1004: 0x4e, 0x1005: 0x71,
		0x8000: 0xff, 0x8001: 0xff, 0x8002: 0x00, 0x8003: 0xff,
	}
	cpu := CPU{Bus: memory, State: State{
		D: [8]uint32{0x0f0f0f0f}, A: [7]uint32{0x8000}, SR: 0x0013,
		PC: 0x1004, Prefetch: [2]uint16{0xc190, 0xabcd},
	}}
	result, err := cpu.Step()
	if err != nil {
		t.Fatal(err)
	}
	if memory[0x8000] != 0x0f || memory[0x8001] != 0x0f || memory[0x8002] != 0x00 || memory[0x8003] != 0x0f {
		t.Fatalf("unexpected AND.L memory: %#v", memory)
	}
	wantKinds := []string{"r", "r", "r", "w", "w"}
	wantAddresses := []uint32{0x8000, 0x8002, 0x1004, 0x8002, 0x8000}
	if result.Clocks != 20 || len(result.Transactions) != len(wantKinds) {
		t.Fatalf("unexpected AND.L bus result: %#v", result)
	}
	for i := range wantKinds {
		if result.Transactions[i].Kind != wantKinds[i] || result.Transactions[i].Address != wantAddresses[i] {
			t.Fatalf("transaction %d = %#v", i, result.Transactions[i])
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
		D:   [8]uint32{1, 2, 3, 4, 5, 6, 7, 8},
		A:   [7]uint32{9, 10, 11, 12, 13, 14, 15},
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
