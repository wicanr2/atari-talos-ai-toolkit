# Atari Talos AI Toolkit

**給大型語言模型與自動化測試使用的無頭 Atari ST／STF 對拍工具。**

Atari Talos 誕生的目的，是讓大型語言模型（LLM）能更可靠地執行、觀察與比較
Atari ST 遊戲。它將提供可重現輸入、逐幀執行、狀態擷取與機器可讀的對拍證據，
協助復古遊戲重製專案更準確地還原原版行為。

傳統模擬器主要讓人透過視窗操作；Codex、Claude Code 與 `go test` 需要的則是
「問得到、量得到、可重播」：精確推進指定畫格、讀取 CPU／記憶體狀態、保存快照，
並以 JSON Lines 交換命令與結果，而不是送按鍵後等待一段牆上時間再猜畫面是否穩定。

## 現況

專案處於 **M2 機器核心**：版本化 JSON Lines 控制契約已建立，MC68000 核心通過
240,000 筆外部語料，且已保留語料中的 idle／active bus 時間軸；machine epoch、
timed Bus 與首個 4-clock prefetch 路徑已接線。ST／STF memory map
、固定 color profile 的 MFP GPIP input sample，與固定 EmuTOS ROM 的早期啟動及
機型探測已逐狀態對上 Hatari；GLUE 在 reset sync mode 下會以 60 Hz recurring VBL 喚醒 MC68000
stopped state，並經 E-clock 對齊的 level-4 autovector 進入真正的 EmuTOS handler。
第二次 handler 在 267,332 clocks 與 Hatari 全狀態一致，且 `$466 frclock` 已由 1 增至 2；
有界續跑亦可跨第三次 VBL，並完成 `$FF8260` Shifter low-resolution 同值初始化。
`$FF820A` 也會在第三幀由 60 Hz 切至 50 Hz，第四個 VBL deadline 已依 Hatari 修正為
535,528。16 色 `$FF8240–$FF825E` palette bank 與 EmuTOS 初始化迴圈也已完整對拍；
`$FF8201/$FF8203` 程式化 framebuffer 基址暫存器亦已寫成 `$0F8000`，且明確與目前
掃描中的作用基址分離；第四個 VBL 現會在精確 deadline 535,528 將其提交為 active base。
首個 320×200、4-plane big-endian 解碼器也已從 active base 產生 64,000 個 palette indices，
並以 Hatari VBL7 真實 framebuffer dump 驗證；RGB／PNG 與非黑 Talos 正常路徑尚未接通。
ST DMA／WD1772開機路徑已完成force-interrupt與restore首切片；restore依固定clock比例
排程，在九次GPIP5 inactive輪詢後由EmuTOS讀到active-low IRQ；後續Type-I status
讀回`$E4`並清除IRQ也已與固定Hatari trace一致。第一顆drive的data-register track 0
設定與same-track seek亦已走完相同IRQ／status垂直鏈；YM2149 port A也已從drive 0
切至drive 1，準備執行第二顆drive探測。
**EmuTOS 1.3 已經開得完**：從 reset 一路走到 GEM 桌面（選單列、兩個磁碟圖示、
垃圾桶與滑鼠游標），1.2 億條指令之內沒有再撞到未建模的存取，畫面內容以 SHA-256
釘在測試裡。鍵盤與滑鼠輸入、掛磁碟載入程式都還沒接，因此目前**還不是可啟動遊戲的
模擬器**；未具備的控制命令持續明確失敗，不會假裝成功。

首版範圍固定為 Atari ST／STF：Motorola 68000、RAM／ROM 位址空間，以及遊戲需要的
GLUE、MMU、Shifter、MFP、PSG、ACIA 與 FDC 路徑。STE、TT、Falcon、DSP 與 Videl
不在第一階段。

Hatari 是外部相容性 oracle，不是程式碼來源。Atari Talos 依公開硬體規格獨立以 Go
重寫；不翻譯、移植或連結 Hatari 的 GPL 程式碼。

## JSON Lines 控制介面

每行是一個 JSON 物件；回覆沿用呼叫端提供的 `id`。未知命令、未知欄位與尚未具備的
能力一律失敗即關閉（fail-closed）。

```console
$ printf '%s\n' '{"id":"1","op":"hello"}' '{"id":"2","op":"capabilities"}' | ataritalos
{"id":"1","ok":true,"result":{"name":"Atari Talos AI Toolkit","protocol":"talos-jsonl/1","version":"0.0.1-dev"}}
{"id":"2","ok":true,"result":{"commands":["hello","capabilities","quit"],"emulation_ready":false,"machine":"atari-stf"}}
```

未完成的命令不會假裝成功：

```console
$ printf '%s\n' '{"id":"3","op":"run_frames","frames":1}' | ataritalos
{"id":"3","ok":false,"error":{"code":"not_implemented","message":"run_frames requires the Atari ST machine core"}}
```

正式控制協定見 [`docs/spec/002-jsonl-control.md`](docs/spec/002-jsonl-control.md)。Go 套件
與 CLI 將使用同一組 request／response 型別，讓 remake 可以直接在 `go test` 內嵌，
也能讓 Codex、Claude Code、shell 或其他語言透過程序介面控制。

## 建置與測試

所有建置與測試都在固定 Go Docker image 中執行：

```sh
tools/go.sh test ./...
tools/go.sh build -o bin/ataritalos ./cmd/ataritalos

# 下載固定版本的外部 CPU 語料，再跑逐 clock／bus 驗收
tools/fetch-m68000-tests.sh
TALOS_M68000_TESTS=workplace/m68000-tests/v1 tools/go.sh test ./internal/m68k

# 用真實出貨的 68000 程式碼驗收：UCSD p-System IV.2.1 直譯器
# （SunDog: Frozen Legacy 的 SYSTEM.INTERP，自備原版磁碟檔案）
TALOS_UCSD_INTERP=workplace/sundog tools/go.sh test ./internal/m68k -run UCSD
```

合成語料逐條窮舉單一指令的狀態空間；`TALOS_UCSD_INTERP` 那組跑的則是一段有目的的
程式——opcode 分派、短常數、區域變數、陣列索引、算術與布林運算，指令彼此有真實的資料
相依。兩者互補。原版素材不進 repository，檔案雜湊在測試裡釘死；未設定環境變數時該組
測試跳過。契約見 [`docs/spec/134-ucsd-psystem-interpreter-execution.md`](docs/spec/134-ucsd-psystem-interpreter-execution.md)
起的四份規格（134 執行、135 分派迴圈、136 算術、137 布林）。

## 里程碑

| 里程碑 | 完成條件 | 狀態 |
|---|---|---|
| M0 | JSON Lines 契約、CLI、Docker 測試、文件、授權與 public repo | **完成** |
| M1 | 68000 核心通過公開指令語料與邊界測試 | **基礎／控制流／control EA、完整 MOVE／MOVEA／MOVEM／MOVE USP／MOVE to CCR／SR／MOVE from SR／LINK／UNLK／EXG／ADDA／SUBA／AND／ANDI／OR／ORI／EOR／EORI／CMP／CMPI／CMPM／CMPA／ADD／ADDI／ADDQ／SUB／SUBI／SUBQ／CLR／TST／TAS／ASL／ASR／LSL／LSR／ROR／ROL／MULS／MULU／DIVS／DIVU／NOT／NEG／Scc／DBcc／BTST／BCHG／BCLR／BSET／TRAP／RTE／STOP／line-F，共 240,000 筆外部語料已通過（TAS memory timing 依 Hatari 勘誤）；另以 UCSD p-System IV.2.1 直譯器的真實出貨程式碼補一組互補驗收** |
| M2 | ST／STF 記憶體、TOS 開機與決定性時鐘 | **基礎 memory map、power-on reset、MMU、exception、`RESET`／`STOP`、空 cartridge、line-F、ST void I/O、`TST.B` bus error、MFP reset bank、固定 GPIP input、首個 external bus slot、level-4 autovector、reset 60 Hz recurring GLUE VBL、stopped-clock 快轉、video mode、16 色 palette、programmed／active framebuffer 基址已 CONFORMED；第四 VBL deadline 為 535,528** |
| M3 | Shifter 畫面、鍵鼠、FDC 與磁碟映像 | **低解析度索引畫面、drive-0 FDC探測鏈及YM2149切至drive 1已 CONFORMED；RGB／PNG、鍵鼠、drive-1探測、跨track seek與磁碟映像尚未完成** |
| M4 | breakpoint、watchpoint、trace、快照與畫面輸出 | 未開始 |
| M5 | 《Dungeon Master》與 Hatari 同狀態原版對拍 | **進行中：raw `.st`、sector DMA、連續 sector 與 A 槽雙面讀取已 CONFORMED；私人混合素材 bootstrap 只用來揭露 emulator gate，尚缺合法原版 ST 磁片，未宣稱可玩或 parity 通過** |

目前真相、限制與下一步以 [`CONTEXT.md`](CONTEXT.md) 為準。

## 原版素材與商標

本 repo 與發行包不包含 TOS ROM、遊戲磁碟、遊戲素材或其他 Atari 原版內容。使用者必須
自備合法取得的檔案。Atari、Atari ST 與相關商標屬於各自權利人；本專案與 Atari、
Hatari 或原版遊戲權利人沒有隸屬、合作、贊助或授權關係。

## 授權

本專案採 [RRSAL-1.0](LICENSE)：非商業使用、修改與再散布免費；實況、影片、評論與
一般平台分潤明示允許；商業使用另洽。這是一份 source-available 授權，不是 OSI
定義的開放原始碼授權。RRSAL-1.0 不涵蓋 TOS ROM、遊戲磁碟、商標或第三方內容。
