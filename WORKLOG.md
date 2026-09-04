# Atari Talos 工作歷程

## 2026-09-05

- 建立專案骨架、RRSAL-1.0、繁中／英文入口與 Docker Go 工具鏈。
- 建立 `talos-jsonl/1` 控制契約及 CLI；未完成模擬能力採失敗即關閉。
- 固定使用既有 `golang:1.24-bookworm`，明示 `/usr/local/go/bin` 後全套測試與 CLI
  實際往返通過。
- 建立 public GitHub repo，`main` 為預設分支；設定 `atari-st`、`emulator`、
  `ai-tools`、`retrocomputing`、`golang`、`testing` 主題。
- 固定 SingleStepTests/m68000 commit 與授權，建立 NOP READY 規格、24-bit sparse bus、
  預取模型與 corpus loader；以 2,500 筆狀態／clock／bus trace 驗收第一條 CPU 指令。
- 修正外部 corpus 的 Docker 邊界：主機端解析與驗檔，容器內固定唯讀 `/corpus`，
  避免相對路徑在 Codex／Claude Code 工作目錄間漂移。
- 新增 MOVEQ READY 規格與實作；以 2,500 筆外部語料驗 Dn、符號延伸、CCR、預取及 bus。
- 依 Dungeon Master DM12EN 重建組語的實際使用情形，新增 SWAP、EXT.W、EXT.L；三份
  外部語料共 7,500 筆完整狀態、RAM、clock 與 bus transaction 全部通過，CPU 累計
  驗收 12,500 筆。
- 固定 NXP／Motorola 官方 M68000PRM 手冊雜湊，建立 Bcc／BRA 正常控制流規格；
  1,830 筆偶數目標語料全部通過，CPU 累計正常單步驗收 14,330 筆。另確認 670 筆
  奇數目標屬 address error，尚未實作例外時採失敗即關閉且不改狀態。
- 完成 MC68000 address error 的 14-byte frame、模式切換、trace 清除、vector 3 與
  handler 預取；訂正 `re` 未定義 data bus 的初步解讀，不對 fault 位址虛構 RAM read。
  670 筆例外語料全數通過，CPU 累計外部單步驗收 15,000 筆。
- 完成 BSR 與 RTS 的 active-stack call／return 垂直鏈；兩份各 2,500 筆語料均包含
  正常與 address-error 路徑並全數通過，CPU 累計外部單步驗收 20,000 筆。
- 完成 JMP／JSR 共用 control-EA 解碼與各自 bus 次序，七種 68000 control mode 共
  5,000 筆全數通過。依官方相容性文字與語料訂正 extension bits 10–8：68000 忽略，
  不誤判為 full-extension 例外。CPU 累計外部單步驗收 25,000 筆。
- 將 control-EA 接到 LEA／PEA；保留 PEA absolute modes 在最後一次預取前插入 stack
  write 的 microcode 次序。兩份共 5,000 筆全部通過，CPU 累計外部單步驗收 30,000 筆。
- 建立第一個 data-EA 垂直切片：`MOVE.B` 全部合法 source modes 至 Dn。加入 byte bus
  read、UDS／LDS lane、A7 byte delta、立即值與 source prefetch 排程；384 筆固定子集
  全部通過，CPU 累計外部單步驗收 30,384 筆。
- 完成 `MOVE.B` 全部記憶體目的端及 byte write bus，保留 predecrement 與 absolute-long
  的不同 prefetch／write 次序，也驗證 source side effect 後才計算 destination EA。
  新增 2,116 筆全狀態、RAM、clock、bus 驗收，完整 MOVE.B 2,500 筆全通過，CPU
  累計外部單步驗收 32,500 筆。
- 完成 `MOVE.W` 全部合法 source／destination EA 與 data address error。固定分母為
  正常 1,013、source read fault 839、destination write fault 648；2,500 筆完整狀態、
  RAM、clock、bus 與 vector-3 frame 全數通過，CPU 累計外部單步驗收 35,000 筆。
- 完成 `MOVE.L` 全部合法 source／destination EA、兩次 word 組成的 long bus access
  與 data address error。固定分母正常 1,013、source fault 869、destination fault 618；
  2,500 筆全數通過，CPU 累計外部單步驗收 37,500 筆。
- 完成 `MOVEA.W`／`MOVEA.L`：重用已驗證的 word／long source EA 與 address-error
  路徑，新增 word 符號延伸及 An／A7 destination。兩份共 5,000 筆全數通過，CPU
  累計外部單步驗收 42,500 筆。
- 依 DM12EN 產生組語盤點，完成高頻 `ADDA.W`／`ADDA.L`。兩份語料共 5,000 筆
  全數通過；其中 `ADDA.L` 的 memory source 比 register／immediate source 少 2 clocks，
  已分開建模。CPU 累計外部單步驗收 47,500 筆。
- 完成 `AND.B/W/L` 與同組語料涵蓋的 `ANDI.B/W/L`；三種寬度共 7,500 筆全數通過。
  讀改寫固定在最後一次 prefetch 後才 write，long write 為 low word 先行；word／long
  postincrement 的 address-error 副作用分開建模。CPU 累計外部單步驗收 55,000 筆。
- 完成 `CMP.B/W/L`、`CMPI.B/W/L` 與 `CMPM.B/W/L`；三種寬度共 7,500 筆全數通過。
  CMPM 的兩次後遞增、alias、來源／目的 odd-address 副作用與 saved PC 採獨立微時序；
  CPU 累計外部單步驗收 62,500 筆。
- 完成 `ADD.B/W/L`、`ADDI.B/W/L` 與 `ADDQ.B/W/L`；三種寬度共 7,500 筆全數通過。
  ADDQ 的 0→8 immediate、An 特例、long 額外時鐘，以及所有記憶體讀改寫與
  address-error 路徑均逐筆驗收；CPU 累計外部單步驗收 70,000 筆。
- 完成 `CLR.B/W/L`；三種寬度共 7,500 筆全數通過。Dn 的窄寬度保留、X／NZVC、
  記憶體讀改寫 bus 次序、long low-word-first 寫回與 address-error 路徑均逐筆驗收；
  CPU 累計外部單步驗收 77,500 筆。
- 完成 `MOVEM.W/L`；兩種寬度共 5,000 筆全數通過。雙向 register mask、word
  符號延伸、predecrement 反序、postincrement、記憶體載入末尾虛讀、PC-relative
  program FC 與 address-error 微時序均逐筆驗收；CPU 累計外部單步驗收 82,500 筆。
- 完成 `LINK` 與 `UNLK` 正常函式框架路徑。LINK 2,500 筆外部語料全數通過；UNLK
  因上游無獨立語料，以官方契約加固定本地 state／RAM／clock／bus 測試，odd frame
  維持失敗即關閉。CPU 累計外部單步驗收 85,000 筆。
- 完成 `TST.B/W/L`；三種寬度共 7,500 筆全數通過。Dn 與全部合法 memory data EA、
  X／NZVC、無寫回、prefetch、clock 與 word／long address-error 均逐筆驗收；CPU
  累計外部單步驗收 92,500 筆。
- 完成 `OR.B/W/L` 與 `ORI.B/W/L`；三種寬度共 7,500 筆全數通過。一般 OR 雙方向、
  immediate、窄寬度 Dn、全部合法 EA、讀改寫 bus 次序、long low-word-first 寫回與
  address-error 均逐筆驗收；CPU 累計外部單步驗收 100,000 筆。
- 完成 `SUB.B/W/L`、`SUBI.B/W/L` 與 `SUBQ.B/W/L`；三種寬度共 7,500 筆全數
  通過。一般 SUB 雙方向、immediate、quick、An 特例、X／NZVC borrow 旗標、全部
  合法 EA、讀改寫 bus 次序與 address-error 均逐筆驗收；CPU 累計外部單步驗收
  107,500 筆。
- 完成 `ASL.B/W/L`；三種寬度共 7,500 筆全數通過。立即值／暫存器 count、低六位
  截斷、零 count、逐步符號變化 V、最後移出位元 C／X、動態 clocks、word memory
  讀改寫與 address-error 均逐筆驗收；CPU 累計外部單步驗收 115,000 筆。
- 完成 `ASR.B/W/L` 與 `LSR.B/W/L`；六份語料共 15,000 筆全數通過。立即值／
  暫存器 count、低六位截斷、零 count、算術符號／邏輯零填入、最後移出位元 C／X、
  動態 clocks、word memory 讀改寫與 address-error 均逐筆驗收；CPU 累計外部單步
  驗收 130,000 筆。
- 完成 `MULS.W` 與 `MULU.W`；兩份語料共 5,000 筆全數通過。signed／unsigned
  16×16→32、全部 word data EA、X／NZVC、prefetch、address-error 與資料相依 clocks
  均逐筆驗收；CPU 累計外部單步驗收 135,000 筆。
- 完成 `NOT.B/W/L` 與 `NEG.B/W/L`；六份語料共 15,000 筆全數通過。Dn／memory、
  三種寬度、邏輯反相、`0−operand`、X／NZVC、long low-word-first 寫回、prefetch、
  clocks 與 address-error 均逐筆驗收；CPU 累計外部單步驗收 150,000 筆。
- 完成全部 16 種 `Scc.B`；固定語料 2,500 筆全數通過。真假條件、Dn 不同 clocks、
  SR 完全不變、所有 memory destination EA、operand read → prefetch → byte write、
  UDS／LDS 與 A7 delta 均逐筆驗收；CPU 累計外部單步驗收 152,500 筆。
- 完成全部 16 種 `DBcc.W`；固定語料 2,500 筆全數通過。condition 成立、計數到期、
  成功分支三條正常時序與奇數目標 vector 3 均逐筆驗收；CPU 累計外部單步驗收
  155,000 筆。
- 訂正先前把 `UNLINK.json.bin` 誤判為不存在的盤點結果；補跑 UNLK 正常 1,385 與
  odd-frame vector-3 1,115 筆，並修正 A7 alias 最終提交讀出 long 的行為。2,500 筆
  全數通過，CPU 累計外部單步驗收 157,500 筆。
- 依 Atari Corporation 1986 硬體規格建立 ST／STF 基礎 memory map；加入兩種 RAM
  容量、reset ROM shadow、TOS ROM、24-bit mask、FC 權限、big-endian word、read-only、
  未映射／保留 I/O typed bus fault 與原子寫入測試。vector 2 與 I/O 裝置仍另待規格。
- 依 DM12EN 靜態使用量選出 bit 指令族；完成 `BTST`／`BCHG`／`BCLR`／`BSET`
  dynamic／immediate、Dn／byte memory、Z、EA、bus 與資料相依 clock，四份固定語料
  10,000 筆全數通過，CPU 累計外部單步驗收 167,500 筆。
- 完成 `DIVS.W`／`DIVU.W` 的成功、overflow、word EA、資料相依 clocks 與 vector 3，
  固定語料 5,000 筆全數通過；再以 Hatari 兩個最小 PRG 補齊 divisor=0 vector 5 的
  40-clock、Z、Dn、6-byte frame 與 handler prefetch，累計外部單步驗收 172,500 筆。
- `go vet ./...` 的 `stdmethods` 會把專案既有 address-aware `ReadByte(address, FC)`／
  `WriteByte(address, value, FC)` 誤認為 `io.ByteReader`／`io.ByteWriter` 慣例；以同一 vet
  停用該不適用分析器後全數通過，沒有將命名警告誤列為產品缺陷。
