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

func testSingleStepCorpus(t *testing.T, name string) {
	t.Helper()
	root := os.Getenv("TALOS_M68000_TESTS")
	if root == "" {
		t.Skip("TALOS_M68000_TESTS is not set; external m68000 corpus not available")
	}
	tests, err := readCorpus(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	if len(tests) != 2500 {
		t.Fatalf("%s corpus has %d tests, want 2500", name, len(tests))
	}
	for _, test := range tests {
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
			if !reflect.DeepEqual(test.Initial.RAM, test.Final.RAM) {
				t.Fatal("NOP corpus unexpectedly changes RAM")
			}
			if result.Clocks != test.Clocks || !reflect.DeepEqual(result.Transactions, test.Transactions) {
				t.Fatalf("bus mismatch\n got: %#v\nwant: clocks=%d transactions=%#v", result, test.Clocks, test.Transactions)
			}
		})
	}
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
		out = append(out, Transaction{Kind: label, Cycle: cycle, FC: uint8(fc), Address: address,
			Size: size, Data: uint16(data), UDS: uds != 0, LDS: lds != 0})
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
