package m68k

import (
	"fmt"
	"reflect"
	"testing"
)

type resetRead struct {
	address uint32
	fc      uint8
}

type resetRecordingBus struct {
	SparseMemory
	reads       []resetRead
	failAddress uint32
}

func (b *resetRecordingBus) ReadWord(address uint32, fc uint8) (uint16, error) {
	b.reads = append(b.reads, resetRead{address: address, fc: fc})
	if b.failAddress != 0 && address == b.failAddress {
		return 0, fmt.Errorf("forced reset read failure at 0x%x", address)
	}
	return b.SparseMemory.ReadWord(address, fc)
}

func TestResetReadsVectorsAndPrefetchWithSupervisorProgramFC(t *testing.T) {
	memory := SparseMemory{
		0: 0x00, 1: 0x01, 2: 0x00, 3: 0x00,
		4: 0x00, 5: 0xfc, 6: 0x00, 7: 0x30,
		0xfc0030: 0x60, 0xfc0031: 0x00, 0xfc0032: 0x00, 0xfc0033: 0x1c,
	}
	bus := &resetRecordingBus{SparseMemory: memory}
	cpu := CPU{Bus: bus, State: State{D: [8]uint32{0x12345678}, A: [7]uint32{0x87654321},
		USP: 0xabcdef, SSP: 1, SR: 2, PC: 3, Prefetch: [2]uint16{4, 5}}}
	if err := cpu.Reset(); err != nil {
		t.Fatal(err)
	}
	wantReads := []resetRead{{0, 6}, {2, 6}, {4, 6}, {6, 6}, {0xfc0030, 6}, {0xfc0032, 6}}
	if !reflect.DeepEqual(bus.reads, wantReads) {
		t.Fatalf("reset reads=%+v want %+v", bus.reads, wantReads)
	}
	if cpu.State.SSP != 0x00010000 || cpu.State.SR != 0x2700 || cpu.State.PC != 0xfc0034 ||
		cpu.State.Prefetch != [2]uint16{0x6000, 0x001c} || cpu.State.D[0] != 0x12345678 ||
		cpu.State.A[0] != 0x87654321 || cpu.State.USP != 0xabcdef {
		t.Fatalf("reset state=%+v", cpu.State)
	}
}

func TestResetFailureDoesNotCommitStagedState(t *testing.T) {
	memory := SparseMemory{
		0: 0x00, 1: 0x01, 2: 0x00, 3: 0x00,
		4: 0x00, 5: 0xfc, 6: 0x00, 7: 0x30,
		0xfc0030: 0x60, 0xfc0031: 0x00, 0xfc0032: 0x00, 0xfc0033: 0x1c,
	}
	initial := State{D: [8]uint32{1}, A: [7]uint32{2}, USP: 3, SSP: 4,
		SR: 5, PC: 6, Prefetch: [2]uint16{7, 8}}
	cpu := CPU{Bus: &resetRecordingBus{SparseMemory: memory, failAddress: 0xfc0032}, State: initial}
	if err := cpu.Reset(); err == nil {
		t.Fatal("reset unexpectedly succeeded")
	}
	if cpu.State != initial {
		t.Fatalf("failed reset committed state: %+v", cpu.State)
	}

	odd := SparseMemory{0: 0, 1: 1, 2: 0, 3: 0, 4: 0, 5: 0xfc, 6: 0, 7: 0x31}
	cpu = CPU{Bus: odd, State: initial}
	if err := cpu.Reset(); err == nil || cpu.State != initial {
		t.Fatalf("odd-PC reset err=%v state=%+v", err, cpu.State)
	}
	cpu = CPU{State: initial}
	if err := cpu.Reset(); err == nil || cpu.State != initial {
		t.Fatalf("nil-bus reset err=%v state=%+v", err, cpu.State)
	}
}

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

func TestMC68000MOVECEntersIllegalInstructionVector4(t *testing.T) {
	for _, test := range []struct {
		name      string
		opcode    uint16
		initialSR uint16
		finalSR   uint16
	}{
		{"control_to_register_supervisor_trace", 0x4e7a, 0xa704, 0x2704},
		{"register_to_control_user_trace", 0x4e7b, 0x8004, 0x2004},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory := SparseMemory{
				0x0010: 0x00, 0x0011: 0x00, 0x0012: 0x10, 0x0013: 0x00,
				0x1000: 0x21, 0x1001: 0xfc, 0x1002: 0x00, 0x1003: 0xfc,
			}
			cpu := CPU{Bus: memory, State: State{
				D: [8]uint32{0x12345678}, A: [7]uint32{0x87654321},
				USP: 0x4000, SSP: 0x3000, SR: test.initialSR, PC: 0x2004,
				Prefetch: [2]uint16{test.opcode, 0x0801},
			}}
			result, err := cpu.Step()
			if err != nil {
				t.Fatal(err)
			}
			if result.Clocks != 36 || cpu.State.SSP != 0x2ffa || cpu.State.USP != 0x4000 ||
				cpu.State.SR != test.finalSR || cpu.State.PC != 0x1004 ||
				cpu.State.Prefetch != [2]uint16{0x21fc, 0x00fc} ||
				cpu.State.D[0] != 0x12345678 || cpu.State.A[0] != 0x87654321 {
				t.Fatalf("result=%+v state=%+v", result, cpu.State)
			}
			wantFrame := []uint16{test.initialSR, 0x0000, 0x2000}
			for index, want := range wantFrame {
				got, readErr := memory.ReadWord(0x2ffa+uint32(index*2), 5)
				if readErr != nil || got != want {
					t.Fatalf("frame[%d]=%04x/%v want %04x", index, got, readErr, want)
				}
			}
			wantTransactions := []Transaction{
				writeTransaction(0x2ffe, 5, 0x2000),
				writeTransaction(0x2ffa, 5, test.initialSR),
				writeTransaction(0x2ffc, 5, 0x0000),
				readTransaction(0x0010, 5, 0x0000),
				readTransaction(0x0012, 5, 0x1000),
				readTransaction(0x1000, 6, 0x21fc),
				readTransaction(0x1002, 6, 0x00fc),
			}
			if !reflect.DeepEqual(result.Transactions, wantTransactions) {
				t.Fatalf("transactions=%+v want %+v", result.Transactions, wantTransactions)
			}
		})
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

func TestBitOperationsRejectIllegalDestinationWithoutConsumingExtension(t *testing.T) {
	for _, opcode := range []uint16{0x08c8, 0x08fa} {
		initial := State{D: [8]uint32{1}, SR: 0x2004, PC: 0x1004,
			Prefetch: [2]uint16{opcode, 0x001f}}
		cpu := CPU{Bus: SparseMemory{0x1004: 0x4e, 0x1005: 0x71}, State: initial}
		if _, err := cpu.Step(); err == nil {
			t.Fatalf("opcode %04x unexpectedly accepted", opcode)
		}
		if cpu.State != initial {
			t.Fatalf("opcode %04x changed state on rejection: %+v", opcode, cpu.State)
		}
	}
}

func TestTASRejectsIllegalDestinationWithoutConsumingExtension(t *testing.T) {
	for _, opcode := range []uint16{0x4ac8, 0x4afa, 0x4afc} {
		initial := State{D: [8]uint32{1}, SR: 0x2004, PC: 0x1004,
			Prefetch: [2]uint16{opcode, 0x001f}}
		cpu := CPU{Bus: SparseMemory{0x1004: 0x4e, 0x1005: 0x71}, State: initial}
		if _, err := cpu.Step(); err == nil {
			t.Fatalf("opcode %04x unexpectedly accepted", opcode)
		}
		if cpu.State != initial {
			t.Fatalf("opcode %04x changed state on rejection: %+v", opcode, cpu.State)
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

// TestLineFEntersEmulatorVector11 covers spec 059. Line 1111 is reserved for
// the coprocessor interface and an MC68000 has none, so the whole $Fxxx range
// is a line-F emulator exception — the three endpoints here stand for the
// range, not for three special encodings.
func TestLineFEntersEmulatorVector11(t *testing.T) {
	for _, test := range []struct {
		name      string
		opcode    uint16
		initialSR uint16
		finalSR   uint16
	}{
		{"low_end_supervisor_trace", 0xf000, 0xa704, 0x2704},
		{"emutos_pmove_probe", 0xf010, 0x2700, 0x2700},
		{"high_end_user_trace", 0xffff, 0x8004, 0x2004},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory := SparseMemory{
				// Vector 11 lives at $2C, which is what EmuTOS writes $FC00D4
				// into; here it points at $1000.
				0x002c: 0x00, 0x002d: 0x00, 0x002e: 0x10, 0x002f: 0x00,
				0x1000: 0x21, 0x1001: 0xfc, 0x1002: 0x00, 0x1003: 0xfc,
			}
			cpu := CPU{Bus: memory, State: State{
				D: [8]uint32{0x12345678}, A: [7]uint32{0x87654321},
				USP: 0x4000, SSP: 0x3000, SR: test.initialSR, PC: 0x2004,
				Prefetch: [2]uint16{test.opcode, 0x0800},
			}}
			result, err := cpu.Step()
			if err != nil {
				t.Fatal(err)
			}
			if result.Clocks != 36 || cpu.State.SSP != 0x2ffa || cpu.State.USP != 0x4000 ||
				cpu.State.SR != test.finalSR || cpu.State.PC != 0x1004 ||
				cpu.State.Prefetch != [2]uint16{0x21fc, 0x00fc} ||
				cpu.State.D[0] != 0x12345678 || cpu.State.A[0] != 0x87654321 {
				t.Fatalf("result=%+v state=%+v", result, cpu.State)
			}
			// The saved PC is the opcode address, not the next one: $2004 - 4.
			wantFrame := []uint16{test.initialSR, 0x0000, 0x2000}
			for index, want := range wantFrame {
				got, readErr := memory.ReadWord(0x2ffa+uint32(index*2), 5)
				if readErr != nil || got != want {
					t.Fatalf("frame[%d]=%04x/%v want %04x", index, got, readErr, want)
				}
			}
			wantTransactions := []Transaction{
				writeTransaction(0x2ffe, 5, 0x2000),
				writeTransaction(0x2ffa, 5, test.initialSR),
				writeTransaction(0x2ffc, 5, 0x0000),
				readTransaction(0x002c, 5, 0x0000),
				readTransaction(0x002e, 5, 0x1000),
				readTransaction(0x1000, 6, 0x21fc),
				readTransaction(0x1002, 6, 0x00fc),
			}
			if !reflect.DeepEqual(result.Transactions, wantTransactions) {
				t.Fatalf("transactions=%+v want %+v", result.Transactions, wantTransactions)
			}
		})
	}
}

// The extension word is not decoded: the exception is decided the moment the
// opcode is fetched, so $F010 behaves the same whatever follows it.
func TestLineFIgnoresTheExtensionWord(t *testing.T) {
	var first State
	for index, extension := range []uint16{0x0800, 0x0000, 0xffff} {
		memory := SparseMemory{
			0x002c: 0x00, 0x002d: 0x00, 0x002e: 0x10, 0x002f: 0x00,
			0x1000: 0x21, 0x1001: 0xfc, 0x1002: 0x00, 0x1003: 0xfc,
		}
		cpu := CPU{Bus: memory, State: State{
			SSP: 0x3000, SR: 0x2700, PC: 0x2004,
			Prefetch: [2]uint16{0xf010, extension},
		}}
		if _, err := cpu.Step(); err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			first = cpu.State
			continue
		}
		if cpu.State != first {
			t.Fatalf("extension %04x changed the outcome: %+v want %+v", extension, cpu.State, first)
		}
	}
}
