# 081 — ST 第四幀 VBL active framebuffer 基址重載

狀態：**CONFORMED**。

## 範圍與證據

本切片承接規格 078／080，只建立固定普通彩色 ST／EmuTOS 1.3 UK 開機路徑中，
第三幀由 60 Hz 切為 50 Hz 後的第四個 VBL，如何把 programmed base `$0F8000`
提交為 active framebuffer base。一般 50 Hz HBL 310／cycle 48 的提前重載、screen
counter registers、raster fetch、像素解碼與 framebuffer 輸出仍未涵蓋。

- **已確認（Atari 一手硬體規格）**：Atari Corporation《Engineering Hardware
  Specification of the Atari ST Computer System》，1986-01-07，既有收據 SHA-256
  `eb3a001ed636123f94c9c612ab33b6de2b1b118177ea01cfb971bf3ae17e6044`；Shifter display
  address 由 `$FF8201/$FF8203` 指定，普通 ST 低 byte 不存在。
- **強證據（固定 Hatari oracle 實作）**：Hatari source commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`、archive SHA-256
  `ed3861b10b05283d0a97df0a9070cef5ae71293ddf4c797a82174ae50ea8877c`。
  `Video_RestartVideoCounter` 以 programmed registers 更新 `VideoBase`；一般硬體重載點
  是新 VBL 前三條 HBL，但 `Video_ClearOnVBL` 亦於每個 VBL 無條件重載一次。
  固定 `video_hbl` trace 的前三次重載都在 HBL 260，值為 0；第三幀切成 50 Hz 後仍於
  HBL 262 結束，沒有 HBL 310 重載紀錄，故第四個 VBL 必須由 `Video_ClearOnVBL`
  提交 `$0F8000`。本專案不複製或連結 GPL 程式碼。
- **已確認（固定 EmuTOS／Hatari 實跑）**：EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。Hatari 在
  CycleCounter=535,524、VBL=3、HBL=263 時 active base仍為 0，下一條
  `$FC299E: 49 EF 00 1C`（`LEA $1C(A7),A4`）跨過既有 VBL deadline 535,528；
  8 clocks 後 CycleCounter=535,532、VBL=4、FrameCycles=68，active base=`$0F8000`，
  programmed registers仍為 `$0F/$80`。

## typed 行為

1. `Memory` 分開保存 programmed 與 active base；cold reset兩者皆為 0。
   `ProgrammedVideoBase()` 不改語意，`ActiveVideoBase()` 只回最後一次 VBL 提交值。
2. 每次 `Machine` 形成 VBL event 時，先把當下 programmed base 原子複製到 active base，
   再 latch level-4 pending並排下一個 deadline。這對應 `Video_ClearOnVBL` 的保底重載；
   不冒稱已實作正常幀更早的 HBL 260／310 reload。
3. CPU instruction 跨過 deadline 時，active base在該 instruction 完成後可觀察為新值；
   event deadline仍是 535,528。Hatari 的邊界是 535,524→535,532；Talos 因規格 080
   已記錄的既有 24-clock 累積差距，正常 guest 邊界是 535,520→535,530。兩者都跨同一
   event deadline，但本切片不得宣稱 CPU instruction boundary 已收斂。
4. CPU 已 STOP 而 `Machine.Step` 快轉到 VBL 時亦走同一提交入口，不建立第二套規則。
   active base更新不讀 RAM、不產生像素，也不改 CPU clock。

## 驗收與停止線

- synthetic tests 覆蓋 reset、programmed／active 分離、running instruction 跨 deadline、
  STOP 快轉、重複同值 VBL，以及 22-bit DMA 高位遮罩後的基址。
- 固定 ROM 必須證明 Talos 535,520 boundary 前 active=0，跨過 deadline 後 clocks=535,530、
  active=`$0F8000`、programmed=`$0F8000` 且 pending VBL；Hatari 的 535,524→535,532
  收據作為同 event oracle，24-clock 累積差距維持已知限制。
- 完整 corpus、ST tests、固定 ROM、`go vet -stdmethods=false ./...` 與 build 全綠後才升
  **CONFORMED**；下一切片必須處理正常 50 Hz 幀的 HBL 310 提前重載或實際 pixel consumer，
  不得以本 VBL 保底重載冒稱 Shifter raster timing 完成。
