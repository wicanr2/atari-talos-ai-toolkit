# 驗證矩陣

| 能力 | 內部測試 | 獨立 oracle | 現況 |
|---|---|---|---|
| JSON Lines 解碼與回覆關聯 | protocol／CLI golden test | 不適用 | 已建立 |
| 未知欄位與未知命令拒絕 | protocol／CLI test | 不適用 | 已建立 |
| 68000 NOP | 本地 pipeline／fail-closed 測試 | SingleStepTests 2,500 筆狀態＋bus trace | 通過 |
| 68000 MOVEQ | 本地 immediate／CCR 測試 | SingleStepTests 2,500 筆狀態＋bus trace | 通過 |
| 68000 SWAP／EXT.W／EXT.L | 本地暫存器／CCR 測試 | SingleStepTests 7,500 筆狀態＋bus trace | 通過 |
| 68000 Bcc／BRA 正常控制流 | condition／位移／pipeline 測試 | SingleStepTests 1,830 筆正常狀態＋bus trace | 通過 |
| 68000 address error | 固定 frame／模式／handler 測試 | Bcc 670 筆例外狀態＋bus trace | 通過 |
| 68000 BSR | stack／return PC／複合例外測試 | SingleStepTests 2,500 筆狀態＋bus trace | 通過 |
| 68000 RTS | stack pop／return 預取／複合例外測試 | SingleStepTests 2,500 筆狀態＋bus trace | 通過 |
| 68000 JMP／JSR control EA | 非 control mode 拒絕測試 | SingleStepTests 5,000 筆七種 mode＋bus trace | 通過 |
| 68000 LEA／PEA | control-EA destination／stack 測試 | SingleStepTests 5,000 筆狀態＋bus trace | 通過 |
| 68000 MOVE.B | A7／alias postincrement、byte lane／CCR 測試 | MOVE.B 2,500 筆全 source／destination EA＋RAM＋bus trace | 通過 |
| 68000 MOVE.W | word EA／CCR／data address-error 測試 | MOVE.W 正常 1,013＋read fault 839＋write fault 648 筆完整 state／RAM／clock／bus trace | 通過 |
| 68000 MOVE.L | 分段 long EA／CCR／data address-error 測試 | MOVE.L 正常 1,013＋read fault 869＋write fault 618 筆完整 state／RAM／clock／bus trace | 通過 |
| 68000 MOVEA.W／MOVEA.L | 符號延伸／A7／alias／CCR 不變測試 | 兩種寬度 5,000 筆完整 state／RAM／clock／bus／read-fault frame | 通過 |
| 68000 ADDA.W／ADDA.L | 符號延伸／active A7／CCR 不變測試 | 兩種寬度 5,000 筆完整 state／RAM／clock／bus／read-fault frame | 通過 |
| 68000 AND／ANDI.B／W／L | Dn、RMW bus 次序、long word 次序／CCR 測試 | 三種寬度 7,500 筆完整 state／RAM／clock／bus／read-fault frame | 通過 |
| 68000 CMP／CMPI／CMPM.B／W／L | 減法旗標、雙後遞增與 fault 副作用測試 | 三種寬度 7,500 筆完整 state／RAM／clock／bus／fault frame | 通過 |
| 68000 ADD／ADDI／ADDQ.B／W／L | 加法旗標、quick、An 特例與 RMW 測試 | 三種寬度 7,500 筆完整 state／RAM／clock／bus／fault frame | 通過 |
| 68000 CLR.B／W／L | Dn 寬度、CCR 與 memory RMW 測試 | 三種寬度 7,500 筆完整 state／RAM／clock／bus／fault frame | 通過 |
| 68000 MOVEM.W／L | mask、雙方向、虛讀、predecrement／postincrement 測試 | 兩種寬度 5,000 筆完整 state／RAM／clock／bus／fault frame | 通過 |
| 68000 LINK／UNLK | active stack、frame pointer、A7 alias、push／pop、odd-frame 測試 | 兩族共 5,000 筆完整 state／RAM／clock／bus／vector-3 frame | 通過 |
| 68000 TST.B／W／L | Dn／memory EA、CCR、無寫回測試 | 三種寬度 7,500 筆完整 state／RAM／clock／bus／fault frame | 通過 |
| 68000 OR／ORI.B／W／L | Dn、雙方向、RMW bus 次序、CCR 測試 | 三種寬度 7,500 筆完整 state／RAM／clock／bus／fault frame | 通過 |
| 68000 SUB／SUBI／SUBQ.B／W／L | 減法旗標、quick、An 特例與 RMW 測試 | 三種寬度 7,500 筆完整 state／RAM／clock／bus／fault frame | 通過 |
| 68000 ASL.B／W／L | immediate／Dn count、X／NZVC、動態 clock、memory RMW 測試 | 三種寬度 7,500 筆完整 state／RAM／clock／bus／fault frame | 通過 |
| 68000 ASR／LSR.B／W／L | immediate／Dn count、符號／零填入、X／NZVC、memory RMW 測試 | 六份語料 15,000 筆完整 state／RAM／clock／bus／fault frame | 通過 |
| 68000 MULS／MULU.W | signed／unsigned 結果、資料相依 clock、word data EA 測試 | 兩份語料 5,000 筆完整 state／RAM／clock／bus／fault frame | 通過 |
| 68000 NOT／NEG.B／W／L | Dn／memory、邏輯／borrow 旗標、RMW bus 次序測試 | 六份語料 15,000 筆完整 state／RAM／clock／bus／fault frame | 通過 |
| 68000 Scc.B | 16 conditions、Dn 真／假 clock、memory RMW、SR 不變測試 | Scc 2,500 筆完整 state／RAM／clock／bus trace | 通過 |
| 68000 DBcc.W | 16 conditions、低 16-bit 計數、三條正常時序與奇數目標測試 | DBcc 2,500 筆完整 state／RAM／clock／bus／vector-3 frame | 通過 |
| ST／STF 基礎 memory map | RAM 容量／邊界、reset shadow、ROM、FC 權限、word 與 fault 測試 | 自建非原版 pattern；Atari 1986 hardware spec 第 25–27 頁 | 通過；CPU vector 2 其餘路徑／I/O 待補 |
| MC68000／ST power-on reset | FC=6 vector 讀取、SR／SSP／PC／prefetch、failure staging、machine counters 與首指令 | synthetic ROM；EmuTOS 1.3 UK 192 KiB；Hatari 2.4.1 同狀態 | 通過；首條 `BRA.W` 後 10 clocks／PC／prefetch 一致 |
| ST MMU `$FF8001` | cold reset、FC 權限、full-byte R/W latch、512 KiB-bank RAS／CAS translation、alias／identity／empty bank | Atari 1986 hardware spec；Hatari I/O trace／`stMemory.c`；EmuTOS 同 ROM | 通過；`$FA` 寫讀與前 7 條至 `$FC0070` state／92 clocks 全同 |
| 68000 `MOVEC` illegal／vector 4 | `$4E7A/$4E7B`、saved opcode PC、format-0 frame、FC／bus、36 clocks | synthetic 雙方向／user／supervisor；Hatari／EmuTOS 同 ROM | 通過；第 8 條／128 clocks 到 `$FC0074` state／frame 全同 |
| 其餘 68000 指令 | 待建立 | SingleStepTests；TAS／TRAPV 暫不採信 | 進行中 |
| TOS 開機 | reset、MMU、`MOVEC`→vector 4、vector 2 fault address 與 `RESET` 已建立；cartridge 區與 I/O 待擴充 | Hatari 2.4.1／EmuTOS 1.3 同 ROM | 進行中；`$FC0074` 完全對拍（8 條／128 clocks），`$FC0080` 的 bus-error frame 與 Hatari 同值（`$FFFF8006`），`$FC0088` 的 `RESET` 已執行（11 條／352 clocks）；下一個停止點是 `$FC008E` 讀 `$FA0000` |
| 畫面 | 待建立 | Hatari 同幀原生 framebuffer | 未開始 |
| 輸入與時序 | 待建立 | Hatari 同事件與狀態點 | 未開始 |
| Dungeon Master | 待建立 | Hatari 正常入口同狀態路徑 | 未開始 |
