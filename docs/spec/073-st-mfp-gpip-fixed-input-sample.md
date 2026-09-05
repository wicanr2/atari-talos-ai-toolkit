# 073 — ST MFP GPIP fixed input sample

狀態：**CONFORMED**。

## 範圍與證據

本切片擴充規格 061：為目前固定的 headless Atari ST／STF color profile，讓 MFP GPIP
`$FFFA01` byte read 依 DDR 合併外部 input pins，而不是永久回傳 MFP reset 時的內部
latch。只定義首段 EmuTOS monitor probe 所需的穩定 idle sample `$A1`；動態 printer、
RS-232、blitter、ACIA、FDC transition、edge interrupt 與其他 monitor profile 不在範圍。

- **已確認（MC68901 一手規格）**：NXP《MC68901 Multi-Function Peripheral》user
  manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`。
  DDR bit=0 時 GPIP 為 high-impedance input，read 應取得外部 pin；DDR bit=1 才回傳
  output latch。寫 GPIP 不得覆蓋 input bits。
- **已確認（固定 EmuTOS 1.3）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/screen.c:119-127
  shifter_get_monitor_type`，檔案 SHA-256
  `1fd34394b4490124d75c8d5d4b0a82bcaaef3efa2ba25053cfca814681e5c4dc`。
  程式直接讀 `$FFFFFA01` bit 7；1 代表 color、0 代表 monochrome。固定 ROM 中對應
  `$FC67B8 MOVE.B $FA01,D0`，隨後以 `EXT.W`／`LSR.W #15` 抽取該 bit。
- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/mfp.c` SHA-256
  `610e30dc75acf0d0f802b0712e899be83f0926b8d1e54d1c0bea85466bcfc69b`。
  `MFP_GPIP_ReadByte_Main:1882-1921` 依 DDR 保留 output bits、更新 input bits；
  `MFP_Main_Compute_GPIP7:1832-1860` 對 ST color monitor 回 bit 7=1；未啟用 printer
  時 bit 0=1。GPIP bit 5 是 active-low FDC/HDC idle signal。
- **已確認（Hatari 外部 oracle）**：固定 image／ROM 在 `$FC67B8` 前 cached GPIP
  顯示 `$20`；執行 GPIP read 及後續抽取後，GPIP=`$A1`、D0=`$1`，FrameCycles
  `45300→45360`。因此本 profile 的首次 bus sample 是 bit 7 color + bit 5 FDC idle +
  bit 0 no-printer busy，即 `$A1`。
- **已確認（Talos 差異定位）**：規格 071 的第 7,563 條邊界 D2/D3 與 Hatari 相同；
  到 STOP 前 Talos 首次分歧源自 `$FC67B8` 仍讀 `$00`。`$FC68FE MOVE SR,D2`
  因前述 LSR 結果而保存 Talos `$2704`、Hatari `$2710`，之後 monitor table 選擇令
  Talos D3=`$0`、Hatari D3=`$1`。不是 STOP 或 shift opcode 本身的差異。

## typed 行為

1. `Memory` 保存獨立的 GPIP output/current latch 與固定 external input sample `$A1`。
   cold reset／MC68000 `RESET` 清內部 GPIP 與 DDR，但外部 idle pins 仍是 `$A1`。
2. supervisor byte read 先套用
   `(currentGPIP & DDR) | (externalInputs & ^DDR)`，保存並回傳合併結果；仍增加 4 wait
   clocks。DDR=`$00` 時固定 profile 首次讀回 `$A1`。
3. GPIP write 沿用規格 061，只修改 DDR=1 的 output bits並保留當前 input bits。
4. 本切片不產生 GPIP edge／pending IRQ；external input sample 固定不變。其他 profile
   或裝置開始驅動 pin 前必須另立 READY 規格，不能沿用 `$A1` 冒充動態周邊。

## 驗收與停止線

- memory test 覆蓋 reset 後 `$A1` sample、DDR output/input merge、write preservation、
  24-bit alias、user／word fault與 4 wait clocks。
- 固定 EmuTOS 應在 STOP 前得到 D2=`$2710`，修正 monitor-detect 分歧。本規格完成時
  D3 因 `$466 frclock` 尚無 VBL producer；規格 075 已由第一個 VBL guest handler
  補上，現為雙方 `$1`。opcode `$FCD09A`、SR／prefetch、D/A／stack 對上固定 Hatari，
  現行 gate 於第 7,604 條進入 stopped state。
- 完整 232,500 筆 CPU corpus、固定 ROM、Go 測試、靜態檢查與建置均須通過，才可
  將狀態改為 **CONFORMED**。

## 玩家路徑、存檔與權利邊界

此切片修正固定 color ST profile 的 monitor detection，直接影響後續 Shifter 模式選擇，
但尚未繪製畫面或建立 IRQ。專案不收錄 ROM、手冊或 Hatari 程式碼；只保存雜湊、
定位與獨立導出的 typed 契約。
