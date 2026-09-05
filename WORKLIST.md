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
| 其餘 68000 指令 | 進行中 | 每組 READY 後實作，逐組通過外部語料 |
| ST／STF 基礎 memory map | **CONFORMED** | 512 KiB／1 MiB RAM、reset shadow、192 KiB TOS ROM、保護與 typed bus fault 測試通過 |
| 68000 bus error／vector 2 | 進行中 | `MOVE.W` user word source read 首切片已 CONFORMED；其餘讀寫、寬度、instruction fetch 與 double fault 仍須逐片驗收 |
| ST／STF I/O memory map | 待辦 | 各晶片 READY 後逐區接入；保留位址維持 bus fault |
| Hatari 外部 oracle | **DRAFT** | 同輸入 metadata、狀態與截圖收據可重跑；公開契約載體待使用者定案 |
