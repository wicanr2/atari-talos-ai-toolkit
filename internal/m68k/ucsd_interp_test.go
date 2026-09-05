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
// 被驗收的這幾支常式在執行時只碰下面明確映射的記憶體，所以只需要 CPU 與記憶體，
// 不依賴 Shifter、FDC 或任何 I/O。這件事由 SparseMemory 保證而不是靠掃描位元組：
// 它對未映射位址回傳錯誤，任何非預期的取值、trap 向量讀取或 I/O 存取都會讓測試失敗。
//
// 整份 SYSTEM.INTERP 並非完全不含 trap——偏移 $008A 有一條由旗標保護的 TRAP #15
// 除錯 hook，不在這裡執行的路徑上。全檔沒有 TRAP #1／#13／#14，所以直譯器不呼叫
// GEMDOS、BIOS 或 XBIOS。
//
// 素材是原版磁碟的檔案，不進 repository：設 TALOS_UCSD_INTERP 指向存放目錄。
// 檔案雜湊在測試裡釘死，換一份就失敗，不會拿別的檔案冒充通過。
//
// 序列測試的期望值另外由 laanwj/sundog 的 psys_interpreter.c（同一套 p-machine 的獨立
// C 重寫）跑同一段 p-code 確認過，六組逐字相同——見 docs/spec/055。那份程式碼的角色與
// Hatari 相同：外部 oracle，本 repository 不連結、不移植也不依賴它。
const (
	interpSHA256 = "a344edfb07d27cafa3dfda68f1854a76f63a0e89cf2e8229dacf5aa64d603c38"
	interpSize   = 11776

	// 直譯器內的偏移。語意由 SunDog 專案人工反組譯 dispatch 表與各常式得出。
	interpDispatch = 0x00ec // 256 項 big-endian word 的 opcode 分派表
	interpSLDC     = 0x00d8 // 短常數 0–31（opcode 0x00–0x1f）
	interpLDL      = 0x0534 // 載入區域變數 1–16（opcode 0x20–0x2f）
	interpIXA      = 0x0952 // 陣列索引（p-code 的 ixa）
	interpLoop     = 0x00de // 分派迴圈：取 opcode、查表、跳進常式
	interpSLLA     = 0x0554 // 取區域變數位址 1–8（opcode $60–$67）
	interpSSTL     = 0x057c // 存入區域變數 1–8（opcode $68–$6f）
	interpInvalid  = 0x0304 // 無效 opcode 的錯誤路徑（錯誤碼 11）
	interpUJP      = 0x0d0e // 無條件跳躍（opcode $8A）
	interpFJP      = 0x0d22 // 假時跳躍（opcode $D4）
	interpTJP      = 0x0d18 // 真時跳躍（opcode $F1）
	interpLDB      = 0x08a8 // 取 byte（opcode $A7）
	interpLAND     = 0x0a02 // 布林 and（opcode $A1）
	interpLOR      = 0x0a08 // 布林 or（opcode $A0）
	interpBNOT     = 0x0a12 // 布林 not（opcode $9F）

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
	h.cpu.State.A[3] = interpBase // a3 是直譯器載入基底；分派表存的是相對它的偏移
	h.cpu.State.A[5] = sentinel   // a5 是常式的回返點
	return h
}

// startDispatch 讓直譯器從分派迴圈開始跑，而不是直接進單一常式。
// a5 改指向迴圈本身，常式結尾的 jmp (a5) 因此構成完整的 fetch-execute 循環。
func (h *harness) startDispatch(pcode []byte, at uint32) {
	for i, b := range pcode {
		h.mem[at+uint32(i)] = b
	}
	h.cpu.State.A[4] = at // a4 是 p-code 的 IP
	h.cpu.State.A[5] = interpBase + interpLoop
	h.start(interpLoop)
}

// runPCode 執行指定數量的 p-code 指令，回傳消耗的 68000 指令數。
// 一條 p-code 指令的邊界是「控制權回到分派迴圈起點」。
func (h *harness) runPCode(ops int, maxSteps int) int {
	h.t.Helper()
	loopPC := uint32(interpBase + interpLoop + 4)
	completed, steps := 0, 0
	for completed < ops {
		if steps >= maxSteps {
			h.t.Fatalf("%d 條 p-code 指令沒有在 %d 步內完成（已完成 %d）", ops, maxSteps, completed)
		}
		if _, err := h.cpu.Step(); err != nil {
			h.t.Fatalf("step %d at PC=0x%06x: %v", steps, h.cpu.State.PC-4, err)
		}
		steps++
		if h.cpu.State.PC == loopPC {
			completed++
		}
	}
	return steps
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

// TestUCSDInterpDispatchLoopRunsPCodeSequence 讓分派迴圈自己跑一串 p-code。
//
// 前面幾個測試各自跳進單一常式；這一個從分派迴圈起步，讓直譯器自己取 opcode、查表、
// 進常式、回迴圈。它證明的東西不同：不是「某支常式做對事」，而是**取指令與分派這個
// 循環本身在 Atari Talos 上成立**，因此可以執行任意 p-code 序列。
//
// 迴圈是 moveq #0,d0 / move.b (a4)+,d0 / add.w d0,d0 / move.w 6(pc,d0.w),d7 /
// jmp 0(a3,d7.w)。`6(pc,d0.w)` 的 PC 是延伸字位址 $00E6，加 6 正好是分派表 $00EC；
// `0(a3,d7.w)` 說明表項是相對載入基底的偏移，所以直譯器不必載在位址 0。
func TestUCSDInterpDispatchLoopRunsPCodeSequence(t *testing.T) {
	image := loadInterp(t)
	h := newHarness(t, image)
	// sldc5、sldc3、sldc31、sldc0：四條短常數，堆疊由下往上是 5、3、31、0。
	h.startDispatch([]byte{0x05, 0x03, 0x1f, 0x00}, operandStream)
	steps := h.runPCode(4, 64)

	const wantPerOp = 8 // 迴圈 5 條 ＋ 常式 3 條
	if steps != 4*wantPerOp {
		t.Fatalf("四條 p-code 花了 %d 條 68000 指令，want %d", steps, 4*wantPerOp)
	}
	if got := h.cpu.State.A[4]; got != operandStream+4 {
		t.Fatalf("a4 停在 0x%06x，want 0x%06x（IP 該前進四個位元組）", got, operandStream+4)
	}
	if got := h.cpu.State.SSP; got != stackTop-8 {
		t.Fatalf("SSP 是 0x%06x，want 0x%06x（四個 word）", got, stackTop-8)
	}
	for i, want := range []uint16{0, 31, 3, 5} { // 由堆疊頂往下
		if got := h.peek(h.cpu.State.SSP + uint32(i)*2); got != want {
			t.Fatalf("堆疊第 %d 個 word 是 %d，want %d", i, got, want)
		}
	}
}

// TestUCSDInterpDispatchLoopMixesOpcodeFamilies 在同一串裡混用不同常式。
//
// 只跑同一種 opcode 證明不了分派表真的被查了——四條 sldc 走同一個表項。這一條混入
// 區域變數載入與陣列索引，讓迴圈必須跳到三支不同的常式，且後面的指令要吃前面留在
// 堆疊上的結果。
func TestUCSDInterpDispatchLoopMixesOpcodeFamilies(t *testing.T) {
	image := loadInterp(t)
	h := newHarness(t, image)
	const frame = 0x028000
	h.cpu.State.A[0] = frame
	h.poke(frame+8+3*2, 0x1000) // 區域變數 3 ＝ 地面圖的基底位址

	// ldl3（opcode $22）取基底、sldc7 推列號、ixa $14 算出「基底 + 7 列 × 40 bytes」。
	// 這正是 SunDog 讀城市地面圖某一列的完整三步。
	h.startDispatch([]byte{0x22, 0x07, 0xd7, 0x14}, operandStream)
	h.runPCode(3, 64)

	if got := h.cpu.State.SSP; got != stackTop-2 {
		t.Fatalf("SSP 是 0x%06x，want 0x%06x（三條指令後只剩一個結果）", got, stackTop-2)
	}
	const want = 0x1000 + 7*20*2
	if got := h.peek(h.cpu.State.SSP); got != want {
		t.Fatalf("地面圖第 7 列的位址是 0x%04x，want 0x%04x", got, want)
	}
	if got := h.cpu.State.A[4]; got != operandStream+4 {
		t.Fatalf("a4 停在 0x%06x，want 0x%06x（ixa 的運算元佔一個位元組）", got, operandStream+4)
	}
}

// TestUCSDInterpDispatchTableShape 驗證整張分派表的形狀。
//
// 254 個 opcode 分派到哪裡，是這份直譯器能力範圍的完整地圖。這一條把它釘住：
// 常式支數、無效 opcode 的集合、以及「opcode $9C 指回迴圈本身」——那是一條 NOP。
// 表是資料，這一條不執行任何指令。
func TestUCSDInterpDispatchTableShape(t *testing.T) {
	image := loadInterp(t)
	entry := func(opcode int) uint16 {
		at := interpDispatch + opcode*2
		return uint16(image[at])<<8 | uint16(image[at+1])
	}
	targets := map[uint16]int{}
	for opcode := 0; opcode < 256; opcode++ {
		targets[entry(opcode)]++
	}
	const wantRoutines, wantInvalid = 107, 45
	if len(targets) != wantRoutines {
		t.Fatalf("分派表指向 %d 支常式，want %d", len(targets), wantRoutines)
	}
	// 無效 opcode 全部收斂到同一支錯誤常式。
	if got := targets[interpInvalid]; got != wantInvalid {
		t.Fatalf("%d 個 opcode 指向錯誤常式，want %d", got, wantInvalid)
	}
	for _, opcode := range []int{0x40, 0x5f, 0xaa, 0xaf, 0xf5, 0xff} {
		if got := entry(opcode); got != interpInvalid {
			t.Fatalf("opcode 0x%02x 指向 0x%04x，want 0x%04x（無效）", opcode, got, interpInvalid)
		}
	}
	// opcode $9C 的表項就是迴圈起點：進去以後直接取下一個 opcode，等於 NOP。
	if got := entry(0x9c); got != interpLoop {
		t.Fatalf("opcode 0x9c 指向 0x%04x，want 0x%04x（迴圈本身）", got, interpLoop)
	}
	// 錯誤常式的第一條是 moveq #11,d0——錯誤碼 11。
	if got := uint16(image[interpInvalid])<<8 | uint16(image[interpInvalid+1]); got != 0x700b {
		t.Fatalf("錯誤常式首指令是 0x%04x，want 0x700b（moveq #11,d0）", got)
	}
}

// TestUCSDInterpStoreLoadRoundTrip 用 p-code 序列做存取往返。
//
// `sldc21` / `sstl1` / `ldl1`：推一個值、存進活動記錄第 1 格、再讀回來。這條同時驗兩件
// 事——sstl 與 ldl 對同一格的位址算法一致，而且值真的落在記憶體 `frame+8+1×2` 上，
// 不是兩支常式互相自洽卻都算錯。
func TestUCSDInterpStoreLoadRoundTrip(t *testing.T) {
	image := loadInterp(t)
	h := newHarness(t, image)
	const frame = 0x028000
	h.cpu.State.A[0] = frame
	h.poke(frame+8+1*2, 0xffff) // 先放不同的值，確保結果來自 sstl 而不是殘留

	h.startDispatch([]byte{0x15, 0x68, 0x20}, operandStream) // sldc21 / sstl1 / ldl1
	h.runPCode(3, 64)

	const want = 21
	if got := h.peek(frame + 8 + 1*2); got != want {
		t.Fatalf("記憶體 frame+8+2 是 %d，want %d（sstl 沒寫對地方）", got, want)
	}
	if got := h.cpu.State.SSP; got != stackTop-2 {
		t.Fatalf("SSP 是 0x%06x，want 0x%06x", got, stackTop-2)
	}
	if got := h.peek(h.cpu.State.SSP); got != want {
		t.Fatalf("ldl1 讀回 %d，want %d", got, want)
	}
}

// TestUCSDInterpLoadLocalAddressIsPSystemRelative 驗證取區域變數位址的常式。
//
// 常式是 `subi.w #$BE,d0` / `add.l a0,d0` / `sub.l a6,d0` / `addq.w #8,d0`：先算出
// 活動記錄裡那一格的主機位址，再減掉 a6。**a6 是 p-system 記憶體基底**，所以推上堆疊的
// 是 p-system 位址而不是主機位址——這條決定了 p-code 看到的位址空間長什麼樣。
func TestUCSDInterpLoadLocalAddressIsPSystemRelative(t *testing.T) {
	image := loadInterp(t)
	const memoryBase = 0x040000
	const framePSys = 0x8000 // 活動記錄在 p-system 位址空間的位置
	for local := 1; local <= 8; local++ {
		h := newHarness(t, image)
		h.cpu.State.A[6] = memoryBase
		h.cpu.State.A[0] = memoryBase + framePSys
		h.startDispatch([]byte{byte(0x60 + local - 1)}, operandStream)
		h.runPCode(1, 32)

		want := uint16(framePSys + 8 + local*2)
		if got := h.peek(h.cpu.State.SSP); got != want {
			t.Fatalf("slla%d 推了 0x%04x，want 0x%04x", local, got, want)
		}
	}
}

// TestUCSDInterpNoOpOpcodeAdvancesOnly 驗證 opcode $9C 是 NOP。
//
// 它的表項指回迴圈起點，所以進去以後直接取下一個 opcode：IP 前進，其他什麼都不做。
func TestUCSDInterpNoOpOpcodeAdvancesOnly(t *testing.T) {
	image := loadInterp(t)
	h := newHarness(t, image)
	h.startDispatch([]byte{0x9c, 0x9c, 0x05}, operandStream) // nop / nop / sldc5
	h.runPCode(3, 64)

	if got := h.cpu.State.A[4]; got != operandStream+3 {
		t.Fatalf("a4 停在 0x%06x，want 0x%06x", got, operandStream+3)
	}
	if got := h.cpu.State.SSP; got != stackTop-2 {
		t.Fatalf("SSP 是 0x%06x，want 0x%06x（只有 sldc5 推了東西）", got, stackTop-2)
	}
	if got := h.peek(h.cpu.State.SSP); got != 5 {
		t.Fatalf("堆疊頂是 %d，want 5", got)
	}
}

// TestUCSDInterpArithmeticAndComparison 驗收算術、比較與堆疊操作族。
//
// 這些 opcode 讓 p-code 序列能表達實際的運算式，而不只是搬移資料。每一列的期望值都由
// laanwj/sundog 的獨立 C 直譯器跑同一段 p-code 確認過。
func TestUCSDInterpArithmeticAndComparison(t *testing.T) {
	image := loadInterp(t)
	cases := []struct {
		name   string
		pcode  []byte
		ops    int
		stack  []uint16 // 由堆疊頂往下
		ipcAdd uint32
	}{
		{"ldcb 40", []byte{0x80, 0x28}, 1, []uint16{40}, 2},
		{"ldci 480", []byte{0x81, 0xe0, 0x01}, 1, []uint16{480}, 3},
		{"5 + 3", []byte{0x05, 0x03, 0xa2}, 3, []uint16{8}, 3},
		{"5 − 3（tos1 減 tos0）", []byte{0x05, 0x03, 0xa3}, 3, []uint16{2}, 3},
		{"31 ÷ 7", []byte{0x1f, 0x07, 0x8d}, 3, []uint16{4}, 3},
		{"31 mod 7", []byte{0x1f, 0x07, 0x8f}, 3, []uint16{3}, 3},
		{"5 = 5", []byte{0x05, 0x05, 0xb0}, 3, []uint16{1}, 3},
		{"3 ≤ 5", []byte{0x03, 0x05, 0xb2}, 3, []uint16{1}, 3},
		{"5 ≤ 3", []byte{0x05, 0x03, 0xb2}, 3, []uint16{0}, 3},
		{"dup1", []byte{0x07, 0xe2}, 2, []uint16{7, 7}, 2},
		{"swap", []byte{0x03, 0x05, 0xbd}, 3, []uint16{3, 5}, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, image)
			h.startDispatch(tc.pcode, operandStream)
			h.runPCode(tc.ops, 128)

			wantSP := uint32(stackTop - 2*len(tc.stack))
			if got := h.cpu.State.SSP; got != wantSP {
				t.Fatalf("SSP 是 0x%06x，want 0x%06x", got, wantSP)
			}
			for i, want := range tc.stack {
				if got := h.peek(h.cpu.State.SSP + uint32(i)*2); got != want {
					t.Fatalf("堆疊第 %d 個 word 是 %d，want %d", i, got, want)
				}
			}
			if got := h.cpu.State.A[4]; got != operandStream+tc.ipcAdd {
				t.Fatalf("a4 前進 %d，want %d", got-operandStream, tc.ipcAdd)
			}
		})
	}
}

// TestUCSDInterpGroundGridArithmetic 用 p-code 重現原版的地面圖座標換算。
//
// SunDog 的 check_exit 把走路人的 sprite 座標換算成 40×24 的格座標，再用它索引地面圖：
//
//	欄 = (捲動x + sprite.x / 步距) mod 40
//	列 = (捲動y + sprite.y / 步距) mod 24
//	那一格 = 地面圖基底 + 列 × 20 words × 2
//
// 這裡的數值不是編出來的：捲動 (7,4)、sprite (175,120)、步距 40 都是在原版執行時用除錯器
// 讀出來的，而換算結果（欄 11、列 7）與同一刻讀到的格座標一致。所以這個測試是拿原版的
// 直譯器、跑原版的算式、餵原版的數值。
func TestUCSDInterpGroundGridArithmetic(t *testing.T) {
	image := loadInterp(t)
	cases := []struct {
		name  string
		pcode []byte
		ops   int
		want  uint16
	}{
		{
			"欄 = (7 + 175/40) mod 40",
			// sldc7 / ldcb 175 / ldcb 40 / dvi / adi / ldcb 40 / modi
			[]byte{0x07, 0x80, 0xaf, 0x80, 0x28, 0x8d, 0xa2, 0x80, 0x28, 0x8f}, 7, 11,
		},
		{
			"列 = (4 + 120/40) mod 24",
			// sldc4 / ldcb 120 / ldcb 40 / dvi / adi / sldc24 / modi
			[]byte{0x04, 0x80, 0x78, 0x80, 0x28, 0x8d, 0xa2, 0x18, 0x8f}, 7, 7,
		},
		{
			"地面圖第 7 列的位址",
			// ldci $1000 / 列的算式 / ixa 20
			[]byte{0x81, 0x00, 0x10, 0x04, 0x80, 0x78, 0x80, 0x28, 0x8d, 0xa2, 0x18, 0x8f, 0xd7, 0x14},
			9, 0x1118,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, image)
			h.startDispatch(tc.pcode, operandStream)
			h.runPCode(tc.ops, 256)

			if got := h.cpu.State.SSP; got != stackTop-2 {
				t.Fatalf("SSP 是 0x%06x，want 0x%06x（該只剩一個結果）", got, stackTop-2)
			}
			if got := h.peek(h.cpu.State.SSP); got != tc.want {
				t.Fatalf("結果是 %d (0x%04x)，want %d (0x%04x)", got, got, tc.want, tc.want)
			}
			if got := h.cpu.State.A[4]; got != operandStream+uint32(len(tc.pcode)) {
				t.Fatalf("a4 前進 %d，want %d", got-operandStream, len(tc.pcode))
			}
		})
	}
}

// TestUCSDInterpBranches 驗收跳躍族。有了分支，p-code 才能表達條件邏輯。
//
// 位移是相對「位移位元組之後」，不是相對 opcode。`fjp` 在堆疊頂為 0 時跳，`tjp` 在非 0
// 時跳，兩者都消耗那個值。
func TestUCSDInterpBranches(t *testing.T) {
	image := loadInterp(t)
	cases := []struct {
		name  string
		pcode []byte
		ops   int
		stack []uint16
	}{
		// sldc5 / ujp +1 / sldc3 / sldc7：跳過 sldc3。
		{"ujp 無條件跳過一個位元組", []byte{0x05, 0x8a, 0x01, 0x03, 0x07}, 3, []uint16{7, 5}},
		// sldc0 是假 → fjp 跳，sldc3 被跳過。
		{"fjp 假時跳", []byte{0x00, 0xd4, 0x01, 0x03, 0x07}, 3, []uint16{7}},
		// sldc1 是真 → fjp 不跳，sldc3 執行。
		{"fjp 真時不跳", []byte{0x01, 0xd4, 0x01, 0x03, 0x07}, 4, []uint16{7, 3}},
		// sldc1 是真 → tjp 跳。
		{"tjp 真時跳", []byte{0x01, 0xf1, 0x01, 0x03, 0x07}, 3, []uint16{7}},
		{"tjp 假時不跳", []byte{0x00, 0xf1, 0x01, 0x03, 0x07}, 4, []uint16{7, 3}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, image)
			h.startDispatch(tc.pcode, operandStream)
			h.runPCode(tc.ops, 128)
			wantSP := uint32(stackTop - 2*len(tc.stack))
			if got := h.cpu.State.SSP; got != wantSP {
				t.Fatalf("SSP 是 0x%06x，want 0x%06x", got, wantSP)
			}
			for i, want := range tc.stack {
				if got := h.peek(h.cpu.State.SSP + uint32(i)*2); got != want {
					t.Fatalf("堆疊第 %d 個 word 是 %d，want %d", i, got, want)
				}
			}
		})
	}
}

// TestUCSDInterpIndirectLoads 驗收間接載入：sind 讀 word，ldb 讀 byte。
func TestUCSDInterpIndirectLoads(t *testing.T) {
	image := loadInterp(t)
	// p-system 位址是相對 a6 的 16-bit 值；harness 的 a6 是 0，所以這裡就是絕對位址。
	const at = 0xa000
	cases := []struct {
		name  string
		pcode []byte
		ops   int
		want  uint16
	}{
		// ldci <at> / sind0：讀 [at]。
		{"sind0 讀該位址的 word", []byte{0x81, byte(at & 0xff), byte(at >> 8), 0x78}, 2, 0x1234},
		// ldci <at-2> / sind1：讀 [at-2 + 1 word] ＝ [at]。
		{"sind1 讀往後一個 word", []byte{0x81, byte((at - 2) & 0xff), byte((at - 2) >> 8), 0x79}, 2, 0x1234},
		// ldci <at> / sldc0 / ldb：讀 [at] 的高位元組（p-system 資料是 big-endian）。
		{"ldb 偏移 0 取高位元組", []byte{0x81, byte(at & 0xff), byte(at >> 8), 0x00, 0xa7}, 3, 0x12},
		{"ldb 偏移 1 取低位元組", []byte{0x81, byte(at & 0xff), byte(at >> 8), 0x01, 0xa7}, 3, 0x34},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, image)
			h.poke(at, 0x1234)
			h.startDispatch(tc.pcode, operandStream)
			h.runPCode(tc.ops, 64)
			if got := h.peek(h.cpu.State.SSP); got != tc.want {
				t.Fatalf("結果是 0x%04x，want 0x%04x", got, tc.want)
			}
		})
	}
}

// TestUCSDInterpGroundCodeNormalisation 用 p-code 重現原版讀地面碼並正規化的完整流程。
//
// check_exit 讀出腳下那一格之後做的第一件事是「大於 15 就減 16」——那 16 是 draw_city_map
// 繪圖時加上的視覺變體偏移。這一段是原版的條件邏輯，用 dup1／leqi／tjp 表達：
//
//	dup1 / sldc15 / leqi / tjp +2 / sldc16 / sbi
//
// 邊界值是這一條的重點：15 不減、16 要減。門表的上界 36 正規化之後正好是 20，與
// check_building 的跳表 max 相同。
func TestUCSDInterpGroundCodeNormalisation(t *testing.T) {
	image := loadInterp(t)
	// dup1 / sldc15 / leqi / tjp +2 / sldc16 / sbi
	tail := []byte{0xe2, 0x0f, 0xb2, 0xf1, 0x02, 0x10, 0xa3}
	cases := []struct {
		code, want uint16
		ops        int
		note       string
	}{
		{5, 5, 5, "碼 5：check_exit 特地為它開的那條 or"},
		{15, 15, 5, "邊界：15 不減"},
		{16, 0, 7, "邊界：16 要減"},
		{35, 19, 7, "Drahew 的倉庫槽位，正規化成門表的 19"},
		{36, 20, 7, "門表上界，等於 check_building 跳表的 max"},
	}
	for _, tc := range cases {
		t.Run(tc.note, func(t *testing.T) {
			h := newHarness(t, image)
			pcode := append([]byte{0x80, byte(tc.code)}, tail...) // ldcb <碼> 起頭
			h.startDispatch(pcode, operandStream)
			h.runPCode(tc.ops, 128)
			if got := h.cpu.State.SSP; got != stackTop-2 {
				t.Fatalf("SSP 是 0x%06x，want 0x%06x（該只剩正規化後的碼）", got, stackTop-2)
			}
			if got := h.peek(h.cpu.State.SSP); got != tc.want {
				t.Fatalf("碼 %d 正規化成 %d，want %d", tc.code, got, tc.want)
			}
		})
	}
}

// TestUCSDInterpDispatchTableBooleanRoutines 先確認這三個 opcode 指到哪裡。
//
// 這一條不執行任何指令，也不看常式做什麼：它把「$A1 走 $0A02、$A0 走 $0A08、
// $9F 走 $0A12」這個定位直接讀出表來。少了它，下面那些執行測試等於同時假設定位與
// 語意都對，兩邊一起錯就看不出來。
func TestUCSDInterpDispatchTableBooleanRoutines(t *testing.T) {
	image := loadInterp(t)
	entry := func(opcode int) uint16 {
		at := interpDispatch + opcode*2
		return uint16(image[at])<<8 | uint16(image[at+1])
	}
	for _, tc := range []struct {
		opcode int
		want   uint16
		name   string
	}{
		{0xa1, interpLAND, "land"},
		{0xa0, interpLOR, "lor"},
		{0x9f, interpBNOT, "bnot"},
		{0xd4, interpFJP, "fjp"},
		{0xf1, interpTJP, "tjp"},
	} {
		if got := entry(tc.opcode); got != tc.want {
			t.Fatalf("dispatch[0x%02x]（%s）= 0x%04x，want 0x%04x", tc.opcode, tc.name, got, tc.want)
		}
	}
	// 三支常式各自獨立，不是同一支的別名。
	if interpLAND == interpLOR || interpLOR == interpBNOT {
		t.Fatal("三個布林 opcode 指到同一支常式")
	}
}

// TestUCSDInterpBooleanOperatorsAreBitwise 驗收 land／lor／bnot 三支常式。
//
// 這三個 opcode 的名字看起來像邏輯運算，實際上不是：分派表指到的 68000 常式是
// `and.w` 與 `or.w`，也就是位元運算；bnot 是 `not.w` 之後 `andi.w #1`，只翻最低位元。
// 配上 fjp／tjp 的 `btst #0`（見 TestUCSDInterpBooleanTruthLivesInBitZero），
// p-system 的真假整套住在 **bit 0**。
//
// 為什麼值得單獨驗收：SunDog 的新遊戲初始損壞（XSTARTUP:0x31）用
// `(欄 = 0) or random()` 決定壞法，而 random() 的值域是 0–8191。把 lor 當成邏輯或、
// 把 fjp 當成「整個 word 為零才跳」來讀，那個條件會幾乎永遠成立，與實測的分布矛盾。
// 讀成位元運算加 bit 0 之後矛盾消失——這裡用真實直譯器把那個讀法跑出來。
func TestUCSDInterpBooleanOperatorsAreBitwise(t *testing.T) {
	image := loadInterp(t)
	cases := []struct {
		name  string
		pcode []byte
		ops   int
		stack []uint16
	}{
		// 1 and 1 = 1；0 and 1 = 0。當成邏輯運算時這兩個也對，所以還不夠。
		{"1 and 1", []byte{0x01, 0x01, 0xa1}, 3, []uint16{1}},
		{"0 and 1", []byte{0x00, 0x01, 0xa1}, 3, []uint16{0}},
		// 這一組把位元與邏輯分開：2 和 1 都非零，邏輯 and 會給 1，位元 and 給 0。
		{"2 and 1 是位元運算", []byte{0x02, 0x01, 0xa1}, 3, []uint16{0}},
		{"3 and 6 是位元運算", []byte{0x03, 0x06, 0xa1}, 3, []uint16{2}},
		{"1 or 0", []byte{0x01, 0x00, 0xa0}, 3, []uint16{1}},
		// 邏輯 or 會給 1，位元 or 給 6。
		{"2 or 4 是位元運算", []byte{0x02, 0x04, 0xa0}, 3, []uint16{6}},
		{"8 or 4 的 bit 0 是 0", []byte{0x08, 0x04, 0xa0}, 3, []uint16{12}},
		// bnot 只翻 bit 0，不是整個 word 取補數。
		{"not 0", []byte{0x00, 0x9f}, 2, []uint16{1}},
		{"not 1", []byte{0x01, 0x9f}, 2, []uint16{0}},
		{"not 2 只看 bit 0", []byte{0x02, 0x9f}, 2, []uint16{1}},
		{"not 7 只看 bit 0", []byte{0x07, 0x9f}, 2, []uint16{0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, image)
			h.startDispatch(tc.pcode, operandStream)
			h.runPCode(tc.ops, 128)
			wantSP := uint32(stackTop - 2*len(tc.stack))
			if got := h.cpu.State.SSP; got != wantSP {
				t.Fatalf("SSP 是 0x%06x，want 0x%06x", got, wantSP)
			}
			for i, want := range tc.stack {
				if got := h.peek(h.cpu.State.SSP + uint32(i)*2); got != want {
					t.Fatalf("堆疊第 %d 個 word 是 %d，want %d", i, got, want)
				}
			}
		})
	}
}

// TestUCSDInterpBooleanTruthLivesInBitZero：fjp 與 tjp 看的是 bit 0，不是整個 word。
//
// 分派表指到的常式是 `move.w (a7)+,d0` / `btst #0,d0` / `beq`（fjp）或 `bne`（tjp）。
// 所以偶數是假、奇數是真，與「非零即真」不同——8 是假，9 是真。
func TestUCSDInterpBooleanTruthLivesInBitZero(t *testing.T) {
	image := loadInterp(t)
	cases := []struct {
		name  string
		pcode []byte
		ops   int
		stack []uint16
	}{
		// ldcb 8 / fjp +1 / sldc3 / sldc7：8 的 bit 0 是 0 → 跳過 sldc3。
		{"fjp 把 8 當成假", []byte{0x80, 0x08, 0xd4, 0x01, 0x03, 0x07}, 3, []uint16{7}},
		// 9 的 bit 0 是 1 → 不跳。
		{"fjp 把 9 當成真", []byte{0x80, 0x09, 0xd4, 0x01, 0x03, 0x07}, 4, []uint16{7, 3}},
		{"tjp 把 8 當成假", []byte{0x80, 0x08, 0xf1, 0x01, 0x03, 0x07}, 4, []uint16{7, 3}},
		{"tjp 把 9 當成真", []byte{0x80, 0x09, 0xf1, 0x01, 0x03, 0x07}, 3, []uint16{7}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, image)
			h.startDispatch(tc.pcode, operandStream)
			h.runPCode(tc.ops, 128)
			wantSP := uint32(stackTop - 2*len(tc.stack))
			if got := h.cpu.State.SSP; got != wantSP {
				t.Fatalf("SSP 是 0x%06x，want 0x%06x", got, wantSP)
			}
			for i, want := range tc.stack {
				if got := h.peek(h.cpu.State.SSP + uint32(i)*2); got != want {
					t.Fatalf("堆疊第 %d 個 word 是 %d，want %d", i, got, want)
				}
			}
		})
	}
}

// TestUCSDInterpNewGameDamageBranch 用原版的那一段條件式與原版的數值驗收。
//
// SunDog 的 XSTARTUP:0x31（段內 0x1acb）決定新遊戲初始損壞長什麼樣：
//
//	sldl4 / sldc0 / equi      ; 壞掉的欄 == 0
//	sldc0 / scxg4 0xb / lor   ; 位元或上一次 random()
//	fjp 0x1ae3                ; 結果的 bit 0 為 0 → 寫替代零件、列狀態 3
//
// 這裡把 random() 的呼叫換成一個常數（呼叫本身是跨段呼叫，不在這個切片的範圍），
// 其餘照原樣跑：欄號與那個常數是輸入，跳不跳是輸出。四種組合涵蓋整張真值表。
//
// p-code 序列（欄號與 random 值由 ldcb 給）：
//
//	ldcb 欄 / sldc0 / equi / ldcb 亂數 / lor / fjp +1 / sldc3 / sldc7
//
// 跳過 sldc3 表示走了「列狀態 3」那條路（fjp 為假時跳）。
func TestUCSDInterpNewGameDamageBranch(t *testing.T) {
	image := loadInterp(t)
	cases := []struct {
		name    string
		column  byte
		random  byte
		status3 bool
	}{
		// 欄 0 的 control node 不吃分流器，所以 equi 給 1，條件恆真。
		{"欄 0、亂數偶數：條件真", 0, 8, false},
		{"欄 0、亂數奇數：條件真", 0, 9, false},
		// 欄非 0 時整個判斷落在亂數的最低位元上。
		{"欄 2、亂數奇數：條件真", 2, 4097 & 0xff, false},
		{"欄 2、亂數偶數：條件假 → 列狀態 3", 2, 8, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, image)
			pcode := []byte{
				0x80, tc.column, // ldcb 欄
				0x00,       // sldc0
				0xb0,       // equi
				0x80, tc.random, // ldcb 亂數
				0xa0,       // lor
				0xd4, 0x01, // fjp +1
				0x03, // sldc3（條件真才執行）
				0x07, // sldc7（哨兵，兩條路都會到）
			}
			ops := 8
			if tc.status3 {
				ops = 7 // 跳過 sldc3 就少一條
			}
			h.startDispatch(pcode, operandStream)
			h.runPCode(ops, 256)
			want := []uint16{7, 3}
			if tc.status3 {
				want = []uint16{7}
			}
			wantSP := uint32(stackTop - 2*len(want))
			if got := h.cpu.State.SSP; got != wantSP {
				t.Fatalf("SSP 是 0x%06x，want 0x%06x（堆疊深度不同表示走了另一條路）", got, wantSP)
			}
			for i, w := range want {
				if got := h.peek(h.cpu.State.SSP + uint32(i)*2); got != w {
					t.Fatalf("堆疊第 %d 個 word 是 %d，want %d", i, got, w)
				}
			}
		})
	}
}
