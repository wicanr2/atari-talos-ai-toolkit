# Atari Talos 工作清單

| 項目 | 狀態 | 完成條件 |
|---|---|---|
| M0 專案與控制契約 | **完成** | public repo、Docker 全測試、JSONL golden test |
| 68000 語料盤點 | **完成** | 固定來源、授權、版本、SHA-256 與測試載入器 |
| 68000 NOP 規格 | **CONFORMED** | 2,500 筆狀態、預取、clock 與 bus transaction 全同 |
| 68000 MOVEQ 規格 | **CONFORMED** | 2,500 筆暫存器、CCR、預取、clock 與 bus transaction 全同 |
| 68000 SWAP／EXT 規格 | **CONFORMED** | 7,500 筆暫存器、CCR、預取、clock 與 bus transaction 全同 |
| 68000 Bcc／BRA 正常控制流 | **CONFORMED** | 1,830 筆正常案例全同 |
| 68000 address error | **CONFORMED** | 14-byte frame 與 670 筆 Bcc address-error 語料全同 |
| 68000 BSR | **CONFORMED** | 2,500 筆正常／例外 call 與 stack transaction 全同 |
| 68000 RTS | **CONFORMED** | 2,500 筆正常／例外 return 與 stack transaction 全同 |
| 68000 JMP／JSR control EA | **CONFORMED** | 七種 mode、5,000 筆正常／例外狀態與 bus 全同 |
| 68000 LEA／PEA | **CONFORMED** | 七種 control mode、5,000 筆 register／stack／bus 全同 |
| 68000 MOVE.B | **CONFORMED** | 全合法 source／destination EA、2,500 筆 state／RAM／clock／bus 全同 |
| 68000 MOVE.W | **CONFORMED** | 正常 1,013、read fault 839、write fault 648；共 2,500 筆全同 |
| 68000 MOVE.L | **CONFORMED** | 正常 1,013、read fault 869、write fault 618；共 2,500 筆全同 |
| 68000 MOVEA.W／MOVEA.L | **CONFORMED** | 兩種寬度共 5,000 筆正常／read fault 全同 |
| 68000 ADDA.W／ADDA.L | **CONFORMED** | 兩種寬度共 5,000 筆正常／read fault、An 結果與 clock 全同 |
| 68000 AND／ANDI.B／W／L | **CONFORMED** | 三種寬度共 7,500 筆雙方向、RMW、CCR、fault 與 bus 全同 |
| 68000 CMP／CMPI／CMPM.B／W／L | **CONFORMED** | 三種寬度共 7,500 筆完整 state／RAM／clock／bus／fault 全同 |
| 68000 ADD／ADDI／ADDQ.B／W／L | **CONFORMED** | 三種寬度共 7,500 筆完整 state／RAM／clock／bus／fault 全同 |
| 68000 CLR.B／W／L | **CONFORMED** | 三種寬度共 7,500 筆 Dn／memory、CCR、RMW 與 fault 全同 |
| 68000 MOVEM.W／L | **CONFORMED** | 兩種寬度共 5,000 筆 mask、雙方向、虛讀、bus 與 fault 全同 |
| 68000 LINK／UNLK | **CONFORMED** | 兩族共 5,000 筆；UNLK 正常／A7 alias／odd-frame vector 3 全同 |
| 68000 TST.B／W／L | **CONFORMED** | 三種寬度共 7,500 筆 Dn／memory EA、CCR、無寫回、bus 與 fault 全同 |
| 68000 OR／ORI.B／W／L | **CONFORMED** | 三種寬度共 7,500 筆雙方向、immediate、RMW、CCR、fault 與 bus 全同 |
| 68000 SUB／SUBI／SUBQ.B／W／L | **CONFORMED** | 三種寬度共 7,500 筆雙方向、immediate、quick、borrow、RMW、fault 與 bus 全同 |
| 68000 ASL.B／W／L | **CONFORMED** | 三種寬度共 7,500 筆 immediate／Dn count、X／NZVC、動態 clock、RMW 與 fault 全同 |
| 68000 ASR／LSR.B／W／L | **CONFORMED** | 六份語料共 15,000 筆 count、符號／零填入、X／NZVC、RMW 與 fault 全同 |
| 68000 MULS／MULU.W | **CONFORMED** | 兩份語料共 5,000 筆 signed／unsigned、資料相依 clock、EA 與 fault 全同 |
| 68000 NOT／NEG.B／W／L | **CONFORMED** | 六份語料共 15,000 筆 Dn／memory、旗標、RMW、long word 次序與 fault 全同 |
| 68000 Scc.B | **CONFORMED** | 16 conditions、Dn／memory、真假 clock、SR、RAM 與 bus 共 2,500 筆全同 |
| 68000 DBcc.W | **CONFORMED** | 16 conditions、計數器、三條正常時序與奇數目標 vector 3 共 2,500 筆全同 |
| 68000 BTST／BCHG／BCLR／BSET | **CONFORMED** | dynamic／immediate bit number、Dn／byte memory、Z、clock 與 bus 共 10,000 筆全同 |
| 68000 DIVS／DIVU.W | **CONFORMED** | 成功／overflow／address error 共 5,000 筆全同；Hatari divisor=0 vector 5 兩案通過 |
| 68000 TRAP／RTE | **CONFORMED** | vectors 32–47、format-0 return、privilege、odd target 共 5,000 筆全同 |
| 68000 EOR／EORI.B／W／L | **CONFORMED** | 三種寬度、Dn／memory destination、immediate、RMW、fault 與 bus 共 7,500 筆全同 |
| 68000 LSL.B／W／L | **CONFORMED** | immediate／Dn count、X／NZVC、動態 clock、memory RMW 與 fault 共 7,500 筆全同 |
| 68000 ROR.B／W／L | **CONFORMED** | immediate／Dn count、X 保留、NZVC、動態 clock、memory RMW 與 fault 共 7,500 筆全同 |
| 68000 SUBA／CMPA.W／L | **CONFORMED** | word sign extension、全部 source EA、CCR、clock、fault 與 bus 共 10,000 筆全同 |
| 68000 EXG | **CONFORMED** | Dn↔Dn、An↔An、Dn↔An、A7 bank、SR 與固定 6 clocks 共 2,500 筆全同 |
| 68000 MOVE USP | **CONFORMED** | 雙方向、A7 bank、supervisor 正常與 user privilege vector 8 共 5,000 筆全同 |
| 68000 MOVE to CCR／SR | **CONFORMED** | masks、word data EA、privilege、pipeline reread、vector 3 與 bus 共 5,000 筆全同 |
| 68000 MOVE from SR | **CONFORMED** | Dn／word memory destination、user mode、RMW bus、clock 與 vector 3 共 2,500 筆全同 |
| 68000 TAS.B | **CONFORMED** | Dn／byte memory、舊值旗標、寫回、EA 與 Hatari 修正 timing 共 2,500 筆全同；pin-level RMW 待 ST bus |
| 其餘 68000 指令 | 進行中 | 每組 READY 後實作，逐組通過外部語料 |
| ST／STF 基礎 memory map | **CONFORMED** | 512 KiB／1 MiB RAM、reset shadow、192 KiB TOS ROM、保護與 typed bus fault 測試通過 |
| MC68000／ST power-on reset／EmuTOS 首指令 | **CONFORMED** | 真實 ROM 的 SSP／PC／prefetch 與首條 `BRA.W` 10 clocks 均對 Hatari 一致 |
| ST MMU `$FF8001`／512 KiB bank translation | **CONFORMED** | cold `$00`、`$0A→$05` trace、512 KiB／1 MiB topology 與 EmuTOS `$FC0070` 邊界同狀態 |
| 68000 `MOVEC` illegal instruction／vector 4 | **CONFORMED** | `$4E7A/$4E7B`、36 clocks；EmuTOS 以 8 條／128 clocks 到 `$FC0074`，frame／state 全同 |
| vector 2 absolute-short fault address | **CONFORMED** | frame 保存 `$FFFF8006`、bus 保存 `$FF8006`；第 10 條／220 clocks 完整對拍 |
| MC68000 `RESET` | **CONFORMED** | privilege、external hook、132 clocks；EmuTOS 第 11 條／352 clocks 同狀態 |
| ST 空 cartridge `$FA0000–$FBFFFF` | **CONFORMED** | 128 KiB `$FF` read-only window；EmuTOS 第 12 條／380 clocks 同狀態 |
| CPU／machine cycle-aware bus 契約 | **READY／進行中** | timeline、machine epoch、TimedBus 與首個 4-clock prefetch 已接線；NOP 2,500 筆 phase 全同，其餘指令待逐族遷移 |
| ST CPU external bus slot alignment | **CONFORMED（首切片）** | `$21FC` 正常偶數 destination 六 phases、phase 0／2=24／26、EmuTOS 390→416；其他指令逐族遷移 |
| MC68000 line-F／vector 11 | **CONFORMED** | `$Fxxx` 2,500 筆全同；核心 34／ST 36 clocks，EmuTOS 第 19 條／532 clocks 的 frame／state／prefetch 對上 Hatari |
| ST Ricoh `$FF860F` void byte read | **CONFORMED** | read `$FF`、8-clock `TST.B`；EmuTOS 第 6,851 條 state／prefetch 對上 Hatari |
| ST 無 Mega-RTC `$FFFC21–$FFFC3F` | **CONFORMED** | byte read `$FF`／write discard；EmuTOS 第 6,879 條 state／prefetch 對上 Hatari |
| 68000 `TST.B (An)` bus error／vector 2 | **CONFORMED** | `$FF8A3C` Blitter probe 64 clocks；byte lane、14-byte frame、state／prefetch 對上 Hatari |
| ST MFP GPIP `$FFFA01` reset write | **CONFORMED** | DDR=0 masking、4 wait clocks；EmuTOS 第 7,475 條／176,638 clocks 對拍，下一停點 `$FFFA03` |
| ST MFP AER `$FFFA03` reset zero write | **CONFORMED** | falling-edge reset、非零改寫 fail-closed、4 wait clocks；第 7,479 條／176,682 clocks 對拍 |
| 68000 bus error／vector 2 | 進行中 | `MOVE.W` 與 `TST.B (An)` read 切片已 CONFORMED；其餘讀寫、寬度、instruction fetch 與 double fault 仍須逐片驗收 |
| ST／STF I/O memory map | 進行中 | 兩個普通 ST void 區與 MFP GPIP／AER reset write 已接；下一停點 `$FFFA05` DDR byte write 需先規格化 |
| Hatari 外部 oracle | **DRAFT** | 同輸入 metadata、狀態與截圖收據可重跑；公開契約載體待使用者定案 |
