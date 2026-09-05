# 076 — ST 50 Hz recurring VBL、E-clock IACK 與 STOP 快轉

狀態：**CONFORMED**。

## 範圍與證據

本切片只延伸規格 075 的固定普通彩色 ST／EmuTOS 1.3 UK 開機 profile：第一個 VBL
之後以 313 lines × 512 clocks 排下一個 50 Hz frame；CPU 已 STOP 且沒有更早事件時，
機器時鐘快轉到該 deadline，再走 ST 視訊 interrupt acknowledge（IACK）與規格 074
的 level-4 autovector。執行期改寫 `$FF820A`、HBL、raster、其他事件競爭及一般化
PAL／NTSC 切換仍未涵蓋，遇到時不得冒充已支援。

- **強證據（固定 Hatari oracle 實作）**：Hatari source commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`、archive SHA-256
  `ed3861b10b05283d0a97df0a9070cef5ae71293ddf4c797a82174ae50ea8877c`。
  `video.h`／`video.c` 定義 50 Hz 為 313×512 clocks 並逐 frame 重排 VBL；
  `m68000.c M68000_WaitEClock` 定義等到下一個 10-clock E-clock 邊界；
  `newcpu.c iack_cycle` 對 HBL／VBL 先走 12-clock IACK start，再等 E-clock，之後走
  實機量得的 10-clock video IACK。Talos 依公開可查契約重寫，不複製或連結 GPL 程式碼。
- **已確認（NXP 一手 CPU 規格）**：NXP《M68000 Family Programmer's Reference
  Manual》SHA-256
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`；STOP 會等待
  可接受的 interrupt，autovector frame／mask 行為沿用規格 072／074。
- **已確認（固定 EmuTOS／Hatari 實跑）**：EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`，Hatari
  `--fast-boot false` 的第二次 `$FC0446` handler 入口為 FrameCycles=124、原 SR
  `$2300`、saved PC `$FCD09E`、SSP `$F70→$F6A`，且 `$466 frclock=1`。第一個 reset
  VBL deadline 133,668 加 50 Hz frame 160,256，第二 deadline 是 293,924；從 event
  到 handler 入口 60 clocks，與下述 E-clock／IACK／ST bus 組合一致。

## typed 行為

1. `Machine.Reset` 將 `nextVBLClock` 設為 133,668。每次 deadline 到達只 latch 一個
   level-4 pending，並把下一個 deadline 增加 160,256 clocks；pending 被 mask 擋住時
   不消失，也不重複計 interrupt。
2. ST 視訊 IACK 額外 clocks 為
   `10 + ((10 - ((epoch + 12) mod 10)) mod 10)`；其中 `epoch` 是開始接受 interrupt 的
   machine clock。這段 idle 放在既有 44-clock MC68000 autovector core 之前，core 的
   timed-bus epoch 同量後移，故 ST RAM slot wait 仍由既有 bus 契約決定。
3. CPU 已 stopped、pending 尚未形成且下一個 VBL 在未來時，單次 `Machine.Step` 先以
   idle phase 快轉到 deadline，再於同一步接受 interrupt。回傳 clocks 與 timeline 必須
   同時包含快轉、視訊 IACK 與 CPU exception；`Instructions` 不增加，`Interrupts` 增加 1。
4. 固定第二 deadline 293,924 的 E-clock wait 為 4，視訊 IACK 額外共 14；CPU core
   因 ST RAM slot wait 為 46，故 handler entry 是 293,984。下一步執行 guest
   `$52B8,$0466`，`frclock` 必須由 1 變 2。
5. 本切片固定第一個 event 後皆採 50 Hz profile。若未來接入 `$FF820A` 寫入或其他視訊
   模式，必須另立 READY 規格取代這項限制，不可靜默沿用本常數。

## 驗收與停止線

- synthetic test 覆蓋 recurring deadline、mask pending、E-clock 公式、STOP 快轉、timeline
  連續性、Reset 清排程狀態。
- 固定 ROM 必須在第二 handler 對上 Hatari 的完整 D/A、SSP、SR、saved PC、prefetch，
  並真正執行 handler 第一條後讀得 `$466=2`。
- 完整 CPU corpus、既有 ST 測試、固定 ROM、`go vet -stdmethods=false ./...` 與 build
  全綠後才升 **CONFORMED**。
- 本切片不宣稱 framebuffer、完整 TOS 開機或任意 video mode；通過第二次 VBL 後只推進
  到下一個實際 typed fail-closed gate。
