package m68k

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// UCSD p-System IV.2.1 直譯器（SunDog: Frozen Legacy 的 SYSTEM.INTERP）的驗收。
//
// 這份語料與 TALOS_M68000_TESTS 的合成語料互補：合成語料逐條窮舉指令的狀態空間，
// 這裡跑的是 1985 年真實出貨的 68000 程式碼——同一段位元組在 Atari ST 上跑了四十年。
// 它不含 trap、line-A 與硬體位址（SunDog 專案全檔掃描的結果），所以只需要 CPU 與
// 記憶體就能完整執行，不依賴 Shifter、FDC 或任何 I/O。
//
// 素材是原版磁碟的檔案，不進 repository：設 TALOS_UCSD_INTERP 指向存放目錄。
// 檔案雜湊在測試裡釘死，換一份就失敗，不會拿別的檔案冒充通過。
const (
	interpSHA256 = "a344edfb07d27cafa3dfda68f1854a76f63a0e89cf2e8229dacf5aa64d603c38"
	interpSize   = 11776

	// 直譯器內的偏移。語意由 SunDog 專案人工反組譯 dispatch 表與各常式得出。
	interpDispatch = 0x00ec // 256 項 big-endian word 的 opcode 分派表
	interpSLDC     = 0x00d8 // 短常數 0–31（opcode 0x00–0x1f）
	interpLDL      = 0x0534 // 載入區域變數 1–16（opcode 0x20–0x2f）
	interpIXA      = 0x0952 // 陣列索引（p-code 的 ixa）

	// 測試選的記憶體佈局。直譯器實際載入位址由 p-system 決定，這裡挑一個非零基底，
	// 若某個切片其實依賴絕對位址或 PC-relative，測試會失敗而不是碰巧通過。
	interpBase    = 0x020000
	stackTop      = 0x030100
	operandStream = 0x031000
	sentinel      = 0x032000
)

func loadInterp(t *testing.T) []byte {
	t.Helper()
	root := os.Getenv("TALOS_UCSD_INTERP")
	if root == "" {
		t.Skip("TALOS_UCSD_INTERP is not set; UCSD p-System interpreter not available")
	}
	image, err := os.ReadFile(filepath.Join(root, "SYSTEM.INTERP"))
	if err != nil {
		t.Fatal(err)
	}
	if len(image) != interpSize {
		t.Fatalf("SYSTEM.INTERP is %d bytes, want %d", len(image), interpSize)
	}
	sum := sha256.Sum256(image)
	if got := hex.EncodeToString(sum[:]); got != interpSHA256 {
		t.Fatalf("SYSTEM.INTERP sha256 is %s, want %s", got, interpSHA256)
	}
	return image
}

// harness 把直譯器映像放進稀疏記憶體，並從任意常式起點單步執行到 jmp (a5)。
type harness struct {
	t   *testing.T
	cpu CPU
	mem SparseMemory
}

func newHarness(t *testing.T, image []byte) *harness {
	t.Helper()
	mem := SparseMemory{}
	for i, b := range image {
		mem[uint32(interpBase+i)] = b
	}
	// 哨兵：常式結尾的 jmp (a5) 跳到這裡，兩個 word 讓它有東西可以預取。
	for i := uint32(0); i < 4; i++ {
		mem[sentinel+i] = 0x4e // 0x4e71 = NOP
		if i%2 == 1 {
			mem[sentinel+i] = 0x71
		}
	}
	h := &harness{t: t, mem: mem}
	h.cpu.Bus = mem
	h.cpu.State.SR = 0x2700 // supervisor，關中斷；A7 用 SSP
	h.cpu.State.SSP = stackTop
	h.cpu.State.A[5] = sentinel // a5 是 dispatch 迴圈的回返點
	return h
}

// poke 寫入一個 big-endian word（68000 與 p-system 的資料都是 big-endian）。
func (h *harness) poke(address uint32, value uint16) {
	h.mem[address] = byte(value >> 8)
	h.mem[address+1] = byte(value)
}

func (h *harness) peek(address uint32) uint16 {
	return uint16(h.mem[address])<<8 | uint16(h.mem[address+1])
}

// push 把一個 word 推上 68000 堆疊，模擬直譯器進常式前的運算元堆疊。
func (h *harness) push(value uint16) {
	h.cpu.State.SSP -= 2
	h.poke(h.cpu.State.SSP, value)
}

// start 設定 PC 與預取。Atari Talos 的 PC 契約是「下一個要預取的位址」，
// 所以從位址 addr 執行等於 PC=addr+4、預取為 addr 起的兩個 word。
func (h *harness) start(offset uint32) {
	address := uint32(interpBase) + offset
	h.cpu.State.PC = address + 4
	h.cpu.State.Prefetch = [2]uint16{h.peek(address), h.peek(address + 2)}
}

// run 單步到常式跳回 a5 為止，回傳執行的指令數與總 clocks。
func (h *harness) run(maxSteps int) (steps int, clocks uint32) {
	h.t.Helper()
	for steps = 0; steps < maxSteps; steps++ {
		if h.cpu.State.PC == sentinel+4 {
			return steps, clocks
		}
		result, err := h.cpu.Step()
		if err != nil {
			h.t.Fatalf("step %d at PC=0x%06x: %v", steps, h.cpu.State.PC-4, err)
		}
		clocks += result.Clocks
	}
	h.t.Fatalf("routine did not reach the dispatch loop within %d steps", maxSteps)
	return 0, 0
}

// TestUCSDInterpDispatchTableShortConstants 驗證分派表的結構本身。
//
// 這一條不執行任何指令：它證明「opcode 0x00–0x1f 全部走同一支短常數常式」這個
// 解讀直接寫在表裡，而不是從常式行為反推出來的。
func TestUCSDInterpDispatchTableShortConstants(t *testing.T) {
	image := loadInterp(t)
	entry := func(opcode int) uint16 {
		at := interpDispatch + opcode*2
		return uint16(image[at])<<8 | uint16(image[at+1])
	}
	for opcode := 0x00; opcode <= 0x1f; opcode++ {
		if got := entry(opcode); got != interpSLDC {
			t.Fatalf("dispatch[0x%02x] = 0x%04x, want 0x%04x", opcode, got, interpSLDC)
		}
	}
	// 表外的第一個 opcode 走別支，否則「前 32 項共用」就不是一個有內容的陳述。
	if entry(0x20) == interpSLDC {
		t.Fatal("dispatch[0x20] also points at the short-constant routine")
	}
	if got := entry(0x20); got != interpLDL {
		t.Fatalf("dispatch[0x20] = 0x%04x, want 0x%04x", got, interpLDL)
	}
}

// TestUCSDInterpShortConstantPushesOpcodeHalf 執行短常數常式。
//
// 直譯器進常式時 d0 是 opcode×2（分派表以 word 為單位索引），常式把它右移一位還原成
// 常數本身再推上運算元堆疊。這是 p-code 的 sldc0–sldc31。
func TestUCSDInterpShortConstantPushesOpcodeHalf(t *testing.T) {
	image := loadInterp(t)
	for opcode := 0; opcode <= 0x1f; opcode++ {
		h := newHarness(t, image)
		h.cpu.State.D[0] = uint32(opcode * 2)
		h.start(interpSLDC)
		steps, clocks := h.run(8)
		const wantSteps = 3 // lsr.w #1,d0 / move.w d0,-(sp) / jmp (a5)
		if steps != wantSteps {
			t.Fatalf("sldc%d took %d instructions, want %d", opcode, steps, wantSteps)
		}
		if got := h.cpu.State.SSP; got != stackTop-2 {
			t.Fatalf("sldc%d left SSP at 0x%06x, want 0x%06x", opcode, got, stackTop-2)
		}
		if got := h.peek(h.cpu.State.SSP); got != uint16(opcode) {
			t.Fatalf("sldc%d pushed %d, want %d", opcode, got, opcode)
		}
		if clocks == 0 {
			t.Fatalf("sldc%d reported no clocks", opcode)
		}
	}
}

// TestUCSDInterpIndexArrayScalesByWords 是這份語料的核心。
//
// p-code 的 `ixa n` 把堆疊上的基底位址前進「索引 × n 個 word」。SunDog 專案用它推出
// 城市地面圖是每列 40 bytes（`ixa 0x14`＝20 words×2），整張圖 40×24——那個結論原本
// 只有閱讀反組譯這一層證據。這裡直接執行原版的常式，證明換算就是 base + index×n×2，
// 且 **base 是 byte 位址**（常式把位移直接加到堆疊上的基底，沒有再乘）。
func TestUCSDInterpIndexArrayScalesByWords(t *testing.T) {
	image := loadInterp(t)
	cases := []struct {
		name          string
		elementWords  uint16
		index         uint16
		base          uint16
		wantByteDelta uint16
	}{
		{"地面圖每列 20 words＝40 bytes", 0x14, 0, 0x1000, 0},
		{"地面圖第 1 列", 0x14, 1, 0x1000, 40},
		{"地面圖第 7 列", 0x14, 7, 0x1000, 280},
		{"地面圖第 23 列（最後一列）", 0x14, 23, 0x1000, 920},
		{"元素 1 word 時只乘 2", 0x01, 5, 0x2000, 10},
		{"元素 6 words（世界資料的 record）", 0x06, 9, 0x0800, 108},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, image)
			// 運算元流：ixa 的元素大小以變長編碼跟在 opcode 之後，a4 是 p-code 的 IP。
			h.mem[operandStream] = byte(tc.elementWords)
			h.cpu.State.A[4] = operandStream
			h.push(tc.base)  // 基底位址（byte 位址）
			h.push(tc.index) // 索引，常式先 pop 它
			h.start(interpIXA)
			h.run(16)

			if got := h.cpu.State.SSP; got != stackTop-2 {
				t.Fatalf("SSP is 0x%06x, want 0x%06x（索引該被 pop 掉，基底留著）", got, stackTop-2)
			}
			want := tc.base + tc.wantByteDelta
			if got := h.peek(h.cpu.State.SSP); got != want {
				t.Fatalf("ixa 0x%02x with index %d gave 0x%04x, want 0x%04x",
					tc.elementWords, tc.index, got, want)
			}
			if got := h.cpu.State.A[4]; got != operandStream+1 {
				t.Fatalf("a4 is 0x%06x, want 0x%06x（單位元組運算元該只吃一個位元組）",
					got, operandStream+1)
			}
		})
	}
}

// TestUCSDInterpIndexArrayReadsTwoByteOperand 驗證變長運算元的第二種長度。
//
// UCSD 的「big」編碼：第一個位元組小於 0x80 就是值本身，否則去掉最高位再吃一個位元組
// 接在低位。這條規則同時決定了反組譯的正確性——吃錯位元組數，後面整段都會偏掉。
func TestUCSDInterpIndexArrayReadsTwoByteOperand(t *testing.T) {
	image := loadInterp(t)
	h := newHarness(t, image)
	// 0x81 0x00 = ((0x81 & 0x7f) << 8) | 0x00 = 0x100 個 word。
	h.mem[operandStream] = 0x81
	h.mem[operandStream+1] = 0x00
	h.cpu.State.A[4] = operandStream
	h.push(0x1000)
	h.push(2)
	h.start(interpIXA)
	h.run(16)

	if got := h.cpu.State.A[4]; got != operandStream+2 {
		t.Fatalf("a4 is 0x%06x, want 0x%06x（兩位元組運算元）", got, operandStream+2)
	}
	const want = 0x1000 + 2*0x100*2
	if got := h.peek(h.cpu.State.SSP); got != want {
		t.Fatalf("ixa 0x100 with index 2 gave 0x%04x, want 0x%04x", got, want)
	}
}

// TestUCSDInterpLoadLocalUsesVarOffsetEight 執行載入區域變數的常式。
//
// 常式是 `subi.w #$3E,d0` 後 `move.w 8(a0,d0.l),-(sp)`：d0 進來是 opcode×2，減掉 0x3E
// 之後就是「區域變數編號×2」，再加上活動記錄標頭的 8 個位元組。這支就是 p-system
// 活動記錄裡「編號 × 2 + 8」那條換算的出處，讀 SunDog 執行時的區域變數靠的是同一條。
func TestUCSDInterpLoadLocalUsesVarOffsetEight(t *testing.T) {
	image := loadInterp(t)
	const frame = 0x028000
	for local := 1; local <= 16; local++ {
		opcode := 0x20 + local - 1 // opcode 0x20–0x2f 對到區域變數 1–16
		h := newHarness(t, image)
		// 整個 frame 先填可辨識的雜訊，確保常式取的是算出來的那一格而不是碰巧命中。
		for slot := 0; slot <= 17; slot++ {
			h.poke(frame+8+uint32(slot)*2, uint16(0xd000+slot))
		}
		want := uint16(0xbe00 + local)
		h.poke(frame+8+uint32(local)*2, want)

		h.cpu.State.A[0] = frame
		h.cpu.State.D[0] = uint32(opcode * 2)
		h.start(interpLDL)
		steps, _ := h.run(8)

		const wantSteps = 3 // subi.w #$3e,d0 / move.w 8(a0,d0.l),-(sp) / jmp (a5)
		if steps != wantSteps {
			t.Fatalf("local %d took %d instructions, want %d", local, steps, wantSteps)
		}
		if got := h.peek(h.cpu.State.SSP); got != want {
			t.Fatalf("local %d pushed 0x%04x, want 0x%04x（該讀 frame+8+%d×2）",
				local, got, want, local)
		}
	}
}
