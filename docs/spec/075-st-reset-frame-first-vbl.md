# 075 — ST 彩色 reset frame 的第一個 VBL pending

狀態：**CONFORMED**。

## 範圍與證據

本切片只建立普通 ST／STF 彩色 profile 從 cold reset 起算的第一個 GLUE VBL event、
level-4 pending latch 與 CPU instruction-boundary 接受。後續 frame、執行期 50／60 Hz
切換、HBL、Shifter raster、畫面輸出及 stopped CPU 尚未到期時的快轉另立規格。

- **已確認（Atari 一手硬體規格）**：Atari Corporation《Engineering Hardware
  Specification of the Atari ST Computer System》，1986-01-07，保存掃描
  `GEM_0904.pdf` SHA-256
  `eb3a001ed636123f94c9c612ab33b6de2b1b118177ea01cfb971bf3ae17e6044`。
  ST video／GLUE 提供第 4 級 vertical blank interrupt；CPU vector 表 `$70` 是 level 4。
- **強證據（固定 Hatari oracle 實作）**：Hatari source commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`、archive SHA-256
  `ed3861b10b05283d0a97df0a9070cef5ae71293ddf4c797a82174ae50ea8877c`。
  普通彩色 ST reset 將 video frequency register 清為 0，初始採 60 Hz；一 frame 是
  263 lines × 508 CPU clocks，STF VBL offset 是 64，因此第一個 event deadline 為
  133,668 clocks。Hatari 保留被 SR mask 擋住的 pending level，直到 instruction
  boundary 可接受。本專案只依此公開可查 oracle 契約重寫，不複製或連結 GPL 程式碼。
- **已確認（固定 EmuTOS／Hatari 同狀態）**：EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
  Hatari 以 `--fast-boot false` 的第一個 `$FC0446` VBL handler 入口為 VBL=1、
  FrameCycles=45,520；原 SR `$2300`、
  saved PC `$FC6904`、SSP `$F74→$F6E`、新 mask 4，handler 第一條
  `ADDQ.L #1,$466` 前 `frclock=0`。其後 `$FC6904` consumer 在 FrameCycles=45,608、
  `frclock=1`，證明事件
  必須在該 instruction 之前插入並由 guest handler 寫入。
- **已確認（可丟棄 Talos 探針與同 profile 勘誤）**：Talos 在 133,668 clocks 形成
  pending 時 SR=`$2700`，故保留到 pipeline PC=`$FC6908`（當前 opcode `$FC6904`）、
  SR=`$2300` 才接受。Talos 此時的 D4／D7／A5 是 `$80000/$1/$FC01F4`；Hatari 必須
  使用 `--fast-boot false` 才是同一個未修改 ROM profile，三者及其餘 D/A 全部相同。
  `--fast-boot true` 會跳過部分開機工作而留下 0，不可拿來反駁 raw-ROM Talos 狀態。

## typed 行為

1. `Machine.Reset` 採用固定 first-VBL deadline 133,668，清 pending、raised 與 interrupt
   counter。這是目前唯一支援的 fixed color ST reset profile；沒有泛化為任意模式。
2. 每條 CPU instruction 完成後，若全機 clocks 首次跨過 deadline，就只設定 level-4
   pending。SR mask 不會讓 event 消失，也不得提前讀 vector 或寫 `frclock`。
3. 每次 `Machine.Step` 在下一條 instruction 前先嘗試接受 pending。mask 阻擋時照常執行
   instruction；可接受時呼叫規格 074 的 CPU autovector，增加 `Interrupts`、不增加
   `Instructions`，並清 pending。此切片原先只驗收 44-clock exception core 加 ST bus
   slot wait；規格 076 已補上位於 core 前的 E-clock／video IACK，固定第一個 handler
   entry 因而由 177,996 clocks 訂正為 178,012。register、frame 與 guest 行為不變。
4. 下一次 Step 才執行 handler 第一條；`$466` 必須由 ROM opcode `$52B8,$0466` 改成 1。
5. 本切片原本在第一個 event raised 後不排下一個 deadline；後續 recurring deadline 與
   STOP 快轉現由規格 076 接手，不得用第一 frame 常數反覆假造 event。

## 驗收與停止線

- synthetic test 覆蓋 deadline 前後、mask 保留、interrupt action 不計 instruction，
  Reset 清 state。
- 固定 ROM 必須對上 Hatari 第一 handler 的完整 D/A、SSP、SR、saved PC、prefetch，且
  真正執行第一條後讀得 `$466=1`。
- 完整 CPU corpus、既有 ST 測試、固定 ROM、vet 與 build 全綠才升 **CONFORMED**。
- 本切片不宣稱 recurring VBL、framebuffer 或完整 TOS 開機；下一 gate 由 handler
  後續實際第一個未支援行為決定。
