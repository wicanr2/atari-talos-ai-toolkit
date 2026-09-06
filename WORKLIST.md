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
| 68000 ROL.B／W／L | **CONFORMED** | immediate／Dn count、X 保留、NZVC、動態 clock、memory RMW 與 fault 共 7,500 筆全同；EmuTOS `$E378` 已跨過 |
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
| ST Ricoh `$FF860F` void byte write | **CONFORMED** | 任意byte忽略、24-bit alias、無裝置狀態；289,521條／2,982,760 clocks |
| ST 無 Mega-RTC `$FFFC21–$FFFC3F` | **CONFORMED** | byte read `$FF`／write discard；EmuTOS 第 6,879 條 state／prefetch 對上 Hatari |
| 68000 `TST.B (An)` bus error／vector 2 | **CONFORMED** | `$FF8A3C` Blitter probe 64 clocks；byte lane、14-byte frame、state／prefetch 對上 Hatari |
| ST MFP GPIP `$FFFA01` reset write | **CONFORMED** | DDR=0 masking、4 wait clocks；EmuTOS 第 7,475 條／176,638 clocks 對拍，下一停點 `$FFFA03` |
| ST MFP AER `$FFFA03` reset zero write | **CONFORMED** | falling-edge reset、非零改寫 fail-closed、4 wait clocks；第 7,479 條／176,682 clocks 對拍 |
| ST MFP DDR `$FFFA05` reset zero write | **CONFORMED** | input reset、非零方向改寫 fail-closed、4 wait clocks；第 7,483 條／176,726 clocks 對拍 |
| ST MFP IERA／IERB reset zero writes | **CONFORMED** | disable/reset、非零 enable fail-closed、4 wait clocks；第 7,491 條／176,814 clocks 對拍 |
| ST MFP IPRA／IPRB write-zero-to-clear | **CONFORMED** | `pending &= value`、reset／read、4 wait clocks；第 7,499 條／176,902 clocks 對拍 |
| ST MFP ISRA／ISRB write-zero-to-clear | **CONFORMED** | `in_service &= value`、reset／read、4 wait clocks；第 7,507 條／176,990 clocks 對拍 |
| ST MFP IMRA／IMRB mask latch | **CONFORMED** | 無 pending 時完整 byte latch、pending 非零 fail-closed、4 wait clocks；第 7,515 條／177,078 clocks 對拍 |
| ST MFP Vector Register | **CONFORMED** | vector base、unused bits read-zero、EOI／ISR clear、pending 非零 fail-closed；第 7,519 條／177,122 clocks 對拍 |
| ST MFP timer control reset-stop | **CONFORMED** | TACR／TBCR／TCDCR `$00→$00`、非零 fail-closed、4 wait clocks；第 7,531 條／177,254 clocks 對拍 |
| ST MFP timer data stopped-load | **CONFORMED** | TADR／TBDR／TCDR／TDDR 停止時同步 data/main counter、active fail-closed；第 7,547 條／177,430 clocks 對拍 |
| ST MFP Timer C delay-mode 啟動 | **CONFORMED** | 固定 `$00→$50`、TCDR/main=`$C0`、÷64 start transition；68,103 條／963,104 clocks 抵達 `$E378` memory `ROL.W` gate |
| ST MFP Timer C interrupt enable | **CONFORMED** | IERB bit 5=`$20`、IPRB 尚無 pending、IMRB=`$20`；68,378 條／966,808 clocks 抵達 Timer D `$50→$51` gate |
| ST MFP Timer D delay-mode 啟動 | **CONFORMED** | TCDCR `$50→$50` stop-D、TDDR/main=`$02`、TCDCR `$50→$51`；68,392 條／966,948 clocks |
| ST MFP USART fixed enable | **CONFORMED** | UCR／RSR／TSR=`$88/$01/$01`；68,451 條／967,594 clocks 抵達 IERA gate |
| ST MFP USART interrupt channels | **CONFORMED** | RBF／TBE 令 IERA／IMRA=`$14/$14`，無 pending；68,518 條／968,318 clocks |
| ST YM2149 boot mixer／port A | **CONFORMED** | `$FF8800/$FF8802` 四筆固定 write，selected/R7/R14=`$0E/$C0/$07`；68,528 條／968,510 clocks 抵達 ACIA |
| ST YM2149 port A首次drive-select更新 | **CONFORMED** | 同值選R14、讀`$07`、寫`$05`；289,556條／2,983,132 clocks |
| ST DMA mode／WD1772 force-interrupt初始化 | **CONFORMED** | mode `$0080`選command/status、`$D0`清IRQ並建立Type-I status；timed word bus接入後289,612條／2,983,704 clocks |
| ST WD1772 restore／GPIP5 IRQ期限 | **CONFORMED** | `$0B`自timed bus phase起729 CPU clocks；九次inactive輪詢後讀`$91`，289,803條／2,985,654 clocks |
| ST WD1772 Type-I status read-clear | **CONFORMED** | mode `$0080`同值write，固定無磁片status `$E4`；read清IRQ／GPIP5，289,865條／2,986,256 clocks |
| ST WD1772 data register／same-track seek | **CONFORMED** | `$86→data 0→$80→seek $13`，729-clock期限、九次inactive poll與`$E4` read-clear；290,223條／2,989,944 clocks |
| ST YM2149 port A切換drive 1 | **CONFORMED** | 同值選R14、讀`$05`、寫`$03`；290,303條／2,990,890 clocks |
| ST IKBD ACIA control init | **CONFORMED** | `$03→$96`、status TDRE=`$02`；68,551 條／968,772 clocks 抵達首筆 data write |
| ST IKBD ACIA first transmit deadline | **CONFORMED** | TDR=`$80`、TDRE clear／1024-clock restore；68,645 條／969,640 clocks 抵達第二 byte `$01` |
| ST IKBD ACIA second TX／reset response | **CONFORMED** | `$01`雙buffer、10-tick frame、513,024-clock `$F1` response；128,378條／21 IRQ／1,509,022 clocks讀取 `$F1` |
| ST MIDI ACIA control／IKBD stale RDR／MFP channel 6 | **CONFORMED** | `$03→$95`、一次`$F1`保留值讀取、IERB/IMRB=`$60`；136,236條／23 IRQ／1,580,634 clocks |
| ST MFP Timer D system-clock start | **CONFORMED** | channel 4 clear、TDDR=`$00`=256、IERB/IMRB=`$70`、TCDCR=`$52`；136,285條／23 IRQ／1,581,256 clocks |
| ST MFP Timer D recurrence／channel 4 IRQ | **CONFORMED** | 2,560 MFP clocks有理數週期、IPRB bit4、level 6 vector 68、software EOI；137,213條／24 IRQ／1,589,660 clocks進`$FC7884` |
| ST MFP Timer C recurrence／channel 5 IRQ | **CONFORMED** | timed phase 962,844、12,288 MFP clocks、B-bank priority、level 6 vector 69；72,342條／1,003,004 clocks進`$FC04DE` |
| ST MFP Timer D normal stop／channel 4 clear | **CONFORMED** | IERB/IMRB `$70→$60`、TCDCR `$52→$50`、vector `$110=$FC03EA`、scheduler停止；289,256條／2,978,730 clocks |
| ST MFP USART第二次設定／baud Timer D重啟 | **CONFORMED** | TSR empty bit、七段同值重設、control1不啟動system IRQ scheduler；289,342條／2,979,680 clocks |
| ST MFP USART reset writes | **CONFORMED** | SCR／UCR／RSR／TSR 軟體清零、TSR 硬體 reset 未定、非零與 UDR fail-closed；第 7,563 條／177,606 clocks 對拍 |
| MC68000 `STOP` | **CONFORMED** | privilege、immediate SR、stopped latch、Reset 清除；2,500 筆語料通過，接入第一 VBL 後 EmuTOS 第 7,604 條／178,228 clocks 進入停機 |
| MC68000 level 4 autovector 接受 | **CONFORMED** | mask 仲裁、44 clocks、6-byte frame、running／STOP saved PC 與 `$70→$FC0446` 對上固定 Hatari |
| ST reset frame 第一個 GLUE VBL | **CONFORMED** | 133,668-clock pending、mask 保留、同 profile handler entry 全狀態、guest `$466:0→1`；E-clock／IACK 由規格 076 補齊 |
| ST reset 60 Hz recurring VBL／STOP 快轉 | **CONFORMED** | 第二 deadline 267,272、E-clock／video IACK、handler entry 267,332 全狀態同 Hatari，guest `$466:1→2` |
| ST Shifter `$FF8260` reset／low-res 同值寫入 | **CONFORMED** | STF read `$FC`、非零 fail-closed；EmuTOS `$FC69E6` 12 clocks 與 Hatari 全狀態同 |
| ST GLUE `$FF820A` 第 0 線 60→50 Hz | **CONFORMED** | `$FC6A02` 12 clocks、register `$00→$02`；第四 deadline 535,528 對 Hatari CycleCounter |
| ST Shifter `$FF8240–$FF825E` 16 色 palette | **CONFORMED** | ST `$0777` mask、word bank；EmuTOS 16 色迴圈完整 state／值同 Hatari |
| ST Shifter `$FF8201/$FF8203` 程式化 framebuffer 基址 | **CONFORMED** | 22-bit DMA mask、256-byte alignment；EmuTOS `$0F8000` 兩筆寫入完整 state／clock 同 Hatari |
| ST Shifter 第四幀 active framebuffer 基址重載 | **CONFORMED** | transition frame 無 HBL310；VBL deadline 535,528 將 programmed `$0F8000` 提交為 active base |
| ST low-res 320×200 4-plane 索引畫面 | **CONFORMED** | 32,000-byte DMA snapshot→64,000 indices；Hatari VBL7 raw／decoded SHA-256 與 histogram 通過 |
| ST MFP GPIP fixed input sample | **CONFORMED** | color／FDC idle／no-printer `$A1` 依 DDR 合併；monitor probe 與 STOP 前 D2=`$2710` 對上 Hatari |
| 68000 bus error／vector 2 | 進行中 | `MOVE.W` 與 `TST.B (An)` read 切片已 CONFORMED；其餘讀寫、寬度、instruction fetch 與 double fault 仍須逐片驗收 |
| ST／STF I/O memory map | 進行中 | recurring VBL、video、MFP、PSG／ACIA／USART、雙drive FDC鏈、空ACSI、parallel strobe與IKBD `$1C` transmit已接；下一步IKBD `$FC + 6-byte` clock response |
| UCSD p-System 直譯器真實碼驗收（規格 134）| **CONFORMED** | SunDog 的 `SYSTEM.INTERP`（固定 SHA-256）：分派表結構、短常數、區域變數 `+8+n×2`、`ixa` 的 `base+index×n×2` 與變長運算元全部通過，每條都有負對照 |
| UCSD p-System 分派迴圈與序列執行（規格 135）| **CONFORMED** | `$00DE` 的 fetch-execute 循環與分派表全形狀（107 支常式、45 個無效 opcode）；短常數、混合族、存取往返、區域變數位址與 NOP 序列全部通過，六組負對照確認會失敗；六組 p-code 與 laanwj/sundog 的獨立 C 直譯器逐字相同 |
| UCSD p-System 算術、分支與真實邏輯（規格 136）| **CONFORMED** | `ldcb`／`ldci`／`adi`／`sbi`／`dvi`／`modi`／`equi`／`leqi`／`dup1`／`swap`；並以原版 `check_exit` 的格座標換算驗收——數值取自原版執行時的除錯器讀值，算出的欄 11／列 7 與當時讀到的格座標一致 |
| UCSD p-System 布林運算（規格 137）| **CONFORMED** | `land`／`lor`／`bnot` 都是位元運算，配上 `fjp`／`tjp` 的 `btst #0`，真假整套住在 bit 0——8 是假、9 是真。用 SunDog `XSTARTUP:0x31` 的初始損壞判斷式跑整張真值表，負對照兩條 |
| Hatari 外部 oracle | **可重跑** | `tools/hatari-oracle/`：Dockerfile 釘住上游 tarball 的 SHA-256（`2a5da193…`），`trace.sh` 用 `--run-vbls` 加 `--trace-file` 無人值守取 CPU trace，`cycles.sh` 挑出指定位址第一次執行的 cycle 數。以已 CONFORMED 的 `$FC0070`／`$FC0074`（92／128 cycles）做過正對照。`--parse` 那條互動路不可用——中斷點觸發後 Hatari 停在提示等 stdin，容器裡既不結束也拿不到輸出。截圖收據與公開契約載體仍待定案 |
