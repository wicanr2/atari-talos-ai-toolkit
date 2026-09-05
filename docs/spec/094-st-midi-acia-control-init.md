# 094 — ST MIDI ACIA control initialization

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定 EmuTOS 1.3 UK 對 MIDI MC6850 ACIA control `$FFFC04` 的 master
reset `$03` 與 configuration `$95`。MIDI data `$FFFC06`、外部clock、TX/RX shift、IRQ
與host MIDI I/O不在範圍。

- **已確認（Atari ST／MC6850平台契約）**：MIDI ACIA control/status位於 `$FFFC04`、
  data位於 `$FFFC06`；control bits 1–0=`11`是master reset，status bit1是TDRE。
- **強證據（固定Hatari 2.4.1外部oracle）**：EmuTOS ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；16-VBL
  `acia,midi,midi_raw,io_write,io_read` trace在VBL10/HBL196由PC `$FC6416`寫
  `$FFFC04=$03`，再由PC `$FC641A`寫 `$FFFC04=$95`。
- **已確認（固定Talos正常路徑）**：規格099接入Timer C後，在136,125 instructions、
  23 interrupts、1,579,268 clocks完成 `$FFFC04` 的 `$03→$95` sequence。

## typed行為與驗收

1. cold reset／MC68000 RESET清MIDI ACIA control、status與configured state。
2. 唯一允許序列為 `$00→$03` master reset，再 `$03→$95` configuration；master reset
   將status設TDRE=`$02`，configuration保留並標記configured。
3. 錯序、重複、其他control、data access、user access與word access均失敗即關閉且
   原子不變。byte access採既有ACIA 4 wait clocks；不宣稱外部MIDI clock parity。
4. synthetic測試、固定ROM、完整corpus、全測試、vet與build通過，並抵達下一個typed
   gate後才升 **CONFORMED**。

## 驗收收據

- synthetic測試確認`$03→$95`、4 wait clocks、reset與所有範圍外存取的fail-closed。
- 固定ROM通過兩筆control write後進入ACIA channel 6 MFP enable序列；完整路徑收據見
  規格096。
- 完整corpus、全測試、`go vet`與建置結果記錄於專案驗證矩陣。
