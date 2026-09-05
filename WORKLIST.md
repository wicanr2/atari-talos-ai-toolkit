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
| vector 2 absolute-short fault address | **CONFORMED** | frame 保存 CPU 內部的 32-bit 有效位址；EmuTOS `$FC0080` `TST.W $8006` 端到端得 `$FFFF8006`、匯流排 `$FF8006`；含暫存器高位元的負對照與三條鑑別力驗證 |
| MC68000 `RESET` | **CONFORMED** | `RESET.json.bin` 2,500 筆（supervisor 132 clocks／user vector 8 34 clocks）全同；EmuTOS `$FC0088` 端到端執行成功，CPU 狀態不變、PC 前進一個 word。外部 `RESET` 線對周邊的效果待 I/O 接入後各自定義 |
| ST cartridge port `$FA0000`–`$FBFFFF` | 待規格 | EmuTOS 第 12 條指令在 `$FC008E` 探測插卡；目前 memory map 不涵蓋該區，回 unmapped fault |
| bus fault → vector 2 的指令涵蓋面 | 待規格 | 目前只有 `MOVE.W` 來源讀取會轉成 vector 2；其餘路徑的 bus fault 直接往外傳，開機路徑因此停在第一個非 `MOVE.W` 的失敗存取 |
| 68000 bus error／vector 2 | 進行中 | `MOVE.W` user word source read 首切片已 CONFORMED；其餘讀寫、寬度、instruction fetch 與 double fault 仍須逐片驗收 |
| ST／STF I/O memory map | 待辦 | 各晶片 READY 後逐區接入；保留位址維持 bus fault |
| UCSD p-System 直譯器真實碼驗收 | **CONFORMED** | SunDog 的 `SYSTEM.INTERP`（固定 SHA-256）：分派表結構、短常數、區域變數 `+8+n×2`、`ixa` 的 `base+index×n×2` 與變長運算元全部通過，每條都有負對照 |
| UCSD p-System 分派迴圈與序列執行 | **CONFORMED** | `$00DE` 的 fetch-execute 循環與分派表全形狀（107 支常式、45 個無效 opcode）；短常數、混合族、存取往返、區域變數位址與 NOP 序列全部通過，六組負對照確認會失敗；六組 p-code 與 laanwj/sundog 的獨立 C 直譯器逐字相同 |
| UCSD p-System 算術、分支與真實邏輯 | **CONFORMED** | `ldcb`／`ldci`／`adi`／`sbi`／`dvi`／`modi`／`equi`／`leqi`／`dup1`／`swap`；並以原版 `check_exit` 的格座標換算驗收——數值取自原版執行時的除錯器讀值，算出的欄 11／列 7 與當時讀到的格座標一致 |
| Hatari 外部 oracle | **可重跑** | `tools/hatari-oracle/`：Dockerfile 釘住上游 tarball 的 SHA-256（`2a5da193…`），`trace.sh` 用 `--run-vbls` 加 `--trace-file` 無人值守取 CPU trace，`cycles.sh` 挑出指定位址第一次執行的 cycle 數。以已 CONFORMED 的 `$FC0070`／`$FC0074`（92／128 cycles）做過正對照。`--parse` 那條互動路不可用——中斷點觸發後 Hatari 停在提示等 stdin，容器裡既不結束也拿不到輸出。截圖收據與公開契約載體仍待定案 |
