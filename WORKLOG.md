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
- 完成 `TRAP #0..#15` 與 MC68000 `RTE`；固定語料 5,000 筆涵蓋 format-0 frame、
  vectors 32–47、normal return、user privilege vector 8、odd target vector 3、SR mask、
  stack 提交點與 bus 次序，全數通過後累計外部單步驗收 177,500 筆。
- 完成 `EOR.B/W/L` 與 `EORI.B/W/L`；固定語料 7,500 筆涵蓋 Dn／memory destination、
  immediate、三種寬度、X／NZVC、RMW、EA clocks、long word 寫回次序與 vector 3，
  全數通過後累計外部單步驗收 185,000 筆。
- 完成 `LSL.B/W/L`；固定語料 7,500 筆涵蓋 immediate／Dn count、三種寬度、
  零 count、X／NZVC、動態 clocks、word memory RMW、EA、bus 與 vector 3，
  全數通過後累計外部單步驗收 192,500 筆。
- 完成 `ROR.B/W/L`；固定語料 7,500 筆涵蓋 immediate／Dn count、三種寬度、
  零 count、X 保留、NZVC、動態 clocks、word memory RMW、EA、bus 與 vector 3，
  全數通過後累計外部單步驗收 200,000 筆。
- 完成 `SUBA.W/L` 與 `CMPA.W/L`；四份固定語料 10,000 筆涵蓋 word sign extension、
  全部 data source EA、CCR／X、clocks、bus 與 vector 3；並以語料找出及修正
  CMPA.L An-direct 與 CMPM mask 的解碼重疊，累計外部單步驗收 210,000 筆。
- 完成 `EXG` 三種合法形式；固定語料 2,500 筆涵蓋 Dn↔Dn、An↔An、Dn↔An、
  A7 stack bank、SR 不變、固定 clocks 與 prefetch bus，全數通過後累計外部單步
  驗收 212,500 筆。
- 完成 `MOVE An,USP` 與 `MOVE USP,An`；固定語料 5,000 筆涵蓋正常 supervisor、
  user privilege vector 8、A7／SSP bank、指令起始 saved PC、frame 與 bus，
  全數通過後累計外部單步驗收 217,500 筆。
- 完成 `MOVE <ea>,CCR` 與 `MOVE <ea>,SR`；固定語料 5,000 筆涵蓋 masks、word data EA、
  vector 3、user privilege vector 8、S bit 切換後 program FC 與 `PC-2` 管線重讀，
  全數通過後累計外部單步驗收 222,500 筆。
- 完成 `MOVE SR,<ea>`；固定語料 2,500 筆涵蓋 Dn／memory destination、user mode、
  SR 保留、RMW bus 次序、EA clocks 與 978 筆 vector 3，全數通過後累計外部單步
  驗收 225,000 筆。
- 完成 `TAS.B`；先以 DM12EN `COMMAND.C:119` 確認遊戲命令佇列鎖用途，再用 Hatari
  兩次最小探針修正上游已知的 memory timing 誤差。固定語料 2,500 筆在局部 `+2`
  clocks 勘誤後涵蓋 Dn／memory、旗標、EA、RAM 與 transaction 全同，累計外部單步
  驗收 227,500 筆。
- `go vet ./...` 的 `stdmethods` 會把專案既有 address-aware `ReadByte(address, FC)`／
  `WriteByte(address, value, FC)` 誤認為 `io.ByteReader`／`io.ByteWriter` 慣例；以同一 vet
  停用該不適用分析器後全數通過，沒有將命名警告誤列為產品缺陷。
- 完成 MC68000／ST power-on reset 與 machine epoch：FC=6 載入 SSP／PC／
  prefetch，失敗不提交 staged CPU state，成功 reset 歸零指令／clock 計數。
  以 EmuTOS 1.3 真實 ROM 執行首條 `BRA.W`，得到與 Hatari 相同的
  SSP／SR／PC／prefetch 與 10 clocks；完整 227,500 筆 CPU 語料回歸、
  Go 靜態檢查與建置均通過。
- 以 EmuTOS 連續單步將開機首個失敗收旂到 `$FF8001` MMU 讀取；
  完成 cold-reset latch、supervisor byte R/W、高位保留但不參與 bank 解碼，以及 512 KiB／1 MiB
  實體 topology 在三種 STF 邏輯 bank 大小下的位址轉換。Hatari I/O trace
  確認 `$00→$0A→$05`序列；同 ROM 前 7 條指令已至 `$FC0070`、
  92 clocks 並全狀態一致。完整 CPU 語料回歸、靜態檢查與建置通過。
- 完成 MC68000 對 `$4E7A/$4E7B` 68010+ `MOVEC` 的 illegal-instruction
  vector 4；synthetic 測試驗證兩方向、user／supervisor、frame、FC、bus 與
  36 clocks。EmuTOS 同 ROM 以 8 條指令／128 clocks 到 `$FC0074`，
  SSP／SR／prefetch／frame 與 Hatari 全同。後續 `$FC0080` bus-error frame 探針
  發現 `$FFFF8006`／`$00FF8006` 差異，已收斂成下一個明確 gate。
- 修正 word-source bus error 的 CPU／bus 位址邊界：absolute-short EA 的
  32-bit `$FFFF8006` 保留於 vector 2 frame，ST backend 與 transaction 仍用
  24-bit `$FF8006`。synthetic bus 與 EmuTOS 第 10 條／220 clocks 完整對拍通過。
- 實作 privileged MC68000 `RESET`：optional external reset hook、user vector 8、
  register preservation、FC=6 prefetch 與 132 clocks；ST memory reset MMU latch
  但不清 RAM。EmuTOS 第 11 條／352 clocks 與 Hatari 全狀態一致，新停點收斂到
  `$FA0000` 空 cartridge window。
- 建立 128 KiB 空 cartridge ROM window：`$FA0000–$FBFFFF` 回 `$FF`，
  user／supervisor 可讀、write typed read-only fault，且與 MMU translation 分離。
  EmuTOS 第 12 條／380 clocks 與 Hatari 全狀態一致。
- 逐條追查新停點，確認第 14 條 RAM write 因全機週期位置比固定 CPU timing
  多 2 clocks；工作轉入 cycle-aware bus／Shifter arbitration 架構決策，
  `$FC00BE` line-F 先保留 oracle 收據而不越過前置差異實作。
- 使用者定案採完整、漸進式 cycle-aware Bus；完成 READY 規格 055，明定 access 前
  傳入 machine epoch＋instruction offset、wait 推移後續 phase，以及未遷移 timed path
  失敗即關閉。新增 `BusPhase`，讓固定 MC68000 語料不再丟棄 idle phase，並在每筆載入時
  驗證完整 timeline duration 等於 instruction clocks；227,500 筆回歸、靜態檢查與建置通過。
- 完成 cycle-aware runtime 首切片：新增 `BusAccess`／`TimedBus`／`CPU.StepAt`，由
  machine 傳入當前 64-bit epoch，並遷移 NOP／MOVEQ／SWAP／EXT 共用 prefetch。
  synthetic 2-clock wait 驗證 access 前 clock、idle＋active timeline 與總 clocks；
  NOP 2,500 筆完整 phase、全 227,500 筆舊驗收、固定 EmuTOS 12 條／380 clocks、
  靜態檢查及建置均通過。其餘指令與 Shifter 仲裁未冒稱完成。
- 固定 Hatari v2.4.1 原始碼顯示 cycle-exact memory read／write 會在 access 前，以
  全機＋指令內 clock 對齊四 clock bus slot；將原本籠統的「Shifter arbitration」
  收窄為 shared-memory bus slot alignment。建立 DRAFT 規格 056 並記錄三份來源檔
  SHA-256；因 `$21FC` 未出現在固定 MOVE.L 語料，access offsets、phase 0／2 探針與
  地址分類仍未完成，故未升 READY、未加入 production wait。
- 建立 `$21FC` phase 0／2 成對 Hatari probe：相同目標 PC／state／operand／destination，
  只以 12-clock `JMP` 與 10-clock `BRA.W` 改變起始 phase，實測為 24／26 clocks。
  結合 MAME microcoded `$2B7C`／`$257C` 六個連續 bus phases，確認 `$21FC` offsets
  0／4／8／12／16／20；並確認對齊位於 CPU external access 外層，不是 RAM 位址特例。
  規格 056 升 READY，production 實作留待下一切片。
- 依 READY 規格 056 實作 ST CPU external 四 clock bus slot 對齊，並將
  `MOVE.L immediate→absolute-short` 六個 phases 接入 timed Bus。synthetic phase 0／2
  為 24／26 clocks；固定 EmuTOS 第 14 條由 390→416，RAM／state／prefetch 全同，
  再逐步驗至第 18 條／496 clocks 的 line-F 邊界。完整 227,500 筆語料、固定 ROM、
  靜態檢查與建置通過；規格 056 的正常偶數 destination 切片升 CONFORMED。
- 完成 MC68000 line-F／vector 11：固定 MAME microcoded 語料 2,500 筆確認核心
  exception 為 34 clocks，ST timed Bus 由 6-clock internal phase 起算並補 2-clock
  slot wait。固定 EmuTOS 第 19 條累計 532 clocks 進 `$FC00D4`，6-byte frame、
  SSP／SR／PC／prefetch 全部通過；CPU 外部語料累計 230,000 筆。向後有界探測至
  第 6,851 條才停在 `$FF860F` 保留 I/O 讀取，作為下一個規格切片。
- 依固定 Hatari／EmuTOS 原始碼與 tracepoint，完成普通 ST／Ricoh `$FF860F`
  void byte read：回 `$FF`、不產生 bus error；固定 EmuTOS 第 6,851 條 `TST.B`
  為 8 clocks 且全狀態對拍。規格 058 升 CONFORMED。
- 完成普通 ST 無 Mega-RTC 的 `$FFFC21–$FFFC3F` void byte range：read `$FF`、
  byte write discard，且明確不接主機 wall-clock。固定 EmuTOS 第 6,879 條對拍通過，
  後續推進到 6,916 條；新停點為 `$FF8A3C` Blitter 探測 bus fault。規格 059 升
  CONFORMED。
- 完成 `TST.B (An)` typed bus fault／vector 2：固定 Hatari `$FF8A3C` Blitter probe
  確認 64 clocks 與 frame `$4A15,$FFFF,$8A3C,$4A10,$2704,$00FC,$0638`；synthetic
  ST memory 與固定 EmuTOS 第 6,917 條的 registers、frame、handler prefetch 全同。
  後續推進到 7,474 條，新停點為 `$FFFA01` MFP byte write；規格 060 升 CONFORMED。
