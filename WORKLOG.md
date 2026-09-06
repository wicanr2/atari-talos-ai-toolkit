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
- 完成 MFP GPIP `$FFFA01` reset-state byte write：先由 NXP MC68901 manual、固定
  Hatari／EmuTOS 原始碼、M68000 corpus 與 Hatari trace 收斂規格 061，再實作 GPIP／DDR
  reset state、DDR-masked write 及 `MOVE.B #imm,(An)` timed MFP phase。固定 EmuTOS
  第 7,475 條為 176,638 clocks，state／prefetch／GPIP 對拍；下一停止點為
  `$FFFA03` AER。完整 230,000 筆 corpus、固定 ROM、全測試、vet 與 build 通過，
  規格 061 升 CONFORMED。
- 完成 MFP AER `$FFFA03` reset-state zero write：依 NXP manual 的 edge polarity 與
  AER-write transition 警告，只放行不產生 transition 的 `$00→$00`，非零值以 typed
  `unsupported_device_state` 失敗。固定 Hatari trace 為 16 clocks；EmuTOS 第 7,479
  條／176,682 clocks 的 state、prefetch、AER 對拍通過，下一停點 `$FFFA05` DDR。
  完整 corpus、固定 ROM、全測試、vet 與 build 通過，規格 062 升 CONFORMED。
- 完成 MFP DDR `$FFFA05` reset-state zero write：依 NXP manual 的 input／output
  direction 契約與 Hatari interrupt reevaluation，只放行 `$00→$00`；非零方向改寫
  typed fail-closed。Hatari trace 為 16 clocks，固定 EmuTOS 第 7,483 條／176,726
  clocks 全狀態對拍；下一停點 `$FFFA07` IERA。完整 corpus、固定 ROM、全測試、
  vet 與 build 通過，規格 063 升 CONFORMED。
- 合併完成 MFP IERA／IERB `$FFFA07/$FFFA09` reset-state zero writes：依 NXP
  manual 的 enable／pending 契約，只放行 `$00→$00`，非零 enable typed fail-closed。
  Hatari 兩次 trace 各 16 clocks；固定 EmuTOS 第 7,491 條／176,814 clocks 全狀態
  對拍，下一停點 `$FFFA0B` IPRA。完整 corpus、固定 ROM、全測試、vet 與 build
  通過，規格 064 升 CONFORMED。
- 完成 MFP IPRA／IPRB `$FFFA0B/$FFFA0D` write-zero-to-clear：依 NXP manual
  的 pending bit 契約與固定 Hatari handler，實作 reset/read latch 與
  `pending &= value`，涵蓋注入 `$A5` 後寫 `$3C` 得 `$24`、write `$FF` 不設 bit、
  alias、wait、保護與寬度錯誤。Hatari 兩次 trace 各 16 clocks；固定 EmuTOS
  第 7,499 條／176,902 clocks 全狀態對拍，下一停點 `$FFFA0F` ISRA。完整 corpus、
  固定 ROM、全測試、vet 與 build 通過，規格 065 升 CONFORMED。
- 完成 MFP ISRA／ISRB `$FFFA0F/$FFFA11` write-zero-to-clear：依 NXP manual
  的 automatic／software EOI 與 in-service 契約，實作 reset/read latch 及
  `in_service &= value`；IACK、priority、IRQ 與 Vector Register 明確留在範圍外。
  Hatari 兩次 trace 各 16 clocks；固定 EmuTOS 第 7,507 條／176,990 clocks 全狀態
  對拍，下一停點 `$FFFA13` IMRA。完整 230,000 筆 corpus、固定 ROM、全測試、
  vet 與 build 通過，規格 066 升 CONFORMED。
- 完成 MFP IMRA／IMRB `$FFFA13/$FFFA15` mask latch：依 NXP manual 的
  mask／pending／IRQ 契約，在 pending 為 0 時接完整 byte latch；pending 非零時
  因 IRQ 尚未建模而 typed fail-closed，失敗不改 IMR／IPR。Hatari 兩次 trace 各
  16 clocks；固定 EmuTOS 第 7,515 條／177,078 clocks 全狀態對拍，下一停點
  `$FFFA17` Vector Register。完整 230,000 筆 corpus、固定 ROM、全測試、vet 與
  build 通過，規格 067 升 CONFORMED。
- 完成 MFP Vector Register `$FFFA17`：依 NXP manual 實作 `$F8` 有效位元、
  vector base、software／automatic EOI 與切回 automatic 清雙 ISR；Hatari 會保存
  unused bits 的差異已明列。pending 非零而需重算 IRQ 的切換維持 typed fail-closed。
  固定 Hatari trace 為 16 clocks；EmuTOS 第 7,519 條／177,122 clocks 全狀態對拍，
  下一停點 `$FFFA19` Timer A Control Register。完整 230,000 筆 corpus、固定 ROM、
  全測試、vet 與 build 通過，規格 068 升 CONFORMED。
- 合併完成 MFP TACR／TBCR／TCDCR `$FFFA19/$FFFA1B/$FFFA1D` reset-stop：
  依 NXP manual 只接 reset/read 與 `$00→$00`，非零 control 因 prescaler、counter、
  output 與 interrupt 尚未建模而 typed fail-closed。固定 Hatari 三次 trace 各
  16 clocks；EmuTOS 第 7,531 條／177,254 clocks 全狀態對拍，下一停點 `$FFFA1F`
  Timer A Data Register。完整 230,000 筆 corpus、固定 ROM、全測試、vet 與 build
  通過，規格 069 升 CONFORMED。
- 合併完成 MFP TADR／TBDR／TCDR／TDDR `$FFFA1F/$FFFA21/$FFFA23/$FFFA25`
  stopped-load：timer stopped 時任意 byte 同步載入 data/main counter，active timer
  因 countdown、capture、reload 與臨界不定值尚未建模而 typed fail-closed。固定
  Hatari 四次 trace 各 16 clocks；EmuTOS 第 7,547 條／177,430 clocks 全狀態對拍，
  下一停點 `$FFFA27` SCR。完整 230,000 筆 corpus、固定 ROM、全測試、vet 與 build
  通過，規格 070 升 CONFORMED。
- 完成 MFP SCR／UCR／RSR／TSR `$FFFA27/$FFFA29/$FFFA2B/$FFFA2D` reset writes：
  依 NXP manual 不把 TSR 冒稱硬體 reset-zero，只在固定 EmuTOS 軟體寫零後標為已知；
  非零 serial state 與 UDR 維持 typed fail-closed。Hatari 四次 trace 各 16 clocks；
  EmuTOS 第 7,563 條／177,606 clocks 全狀態對拍，有界續跑至第 7,598 條後停在
  `STOP` `$4E72`。完整 230,000 筆 corpus、固定 ROM、全測試、vet 與 build 通過，
  規格 071 升 CONFORMED。
- 完成 MC68000 `STOP`：以 NXP manual 與固定 `STOP.json.bin` 建立 privilege、immediate
  SR、4-clock 與 stopped latch 契約；CPU Reset 清 latch，未建模 interrupt 喚醒前後續
  Step 明確回 `ErrStopped`。釐清 Talos PC 是 next-instruction `$FCD09E`、實際 opcode
  位址為 `$FCD09A`，並保留 Hatari D2／D3 差異。2,500 筆 STOP 語料加入後累計
  232,500 筆；固定 EmuTOS 第 7,599 條／178,096 clocks 進入停機。全測試、vet、build
  通過，規格 072 升 CONFORMED。
- 完成固定 color ST profile 的 MFP GPIP input sample：依 MC68901 DDR 契約合併
  `$A1` 外部 pins，對上 Hatari `$FC67B8` monitor probe，STOP 前 D2 從 `$2704`
  收斂為 `$2710`。剩餘 D3 差異由 ROM bytes 與 EmuTOS producer／consumer 證實為
  `$466 frclock` 尚無 VBL producer，不再歸咎 CPU／GPIP。完整 232,500 筆 corpus、
  固定 ROM、全測試、vet、build 通過，規格 073 升 CONFORMED。
- 完成 MC68000 level 4 autovector CPU 接受層：NXP manual 確認 STOP／mask／vector 28，
  固定 Hatari 量得 `$70=$FC0446`、STOP 後 saved PC `$FCD09E`、SSP `$F70→$F6A`、
  SR mask 4 與 handler prefetch `$52B8,$0466`。實作遮罩拒絕、非法 level fail-closed、
  44 clocks、format-0 frame 與成功後解除 stopped latch；完整 232,500 筆 corpus、固定
  EmuTOS STOP gate、全測試、vet、build 通過，規格 074 升 CONFORMED。GLUE frame
  phase 尚無 READY 規格，故未猜測 VBL deadline 或直接改 `$466`。
- 完成 ST reset frame 第一個 GLUE VBL：固定 Hatari source 確認 color ST reset 的
  263×508+64=133,668 clock deadline與 pending 保留；先以可丟棄探針發現
  `--fast-boot true` profile 不同，再用 `false` 重拍，第一 `$FC0446` handler 的完整
  D/A、SSP、SR、saved PC、prefetch 全與 Talos 相同。對拍同時修正 running interrupt
  saved PC 為 pipeline `State.PC-4`，STOP 路徑仍保存已推進的 `State.PC`。guest handler
  真正令 `$466` 由 0 變 1，返回後 D3=1；新 STOP gate 為 7,604 instructions、1 interrupt、
  178,228 clocks。完整 232,500 筆 corpus、固定 ROM、全測試、
  `go vet -stdmethods=false ./...` 與 CLI build 通過，規格 075 升 CONFORMED。
- 完成 recurring VBL、STOP 快轉與 ST 視訊 IACK的首版；當時未讀 `$FF820A`，誤把
  reset 後 frame 寫成 50 Hz。由 Hatari 固定 source
  查實 `12-clock IACK start → 10-clock E-clock 對齊 → 10-clock video IACK`，沒有以固定
  16 clocks 猜補。該輪曾記錄第二 deadline 293,924／handler 293,984；後續第三 VBL
  trace 證明此數值錯誤，現行收據已依規格 076 訂正為 267,272／267,332。
  `$FC0446`，完整 D/A、SSP、SR、saved PC 與 prefetch 對上 Hatari，guest handler 真正令
  `$466 frclock` 由 1 變 2。同步訂正规格 075 首次 handler 的 machine clock 為 178,012；
  register／frame 結論不變。有界續跑可跨第三次 VBL，於 7,654 instructions、3 interrupts、
  當時記錄的 454,504-clock `$FF8260` gate 也隨同失效，訂正值見下一項。
  規格 076 升 CONFORMED。
- 完成 Shifter resolution `$FF8260` 的 reset／低解析度同值初始化首切片：Atari hardware
  map 與固定 Hatari source 確認 byte R/W、bits 1–0 模式及 STF read unused bits 為 1；
  固定 Hatari／EmuTOS 在 VBL=3、FrameCycles 384 由 `$FC69E6: 11 C0 82 60` 寫 D0=0，
  12 clocks 後抵 `$FC69EA`。Talos 對上完整 D/A、SSP、SR、prefetch 與 register read `$FC`；
  medium／high／非法模式仍 typed fail-closed，未冒稱 framebuffer 已完成。有界續跑至
  第三 VBL trace 同時證實 `$FF820A` 此時仍為 0，故修正 reset recurring frame 為 60 Hz；
  7,662 instructions／3 interrupts／401,270 clocks，在 `$FF820A` video sync byte write
  成為下一 gate。規格 077 升 CONFORMED。
- 完成 GLUE video sync `$FF820A` 固定第 0 線 60→50 Hz transition：Hatari
  CycleCounter 直接量得 `$FC6A02` 前 401,272／register 0、12 clocks 後 401,284／register 2；
  VBL4 第一 instruction boundary 535,532、FrameCycles=68，故 event deadline=535,528。
  這證明切換當幀是原 133,604 clocks 加剩餘 262 lines×4=1,048，而非立刻套完整
  160,256-clock frame。Talos 在固定 guest path 以 401,270→401,282 完成同狀態寫入，
  排程改為 next=535,528、後續 period=160,256。下一 gate 為 7,671 instructions／
  401,366 clocks 的 `$FF8240` palette word write。規格 078 升 CONFORMED。
- 完成 ST Shifter `$FF8240–$FF825E` 16 色 palette word bank：一手 hardware map 與
  Hatari 固定 source 確認 ST `$0777` mask、4-clock bus boundary、byte mirroring 特例與
  STF read unused bits限制；本切片只接 deterministic word path。固定 EmuTOS 首筆
  `$FC671A: MOVE.W D1,(A0)+` 以 D1=`$0777`、8 clocks 寫 color 0；完整 16 色表與
  Hatari 相同，Talos 在 7,749 instructions／402,052 clocks 抵達迴圈後 `$FC6722` 狀態。
  palette state 已可供未來 framebuffer 消費，但尚未宣稱像素輸出。下一 gate 為
  7,896 instructions／403,900 clocks 的 `$FF8201` framebuffer base high byte write。
  規格 079 升 CONFORMED。
- 完成 ST Shifter `$FF8201/$FF8203` 程式化 framebuffer base registers：依一手
  hardware map 與固定 Hatari source 接入 high `$3F` mask、middle byte、reset、readback、
  alias、權限、寬度、bus wait 與 `ProgrammedVideoBase()`。固定 EmuTOS 在 Talos
  403,900→403,948 clocks 依序完成兩筆 write 與中間位移，所得 `$0F8000`、完整 CPU
  state 與 Hatari 403,924→403,972 的同路徑一致；兩邊皆未把 write 當成 active base
  立即切換。有界續跑至 68,079 instructions／962,832 clocks 才遇 `$FFFA1D` 非零 timer
  control，但時間序上先處理第四幀前 active base reload。規格 080 升 CONFORMED。
- 完成第四幀 VBL active framebuffer base reload：固定 Hatari trace證實 transition
  frame 沒有 HBL 310，debugger 量得共同 event deadline 535,528 前 active=0、後為
  `$0F8000`。Memory 分離 programmed／active state，Machine 的 running crossing 與
  STOP 快轉共用 VBL 提交入口。Talos 可觀察邊界為 535,520→535,530，Hatari 為
  535,524→535,532；既有 24-clock 累積差距明列而未冒稱收斂。完整 corpus、固定 ROM、
  全測試、vet 與 build 通過後，規格 081 升 CONFORMED。
- 完成 ST low-resolution 4-plane indexed frame首切片：新增精確 32,000-byte→64,000-index
  decoder與 active-base DMA snapshot，輸出 palette copy而不猜 RGB。實作審查抓出 CPU
  reset ROM shadow不適用 Shifter DMA，改由 MMU RAM topology直接讀取並加 base0回歸測試。
  Hatari VBL7 外部 fixture 的 raw／decoded SHA-256、histogram與首非零座標全過；Talos
  正常 VBL4 路徑亦產出全黑 320×200 snapshot。完整 232,500 筆 corpus、固定 ROM、
  全測試、vet、build通過，規格 082 升 CONFORMED；fixture與 ROM 均未入 Git。
- 完成 MFP Timer C delay-mode 啟動首切片：NXP 手冊確認 control 5 是 ÷64；固定
  EmuTOS `xbtimer(2, 0x50, 192, int_timerc)` 與 ROM `$FC62AA` 確認 `$C0/$50`；
  Hatari trace 在 FrameCycles 124,038 顯示 12,288 MFP ticks，固定 clock 比例換算為
  40,106 CPU clocks。Talos 僅接 `$00→$50` 與 start transition，其他 control、
  countdown／timeout／IRQ 仍失敗即關閉。正常路徑前進到 68,103 instructions、
  963,104 clocks，下一 gate 為 `$FC6192` memory `ROL.W` opcode `$E378`。規格 083
  升 CONFORMED。
- 完成 MC68000 ROL.B／W／L：依官方 manual 與固定 SingleStepTests 三份語料實作
  immediate／register count、memory word RMW、C／X／NZV、EA、alignment、clock 與
  bus fault；7,500／7,500 通過，完整 corpus 累計 240,000 筆。固定 EmuTOS
  `$FC6192: E378 1238` 以 16 clocks 執行後，再前進到 68,131 instructions、
  963,388 clocks 的 `$FFFA09` IERB gate。規格 084 升 CONFORMED。
- 完成 MFP Timer C interrupt enable 首切片：NXP manual 確認 IERB bit 5 與兩級
  enable/mask；EmuTOS `mfpint()` 與 Hatari trace 確認先清 IPRB／ISRB，再依序寫
  IERB／IMRB=`$20`，期間沒有 timeout。Talos 只在 TCDCR=`$50`、IPRB=0 接受該 bit，
  正常路徑前進到 68,378 instructions、4 interrupts、966,808 clocks，下一 gate
  是 Timer D 的 TCDCR `$50→$51`。規格 085 升 CONFORMED。
- 完成 MFP Timer D delay-mode boot切片：依 NXP／EmuTOS／Hatari 修正第一筆其實是
  TCDCR `$50→$50` stop-D，再寫 TDDR=`$02` 與 TCDCR=`$51`；沒有把同值 write
  誤當重複啟動。完成 fixed USART UCR／RSR／TSR=`$88/$01/$01`，並依 RBF=12、
  TBE=10 將 IERA／IMRA 接到 `$14/$14`，全程無 pending。規格 086–088 升 CONFORMED。
- 完成 YM2149 boot mixer／port A 固定序列：Hatari `psg_write` trace確認 select 7、
  data `$C0`、select 14、data `$07`；Talos以 typed state與錯序 fail-closed 接線，
  正常路徑前進到 68,528 instructions、4 interrupts、968,510 clocks，下一 gate
  是 ACIA `$FFFC00`。一般 MOVE.B timed-I/O 的 cycle差異明列，規格 089 升 CONFORMED。
- 完成 IKBD MC6850 ACIA control 初始化與第一 transmit deadline：Hatari
  `acia,ikbd_acia` trace確認 `$03` master reset、`$96` configuration、status `$02`、
  TDR=`$80` 後 TDRE 清零，以及 1024 clocks 後恢復。Talos接入 typed control sequence、
  TDR pending state與 recurrent deadline；synthetic memory／scheduler、固定 ROM、完整
  240,000 筆 corpus、全測試、`go vet -stdmethods=false ./...` 與 CLI build通過。
  正常路徑抵達 68,645 instructions、4 interrupts、969,640 clocks 的第二 byte `$01`
  gate；指令邊界近似與 Hatari device-write phase差異明列。規格 090–091 升 CONFORMED。
- 完成IKBD ACIA第二 TX buffer與warm-reset response：Hatari 16-VBL trace確認 `$01`
  在 `$80` frame busy期間只進TDR，經10個serial ticks才移入shift stage；第二 frame收完後
  以固定color-ST的1,002 scanlines／513,024 clocks延遲回傳 `$F1`。Talos新增一次性
  command-consumed latch、RDRF／IRQ status與read-clear，並把device deadline和CPU
  observation boundary分開記錄。正常EmuTOS在128,313 instructions／1,507,268 clocks
  讀取 `$F1`，再前進到136,048 instructions／1,577,208 clocks的MIDI ACIA `$FFFC04`
  gate。完整240,000筆corpus、全測試、`go vet -stdmethods=false ./...`與CLI build通過，
  規格092–093升CONFORMED。
- 完成MIDI ACIA control、IKBD stale RDR與MFP ACIA channel 6：固定Hatari trace確認
  MIDI `$03→$95`，以及IKBD RDRF清除後`$FC06CE`仍讀一次保留的`$F1`。MFP channel 6
  依序以`$BF`清IPRB/ISRB，再將IERB/IMRB從`$20`升為`$60`；stage latch禁止跳步。
  正常路徑抵達136,182 instructions／1,578,882 clocks，下一gate是channel 4／Timer D
  重設的IERB同值`$60`。完整目前corpus、全測試、vet與CLI build通過後，
  規格094–096升CONFORMED。
- 完成MFP Timer D系統時鐘重設：固定Hatari trace確認channel 4先以`$EF`清IPRB/ISRB，
  TCDCR從`$51`停至`$50`，TDDR由`$02`寫成`$00`（MC68901語意256），IERB/IMRB升為
  `$70/$70`，最後TCDCR=`$52`。Hatari記錄`data=256 ctrl=2 timer_cyc=2560`；Talos以
  八段stage防止跳步，固定ROM在136,210 instructions／1,579,228 clocks完成啟動。
  完整目前corpus、全測試、vet與CLI build通過後，規格097升CONFORMED；recurrence、
  pending與MFP IACK明列為下一切片。
- 完成 MFP Timer D recurrence 與 channel 4 向量中斷：MC68901 比例以累積有理數
  `2560*8021248/2457600` 排程，避免逐期 floor 漂移；timeout 設 IPRB bit 4，
  IERB／IMRB 仲裁後由新增的 CPU vectored-interrupt API 以 level 6、vector 68 建立
  44-clock frame。Hatari 固定 trace 顯示暫定 autovector 30 的 `$78` 在 IACK 改成
  `$110`，handler 為 `$FC7884`；Talos 正常 EmuTOS 在 137,138 instructions、
  9 interrupts、1,587,632 clocks 抵達相同 handler，pending 轉入 ISRB bit 4，guest
  隨後自行寫 `$EF` 清除。固定路徑的 `MOVE.B` 尚未供應 timed access，故 start phase
  明列為 instruction-boundary hardware-spec approximation。完整 240,000 筆 corpus、
  固定 ROM、全測試、vet 與 CLI build 通過，規格 098 升 CONFORMED。
- 完成MFP Timer C recurrence與channel 5向量中斷：timed TCDCR start access提供
  phase 962,844，以累積有理數`12288*8021248/2457600`排程；第一個deadline
  1,002,950後，Talos在72,342 instructions／5 interrupts／1,003,004 clocks進入
  vector 69／`$FC04DE`，guest於`$FC050A`寫ISRB=`$DF`清除。B-bank仲裁共用
  channel 5／4，選較高pending且尊重software-EOI in-service priority。此接線證實
  舊路徑漏掉200 Hz tick：IKBD `$F1`、MIDI、MFP channel 6、Timer D啟動與handler
  的現行收據已在CONTEXT／spec／矩陣訂正；歷史舊數字保留於本檔供追溯。完整
  240,000筆corpus、固定ROM、全測試、vet與CLI build通過，規格099升CONFORMED。
- 完成MFP Timer D正常停止與channel 4清除：Hatari完整CPU trace證實`$FC7862`
  `BCLR #4`將IERB `$70→$60`、`$FC786A`將IMRB `$70→$60`、`$FC7872`
  將TCDCR `$52→$50`；後續更新vector `$110=$FC03EA`並以共用`mfpint`做`$EF`
  clear。Talos新增七段stop stage，machine在stop transition清除running phase／period／
  deadline；另依正常路徑證實，Timer C bit5 pending時仍允許IMRB同值`$60`且保留pending，
  未放寬其他mask改寫。固定ROM在289,256 instructions／234 interrupts／2,978,730 clocks
  完成，下一gate為289,332／2,979,596的UCR `$88→$88`。完整240,000筆corpus、固定ROM、
  全測試、vet與CLI build通過，規格100升CONFORMED。
- 完成MFP USART第二次設定與baud Timer D重啟：MC68901契約與固定Hatari trace確認
  TSR bit7為transmit-buffer-empty，EmuTOS依序寫TCDCR `$50`、TDDR `$02`、TCDCR
  `$51`及UCR／RSR／TSR／SCR `$88/$01/$01/$00`。Talos以七段stage接線，並把
  control1 baud timer與control2 system Timer D IRQ scheduler分離。固定ROM在289,342
  instructions／234 interrupts／2,979,680 clocks完成；後續有界探測將下一gate定位為
  289,520 instructions／2,982,748 clocks的DMA／FDC `$FF860F` byte write。完整240,000
  筆corpus、固定ROM、全測試、vet與CLI build通過，規格101升CONFORMED。
- 完成普通ST／Ricoh `$FF860F` void byte write：固定Hatari原始碼證實該位址同時
  安裝void read/write handler，write不建立register state；固定CPU／I/O trace在
  `$FC3788`寫`$00`後自然繼續。Talos在289,521 instructions／2,982,760 clocks完成，
  規格102升CONFORMED。
- 完成YM2149 port A首次drive-select更新：固定EmuTOS依序同值選register 14、讀回
  `$07`、寫成`$05`；以三段stage避免放寬未證實PSG行為。正常路徑在289,556
  instructions／2,983,132 clocks完成，下一gate定位為DMA mode/status
  `$FF8606=$0080`。完整240,000筆corpus、固定ROM、全測試、vet與CLI build通過，
  規格103升CONFORMED。
- 完成ST DMA mode與WD1772 force-interrupt初始化：固定Hatari `fdc.c`證實
  `$0080`選command/status、兩個word I/O各增加4 wait clocks；`$D0`為condition 0
  Type IV，idle時建立Type-I motor-on status並清IRQ。Talos保存可供後續restore消費的
  typed狀態，正常ROM在289,612 instructions／2,983,694 clocks完成；下一gate是第二次
  mode `$0080`，後接restore `$0B`。完整240,000筆corpus、固定ROM、全測試、vet與
  CLI build通過，規格104升CONFORMED。
- 完成WD1772 restore與GPIP5 IRQ期限：固定Hatari `fdc.c`的prepare／track-zero／complete
  共728 FDC clocks，依固定ST clock比例換算729 CPU clocks；Talos將EmuTOS實際使用的
  `MOVE.W 6(A7),abs.w`接到timed word bus，未以指令結束時間猜補。固定ROM自clock
  2,984,902開始，九次讀得`$B1`後於289,803 instructions／2,985,654 clocks讀得`$91`；
  下一gate定位為289,818／2,985,802的`$FF8606` word write。完整240,000筆corpus亦確認
  奇數來源／目的位址例外未被timed path繞過；全測試、vet與build通過，規格105升
  CONFORMED。
- 完成WD1772 restore後Type-I status read與IRQ清除：EmuTOS原始碼確認
  `get_fdc_reg(FDC_CS)`的selector／delay／read順序，固定Hatari trace在`$FC3888`
  寫mode `$0080`，於`$FC3898`回`$00E4`並清IRQ。Talos新增exact word-read bus phase；
  固定ROM在clock 2,986,242取樣，於289,865 instructions／2,986,256 clocks完成，
  D0=`$FFFF00E4`且GPIP5恢復inactive。下一gate為289,982／2,987,452的
  `$FF8606=$0086`。完整240,000筆corpus、固定ROM、全測試、vet與build通過，規格106
  升CONFORMED。
- 完成第一顆drive的WD1772 data-register與same-track seek：固定EmuTOS／Hatari依序
  寫`$0086/$0000/$0080/$0013`；Hatari state machine證實motor已開、TR=DR=0、verify
  off時仍是728 FDC clocks，CPU＋FDC trace另證實九次inactive GPIP read後才讀到active。
  Talos seek自clock 2,988,614起算，於2,989,930讀status `$E4`並清IRQ，290,223
  instructions／2,989,944 clocks完成。下一gate離開FDC，為290,296／2,990,830的
  YM2149 `$FF8800` byte write。完整240,000筆corpus、固定ROM、全測試、vet與build
  通過，規格107升CONFORMED。
- 完成YM2149 port A切至第二顆drive：固定CPU／PSG trace證實`$FC36CA`同值選R14、
  `$FC36CE`讀回`$05`、`$FC36DC`寫`$03`，Hatari記錄drive `0→1`且side維持0。
  Talos於290,303 instructions／2,990,890 clocks完成，下一gate為290,312／
  2,990,998的第二顆drive `$FF8606=$0080`。完整240,000筆corpus、固定ROM、全測試、
  vet與build通過，規格108升CONFORMED。
- 完成第二顆drive的WD1772探測鏈重用：新增明確probe drive身分，
  由drive 0切到drive 1時只清除每顆drive的restore／seek收據，不重置
  CPU、MFP、PSG或全局clock。固定ROM於290,970 instructions／2,997,708 clocks
  完成stage14；restore與seek各九次inactive poll，兩次status `$E4`均清IRQ。
  下一gate為291,291 instructions／3,001,516 clocks的DMA位址暫存器
  `$FF860D` byte write。完整240,000筆corpus、固定ROM、全測試、vet與build
  通過，規格109升CONFORMED。
- 完成ST floppy／ACSI DMA位址暫存器：依Hatari固定原始碼實作
  `$FF8609/$FF860B/$FF860D` byte R/W、high `$3F` mask、low bit 0對齊與ST
  low／middle ripple carry。固定EmuTOS依low→middle→high寫`$04/$10/$00`，
  Talos於291,294 instructions／3,001,576 clocks形成`$001004`，三條指令各
  20 clocks。下一gate為291,343 instructions／3,002,130 clocks的
  `$FF8606=$0190` DMA reset。完整240,000筆corpus、固定ROM、全測試、
  vet與build通過，規格110升CONFORMED。
- 完成ST DMA direction-toggle reset與sector-count zero序列：固定EmuTOS寫
  `$0190→$0090`，兩次切換bit 8皆依Hatari `FDC_ResetDMA()`契約清sector
  count，再以mode bit 4將`$FF8604=$0000`路由為sector-count write。
  Talos於291,376 instructions／3,002,468 clocks完成，mode=`$0090`、count=0、
  reset count=2。下一gate為291,386 instructions／3,002,576 clocks的
  `$FF8606=$0088` ACSI mode。完整240,000筆corpus、固定ROM、全測試、
  vet與build通過，規格111升CONFORMED。
- 完成空ACSI bus的target-0 command開始與guest timeout：固定Hatari在無
  ACSI image時不接受command也不設HDC IRQ；EmuTOS依次寫
  `$0088→data $0000→$008A`後以Timer C／`hz_200`等待。Talos於291,404
  instructions／3,002,700 clocks完成command start，於clock 3,771,064自然走
  timeout並寫`$0080`。修正這筆同值mode被舊drive-1 probe誤混的條件，
  FDC保持stage14。下一gate為361,268 instructions／4,062,736 clocks的
  target-1 `$FF8606=$0088`。完整240,000筆corpus、固定ROM、全測試、
  vet與build通過，規格112升CONFORMED。
- 完成空ACSI bus target 1–7掃描：將規格112的單一attempt參數化，逐target
  保存command與timeout clock收據，錯command與第九個target皆原子拒絕。
  固定ROM自然送出`$20/$40/$60/$80/$A0/$C0/$E0`，於866,723 instructions／
  461 interrupts／11,591,284 clocks完成target 7；八筆timeout-return clocks
  與固定Hatari 65-VBL trace順序一致，全程不設IRQ且FDC保持stage14。下一gate為
  867,255 instructions／11,598,096 clocks的YM2149 `$FF8800` byte write。
  完整240,000筆corpus、固定ROM、全測試、vet與build通過，規格113升CONFORMED。
- 完成YM2149 parallel-port strobe初始化：固定EmuTOS `parport_init()`經
  `ongibit(GI_STROBE)`執行`$FF8800=$0E→read $03→$FF8802=$23`，只設port A
  bit 5並保留drive／side low三位。Talos於867,260 instructions／462 interrupts／
  11,598,144 clocks完成；下一gate為867,320／11,599,192的IKBD ACIA data
  `$FFFC02` byte write。完整240,000筆corpus、固定ROM、全測試、vet與build通過，
  規格114升CONFORMED。
- 完成IKBD clock request `$1C`傳送：固定EmuTOS `igetregs()`與Hatari
  ACIA／IKBD trace證實單byte request及10-tick 8N1 frame。Talos於867,321
  instructions／11,599,204 clocks接受TDR write，在typed device clock
  11,609,950完成frame，觀察邊界為868,214／11,609,966。response `$FC + 6-byte`
  明確留給下一規格，未以guest timeout冒充原版路徑。完整240,000筆corpus、
  固定ROM、全測試、vet與build通過，規格115升CONFORMED。
- 完成IKBD clock response與MFP channel 6：固定profile逐筆送出
  `$FC,$00,$00,$00,$00,$00,$01`，每筆建立ACIA RDRF／IRQ、GPIP4 active-low與
  vector `$46`，EmuTOS共用handler先讀MIDI status `$02`，再由`_ikbdsys`消費RDR。
  七筆於874,579 instructions／471 interrupts／11,688,070 clocks收齊，下一gate為
  874,900／11,691,528的set-clock `$FFFC02=$1B`。完整240,000筆corpus、固定ROM、
  全測試、vet與build通過，規格116升CONFORMED。
- 建立IKBD set-clock規格117並接入七筆TDR／shift-register typed state。固定ROM已寫
  `$1B,$24,$03,$17,$00,$00,$00`，前六筆frame completion clocks鎖定；第七筆尚餘
  10 ticks時，guest即於881,554 instructions／11,753,400 clocks嘗試buffer下一個
  `$1C`。因此117保持READY，跨packet TDR buffering另立下一規格，不把write receipt
  冒充完整firmware consumption。
- 完成規格117／118：MC6850跨packet TDR buffering先在11,763,550提交set-clock最後
  frame，再於同一deadline載入第二個`$1C`，並在11,773,790完成request。固定profile
  readback `$FC,$24,$03,$17,$00,$00,$00`七筆皆經MFP channel 6由EmuTOS消費，完成點
  889,609 instructions／483 interrupts／11,851,910 clocks。下一gate為
  1,005,202／13,036,392的YM2149 `$FF8800=$05`。完整240,000筆corpus、固定ROM、
  全測試、vet與build通過，117與118均升CONFORMED。
- 完成規格119：依固定EmuTOS `flopvbl()`與Hatari bus trace接入VBL66的drive-0
  media-change輪詢。Talos依序將YM2149 port A `$23→$25`、以DMA mode `$0080`
  讀WD1772 status `$E4`、再恢復`$23`，於1,005,296 instructions／521 interrupts／
  13,037,306 clocks完成；同時訂正先前把CPU D0=`$05`誤當control bus value的紀錄，
  實際register select為`$0E`。下一gate為VBL77再次寫IKBD `$1C`；95-VBL Hatari
  trace確認回傳仍是`$FC,$24,$03,$17,$00,$00,$00`，留待下一個READY規格泛化。
- 完成規格120：將第三輪起的IKBD `$1C`讀時鐘實作為可重入週期，以單調request／
  response completion counters排程，且每輪重置獨立payload／delivery receipt，不覆寫
  前兩輪歷史。固定ROM於13,937,502完成request，七筆
  `$FC,$24,$03,$17,$00,$00,$00`於14,015,326送畢，guest在1,092,926 instructions／
  558 interrupts／14,015,626 clocks收齊。下一gate為1,120,640／14,318,580的
  `$FF8800=$0E`；Hatari VBL90顯示為port A維持`$23`的週期性`flopvbl()`檢查。
- 完成規格121：將單次`flopvbl()`擴充為依count輪替drive `0,1,0,1`的可重入媒體檢查，
  每輪保存drive／status clock並恢復port A `$23`。第二輪在1,120,734 instructions／
  14,319,494 clocks完成；固定ROM自然連續完成73輪後，於1,285,863 instructions／
  1,761 interrupts／106,337,672 clocks抵達新的`$FF8606` word write。synthetic四輪、
  完整CPU語料、全測試、vet與build均通過。
- 完成規格122：680-VBL Hatari trace與EmuTOS `flop_mediach/flopio/floplock`證實下一段
  是`$0082`選WD1772 track register、寫track 0，再由YM2149 `$23→$25`選drive 0。
  Talos於clock 106,338,122提交track data，1,286,016 instructions／1,761 interrupts／
  106,339,274 clocks完成read stage 5；下一gate為`$0084` sector selector。既有73輪
  media-check count未被這條非VBL序列污染。
- 完成規格123：固定EmuTOS依序寫sector 1、DMA address `$001004`、兩次DMA direction
  toggle、sector count 1與WD1772 Type-II `$80`。Talos於1,286,164 instructions／1,761
  interrupts／106,340,824 clocks完成read stage 15，command clock為106,340,810；
  synthetic另驗證錯序拒絕、cold reset與無磁片時DMA buffer不變。無磁片timeout與
  force-interrupt留給獨立規格，沒有把command receipt寫成成功讀取證據。
- 完成規格124：固定Hatari 680-VBL FDC trace證實無磁片Type-II `$80`保持busy，EmuTOS
  依motor-on期限等待75個50 Hz VBL／1.5秒後才寫`$0080/$D0`。Talos沿既有Timer C／
  `hz_200`自然抵達相同分支，於2,370,884 instructions／2,136 interrupts／
  118,354,544 clocks完成force-interrupt；status `$80`、Type-II型別與inactive IRQ均
  鎖入回歸測試。下一gate為`$0086` data-register selector。
- 完成規格125：EmuTOS `flopunlk/dummy_seek`與固定Hatari VBL310 trace證實timeout後
  依序送data 0、seek `$13`、等待IRQ並讀status `$E4`。Talos重用既有seek scheduler，
  記錄九次inactive GPIP poll，於2,371,204 instructions／2,136 interrupts／
  118,357,780 clocks完成。跨裝置驗證另發現真正下一gate是YM2149 `$FF8800` byte
  write，不可由FDC-only trace直接外推為sector selector。
- 完成規格126：固定Hatari PSG＋FDC trace補出dummy seek與下一次sector讀取之間的
  R14同值重選`$0E→read $25→write $25`。Talos於2,371,990 instructions／2,136
  interrupts／118,369,170 clocks完成，R14與media-check count均不變；下一gate為
  `$FF8606=$0084`。由於`MOVE.B Dn,d(An)`仍走未定時byte bus路徑，本輪明確保存
  instruction epoch 118,369,158而非虛構精確bus phase。
- 完成規格127：官方EmuTOS 1.3 192K UK ROM雜湊確認為既有固定
  `ad64942f…135`，固定Hatari VBL310 trace證實retry完整重送sector 1／DMA設定與
  Type-II `$80`。Talos新增不覆寫第一次transaction的retry收據，並把stage 27–37
  的非預期DMA byte寫入改為失敗即關閉；於2,372,203 instructions／2,136 interrupts／
  118,371,412 clocks完成command，RAM buffer仍全零。下一gate為3,456,990／
  130,385,952的第二次timeout selector；完整CPU corpus、全測試、vet與build均通過。
- 完成規格128：固定Hatari 400-VBL trace證實第二次Type-II `$80`仍等待75個50 Hz
  VBL，到VBL385才由`$0080/$D0`中斷；下一筆是dummy-seek `$0086` selector。Talos
  以獨立retry timeout收據於3,457,037 instructions／2,511 interrupts／130,386,416
  clocks完成，第一次timeout收據未被覆寫；下一gate為3,457,115／130,387,154。
  synthetic、固定ROM、完整CPU corpus、全測試、vet與build均通過。
- 完成規格129：固定Hatari VBL385 trace與EmuTOS `flopunlk/dummy_seek`證實第二次
  `$0086/$0000/$0080/$0013`、IRQ及status `$E4` read-clear。Talos重用既有
  728-FDC-clock scheduler，保存獨立第二組data／seek／九次poll／IRQ／read-clock
  收據，於3,457,357 instructions／2,511 interrupts／130,389,652 clocks完成。
  進一步固定ROM探測訂正下一gate：不是緊接的已支援status流程，而是3,516,206／
  130,971,490的第三次retry PSG write。完整CPU corpus、全測試、vet與build均通過。
- 完成規格130：依固定Hatari VBL389 trace接入第三次R14 `$25`同值重選、sector 1、
  DMA `$001004`、direction toggle、count 1與Type-II `$80`，以第三組獨立收據避免
  覆寫前兩輪。固定ROM於3,516,426 instructions／2,528 interrupts／130,973,792
  clocks完成command提交；下一gate為4,600,388／142,979,288的第三次timeout selector。
  synthetic、固定ROM、完整CPU corpus、全測試、vet與build均通過。
- 完成規格131：第三次Type-II `$80`沿guest既有`hz_200`期限再等待75個50 Hz VBL，
  以第三組獨立timeout收據接受`$0080/$D0`。固定ROM於4,600,435 instructions／
  2,903 interrupts／142,979,752 clocks完成，下一gate為4,600,513／142,980,490的
  第三次dummy-seek `$0086` selector。synthetic、固定ROM、完整CPU corpus、全測試、
  vet與build均通過。
- 完成規格132：第三次dummy seek重用728-FDC-clock scheduler，以第三組獨立收據記錄
  `$0086/$0000/$0080/$0013`、九次poll、IRQ與status `$E4` read-clear。固定ROM於
  4,600,755 instructions／2,903 interrupts／142,982,988 clocks完成；下一gate為
  4,601,570／142,994,602的YM2149 byte write。完整CPU corpus、全測試、vet與build通過。
- 啟動規格133第一階段：使用者確認以可重入循環取代全部固定軟碟stage／約50個逐輪
  欄位，不保留永久相容層。先建立單一phase列舉、完整同型receipt與容量8 ring；12輪
  synthetic wrap／lookup／reset、完整CPU corpus、全測試、vet與build均通過。正式FDC／
  PSG分支尚未遷移，規格維持READY，不宣稱循環已接線。
- 規格133第二階段前半：移除`Memory`內約50個分散的逐輪receipt欄位，將前三次已驗證
  收據機械遷入`[3]floppyMediaReceipt`；所有既有分支與測試改查同型欄位。固定ROM前三輪
  instructions／clock錨點完全不變，完整CPU corpus、全測試、vet與build均通過。這個
  `floppyMediaLegacy`陣列只是下一個可回退遷移點，尚未取代固定stage，規格維持READY。
- 規格133第二階段後半：接入第四輪起的單一可重入phase，完整涵蓋PSG、sector／DMA、
  Type-II read、guest timeout／`$D0`、dummy seek scheduler、九次poll與status read-clear。
  synthetic連續12輪驗證ring wrap與RAM不變；固定ROM自然完成第四、第五輪，於
  6,779,282 instructions／3,656 interrupts／167,143,396 clocks才抵達新IKBD gate。
  完整CPU corpus、全測試、vet與build通過；前三輪固定stage尚待遷移，規格維持READY。

- 新增 UCSD p-System 直譯器真實碼驗收（規格 134）。以 SunDog 的 `SYSTEM.INTERP`
  作為與合成語料互補的驗收來源：合成語料窮舉單一指令，這裡跑一段指令彼此有真實資料
  相依的程式。四組測試涵蓋分派表結構、短常數、區域變數與陣列索引，每條斷言都做過
  負對照（期望值差 1、指令數差 1、活動記錄偏移差 2 都確認會失敗）。`tools/go.sh`
  新增 `TALOS_UCSD_INTERP` 掛載，與既有的 `TALOS_M68000_TESTS` 可並存。原版素材
  不進 repository，雜湊在測試裡釘死。完整 227,500 筆語料回歸與靜態檢查通過。
- 接著完成分派迴圈（規格 135）。從單一常式提升到 fetch-execute 循環之後，可以執行任意
  由已驗收常式組成的 p-code 序列——這是把 p-code 當成對拍載體的前提。兩組序列測試
  （四條短常數、三條混合族）加三組負對照。同一輪查證後更正了 054 的論證基礎：原本
  引用「`SYSTEM.INTERP` 零個 trap」的外部結論，實際上有一條由旗標保護的除錯 hook；
  改成由 `SparseMemory` 的 fail-closed 自證，不依賴外部陳述。
- 擴充 135 的覆蓋：解出分派表的完整形狀（107 支常式、45 個無效 opcode、`$9C` 是 NOP），
  並把區域變數的存入與取址納入驗收。有了 `sstl` 之後可以做存取往返，期望值直接對記憶體
  查而不只對堆疊查，擋掉「兩支常式互相自洽卻都算錯位址」。負對照累計六組。
- 為 135 補上第二個實作的對拍。六組 p-code 另外餵給 laanwj/sundog 的獨立 C 直譯器，
  結果與 Atari Talos 執行原版 68000 直譯器逐字相同，期望值因此不再是單方面算出來的。
  那份程式碼的角色與 Hatari 相同——外部 oracle，不連結、不移植、不依賴。
- 完成算術族（規格 136）。p-code 序列因此能表達實際運算式，驗收用的是原版 `check_exit`
  的格座標換算——數值取自原版執行時的除錯器讀值，算出的欄 11、列 7 與當時讀到的格座標
  一致。三方對上：原版直譯器（由 Atari Talos 執行）、獨立 C 重寫、原版實跑。
- 056 再擴到分支與間接載入（`ujp`／`fjp`／`tjp`／`sind`／`ldb`），因此能表達原版的條件
  邏輯。驗收用 `check_exit` 的地面碼正規化「大於 15 就減 16」，邊界 15 與 16 都查。

- 完成布林族（規格 137）。`land`／`lor`／`bnot` 三支常式都是位元運算，配上 `fjp`／`tjp`
  的 `btst #0`，p-system 的真假整套住在 bit 0——8 是假、9 是真。驗收用 SunDog
  `XSTARTUP:0x31` 的初始損壞判斷式跑整張真值表。這解掉了 remake 專案掛著的一個矛盾：
  `(欄 = 0) or random()` 當成邏輯或加「非零即真」讀時幾乎永遠成立，與原版實測的分布
  不合；照位元或加 bit 0 讀，3/8 的比例與實測的 8/23 相符。負對照兩條。

- 完成 line-F emulator（規格 059）。MC68000 的 `$Fxxx` 整段保留給 coprocessor 介面而這顆
  CPU 沒有 coprocessor，所以那一段一條合法指令都沒有，全部走 vector 11——把整段路由過去
  不是拿例外掩蓋未實作，這一點與 051 只能路由 `$4E7A`／`$4E7B` 兩個 opcode 的情況不同。
- 合併回 main（2026-09-06）。這條線的 `MOVE.W` bus-error frame、`RESET`、cartridge port
  與 line-F 四片，main 在同一段期間各自做過而且走得更完整（cycle-aware bus、MFP、
  floppy 都疊在上面），所以合併時取 main 的實作，只留下這條線獨有的 UCSD p-System
  四片，規格重編號成 134–137。`tools/hatari-oracle/` 一併留下，`bus_error_address_test.go`
  的兩條 synthetic 負對照（sign-extend 與位址暫存器高位元）也留著——main 那邊沒有。
- 同步合併後的`main`至`e957113`並確認全測試通過；規格133繼續採可回復遷移。
  共用軟碟phase補齊第一輪才有的`$0082` track selector與track 0 data前綴，新增錯誤
  即關閉的狀態／mode測試，完整`go test ./...`與CLI建置通過。Go 1.24的`go vet ./...`
  會在既有`ReadByte`／`WriteByte`方法觸發標準介面簽章警告，屬同步後基線問題；正式
  入口與前三輪舊stage尚未切換，因此規格維持READY，不提前宣稱已完成統一。
- 修正同步後的Go 1.24 `go vet`基線：內部bus API的`ReadByte`／`WriteByte`因方法名
  與標準`io.ByteReader`／`io.ByteWriter`相同但簽章不同而觸發`stdmethods`；純重命名為
  `ReadByteFC`／`WriteByteFC`，名稱也明示額外的function code參數。全程未改bus值、
  clock、fault或交易次序；完整`go test ./...`、`go vet ./...`與CLI建置均通過。
