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
