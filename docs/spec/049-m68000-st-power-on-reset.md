# 049 — MC68000／Atari ST power-on reset

狀態：**CONFORMED**。

## 範圍與證據

本規格涵蓋從已建立的 ST reset ROM shadow 載入 SSP／PC、建立第一個 prefetch，並以
可重複的 machine reset epoch 執行第一條 TOS 指令；不涵蓋 `RESET` opcode 或周邊晶片 reset。

- NXP 官方《M68000 Family Programmer's Reference Manual》附錄 B：vector 0／offset
  `0x000` 是 Reset Initial Interrupt Stack Pointer，vector 1／offset `0x004` 是 Reset
  Initial Program Counter。PDF SHA-256：
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- CONFORMED spec 035：Atari ST reset 時低 8 bytes 讀到主 TOS ROM 開頭；主 ROM 位於
  `0xFC0000–0xFEFFFF`，reset 期間為 supervisor program access。
- EmuTOS 1.3 UK 192 KiB ROM SHA-256：
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。前 8 bytes 為
  `60 2e 01 04 00 fc 00 30`，因此 reset SSP 是 `0x602e0104`、初始 PC 是
  `0x00fc0030`；handler 前兩個 words 是 `0x6000 0x001c`。
- Hatari 2.4.1 image ID
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`，
  `--machine st --memsize 1 --fast-boot true`。首次 debugger 可觀測的 ROM 範圍狀態在
  `PC=0xfc004e`、FrameCycles=10，仍保留 `ISP/A7=0x602e0104`，且 S=1、IMASK=7、
  SR=`0x2700`；ROM bytes 確認首指令是 `BRA.W $001c`。這同時確認首條指令的目標與
  10 clocks，且 TOS 在自行重設 stack 前確實接受上述暫時無效 SSP，不得由模擬器代換。

## typed 行為

1. `CPU.Reset` 以 FC=6 依序讀取 words `0x000000`、`0x000002`、`0x000004`、
   `0x000006`，組成 big-endian SSP 與初始 PC；再由初始 PC 與 `PC+2` 讀兩個
   supervisor program words。
2. 成功後 SR 固定為 `0x2700`，SSP 使用完整 vector 0 long，prefetch 為目標前兩個
   words，CPU 的 next-prefetch PC 為初始 PC+4，符合既有 `State.PC` 契約。
3. D0–D7、A0–A6 與 USP 不由 reset API 猜測清零；新建 machine 的零值來自建構狀態，
   反覆 hardware reset 則保留未由 reset 契約定義的暫存器。
4. 初始 PC 為奇數、bus 缺失或任何 vector／prefetch read 失敗時，reset 失敗即關閉，
   不提交 staged CPU reset state。
5. `st.Machine.Reset` 成功後將 machine 的 instruction／clock 計數歸零；reset exception
   自身的 pin-level clocks 定義在 epoch 之前，本切片不以未證實常數灌入執行時間。
6. `st.Machine.Step` 只在 CPU instruction 成功時提交 instruction count 與 clocks；真實
   EmuTOS 首條 `BRA.W` 後計數為 1／10，CPU next-prefetch PC=`0xfc0052`，prefetch=
   `0x46fc,0x2700`，SSP／SR 不變。

## 驗收與排除範圍

- synthetic ROM 驗 vector shadow、FC、SSP、SR、PC、prefetch、暫存器保留與 counters。
- 指定 `TALOS_TOS_ROM` 時，以固定 SHA-256 的真實 EmuTOS 驗 reset 的
  `SSP=602e0104`、`PC=fc0034`、prefetch=`6000,001c`，以及首指令後的
  `PC=fc0052`、prefetch=`46fc,2700`、10 clocks。
- CPU `RESET`／STOP、GLUE／MMU／Shifter／MFP／PSG／ACIA／FDC reset、pin-level reset
  clocks、TOS 後續 opcode 與可開機聲明均不在本切片。

## CONFORMED 收據

- 2026-08-23：synthetic ROM 通過 reset vector／FC／SSP／SR／PC／prefetch、
  staged failure、重複 reset 與 machine counters 驗收。
- 同一實作以唯讀 EmuTOS 1.3 UK 192 KiB ROM 驗證：reset 後
  `SSP=602e0104`、`SR=2700`、`PC=fc0034`、prefetch=`6000,001c`；第一條
  `BRA.W` 後為 10 clocks、`PC=fc0052`、prefetch=`46fc,2700`，與 Hatari 觀測一致。
- 完整 Go 測試含 227,500 筆既有 MC68000 外部單步語料全數通過；
  `go vet -stdmethods=false ./...` 與 `go build ./...` 通過。
