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
| 68000 ROL.B／W／L | immediate／Dn count、X 保留、NZVC、memory RMW 測試 | 三份語料 7,500 筆完整 state／RAM／clock／bus／fault frame | 通過；固定 EmuTOS `$E378` 16 clocks |
| 68000 MULS／MULU.W | signed／unsigned 結果、資料相依 clock、word data EA 測試 | 兩份語料 5,000 筆完整 state／RAM／clock／bus／fault frame | 通過 |
| 68000 NOT／NEG.B／W／L | Dn／memory、邏輯／borrow 旗標、RMW bus 次序測試 | 六份語料 15,000 筆完整 state／RAM／clock／bus／fault frame | 通過 |
| 68000 Scc.B | 16 conditions、Dn 真／假 clock、memory RMW、SR 不變測試 | Scc 2,500 筆完整 state／RAM／clock／bus trace | 通過 |
| 68000 DBcc.W | 16 conditions、低 16-bit 計數、三條正常時序與奇數目標測試 | DBcc 2,500 筆完整 state／RAM／clock／bus／vector-3 frame | 通過 |
| ST／STF 基礎 memory map | RAM 容量／邊界、reset shadow、ROM、FC 權限、word 與 fault 測試 | 自建非原版 pattern；Atari 1986 hardware spec 第 25–27 頁 | 通過；CPU vector 2 其餘路徑／I/O 待補 |
| MC68000／ST power-on reset | FC=6 vector 讀取、SR／SSP／PC／prefetch、failure staging、machine counters 與首指令 | synthetic ROM；EmuTOS 1.3 UK 192 KiB；Hatari 2.4.1 同狀態 | 通過；首條 `BRA.W` 後 10 clocks／PC／prefetch 一致 |
| ST MMU `$FF8001` | cold reset、FC 權限、full-byte R/W latch、512 KiB-bank RAS／CAS translation、alias／identity／empty bank | Atari 1986 hardware spec；Hatari I/O trace／`stMemory.c`；EmuTOS 同 ROM | 通過；`$FA` 寫讀與前 7 條至 `$FC0070` state／92 clocks 全同 |
| 68000 `MOVEC` illegal／vector 4 | `$4E7A/$4E7B`、saved opcode PC、format-0 frame、FC／bus、36 clocks | synthetic 雙方向／user／supervisor；Hatari／EmuTOS 同 ROM | 通過；第 8 條／128 clocks 到 `$FC0074` state／frame 全同 |
| vector 2 absolute-short address | CPU 32-bit EA、24-bit bus、14-byte frame、68 clocks | synthetic `$4A78,$8006`；Hatari／EmuTOS 同 ROM | 通過；第 10 條／220 clocks 完整 frame 全同 |
| MC68000 `RESET` | privilege、external reset hook、register preservation、prefetch、132 clocks | synthetic user／supervisor；Hatari／EmuTOS 同 ROM | 通過；第 11 條／352 clocks state／prefetch 全同 |
| MC68000 level 4 autovector 接受 | mask／非法 level fail-closed、running／STOP saved PC、format-0 frame、timed bus 44 clocks | 固定 EmuTOS `$70=$FC0446`；Hatari 第一／第二 VBL handler 入口 SR／SSP／frame／prefetch | 通過；CPU 接受層與第一 GLUE VBL 已接線 |
| ST reset frame 第一個 GLUE VBL | deadline crossing、mask pending、interrupt／instruction 分帳、guest handler 寫入 | 固定 EmuTOS＋Hatari 2.4.1 `--fast-boot false` 第一 `$FC0446` D/A／SSP／SR／frame／prefetch與 `$466` consumer | 通過；E-clock／video IACK machine clock 由 recurring 規格補齊 |
| ST reset 60 Hz recurring VBL／STOP 快轉 | 263×508 deadline、E-clock 公式、video IACK、idle timeline、STOP 喚醒與 guest handler | 固定 EmuTOS＋Hatari 第二 `$FC0446` 完整 state／frame／prefetch；`$466:1→2` | 通過；第二 handler 267,332 clocks，後續由規格 077 接管 |
| ST Shifter `$FF8260` reset／low-res 同值寫入 | reset/read、`0→0`、非零 fail-closed、alias／權限／寬度／bus wait | Atari hardware map；固定 Hatari／EmuTOS `$FC69E6` 前後完整 state | 通過；12 clocks 到 `$FC69EA`，後續由規格 078 接管 |
| ST GLUE `$FF820A` 第 0 線 60→50 Hz | reset/read、typed transition、same-value／反向 fail-closed、deadline／period 修正 | Hatari `$FC6A02` CycleCounter 401,272→401,284；VBL4 event 535,528 | 通過；Talos 固定寫入 12 clocks，next deadline 535,528 |
| ST Shifter 16 色 palette | 16 word registers、`$0777` mask、reset／alias／權限／寬度／timed wait | Hatari／EmuTOS `$FC671A` 首筆與 `$FC6722` 完整 palette／state | 通過；Talos 401,366→402,052 完成迴圈，後續由 framebuffer base 切片接管 |
| ST Shifter 程式化 framebuffer 基址 | `$FF8201/$FF8203` reset／byte R/W、`$3F` high mask、組址、alias／權限／寬度／timed wait | Hatari／EmuTOS `$FC67FA/$FC6800` 完整 state、register 與 active base 分離 | 通過；Talos 403,900→403,948 得 `$0F8000`，Hatari 403,924→403,972 |
| ST Shifter 第四幀 active framebuffer 基址重載 | programmed／active 分離、VBL 原子提交、running crossing、STOP 快轉、reset | Hatari `video_hbl` trace；535,524→535,532 前後 `info video` 0→`$0F8000` | 通過；共同 deadline 535,528，Talos 可觀察邊界 535,520→535,530 |
| ST low-res 4-plane 索引畫面 | 16 indices、plane／bit／group／line／frame 邊界、DMA RAM、fault、snapshot isolation | Hatari／EmuTOS VBL7 32,000-byte dump；raw／decoded SHA-256、histogram、首非零座標 | 通過；320×200 indices，Talos VBL4 正常路徑全黑 snapshot亦通過 |
| ST MFP Timer C 啟動／channel enable | `$C0` main、TCDCR `$00→$50`、IERB bit 5、IMRB latch、fail-closed 邊界 | NXP MC68901 manual；EmuTOS `xbtimer`；Hatari `mfp_start/mfp_write` trace | 部分通過；start 與 enable 已接，countdown／timeout／IRQ 待補 |
| ST MFP Timer D／USART boot init | TCDCR `$51/$52`、2,560-MFP-clock recurrence、channel 4 pending／level 6 vector 68／software EOI、UCR／RSR／TSR | NXP MC68901 manual；EmuTOS `rsconf1/mfpint`；Hatari MFP exception／IACK trace | 部分通過；Timer D 至 guest `$FC788A` clear 已接，USART data/IRQ 待補；start phase 暫為 instruction-boundary approximation |
| ST YM2149 boot ports | select/data序列、reset、權限、寬度與 fail-closed | Atari hardware map；Hatari `psg_write` trace；固定 EmuTOS ROM | 通過；音訊合成與 port side effects待補 |
| ST IKBD ACIA control／first TX | `$03→$96`、TDRE、TDR=`$80`、1024-clock deadline、fail-closed | MC6850 契約；Hatari `acia,ikbd_acia` trace；固定 EmuTOS ROM | 通過；68,645 instructions／969,640 clocks 抵達第二 byte `$01`，完整 serial／RX／IRQ 待補 |
| ST IKBD ACIA second TX／reset RX | `$01` TDR buffer、10-tick frame、513,024-clock response、RDRF／IRQ status與read-clear | MC6850契約；Hatari 16-VBL trace；固定EmuTOS ROM | 通過；guest讀 `$F1` 後前進至136,048 instructions／1,577,208 clocks的MIDI ACIA gate |
| ST MIDI ACIA control／MFP channel 6 | `$03→$95`、IKBD stale RDR、IPRB/ISRB clear、IERB/IMRB=`$60` staged enable | MC6850／MC68901契約；Hatari MIDI/I/O trace；固定EmuTOS ROM | 通過；136,182 instructions／1,578,882 clocks抵達channel 4／Timer D重設 |
| ST MFP Timer D system-clock start／IRQ | `$EF` clear、`$51→$50→$52`、TDDR=256、有理數 recurrence、IPRB/ISRB、level 6 vector 68 | MC68901契約；Hatari MFP start／exception／IACK trace；固定EmuTOS ROM | 通過；136,210條啟動，137,138條／1,587,632 clocks 進 `$FC7884` 並由 guest 清 ISRB |
| ST 空 cartridge window | 128 KiB `$FF`、FC、MMU 獨立、ROM write fault、邊界 | Hatari v2.4.1 固定原始碼；Hatari／EmuTOS 同 ROM | 通過；第 12 條／380 clocks state／prefetch 全同 |
| 其餘 68000 指令 | 待建立 | SingleStepTests；TAS／TRAPV 暫不採信 | 進行中 |
| TOS 開機 | reset、MMU、exceptions、`RESET`、VBL、Shifter、部分 MFP／PSG／ACIA 已建立；bus arbitration／I/O 待擴充 | Hatari 2.4.1／EmuTOS 1.3 同 ROM | 進行中；正常路徑至137,138 instructions／1,587,632 clocks 的 Timer D handler，下一缺口是後續 Timer C timeout／IRQ |
| 畫面 | low-res 4-plane→palette index 與 VBL snapshot 已建立 | Hatari VBL7 raw framebuffer、decoded index hash | 進行中；RGB／PNG、border、raster palette與遊戲畫面待補 |
| 輸入與時序 | 待建立 | Hatari 同事件與狀態點 | 未開始 |
| Dungeon Master | 待建立 | Hatari 正常入口同狀態路徑 | 未開始 |
