# 077 — ST Shifter resolution reset／低解析度同值寫入

狀態：**CONFORMED**。

## 範圍與證據

本切片只建立普通彩色 ST／STF 的 `$FF8260` byte register、cold／external reset
低解析度值，以及固定 EmuTOS 的 `$00→$00` 同值寫入。切換至 medium／high resolution、
非法值 3、frame 中途 GLUE／Shifter 生效差、palette mask、raster 與 framebuffer 均未涵蓋；
非零寫入必須以 `unsupported_device_state` 失敗即關閉。

- **已確認（Atari 一手硬體規格）**：Atari Corporation《Engineering Hardware
  Specification of the Atari ST Computer System》，1986-01-07，既有專案收據 SHA-256
  `eb3a001ed636123f94c9c612ab33b6de2b1b118177ea01cfb971bf3ae17e6044`；hardware map
  將 `$FF8260` 定義為 Shifter／GLUE resolution byte register，bits 1–0 的 0／1／2
  分別選 low／medium／high resolution。
- **強證據（固定 Hatari oracle 實作）**：Hatari source commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`、archive SHA-256
  `ed3861b10b05283d0a97df0a9070cef5ae71293ddf4c797a82174ae50ea8877c`。
  `ioMemTabST.c` 只映射 `$FF8260` 為 byte R/W；`video.c Video_Reset_Glue` 在彩色 profile
  reset 寫 low-res 0，`Video_Res_ReadByte` 對 STF read 將 unused bits 7–2 設為 1，且
  `Video_Res_WriteByte` 說明 GLUE 先取值、Shifter bus access 再對齊 4-clock boundary。
  本專案只依公開契約重寫，不複製或連結 GPL 程式碼。
- **已確認（固定 EmuTOS／Hatari 實跑）**：EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
  ROM `$FC69E6` bytes `11 C0 82 60` 是 `MOVE.B D0,$8260`（absolute-short sign extension
  後為 `$FFFF8260`）。Hatari 2.4.1、`--fast-boot false` 在 VBL=3、FrameCycles=384
  執行前 `D0=0`、register read 低兩 bits=0；指令後 FrameCycles=396，故為 12 clocks，
  D/A、SR、SSP 不變並前進至 `$FC69EA`。

## typed 行為

1. `Memory.ColdReset` 與 MC68000 external `RESET` 將 Shifter resolution state 設為 0；
   RAM 不受影響。
2. supervisor byte read `$FF8260` 在普通 STF profile 回 `state | $FC`。目前 state 只能是 0，
   因此回 `$FC`；24-bit alias 相同。user access仍依 I/O protection fault。
3. supervisor byte write只接受目前 state=0 且 value=0，不改其他 state；非零 value 或未來
   非零 state 一律回 `unsupported_device_state` 且不提交。`$FF8261` 與 word access 不因
   本切片擴張。
4. `ReadByteAt`／`WriteByteAt` 沿用 Shifter register 的 4-clock bus boundary wait。
   固定 EmuTOS opcode 使用既有 `MOVE.B Dn,abs.w` 12-clock CPU 路徑，不新增 opcode 特例。
5. 寫入成功後只證明低解析度初始化同值；不得據此宣稱 framebuffer 或執行期 resolution
   switching 已完成。

## 驗收與停止線

- synthetic tests 覆蓋 reset/read、`0→0`、非零原子失敗、24-bit alias、user protection、
  相鄰 `$FF825F/$FF8261`、word width 與 timed bus wait。
- 固定 ROM 必須完成 `$FC69E6`，以 12 clocks 到 `$FC69EA`，並對上 Hatari 的完整
  D/A、SSP、SR、prefetch 與 register state。
- 完整 CPU corpus、ST tests、固定 ROM、`go vet -stdmethods=false ./...` 與 build
  全綠後才升 **CONFORMED**；之後有界續跑到下一個 typed gate。
