# Atari Talos 目前狀態

更新日期：2026-09-06。

## 已定案

- 專案名為 Atari Talos AI Toolkit；repo 為 `atari-talos-ai-toolkit`，CLI 為 `ataritalos`。
- 目標是讓 LLM、Codex、Claude Code、`go test` 與 shell 能決定性控制 Atari ST 原版，
  產生 remake 同狀態對拍證據。
- 第一階段為無頭 Atari ST／STF，不含 STE、TT、Falcon。
- Go library 與 CLI 共用協定型別；CLI 的穩定公開介面是 JSON Lines。
- Hatari 只作外部 oracle，不使用其 GPL 程式碼。
- 授權採 RRSAL-1.0；原版 TOS、磁碟與遊戲素材由使用者自備。
- CPU／machine／bus 採完整、漸進式 cycle-aware access：bus 在 access 前取得
  全機 epoch 加指令內 offset，動態 wait 會推移同一指令的後續 phase；不採事後補 clocks。

## 現況

- M0 控制契約已建立：`hello`、`capabilities`、`quit` 可用。
- `run_frames` 等需要機器核心的命令明確回傳 `not_implemented`。
- public repo 已建立：<https://github.com/wicanr2/atari-talos-ai-toolkit>，預設分支 `main`。
- Motorola 68000 的 NOP、MOVEQ、SWAP、EXT.W 與 EXT.L 已完成狀態、預取、clock
  與 program bus read 驗收；每份指令語料各 2,500 筆，外部單步驗收累計 12,500 筆。
- SWAP 與 EXT 族涵蓋暫存器轉換、不同運算寬度的 N／Z 判定及 X 保留；尚未延伸至
  68020 的 EXTB.L 或任何記憶體定址模式。
- Bcc／BRA 的 2,500 筆外部語料已全部通過：1,830 筆正常控制流涵蓋條件成立／不成立、
  byte／word 位移、預取與 clocks；670 筆奇數目標涵蓋 MC68000 address-error frame、
  supervisor 切換、vector 3 與 handler 預取。CPU 外部單步驗收累計 15,000 筆。
- BSR 與 RTS 各 2,500 筆全部通過，涵蓋 user／supervisor stack、return PC、正常預取，
  以及 push／pop 完成後才發生的奇數目標 address error；CPU 累計驗收 20,000 筆。
- JMP／JSR 各 2,500 筆全部通過，涵蓋七種 68000 control effective address、PC-relative、
  Dn／An word／long index、A7、absolute long、stack 次序與 address error；CPU 累計
  外部單步驗收 25,000 筆。
- LEA／PEA 各 2,500 筆全部通過；control EA 已可寫入 An／A7 或依真實 microcode
  排程推入 active stack。CPU 外部單步驗收累計 30,000 筆。
- 完整 `MOVE.B` 2,500 筆已全部通過：384 筆 Dn 目的端及 2,116 筆全部合法記憶體
  目的端，涵蓋所有 source EA、register alias、UDS／LDS lane、A7 byte delta、
  program／data FC、prefetch 與 write 排程。CPU 外部單步驗收累計 32,500 筆。
- 完整 `MOVE.W` 2,500 筆已全部通過：1,013 筆正常執行、839 筆來源讀取位址錯誤、
  648 筆目的寫入位址錯誤；涵蓋全合法 source／destination EA、An direct、16-bit
  CCR、資料 FC、vector 3 框架、saved PC 與 fault 微時序。CPU 累計驗收 35,000 筆。
- 完整 `MOVE.L` 2,500 筆已全部通過：1,013 筆正常執行、869 筆來源讀取位址錯誤、
  618 筆目的寫入位址錯誤；涵蓋分段 long bus access、predecrement 反向 word-write、
  fault-time CCR／saved PC／An 副作用與 absolute-long 特殊管線。CPU 累計驗收 37,500 筆。
- `MOVEA.W`／`MOVEA.L` 各 2,500 筆全部通過：word 正常 1,658／fault 842，long
  正常 1,655／fault 845；涵蓋符號延伸、全部 source EA、A0–A7、active stack、alias
  與不改 CCR。CPU 累計外部單步驗收 42,500 筆。
- `ADDA.W`／`ADDA.L` 各 2,500 筆全部通過：word 正常 1,683／fault 817，long
  正常 1,675／fault 825；涵蓋 word 符號延伸、32-bit 回繞、全部 source EA、A0–A7、
  active stack、alias、不改 CCR，以及 long memory source 的獨立 clock 規則。CPU
  累計外部單步驗收 47,500 筆。
- `AND.B/W/L` 與同語料內的 `ANDI.B/W/L` 共 7,500 筆全部通過；涵蓋 `<ea>→Dn`、
  `Dn→memory`、immediate、全部合法 EA、讀改寫 bus 次序、long low-word-first 寫回、
  CCR 與 address error。CPU 累計外部單步驗收 55,000 筆。
- `CMP.B/W/L`、`CMPI.B/W/L` 與 `CMPM.B/W/L` 共 7,500 筆全部通過；涵蓋不寫回的
  減法旗標、全部合法 EA、immediate、CMPM alias／雙後遞增，以及來源與目的 odd-address
  fault 的不同副作用與 saved PC。CPU 累計外部單步驗收 62,500 筆。
- `ADD.B/W/L`、`ADDI.B/W/L` 與 `ADDQ.B/W/L` 共 7,500 筆全部通過；涵蓋雙向
  ADD、immediate、quick immediate 0→8、ADDQ 寫 An 不改 CCR、讀改寫 bus 次序與
  vector-3 位址錯誤。CPU 累計外部單步驗收 70,000 筆。
- `CLR.B/W/L` 共 7,500 筆全部通過；涵蓋 Dn 與全部合法記憶體目的 EA、X 保留、
  固定 Z、MC68000 operand read／prefetch／zero write 次序與 vector-3 位址錯誤。
  CPU 累計外部單步驗收 77,500 筆。
- `MOVEM.W/L` 共 5,000 筆全部通過；涵蓋雙向 register mask、word 符號延伸、
  predecrement 反序、postincrement、額外虛讀、PC-relative program FC、完整 bus 次序與
  vector-3 位址錯誤。CPU 累計外部單步驗收 82,500 筆。
- `LINK` 與 `UNLK` 各 2,500 筆外部語料全部通過；UNLK 語料實際檔名為
  `UNLINK.json.bin`，涵蓋正常 1,385、odd-frame vector-3 1,115、A7 alias、
  user／supervisor active stack、完整 state／RAM／clock／bus。依原里程碑順序計入後，
  CPU 累計外部單步驗收 87,500 筆。
- `TST.B/W/L` 共 7,500 筆全部通過；涵蓋 Dn、全部合法 memory data EA、X 保留、
  NZVC、無寫回、program prefetch 與 vector-3 位址錯誤。CPU 累計外部單步驗收
  92,500 筆。
- `OR.B/W/L` 與同語料內的 `ORI.B/W/L` 共 7,500 筆全部通過；涵蓋雙方向 OR、
  immediate、全部合法 EA、讀改寫 bus 次序、long low-word-first 寫回、CCR 與
  address error。CPU 累計外部單步驗收 100,000 筆。
- `SUB.B/W/L`、`SUBI.B/W/L` 與 `SUBQ.B/W/L` 共 7,500 筆全部通過；涵蓋雙向
  SUB、immediate、quick immediate 0→8、SUBQ 寫 An 不改 CCR、borrow 旗標、
  讀改寫 bus 次序與 vector-3 位址錯誤。CPU 累計外部單步驗收 107,500 筆。
- `ASL.B/W/L` 共 7,500 筆全部通過；涵蓋立即值／Dn 低六位 count、零 count、
  byte／word／long 暫存器型、word 記憶體讀改寫、X／NZVC、動態 clocks 與 vector-3
  位址錯誤。CPU 累計外部單步驗收 115,000 筆。
- `ASR.B/W/L` 與 `LSR.B/W/L` 共 15,000 筆全部通過；涵蓋立即值／Dn 低六位
  count、零 count、算術符號填入／邏輯零填入、word 記憶體讀改寫、X／NZVC、
  動態 clocks 與 vector-3 位址錯誤。CPU 累計外部單步驗收 130,000 筆。
- `MULS.W` 與 `MULU.W` 共 5,000 筆全部通過；涵蓋 signed／unsigned 16×16→32、
  全部合法 word data EA、X 保留、NZVC、資料相依迭代 clocks 與 vector-3 位址錯誤。
  CPU 累計外部單步驗收 135,000 筆。
- `NOT.B/W/L` 與 `NEG.B/W/L` 共 15,000 筆全部通過；涵蓋 Dn、所有合法可修改
  memory EA、三種寬度、NOT 邏輯旗標、NEG borrow／overflow、讀改寫 bus 次序、
  long low-word-first 寫回與 vector-3 位址錯誤。CPU 累計外部單步驗收 150,000 筆。
- 完整 16 種 `Scc.B` 共 2,500 筆全部通過；涵蓋真假 condition、Dn 真／假不同
  clocks、所有合法可修改 memory EA、SR 不變、operand read、prefetch、byte write、
  UDS／LDS 與 A7 byte delta。CPU 累計外部單步驗收 152,500 筆。
- 完整 16 種 `DBcc.W` 共 2,500 筆全部通過；涵蓋 condition 成立、低 16-bit 計數
  到期／成功分支、10／12／14 clocks 與奇數目標 vector 3。例外不提交 Dn 遞減，
  fault address 與 frame saved PC 分開建模。CPU 累計外部單步驗收 155,000 筆。
- 補入先前漏列的 UNLK 外部語料 2,500 筆後，CPU 累計外部單步驗收為 157,500 筆；
  原限定完成狀態已撤除。
- `BTST`／`BCHG`／`BCLR`／`BSET` 四份語料共 10,000 筆全部通過；涵蓋 dynamic／
  immediate bit number、Dn modulo 32、byte memory modulo 8、Z、A7 byte delta、
  PC-relative program FC、讀改寫 bus 次序與 bit 16–31 的資料相依 clock。DM12EN
  靜態使用 96 點；CPU 累計外部單步驗收 167,500 筆。
- `DIVS.W`／`DIVU.W` 共 5,000 筆全部通過，涵蓋成功、quotient overflow、全部
  word data EA、資料相依迭代 clocks 與 1,936 筆 vector 3；另以 Hatari 各驗一個
  divisor=0 案例，兩者皆為 40 clocks、Z=1、Dn 不變及 6-byte vector 5 frame。
  DM12EN 靜態使用 92 點；CPU 累計外部單步驗收 172,500 筆。
- `TRAP`／`RTE` 共 5,000 筆全部通過：TRAP vectors 32–47 與 34-clock format-0
  exception；RTE 正常 600、奇數 PC vector 3 共 614、user privilege vector 8 共
  1,286 筆。涵蓋 SR mask、SSP 提交點、restored program FC、frame 與完整 bus 次序。
  DM12EN 靜態使用 14 點；CPU 累計外部單步驗收 177,500 筆。
- `EOR.B/W/L`／`EORI.B/W/L` 共 7,500 筆全部通過；涵蓋 Dn 與 memory destination、
  immediate、X／NZVC、read-modify-write、long low-word-first、EA clocks、完整 bus
  次序與 vector 3。DM12EN 靜態使用 35 點；CPU 累計外部單步驗收 185,000 筆。
- `LSL.B/W/L` 共 7,500 筆全部通過；涵蓋 immediate／Dn count、低六位截斷、
  零 count、X／NZVC、動態 clocks、word memory RMW、EA 與 vector 3。DM12EN
  靜態使用 16 點；CPU 累計外部單步驗收 192,500 筆。
- `ROR.B/W/L` 共 7,500 筆全部通過；涵蓋 immediate／Dn count、低六位截斷、
  零 count、X 保留、NZVC、動態 clocks、word memory RMW、EA 與 vector 3。
  DM12EN 靜態使用 10 點；CPU 累計外部單步驗收 200,000 筆。
- `SUBA.W/L`／`CMPA.W/L` 共 10,000 筆全部通過；涵蓋 word sign extension、
  全部 data source EA、CCR／X、clocks、完整 bus 次序與 vector 3，並修正 CMPA.L
  An-direct 與 CMPM mask 的解碼優先序。DM12EN 靜態使用 16 點；CPU 累計外部
  單步驗收 210,000 筆。
- `EXG` 共 2,500 筆全部通過；涵蓋 Dn↔Dn、An↔An、Dn↔An、A7 目前 stack bank、
  SR 不變、固定 6 clocks 與完整 prefetch bus。DM12EN 靜態使用 6 點；CPU 累計
  外部單步驗收 212,500 筆。
- `MOVE An,USP`／`MOVE USP,An` 共 5,000 筆全部通過；正常 supervisor 2,567 筆，
  user privilege vector 8 共 2,433 筆，涵蓋 A7 的 SSP bank、USP、指令起始 saved PC、
  format-0 frame 與完整 bus 次序。CPU 累計外部單步驗收 217,500 筆。
- `MOVE <ea>,CCR`／`MOVE <ea>,SR` 共 5,000 筆全部通過；涵蓋 CCR／SR mask、
  全部 word data EA、1,440 筆 vector 3、1,290 筆 user privilege vector 8、
  新 program FC 與 `PC-2` 管線重讀。CPU 累計外部單步驗收 222,500 筆。
- `MOVE SR,<ea>` 共 2,500 筆全部通過；涵蓋 Dn／全部可修改 word memory EA、
  user mode、SR 保留、目的舊值讀取、prefetch、寫回、EA clocks 與 978 筆 vector 3。
  CPU 累計外部單步驗收 225,000 筆。
- `TAS.B` 共 2,500 筆全部通過；涵蓋 Dn／全部可修改 byte memory EA、舊值旗標、
  bit 7 寫回、A7 delta 與 transaction。上游已知 memory timing 少 2 clocks，已由
  Hatari／Atari ST 兩次 `(A0)` 實測確認為 16 clocks 後局部勘誤；CPU 累計外部單步
  驗收 227,500 筆。pin-level 5-cycle RMW 波形仍未建模。
- ST／STF 基礎 memory map 已完成：可配置 512 KiB／1 MiB RAM、192 KiB TOS ROM、
  reset SSP／PC shadow、24-bit masking、低 2 KiB／I/O supervisor protection、ROM
  read-only 與 typed bus fault 均有測試。
- MC68000／ST power-on reset 已完成：以 FC=6 載入 SSP／PC 與 prefetch、
  staged failure 不提交、machine epoch counters 歸零；EmuTOS 1.3 真實 ROM
  第一條 `BRA.W` 後的 PC／prefetch／10 clocks 與 Hatari 一致。
- ST MMU `$FF8001` 已完成 cold-reset latch、supervisor byte R/W、兩個
  512 KiB 實體 bank 在 128 KiB／512 KiB／2 MiB 邏輯設定下的 STF
  位址轉換。Hatari trace 的 `$00→$0A→$05` 序列已收據；Atari Talos
  現可與 Hatari 同狀態到達 `$FC0070`（7 條指令、92 clocks）。
- MC68000 對 68010+ `$4E7A/$4E7B` `MOVEC` 已正確進 illegal-instruction
  vector 4；EmuTOS 與 Hatari 均以 8 條指令、128 clocks 到 `$FC0074`，
  format-0 frame、SSP／SR 與 handler prefetch 全同。繼續至 `$FC0088`
  時發現中間 vector 2 frame 的 fault address 高 byte 仍有
  `$FFFF8006`（Hatari）／`$00FF8006`（Atari Talos）差異。
- 上述 vector 2 fault address 已修正為保留 CPU 的 `$FFFF8006`，同時 bus
  transaction 維持 24-bit `$FF8006`；第 10 條／220 clocks 的完整 frame 已對拍。
- supervisor `RESET` 已實作 external reset hook、132 clocks 與 sequential
  prefetch；EmuTOS 第 11 條／352 clocks 與 Hatari 同狀態。新的第一失敗點是
  `$FC008A` `CMPI.L` 讀取 `$FA0000` cartridge window；Hatari 回 `$FFFFFFFF`，
  Atari Talos 目前回 unmapped bus fault。
- 空 cartridge `$FA0000–$FBFFFF` 已建立為 128 KiB `$FF` read-only window；
  EmuTOS 第 12 條／380 clocks 與 Hatari 同狀態。逐條向後比對後，第 14 條
  RAM write 首次出現 Hatari 26／Atari Talos 24 clocks。Hatari 固定原始碼將差異
  收窄為依全機與指令內 clock 對齊四 clock 的 ST CPU external bus slot，而非已證實的
  即時 Shifter 搶占；它仍不是該 MOVE opcode 的固定 timing。phase 0／2 相鄰探針已
  確認 24／26 clocks，`$21FC` 正常偶數 destination 切片已依規格 056 CONFORMED。
- cycle-aware Bus 首個 runtime 切片已接線：machine 將 64-bit 全機 epoch 傳入 CPU，
  `TimedBus` 在 access 前收到絕對 clock 並回傳 wait；NOP／MOVEQ／SWAP／EXT 共用的
  4-clock prefetch 已遷移。NOP 2,500 筆完整 phase timeline 全同，synthetic 2-clock
  wait 與 machine epoch 整合測試通過；`$21FC` 六 phases 已遷移，其他指令尚未遷移。
- 固定 EmuTOS 第 14 條現在由 390→416，並繼續與 Hatari 同 clocks 至第 18 條／496；
  line-F 前 PC=`$FC00C2`、prefetch=`$F010,$0800` 及 D0／A0／SSP／SR 均一致。
  `$FC00BE` line-F／vector 11 已完成：MC68000 核心 34 clocks，ST bus phase 加 2 clocks，
  第 19 條累計 532 clocks 進 `$FC00D4`；frame、SSP、SR 與 prefetch 對上 Hatari。
- line-F 外部語料 `ILLEGAL_LINEF.json.bin` 2,500 筆 state／RAM／clocks／bus transaction
  全同；新增 STOP 2,500 筆後，CPU 累計外部單步驗收 232,500 筆。
- 普通 ST／Ricoh `$FF860F` void byte read 與無 Mega-RTC 的 `$FFFC21–$FFFC3F`
  void byte range已依固定 Hatari／EmuTOS 原始碼與 tracepoint CONFORMED；read 回 `$FF`，
  RTC range byte write discard，且不取用主機 wall-clock。固定 EmuTOS 可成功完成
  6,916 條。
- `TST.B (An)` byte source typed bus fault／vector 2 已以 `$FF8A3C` Blitter 探測
  CONFORMED：64 clocks、byte UDS lane、SSW `$4A15`、fault address `$FFFF8A3C`、
  saved PC `$FC0638`、完整 frame 與 handler state 對上 Hatari。固定 EmuTOS 可成功
  完成 7,474 條，新第一停點是對 `$FFFA01` 的 MFP byte write。
- MFP GPIP `$FFFA01` reset-state byte write 已 CONFORMED：MC68901 一手規格確認
  DDR=0 為 input／high impedance，寫入只改 DDR=1 的 output bits；固定 Hatari trace
  確認 EmuTOS `$FC614A` 的 `MOVE.B #$00,(A0)` 為 16 clocks，GPIP／DDR 前後均為 `$00`。
  Atari Talos 現可完成 7,475 條／176,638 clocks，flags、prefetch、GPIP 與 Hatari
  一致；再三條後，第 7,479 條嘗試停在 `$FFFA03` AER write，未泛化其餘 MFP bank。
- MFP AER `$FFFA03` reset-state zero write 已 CONFORMED：官方手冊確認 reset=`$00`、
  bit 0/1 分別選 falling/rising edge，且改寫 AER 本身可能觸發 transition。因 pending
  interrupt 與 timer B 尚未建模，目前只接受 `$00→$00`；非零 write 明確回
  `unsupported_device_state`。固定 EmuTOS 現可完成 7,479 條／176,682 clocks，
  下一次未支援寫入為 `$FFFA05` DDR。
- MFP DDR `$FFFA05` reset-state zero write 已 CONFORMED：官方手冊確認 reset=`$00`
  代表八條 GPIP pin 均為 high-impedance input；因非零值會切換 pin drive 並重新評估
  interrupt，目前只接受 `$00→$00`，其他 write 回 `unsupported_device_state`。
  固定 EmuTOS 現可完成 7,483 條／176,726 clocks，下一未支援寫入為 `$FFFA07` IERA。
- MFP IERA／IERB `$FFFA07/$FFFA09` reset-state zero writes 已 CONFORMED：官方手冊
  確認 reset/disable=`$00`，寫 0 同時清相應 pending bit但不清 in-service。因 interrupt
  sources、pending 與 IRQ 尚未建模，目前只接受 `$00→$00`；非零 enable 回
  `unsupported_device_state`。固定 EmuTOS 現可完成 7,491 條／176,814 clocks，
  下一未支援寫入為 `$FFFA0B` IPRA。
- MFP IPRA／IPRB `$FFFA0B/$FFFA0D` software clear 已 CONFORMED：reset/read latch
  與 `pending &= value` 皆已接線，software 的 0 清除既有 pending、1 保留，不能
  將 cleared bit 設為 pending。固定 EmuTOS 現可完成 7,499 條／176,902 clocks；
  再完成三條控制指令後，第 7,503 次嘗試在 `$FFFA0F` ISRA 失敗即關閉。
- MFP ISRA／ISRB `$FFFA0F/$FFFA11` software clear 已 CONFORMED：NXP 手冊確認
  automatic／software EOI 關係及只能寫 0 清除；目前只接 reset/read latch 與
  `in_service &= value`，未冒稱 IACK、priority 或 Vector Register 已完成。固定
  EmuTOS 現可完成 7,507 條／176,990 clocks；再完成三條控制指令後，第 7,511
  次嘗試在 `$FFFA13` IMRA 失敗即關閉。
- MFP IMRA／IMRB `$FFFA13/$FFFA15` mask latch 已 CONFORMED：reset/read 與無 pending
  狀態下的完整 byte write 已接線；因 Talos 尚未有 IRQ 重新評估，對應 pending
  非零時的 mask write 明確失敗且不改 state。固定 EmuTOS 現可完成 7,515 條／
  177,078 clocks；再完成三條控制指令後，第 7,519 次嘗試在 `$FFFA17` Vector
  Register 失敗即關閉。
- MFP Vector Register `$FFFA17` 已 CONFORMED：依 NXP 一手規格只保存 `$F8`
  mask，bits 2–0 讀回 0；vector base、software／automatic EOI 及切回 automatic
  清 ISRA／ISRB 已接。pending 非零且切 automatic 時因 IRQ 尚未建模而原子失敗。
  固定 EmuTOS 現可完成 7,519 條／177,122 clocks；再完成三條控制指令後，
  第 7,523 次嘗試在 `$FFFA19` Timer A Control Register 失敗即關閉。
- MFP TACR／TBCR／TCDCR `$FFFA19/$FFFA1B/$FFFA1D` reset-stop 已 CONFORMED：
  只接 reset/read 與 `$00→$00` 停止寫入；所有會啟動 prescaler、counter、output
  或 interrupt 的非零 control 都明確失敗。固定 EmuTOS 現可完成 7,531 條／
  177,254 clocks；再完成三條控制指令後，第 7,535 次嘗試在 `$FFFA1F` Timer A
  Data Register 失敗即關閉。
- MFP TADR／TBDR／TCDR／TDDR `$FFFA1F/$FFFA21/$FFFA23/$FFFA25` stopped-load 已
  CONFORMED：timer 停止時任意 byte 同步載入 data register 與 main counter；active
  timer 的捕捉、延後 reload 與臨界不定值尚未建模，明確失敗。固定 EmuTOS 現可完成
  7,547 條／177,430 clocks；再完成三條控制指令後，第 7,551 次嘗試在 `$FFFA27`
  Synchronous Character Register 形成本規格當時的驗收停止點；後續由規格 071 接管。
- MFP Timer C delay-mode 啟動已 CONFORMED：固定 EmuTOS 以 TCDR/main=`$C0`、
  TCDCR `$00→$50` 啟動 control 5（÷64），Hatari trace 為 12,288 MFP ticks，依固定
  8,021,248／2,457,600 Hz 比例是 40,106 CPU clocks。此切片只 latch start transition，
  countdown／timeout／IRQ 尚未接線；正常路徑已前進到 68,103 instructions、3 interrupts、
  963,104 clocks，下一 gate 是 `$FC6192` 的 memory `ROL.W` opcode `$E378`。
- MC68000 ROL.B／W／L 已 CONFORMED：三份固定外部語料 7,500／7,500 通過，完整
  corpus 累計 240,000 筆。固定 EmuTOS `$FC6192: E378 1238` 以 16 clocks 執行；
  正常路徑再前進到 68,131 instructions、3 interrupts、963,388 clocks，下一 gate
  是 Timer C channel 的 `$FFFA09` IERB 非零寫入。
- MFP Timer C interrupt enable 已 CONFORMED：固定路徑只在 TCDCR=`$50`、IPRB=0
  時允許 IERB bit 5 `$00→$20`，後續既有 IMRB latch 寫成 `$20`；尚未產生 timeout
  或 IRQ。正常路徑跨過第四次 VBL，抵達 68,378 instructions、4 interrupts、
  966,808 clocks；下一 gate 是 Timer D 啟動的 TCDCR `$50→$51`。
- MFP Timer D delay-mode 啟動已 CONFORMED：固定序列是 TCDCR `$50→$50` stop-D、
  TDDR/main=`$02`、TCDCR `$50→$51` start-D（control 1、÷4）；26-clock recurrence
  尚未接 scheduler。正常路徑抵達 68,392 instructions、966,948 clocks。
- MFP USART fixed enable與 interrupt channels 已 CONFORMED：UCR／RSR／TSR=
  `$88/$01/$01`，RBF/TBE 將 IERA／IMRA 依序收斂到 `$14/$14`，沒有憑空產生 pending。
- YM2149 固定 boot ports 已 CONFORMED：Hatari trace確認 select 7／data `$C0`／select 14／
  data `$07`；Talos最後 selected/R7/R14=`$0E/$C0/$07`。一般 MOVE.B timed-I/O 尚未接線，
  故不宣稱四筆 cycle parity。正常路徑抵達 68,528 instructions、4 interrupts、
  968,510 clocks，下一 gate 是 ACIA `$FFFC00`。
- IKBD MC6850 ACIA control 與第一 transmit deadline 已 CONFORMED：固定序列 `$03→$96`
  建立 configured status `$02`，首筆 TDR=`$80` 清 TDRE，第一個 1024-clock deadline 將
  TDR 移入 shift stage 並恢復 TDRE。排程以指令邊界作可重現近似，不宣稱 Hatari device
  write phase 的逐 cycle parity。正常路徑抵達 68,645 instructions、4 interrupts、
  969,640 clocks，下一 gate 是第二個 IKBD command byte `$01` 寫入 `$FFFC02`。
- IKBD第二 TX buffer與warm-reset response已CONFORMED：`$01`等待前一個8N1 frame的
  10個serial ticks後，於device clock 979,806移入shift stage；完整第二 frame後依固定
  color-ST profile的513,024-clock reset delay送回RDR=`$F1`。接入Timer C後guest在
  128,378 instructions／21 interrupts／1,509,022 clocks讀取status `$83`與RDR，read後
  status回 `$02`；正常路徑在136,125／23／1,579,268完成MIDI ACIA `$03→$95`。
- MIDI ACIA control、IKBD stale RDR與MFP ACIA channel 6已CONFORMED：Hatari確認
  `$FFFC04`固定序列為`$03→$95`；IKBD RDRF清除後，`$FC06CE`仍讀到一次保留值`$F1`
  而不產生新RX。MFP依序保留IERB/IMRB=`$20`、以`$BF`清IPRB/ISRB bit6，再升為
  `$60/$60`。正常路徑抵達136,236 instructions、23 interrupts、1,580,634 clocks，
  下一gate是channel 4／Timer D序列的IERB同值`$60`。
- MFP Timer D系統時鐘重設已CONFORMED：channel 4序列以`$EF`清IPRB/ISRB，停止
  TCDCR `$51→$50`，把TDDR/main由`$02`重載為`$00`（typed語意256），IERB/IMRB升至
  `$70/$70`後以TCDCR=`$52`啟動control 2（÷10）。接入Timer C後正常路徑在
  136,285 instructions、23 interrupts、1,581,256 clocks抵達啟動邊界；Timer D
  recurrence與MFP IACK已由規格098接線。
- MFP SCR／UCR／RSR／TSR `$FFFA27/$FFFA29/$FFFA2B/$FFFA2D` reset write 已
  CONFORMED：依 NXP 手冊保留 TSR 硬體 reset 未定的事實，只接受固定 EmuTOS 的
  軟體 `$00` 初始化；非零 USART 狀態與 UDR 仍失敗即關閉。四次 MOVE 各 16 clocks，
  第 7,563 條／177,606 clocks 的 state／prefetch 對上 Hatari。這是 USART 切片當時的
  停止點；接入規格 075 的第一 VBL 後，現行後續 gate 改由下列 STOP 項目描述。
- MC68000 `STOP` 已 CONFORMED：執行前 supervisor 判權、immediate SR `$A71F`
  masking、4 clocks、stopped latch、重複 Step 原子停止與 CPU Reset 清除均已接線；
  2,500 筆固定外部語料全過。固定 EmuTOS 的 opcode 位址是 `$FCD09A`，Talos pipeline
  PC `$FCD09E` 指向下一指令；規格 075／076 接入 VBL 與 IACK 後，第 7,604 條／178,244 clocks
  再次以 SR=`$2300` 停機。Hatari 同點 opcode／SR／prefetch 一致，D2=`$2710`、
  D3=`$1` 也已收斂；舊 7,599／178,096 與 D3=0 收據已被規格 075 取代。
- 固定 color ST profile 的 MFP GPIP input sample 已 CONFORMED：bus read 依 DDR 合併
  `$A1`（color monitor bit 7、FDC idle bit 5、no-printer busy bit 0），修正 `$FC67B8`
  monitor detection。STOP 前 D2 已由 `$2704` 收斂為 Hatari `$2710`。
- MC68000 level 4 autovector 接受已 CONFORMED：外部仲裁後的 CPU API 依 SR mask 決定
  接受，使用 vector 28、建立 6-byte frame、切 mask 4、44 clocks 並解除 STOP。固定
  Hatari 在第二個 VBL 進 `$FC0446`，frame 為 `$2300,$00FC,$D09E`，與 Talos typed
  測試一致。第一個 GLUE VBL 已由規格 075 接線。
- ST reset frame 第一個 GLUE VBL 已 CONFORMED：cold color ST 在 133,668 clocks
  raise pending，mask 7 期間保留，於 `$FC6904` 前 mask 3 接受。`--fast-boot false`
  的 Hatari 與 Talos 在 `$FC0446` handler 入口 D/A、SSP、SR、frame、prefetch 全同；
  guest opcode 真正令 `$466 frclock` 從 0 變 1，沒有由 host 直接寫記憶體。規格 076
  已補上 reset 60 Hz recurring deadline、E-clock／video IACK 與 STOP 快轉；第二次 handler
  在 267,332 clocks 完整 state／frame 對上 Hatari，guest 將 `$466` 由 1 增至 2。
  有界續跑跨過第三次 VBL 後，規格 077 已接 `$FF8260` reset／low-resolution 同值寫入：
  `$FC69E6` 以 12 clocks 完成且完整 state 對上 Hatari。再續跑至 7,662 instructions／
  3 interrupts／401,270 clocks 抵達 `$FF820A` video sync byte write；規格 078 已接
  60→50 Hz transition，12 clocks 後把第四 VBL deadline 從 534,480 修正為 Hatari 的
  535,528，之後 period 為 160,256。規格 079 已接 `$FF8240–$FF825E` 16 色 word bank；
  EmuTOS `$FC671A` 首筆 8 clocks，完整迴圈在 7,749 instructions／402,052 clocks 結束，
  palette、A0/A1、D1 與 Hatari 相同。規格 080 已接 `$FF8201/$FF8203` 程式化
  framebuffer base：Talos 在 7,896 instructions／403,900 clocks 進入，依序以
  12／24／12 clocks 完成 high write、`LSR.L #8` 與 middle write，最後為 `$0F8000`；
  Hatari 同段為 403,924→403,972，且 active `VideoBase` 在兩次寫後仍為 0。
  規格 081 進一步確認 transition frame 因 HBL 262 即結束，未到 50 Hz 的 HBL 310；
  Hatari 在 535,524 active仍為 0，跨共同 VBL deadline 535,528 後於 535,532 成為
  `$0F8000`。Talos 既有 24-clock 累積差距使相鄰 guest boundaries 為
  535,520→535,530，但同一 deadline 已提交 programmed→active，差異未被掩飾。
  規格 082 已接 320×200 low-res 4-plane big-endian decoder與 active-base DMA snapshot；
  base 0 明確讀 RAM而非 CPU reset ROM shadow。Hatari VBL7 raw dump SHA-256
  `98dcbfd3…a0570f` 解成 index SHA-256 `6157070b…10444`，分布為 color 0=63,679、
  color 15=321、第一個非零像素 `(1,0)`；Talos 正常 VBL4 snapshot為全黑且 palette一致。
- CPU vector 2 已完成第一條 Hatari 對拍切片：`MOVE.W` absolute-long user word source
  讀取低記憶體保護區時，72 clocks 後進入 handler；SSW、fault address、opcode、原 SR、
  saved PC、14-byte frame、supervisor 切換與預取皆有整合測試。其他 bus-error 讀寫路徑、
  實際 I/O 裝置與 TOS 後續開機仍未完成。
- MFP Timer D 系統時鐘週期與 channel 4 向量中斷已 CONFORMED：依 MC68901 固定
  2,560 MFP clocks，自動 reload 並設定 IPRB bit 4；IERB／IMRB 仲裁後以 level 6、
  vector 68 進 `$FC7884`，承認時 pending 轉入 software-EOI in-service。固定 EmuTOS
  接入Timer C後在137,213 instructions／24 interrupts／1,589,660 clocks抵達handler，guest自行
  寫 ISRB=`$EF` 清除。因該 `MOVE.B` path 尚未供應 timed access，啟動 phase 暫採
  instruction boundary 的 hardware-spec approximation，未宣稱逐 cycle parity。
- MFP Timer C 200 Hz週期與channel 5向量中斷已CONFORMED：timed start phase=962,844，
  12,288 MFP clocks的第一個deadline=1,002,950；Talos在72,342 instructions、
  5 interrupts、1,003,004 clocks自然進入vector 69／`$FC04DE`，pending轉入software-EOI
  in-service，guest由`$FC050A`寫`$DF`清除。B-bank仲裁在Timer C／D同時pending時選
  較高的channel 5，且不越過相同或更高的in-service channel。
- MFP Timer D正常停止與channel 4清除已CONFORMED：固定EmuTOS依序將IERB／IMRB
  `$70→$60`、TCDCR `$52→$50`，更新vector 68 table為`$00FC03EA`，再由共用
  `mfpint`做IMRB／IERB同值`$60`與IPRB／ISRB `$EF` clear。Talos在289,256
  instructions／234 interrupts／2,978,730 clocks完成；Timer D running、phase、scheduler
  與deadline均清除，Timer C pending可跨同值mask write保留。
- MFP USART第二次設定與baud Timer D重啟已CONFORMED：TSR讀取依硬體契約加入
  transmit-buffer-empty bit `$80`；固定EmuTOS完成TCDCR `$50→$50`、TDDR=`$02`、
  TCDCR=`$51`與UCR／RSR／TSR／SCR同值重寫。完成點為289,342 instructions、
  234 interrupts、2,979,680 clocks；control1不會誤啟動system channel 4 scheduler。
  後續`$FF860F` ordinary-ST void byte write亦已CONFORMED：任意byte忽略、不建立
  Falcon／FDC狀態，完成點289,521 instructions／2,982,760 clocks。
- YM2149 port A首次drive-select更新已CONFORMED：固定EmuTOS同值選R14、讀回`$07`、
  寫成`$05`；完成點289,556 instructions／2,983,132 clocks。
- ST DMA／WD1772第一組初始化已CONFORMED：`$FF8606=$0080`選command/status，
  `$FF8604=$00D0`執行condition 0 force-interrupt；保存mode／command／Type-I motor-on
  status `$80`，IRQ與GPIP5維持inactive。接入word-write bus phase後，完成點為
  289,612 instructions／2,983,704 clocks。
- ST WD1772 restore與IRQ期限已CONFORMED：第二次mode `$0080`後接受`$0B`，以
  `floor(728*8021248/8000000)=729` CPU clocks排程；status由`$81`完成為`$84`，
  GPIP5由inactive high轉active low。固定EmuTOS自clock 2,984,902起算，完成九次
  inactive輪詢後於289,803 instructions／2,985,654 clocks實際讀到`$91`。
- ST WD1772 Type-I status read-clear已CONFORMED：EmuTOS同值重寫mode `$0080`後，
  於clock 2,986,242讀`$FF8604`；固定無磁片profile回`$00E4`，再清IRQ並將GPIP5
  恢復inactive high。完成點289,865 instructions／2,986,256 clocks；下一gate為
  289,982 instructions／2,987,452 clocks的`$FF8606=$0086` data-register selector。
- ST WD1772 same-track seek已CONFORMED：第一顆drive依序寫mode `$0086`、data `$0000`、
  mode `$0080`、command `$0013`；motor已開、TR=DR=head track 0、verify off，故沿固定
  729 CPU-clock Type-I期限完成。Talos自2,988,614起算，九次inactive poll後由
  2,989,930的status read回`$E4`並清IRQ；完成點290,223 instructions／2,989,944
  clocks。下一gate為290,296／2,990,830的YM2149 `$FF8800` byte write，開始切至drive 1。
- YM2149 port A切至drive 1已CONFORMED：固定EmuTOS再次同值選R14、讀`$05`、寫
  `$03`；Talos於290,303 instructions／2,990,890 clocks完成，R14=`$03`。下一gate
  為290,312／2,990,998的第二顆drive `$FF8606=$0080`，將重新執行FDC探測鏈。
- WD1772 drive 1探測鏈已CONFORMED：Talos保留明確probe drive身分，切換時清除
  drive 0的每顆drive收據，再重用force-interrupt／restore／status／data／seek
  完整stage1→14。固定ROM於290,970 instructions／2,997,708 clocks完成，
  restore與seek各九次inactive poll，兩次status `$E4`均清IRQ。下一gate是
  291,291 instructions／3,001,516 clocks的`$FF860D` supervisor byte write。
- ST floppy／ACSI DMA位址暫存器已CONFORMED：`$FF8609/$FF860B/$FF860D`
  實作byte R/W、22-bit high mask、even-address mask與ST ripple carry。固定EmuTOS
  依low→middle→high寫成`$001004`，Talos於291,294 instructions／
  3,001,576 clocks完成。下一gate是291,343 instructions／3,002,130 clocks的
  `$FF8606=$0190`；固定Hatari trace顯示它會reset DMA。
- ST DMA toggle reset與sector-count zero已CONFORMED：固定EmuTOS依次寫
  `$0190→$0090→$FF8604=$0000`；Talos依bit 8兩次toggle清sector count並保存
  reset收據，於291,376 instructions／3,002,468 clocks完成。下一gate是
  291,386 instructions／3,002,576 clocks的`$FF8606=$0088` ACSI command mode。
- 空ACSI bus target 0探測已CONFORMED：Talos完成`$0088→data $0000→$008A`，
  依Hatari契約不設IRQ；guest於clock 3,771,064由Timer C／`hz_200`自然timeout後
  寫`$0080`。該同值mode已與drive-1 FDC初次probe明確分流，FDC保持stage14。
  第二次DMA setup後，下一gate是361,268 instructions／4,062,736 clocks的
  target-1 `$FF8606=$0088`。
- 空ACSI bus target 1–7掃描已CONFORMED：八筆command為
  `$00/$20/$40/$60/$80/$A0/$C0/$E0`，各自有typed timeout clock收據，期間
  IRQ保持inactive且FDC維持stage14。target 7於866,723 instructions／461 interrupts／
  11,591,284 clocks完成；下一gate是867,255 instructions／11,598,096 clocks的
  YM2149 `$FF8800` byte write。
- YM2149 parallel-port strobe初始化已CONFORMED：固定EmuTOS依
  `$0E→read $03→write $23`將port A bit 5設為1，既有drive／side low三位保持不變。
  Talos於867,260 instructions／462 interrupts／11,598,144 clocks完成；下一gate是
  867,320 instructions／11,599,192 clocks的IKBD ACIA data `$FFFC02` byte write。
- IKBD clock request與response已CONFORMED：Talos於typed device clock 11,609,950
  完成`$1C` frame，從11,626,334起依固定16／10 serial-tick期限逐筆回
  `$FC,$00,$00,$00,$00,$00,$01`；每筆經MFP channel 6 vector `$46`由EmuTOS正常
  handler消費。七筆於874,579 instructions／471 interrupts／11,688,070 clocks收齊；
  下一gate是874,900／11,691,528的IKBD set-clock `$FFFC02=$1B` write。
- IKBD set-clock與clock readback已CONFORMED：固定ROM寫入
  `$1B,$24,$03,$17,$00,$00,$00`，第七筆於11,763,550完成並在同一serial deadline
  載入預先buffer的第二個`$1C`；request於11,773,790完成。readback
  `[FC,24,03,17,00,00,00]`於889,609 instructions／483 interrupts／11,851,910 clocks
  全數由EmuTOS消費。下一gate為1,005,202／521／13,036,392的YM2149 control write；
  先前依D0誤記為`$05`，固定Hatari bus trace已訂正實際值為register 14的`$0E`。
- ST `flopvbl()` drive-0媒體輪詢已CONFORMED：固定EmuTOS在VBL66將YM2149 port A
  `$23→$25`暫選drive 0，以DMA mode `$0080`讀WD1772 status `$E4`，再恢復`$23`
  （drive 1）。Talos於1,005,296 instructions／521 interrupts／13,037,306 clocks完成；
  status讀取clock為13,036,978。下一gate為1,085,703／548／13,927,048的IKBD
  `$FFFC02=$1C`；Hatari VBL77確認這是可重入的第三次讀時鐘請求，回傳值仍為
  `$FC,$24,$03,$17,$00,$00,$00`。
- IKBD可重入讀時鐘週期已CONFORMED：第三輪`$1C` frame於13,937,502完成，固定profile
  七筆回應於14,015,326送達完畢，EmuTOS於1,092,926 instructions／558 interrupts／
  14,015,626 clocks收齊。實作以單調request／response counters與每輪獨立receipt取代
  一次性特例，synthetic已連續跑過兩輪。下一gate為1,120,640／568／14,318,580的
  `$FF8800=$0E`；Hatari VBL90證實這是port A保持`$23`的下一輪`flopvbl()`檢查。
- `flopvbl()`雙磁碟機媒體檢查已CONFORMED：以單調count輪替drive `0,1,0,1`，每輪
  保存drive與status clock，並恢復原port `$23`。第二輪於1,120,734 instructions／
  14,319,494 clocks完成；固定ROM正常路徑其後連續完成至第73輪，才在1,285,863／
  1,761／106,337,672抵達新的`$FF8606` word write。該gate發生時media stage已是8，
  因此屬另一種FDC transaction，尚待原版trace確認。
- floppy媒體確認讀取的lock／track設定已CONFORMED：固定EmuTOS `flop_mediach()`進入
  `flopio()`後，先以`$0082`選WD1772 track register、寫track 0，再將YM2149 port A
  `$23→$25`選drive 0。Talos於1,286,016 instructions／1,761 interrupts／106,339,274
  clocks完成，track data bus clock為106,338,122；下一gate已由Hatari確認為
  `$FF8606=$0084` sector-register selector。
- 尚未實作完整 68000 或 Atari ST 周邊硬體，不宣稱可開機或執行遊戲。

## 下一步

1. 依已驗證的 pipeline／bus 模型，逐組擴充 Dungeon Master 實際需要的 68000 opcode；
   每組先寫 READY 規格。
2. 依 Dungeon Master DM12EN 產生組語的靜態使用次數選下一批，優先補齊仍缺的
   高頻指令族，並維持完整固定語料驗收。
3. 另建 Hatari 外部 oracle 收據格式，不讓 Hatari 成為 library dependency。
4. 為解鎖第一張 Talos 非黑正常路徑畫面，下一步接入固定Hatari已確認的sector 1、
   DMA read setup與Type-II `$80`空磁碟機timeout序列；不可把無磁片當成成功讀取。
   RGB／PNG
   色階契約與正常50 Hz HBL310提前重載仍須各自READY，不得由palette index或
   VBL 保底提交外推。
