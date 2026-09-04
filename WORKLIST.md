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
| 68000 LINK／UNLK | **CONFORMED（限定範圍）** | LINK 2,500 筆外部全同；UNLK 偶數 frame 本地通過，odd frame fail-closed |
| 68000 TST.B／W／L | **CONFORMED** | 三種寬度共 7,500 筆 Dn／memory EA、CCR、無寫回、bus 與 fault 全同 |
| 68000 OR／ORI.B／W／L | **CONFORMED** | 三種寬度共 7,500 筆雙方向、immediate、RMW、CCR、fault 與 bus 全同 |
| 68000 SUB／SUBI／SUBQ.B／W／L | **CONFORMED** | 三種寬度共 7,500 筆雙方向、immediate、quick、borrow、RMW、fault 與 bus 全同 |
| 68000 ASL.B／W／L | **CONFORMED** | 三種寬度共 7,500 筆 immediate／Dn count、X／NZVC、動態 clock、RMW 與 fault 全同 |
| 68000 ASR／LSR.B／W／L | **CONFORMED** | 六份語料共 15,000 筆 count、符號／零填入、X／NZVC、RMW 與 fault 全同 |
| 68000 MULS／MULU.W | **CONFORMED** | 兩份語料共 5,000 筆 signed／unsigned、資料相依 clock、EA 與 fault 全同 |
| 68000 NOT／NEG.B／W／L | **CONFORMED** | 六份語料共 15,000 筆 Dn／memory、旗標、RMW、long word 次序與 fault 全同 |
| 其餘 68000 指令 | 進行中 | 每組 READY 後實作，逐組通過外部語料 |
| ST／STF memory map | 待辦 | TOS ROM 唯讀、RAM 邊界與 bus error 有規格及測試 |
| Hatari 外部 oracle | **DRAFT** | 同輸入 metadata、狀態與截圖收據可重跑；公開契約載體待使用者定案 |
