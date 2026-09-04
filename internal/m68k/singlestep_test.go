package m68k

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type corpusState struct {
	CPU State
	RAM SparseMemory
}

type corpusTest struct {
	Name         string
	Initial      corpusState
	Final        corpusState
	Transactions []Transaction
	Clocks       uint32
}

func TestSingleStepNOP(t *testing.T) {
	testSingleStepCorpus(t, "NOP.json.bin")
}

func TestSingleStepMOVEQ(t *testing.T) {
	testSingleStepCorpus(t, "MOVE.q.json.bin")
}

func TestSingleStepSWAP(t *testing.T) {
	testSingleStepCorpus(t, "SWAP.json.bin")
}

func TestSingleStepEXTW(t *testing.T) {
	testSingleStepCorpus(t, "EXT.w.json.bin")
}

func TestSingleStepEXTL(t *testing.T) {
	testSingleStepCorpus(t, "EXT.l.json.bin")
}

func TestSingleStepCLRByte(t *testing.T) {
	testSingleStepCorpus(t, "CLR.b.json.bin")
}

func TestSingleStepCLRWord(t *testing.T) {
	testSingleStepCorpus(t, "CLR.w.json.bin")
}

func TestSingleStepCLRLong(t *testing.T) {
	testSingleStepCorpus(t, "CLR.l.json.bin")
}

func TestSingleStepMOVEMWord(t *testing.T) {
	testSingleStepCorpus(t, "MOVEM.w.json.bin")
}

func TestSingleStepMOVEMLong(t *testing.T) {
	testSingleStepCorpus(t, "MOVEM.l.json.bin")
}

func TestSingleStepLINK(t *testing.T) {
	testSingleStepCorpus(t, "LINK.json.bin")
}

func TestSingleStepBcc(t *testing.T) {
	testSingleStepCorpus(t, "Bcc.json.bin")
}

func TestSingleStepBSR(t *testing.T) {
	testSingleStepCorpus(t, "BSR.json.bin")
}

func TestSingleStepRTS(t *testing.T) {
	testSingleStepCorpus(t, "RTS.json.bin")
}

func TestSingleStepJMP(t *testing.T) {
	testSingleStepCorpus(t, "JMP.json.bin")
}

func TestSingleStepJSR(t *testing.T) {
	testSingleStepCorpus(t, "JSR.json.bin")
}

func TestSingleStepLEA(t *testing.T) {
	testSingleStepCorpus(t, "LEA.json.bin")
}

func TestSingleStepPEA(t *testing.T) {
	testSingleStepCorpus(t, "PEA.json.bin")
}

func TestSingleStepMOVEByteSourcesToDn(t *testing.T) {
	testSingleStepCorpusFiltered(t, "MOVE.b.json.bin", 384, func(test corpusTest) bool {
		return test.Initial.CPU.Prefetch[0]>>6&7 == 0
	})
}

func TestSingleStepMOVEByteMemoryDestinations(t *testing.T) {
	testSingleStepCorpusFiltered(t, "MOVE.b.json.bin", 2116, func(test corpusTest) bool {
		return test.Initial.CPU.Prefetch[0]>>6&7 != 0
	})
}

func TestSingleStepMOVEWordNormal(t *testing.T) {
	testSingleStepCorpusFiltered(t, "MOVE.w.json.bin", 1013, func(test corpusTest) bool {
		return !hasTransactionKind(test.Transactions, "re") && !hasTransactionKind(test.Transactions, "we")
	})
}

func TestSingleStepMOVEWordReadAddressErrors(t *testing.T) {
	testSingleStepCorpusFiltered(t, "MOVE.w.json.bin", 839, func(test corpusTest) bool {
		return hasTransactionKind(test.Transactions, "re")
	})
}

func TestSingleStepMOVEWordWriteAddressErrors(t *testing.T) {
	testSingleStepCorpusFiltered(t, "MOVE.w.json.bin", 648, func(test corpusTest) bool {
		return hasTransactionKind(test.Transactions, "we")
	})
}

func TestSingleStepMOVELongNormal(t *testing.T) {
	testSingleStepCorpusFiltered(t, "MOVE.l.json.bin", 1013, func(test corpusTest) bool {
		return !hasTransactionKind(test.Transactions, "re") && !hasTransactionKind(test.Transactions, "we")
	})
}

func TestSingleStepMOVELongReadAddressErrors(t *testing.T) {
	testSingleStepCorpusFiltered(t, "MOVE.l.json.bin", 869, func(test corpusTest) bool {
		return hasTransactionKind(test.Transactions, "re")
	})
}

func TestSingleStepMOVELongWriteAddressErrors(t *testing.T) {
	testSingleStepCorpusFiltered(t, "MOVE.l.json.bin", 618, func(test corpusTest) bool {
		return hasTransactionKind(test.Transactions, "we")
	})
}

func TestSingleStepMOVEAWordNormal(t *testing.T) {
	testSingleStepCorpusFiltered(t, "MOVEA.w.json.bin", 1658, func(test corpusTest) bool {
		return !hasTransactionKind(test.Transactions, "re")
	})
}

func TestSingleStepMOVEAWordReadAddressErrors(t *testing.T) {
	testSingleStepCorpusFiltered(t, "MOVEA.w.json.bin", 842, func(test corpusTest) bool {
		return hasTransactionKind(test.Transactions, "re")
	})
}

func TestSingleStepMOVEALongNormal(t *testing.T) {
	testSingleStepCorpusFiltered(t, "MOVEA.l.json.bin", 1655, func(test corpusTest) bool {
		return !hasTransactionKind(test.Transactions, "re")
	})
}

func TestSingleStepMOVEALongReadAddressErrors(t *testing.T) {
	testSingleStepCorpusFiltered(t, "MOVEA.l.json.bin", 845, func(test corpusTest) bool {
		return hasTransactionKind(test.Transactions, "re")
	})
}

func TestSingleStepADDAWordNormal(t *testing.T) {
	testSingleStepCorpusFiltered(t, "ADDA.w.json.bin", 1683, func(test corpusTest) bool {
		return !hasTransactionKind(test.Transactions, "re")
	})
}

func TestSingleStepADDAWordReadAddressErrors(t *testing.T) {
	testSingleStepCorpusFiltered(t, "ADDA.w.json.bin", 817, func(test corpusTest) bool {
		return hasTransactionKind(test.Transactions, "re")
	})
}

func TestSingleStepADDALongNormal(t *testing.T) {
	testSingleStepCorpusFiltered(t, "ADDA.l.json.bin", 1675, func(test corpusTest) bool {
		return !hasTransactionKind(test.Transactions, "re")
	})
}

func TestSingleStepADDALongReadAddressErrors(t *testing.T) {
	testSingleStepCorpusFiltered(t, "ADDA.l.json.bin", 825, func(test corpusTest) bool {
		return hasTransactionKind(test.Transactions, "re")
	})
}

func TestSingleStepANDByteEAToDn(t *testing.T) {
	testSingleStepCorpusFiltered(t, "AND.b.json.bin", 1317, andEAToDn)
}

func TestSingleStepANDByteDnToEA(t *testing.T) {
	testSingleStepCorpusFiltered(t, "AND.b.json.bin", 1007, func(test corpusTest) bool {
		opcode := test.Initial.CPU.Prefetch[0]
		return opcode&0xf000 == 0xc000 && opcode>>6&7 == 4
	})
}

func TestSingleStepANDImmediateByte(t *testing.T) {
	testSingleStepCorpusFiltered(t, "AND.b.json.bin", 176, func(test corpusTest) bool {
		return test.Initial.CPU.Prefetch[0]&0xff00 == 0x0200
	})
}

func TestSingleStepANDImmediateWord(t *testing.T) {
	testSingleStepCorpusFiltered(t, "AND.w.json.bin", 158, func(test corpusTest) bool {
		return test.Initial.CPU.Prefetch[0]&0xff00 == 0x0200
	})
}

func TestSingleStepANDImmediateLong(t *testing.T) {
	testSingleStepCorpusFiltered(t, "AND.l.json.bin", 129, func(test corpusTest) bool {
		return test.Initial.CPU.Prefetch[0]&0xff00 == 0x0200
	})
}

func TestSingleStepADDByte(t *testing.T) {
	testSingleStepCorpus(t, "ADD.b.json.bin")
}

func TestSingleStepADDWord(t *testing.T) {
	testSingleStepCorpus(t, "ADD.w.json.bin")
}

func TestSingleStepADDLong(t *testing.T) {
	testSingleStepCorpus(t, "ADD.l.json.bin")
}

func TestSingleStepCMPByte(t *testing.T) {
	testSingleStepCorpusFiltered(t, "CMP.b.json.bin", 1991, isCMP)
}

func TestSingleStepCMPWord(t *testing.T) {
	testSingleStepCorpusFiltered(t, "CMP.w.json.bin", 2032, isCMP)
}

func TestSingleStepCMPLong(t *testing.T) {
	testSingleStepCorpusFiltered(t, "CMP.l.json.bin", 2063, isCMP)
}

func TestSingleStepCMPMemoryByte(t *testing.T) {
	testSingleStepCorpusFiltered(t, "CMP.b.json.bin", 276, isCMPMemory)
}

func TestSingleStepCMPMemoryWord(t *testing.T) {
	testSingleStepCorpusFiltered(t, "CMP.w.json.bin", 261, isCMPMemory)
}

func TestSingleStepCMPMemoryLong(t *testing.T) {
	testSingleStepCorpusFiltered(t, "CMP.l.json.bin", 247, isCMPMemory)
}

func TestSingleStepCMPImmediateByte(t *testing.T) {
	testSingleStepCorpusFiltered(t, "CMP.b.json.bin", 233, isCMPImmediate)
}

func TestSingleStepCMPImmediateWord(t *testing.T) {
	testSingleStepCorpusFiltered(t, "CMP.w.json.bin", 207, isCMPImmediate)
}

func TestSingleStepCMPImmediateLong(t *testing.T) {
	testSingleStepCorpusFiltered(t, "CMP.l.json.bin", 190, isCMPImmediate)
}

func isCMP(test corpusTest) bool {
	opcode := test.Initial.CPU.Prefetch[0]
	return opcode&0xf000 == 0xb000 && opcode>>6&7 <= 2
}

func isCMPMemory(test corpusTest) bool {
	return test.Initial.CPU.Prefetch[0]&0xf138 == 0xb108
}

func isCMPImmediate(test corpusTest) bool {
	return test.Initial.CPU.Prefetch[0]&0xff00 == 0x0c00
}

func TestSingleStepANDWordEAToDn(t *testing.T) {
	testSingleStepCorpusFiltered(t, "AND.w.json.bin", 1333, andEAToDn)
}

func TestSingleStepANDLongEAToDn(t *testing.T) {
	testSingleStepCorpusFiltered(t, "AND.l.json.bin", 1315, andEAToDn)
}

func TestSingleStepANDWordDnToEANormal(t *testing.T) {
	testSingleStepCorpusFiltered(t, "AND.w.json.bin", 512, func(test corpusTest) bool {
		opcode := test.Initial.CPU.Prefetch[0]
		return opcode&0xf000 == 0xc000 && opcode>>6&7 == 5 && !hasTransactionKind(test.Transactions, "re")
	})
}

func TestSingleStepANDLongDnToEANormal(t *testing.T) {
	testSingleStepCorpusFiltered(t, "AND.l.json.bin", 597, func(test corpusTest) bool {
		opcode := test.Initial.CPU.Prefetch[0]
		return opcode&0xf000 == 0xc000 && opcode>>6&7 == 6 && !hasTransactionKind(test.Transactions, "re")
	})
}

func TestSingleStepANDWordDnToEAReadAddressErrors(t *testing.T) {
	testSingleStepCorpusFiltered(t, "AND.w.json.bin", 497, func(test corpusTest) bool {
		opcode := test.Initial.CPU.Prefetch[0]
		return opcode&0xf000 == 0xc000 && opcode>>6&7 == 5 && hasTransactionKind(test.Transactions, "re")
	})
}

func TestSingleStepANDLongDnToEAReadAddressErrors(t *testing.T) {
	testSingleStepCorpusFiltered(t, "AND.l.json.bin", 459, func(test corpusTest) bool {
		opcode := test.Initial.CPU.Prefetch[0]
		return opcode&0xf000 == 0xc000 && opcode>>6&7 == 6 && hasTransactionKind(test.Transactions, "re")
	})
}

func andEAToDn(test corpusTest) bool {
	opcode := test.Initial.CPU.Prefetch[0]
	return opcode&0xf000 == 0xc000 && opcode>>6&7 <= 2
}

func hasTransactionKind(transactions []Transaction, kind string) bool {
	for _, transaction := range transactions {
		if transaction.Kind == kind {
			return true
		}
	}
	return false
}

func testSingleStepCorpus(t *testing.T, name string) {
	testSingleStepCorpusFiltered(t, name, 2500, func(corpusTest) bool { return true })
}

func testSingleStepCorpusFiltered(t *testing.T, name string, want int, accept func(corpusTest) bool) {
	t.Helper()
	root := os.Getenv("TALOS_M68000_TESTS")
	if root == "" {
		t.Skip("TALOS_M68000_TESTS is not set; external m68000 corpus not available")
	}
	tests, err := readCorpus(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	accepted := 0
	for _, test := range tests {
		if !accept(test) {
			continue
		}
		accepted++
		test := test
		t.Run(test.Name, func(t *testing.T) {
			cpu := CPU{State: test.Initial.CPU, Bus: test.Initial.RAM}
			result, err := cpu.Step()
			if err != nil {
				t.Fatal(err)
			}
			if cpu.State != test.Final.CPU {
				t.Fatalf("state mismatch\n got: %#v\nwant: %#v", cpu.State, test.Final.CPU)
			}
			if !equalSparseMemory(test.Initial.RAM, test.Final.RAM) {
				t.Fatalf("RAM mismatch\n got: %#v\nwant: %#v", test.Initial.RAM, test.Final.RAM)
			}
			if result.Clocks != test.Clocks || !reflect.DeepEqual(result.Transactions, test.Transactions) {
				t.Fatalf("bus mismatch\n got: %#v\nwant: clocks=%d transactions=%#v", result, test.Clocks, test.Transactions)
			}
		})
	}
	if accepted != want {
		t.Fatalf("%s accepted %d tests, want %d", name, accepted, want)
	}
}

func equalSparseMemory(got, want SparseMemory) bool {
	// Corpus byte writes retain the inactive bus lane as an explicit zero;
	// SparseMemory leaves the same unknown zero lane absent.
	for address, value := range got {
		if want[address] != value {
			return false
		}
	}
	for address, value := range want {
		if got[address] != value {
			return false
		}
	}
	return true
}

func readCorpus(path string) ([]corpusTest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	r := bufio.NewReader(file)
	magic, err := readU32(r)
	if err != nil || magic != 0x1a3f5d71 {
		return nil, fmt.Errorf("m68000 corpus header: magic=0x%08x err=%v", magic, err)
	}
	count, err := readU32(r)
	if err != nil {
		return nil, err
	}
	tests := make([]corpusTest, 0, count)
	for i := uint32(0); i < count; i++ {
		test, err := readCorpusTest(r)
		if err != nil {
			return nil, fmt.Errorf("test %d: %w", i, err)
		}
		tests = append(tests, test)
	}
	if _, err := r.ReadByte(); err != io.EOF {
		return nil, fmt.Errorf("m68000 corpus has trailing data")
	}
	return tests, nil
}

func readCorpusTest(r io.Reader) (corpusTest, error) {
	if err := readBlockHeader(r, 0xabc12367); err != nil {
		return corpusTest{}, err
	}
	name, err := readName(r)
	if err != nil {
		return corpusTest{}, err
	}
	initial, err := readCorpusState(r)
	if err != nil {
		return corpusTest{}, err
	}
	final, err := readCorpusState(r)
	if err != nil {
		return corpusTest{}, err
	}
	transactions, clocks, err := readTransactions(r)
	return corpusTest{Name: name, Initial: initial, Final: final, Transactions: transactions, Clocks: clocks}, err
}

func readName(r io.Reader) (string, error) {
	if err := readBlockHeader(r, 0x89abcdef); err != nil {
		return "", err
	}
	n, err := readU32(r)
	if err != nil || n > 4096 {
		return "", fmt.Errorf("invalid name length %d: %v", n, err)
	}
	name := make([]byte, n)
	_, err = io.ReadFull(r, name)
	return string(name), err
}

func readCorpusState(r io.Reader) (corpusState, error) {
	if err := readBlockHeader(r, 0x01234567); err != nil {
		return corpusState{}, err
	}
	var state State
	for i := range state.D {
		v, err := readU32(r)
		if err != nil {
			return corpusState{}, err
		}
		state.D[i] = v
	}
	for i := range state.A {
		v, err := readU32(r)
		if err != nil {
			return corpusState{}, err
		}
		state.A[i] = v
	}
	var err error
	if state.USP, err = readU32(r); err != nil {
		return corpusState{}, err
	}
	if state.SSP, err = readU32(r); err != nil {
		return corpusState{}, err
	}
	sr, err := readU32(r)
	if err != nil {
		return corpusState{}, err
	}
	state.SR = uint16(sr)
	if state.PC, err = readU32(r); err != nil {
		return corpusState{}, err
	}
	for i := range state.Prefetch {
		v, err := readU32(r)
		if err != nil {
			return corpusState{}, err
		}
		state.Prefetch[i] = uint16(v)
	}
	ramCount, err := readU32(r)
	if err != nil || ramCount > 1<<20 {
		return corpusState{}, fmt.Errorf("invalid RAM count %d: %v", ramCount, err)
	}
	ram := make(SparseMemory, ramCount*2)
	for i := uint32(0); i < ramCount; i++ {
		address, err := readU32(r)
		if err != nil {
			return corpusState{}, err
		}
		var data uint16
		if err := binary.Read(r, binary.LittleEndian, &data); err != nil {
			return corpusState{}, err
		}
		ram[address] = byte(data >> 8)
		ram[address|1] = byte(data)
	}
	return corpusState{CPU: state, RAM: ram}, nil
}

func readTransactions(r io.Reader) ([]Transaction, uint32, error) {
	if err := readBlockHeader(r, 0x456789ab); err != nil {
		return nil, 0, err
	}
	clocks, err := readU32(r)
	if err != nil {
		return nil, 0, err
	}
	count, err := readU32(r)
	if err != nil || count > 1<<20 {
		return nil, 0, fmt.Errorf("invalid transaction count %d: %v", count, err)
	}
	out := make([]Transaction, 0, count)
	for i := uint32(0); i < count; i++ {
		var kind uint8
		if err := binary.Read(r, binary.LittleEndian, &kind); err != nil {
			return nil, 0, err
		}
		cycle, err := readU32(r)
		if err != nil {
			return nil, 0, err
		}
		if kind == 0 {
			continue
		}
		fc, err := readU32(r)
		if err != nil {
			return nil, 0, err
		}
		address, err := readU32(r)
		if err != nil {
			return nil, 0, err
		}
		data, err := readU32(r)
		if err != nil {
			return nil, 0, err
		}
		uds, err := readU32(r)
		if err != nil {
			return nil, 0, err
		}
		lds, err := readU32(r)
		if err != nil {
			return nil, 0, err
		}
		kinds := map[uint8]string{1: "w", 2: "r", 3: "t", 4: "re", 5: "we"}
		label, ok := kinds[kind]
		if !ok {
			return nil, 0, fmt.Errorf("unknown transaction kind %d", kind)
		}
		size := uint8(1)
		if uds+lds == 2 {
			size = 2
		}
		busData := uint16(data)
		if label == "re" || label == "we" {
			busData = 0
		}
		out = append(out, Transaction{Kind: label, Cycle: cycle, FC: uint8(fc), Address: address,
			Size: size, Data: busData, UDS: uds != 0, LDS: lds != 0})
	}
	return out, clocks, nil
}

func readBlockHeader(r io.Reader, want uint32) error {
	if _, err := readU32(r); err != nil {
		return err
	}
	magic, err := readU32(r)
	if err != nil {
		return err
	}
	if magic != want {
		return fmt.Errorf("block magic 0x%08x, want 0x%08x", magic, want)
	}
	return nil
}

func readU32(r io.Reader) (uint32, error) {
	var value uint32
	err := binary.Read(r, binary.LittleEndian, &value)
	return value, err
}
