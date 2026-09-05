# 078 — ST `$FF820A` 第 0 線 60→50 Hz 切換

狀態：**CONFORMED**。

## 範圍與證據

本切片只建立普通彩色 ST／STF video synchronization byte register，以及固定 EmuTOS
在第三個 VBL 的第 0 掃描線末端由 60 Hz（0）寫成 50 Hz（2）的轉換。任意 raster
位置、50→60、external sync、high-resolution interaction、border trick 與 framebuffer
均未涵蓋；超出固定轉換者失敗即關閉。

- **已確認（Atari 一手硬體規格）**：Atari Corporation《Engineering Hardware
  Specification of the Atari ST Computer System》，1986-01-07，既有收據 SHA-256
  `eb3a001ed636123f94c9c612ab33b6de2b1b118177ea01cfb971bf3ae17e6044`；`$FF820A`
  是 video synchronization mode byte register，bit 1 選 50／60 Hz。
- **強證據（固定 Hatari oracle 實作）**：Hatari source commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`、archive SHA-256
  `ed3861b10b05283d0a97df0a9070cef5ae71293ddf4c797a82174ae50ea8877c`。
  `ioMemTabST.c` 映射 `$FF820A` byte R/W；`Video_Reset_Glue` reset 寫 0；
  `Video_Sync_ReadByte` 對 ST／STE 將 unused bits 7–2 設 1；`Video_Sync_WriteByte`
  只採 bit 1、同值不動作，異值依當下 frame／line position 更新 GLUE state。
- **已確認（固定 EmuTOS／Hatari 實跑）**：EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
  ROM `$FC6A02` bytes `11 C1 82 0A` 是 `MOVE.B D1,$820A`，Hatari VBL=3 執行前
  D1=2、FrameCycles=460、CycleCounter=401,272、register=0；12 clocks 後
  `$FC6A06`、FrameCycles=472、CycleCounter=401,284、register=2。下一 VBL counter=4
  的第一個 instruction boundary 是 CycleCounter=535,532、FrameCycles=68，故 event
  deadline 是 535,528。相對第三 event 400,876 增加 134,652；這等於原 60 Hz frame
  133,604 加上剩餘 262 lines 各延長 4 clocks（1,048）。

## typed 行為

1. `Memory.ColdReset`／external `RESET` 將 sync state 設為 0。STF supervisor byte read
   回 `state | $FC`，故 reset 為 `$FC`、50 Hz 為 `$FE`；user、alias、相鄰位址及寬度
   沿用 I/O memory map 契約。
2. byte write接受 0→0、0→2 與 2→2；只保存 bit 1。其他 value 或 2→0 因未建模
   external sync／反向 raster transition，回 `unsupported_device_state` 且不提交。
3. 0→2 形成一次 typed transition latch。`Machine.Step` 在成功 instruction 後消費它：
   固定本 profile 將尚未到期的 `nextVBLClock` 加 1,048，從 534,480 修正為 535,528，
   並將後續 frame period 由 133,604 改為 160,256。
4. transition 只允許在第三 event 已 raise、`nextVBLClock=534,480` 的固定開機狀態消費；
   其他 machine epoch／deadline 組合回 typed machine error，保留已完成的 CPU instruction
   與 register write但不改 scheduler，不可用同一修正常數外推任意 raster。
5. 固定 `$FC6A02` 沿用既有 `MOVE.B Dn,abs.w` 12-clock 路徑；寫入後 D/A、SSP、SR
   不變，PC／prefetch 前進至 `$FC6A06`。

## 驗收與停止線

- synthetic tests 覆蓋 reset/read、同值、0→2、非零非法／反向原子失敗、reset 清 transition、
  alias／權限／寬度與 machine deadline／period 調整。
- 固定 ROM 必須以完整 D/A、SSP、SR、prefetch 在 401,270 clocks 抵達 `$FC6A02`
  （Talos instruction boundary 比 Hatari debugger 收據早 2 clocks），以 12 clocks 完成，
  register read `$FE`，且 `nextVBLClock=535,528`、後續 period=160,256。
- machine integration 必須在固定寫入後排出第四 VBL deadline 535,528；正常 guest 路徑
  會先遇到 `$FF8240` palette write，因此不得用 direct-entry 冒稱已跑抵第四 VBL。
  完整 corpus、ST tests、固定 ROM、`go vet -stdmethods=false ./...` 與 build全綠後才升
  **CONFORMED**。
