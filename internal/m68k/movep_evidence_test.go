package m68k

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func TestMOVEPEvidence(t *testing.T) {
	root := os.Getenv("TALOS_M68000_TESTS")
	if root == "" {
		t.Skip("no corpus")
	}
	for _, name := range []string{"MOVEP.w.json.bin", "MOVEP.l.json.bin"} {
		p := filepath.Join(root, name)
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		t.Logf("%s sha256=%x", name, sha256.Sum256(b))
		tests, e := readCorpus(p)
		if e != nil {
			t.Fatal(e)
		}
		seen := map[uint16]bool{}
		for _, v := range tests {
			mode := v.Initial.CPU.Prefetch[0] & 0x00c0
			if seen[mode] {
				continue
			}
			seen[mode] = true
			t.Logf("name=%s opcode=%04x initial=%+v final=%+v clocks=%d tx=%+v", v.Name, v.Initial.CPU.Prefetch[0], v.Initial.CPU, v.Final.CPU, v.Clocks, v.Transactions)
		}
	}
}
