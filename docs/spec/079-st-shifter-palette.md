# 079 — ST Shifter 16 色 palette word bank

狀態：**CONFORMED**。

## 範圍與證據

本切片建立普通 ST／STF `$FF8240–$FF825E` 的 16 個 word palette registers、ST 9-bit
color mask、cold reset state 及固定 EmuTOS 的完整初始化迴圈。palette byte write 的雙 byte
鏡像、STF read unused bits 的 bus-dependent 不定值、raster palette history、Spec512 與實際
RGB framebuffer 轉換均未涵蓋。

- **已確認（Atari 一手硬體規格）**：Atari Corporation《Engineering Hardware
  Specification of the Atari ST Computer System》，1986-01-07，既有收據 SHA-256
  `eb3a001ed636123f94c9c612ab33b6de2b1b118177ea01cfb971bf3ae17e6044`；hardware map
  定義 `$FF8240–$FF825E` 16 個 palette words，ST color layout 為 `RRr GGr BBb`，
  即有效 mask `$0777`。
- **強證據（固定 Hatari oracle 實作）**：Hatari source commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`、archive SHA-256
  `ed3861b10b05283d0a97df0a9070cef5ae71293ddf4c797a82174ae50ea8877c`。
  `ioMemTabST.c` 將 16 個偶數位址逐一映射為 word R/W；`Video_ColorReg_WriteWord`
  先做 Shifter 4-clock bus alignment，再對 ST 值 mask `$0777`。byte write 會把被寫 byte
  同時複製到另一 byte；`Video_ColorReg_ReadWord` 的 bits 11／7／3 在 STF 依 bus 活動不定，
  固定 ROM 取指路徑則保留為 0。本專案不複製或連結 GPL 程式碼。
- **已確認（固定 EmuTOS／Hatari 實跑）**：EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
  `$FC671A: 30 C1` 是 `MOVE.W D1,(A0)+`。Hatari 首次執行前 CycleCounter=401,372、
  `A0=$FFFF8240`、`D1=$0777`；8 clocks 後 `$FF8240=$0777`、A0=`$FFFF8242`。
  迴圈在 `$FC6722` 結束時 CycleCounter=402,058、A0=`$FFFF8260`，16 words 為
  `0777,0700,0070,0770,0007,0707,0077,0555,0333,0733,0373,0773,0337,0737,0377,0000`。

## typed 行為

1. `Memory` 保存 `[16]uint16` palette；cold reset 清零。合法 register index 為
   `(address-$FF8240)/2`，只接受 supervisor、偶數 word access與24-bit alias。
2. word write原子保存 `value & $0777`；word read回保存值，unused bits固定為 0。
   這個 deterministic read只涵蓋固定 ROM／非 RAM protection path，不冒稱重現 STF
   bus-dependent bits 11／7／3。
3. `$FF823E/$FF8260` 不因 palette bank 擴張；palette byte access保持 fail-closed，直到
   特殊 byte mirroring 另有 READY 規格。long access若由 CPU 拆為兩個合法 word，依序
   更新兩色；本切片不另加原子 long transaction。
4. `ReadWordAt`／`WriteWordAt` 沿用 Shifter 4-clock boundary wait。固定 guest 的
   `MOVE.W D1,(A0)+` 在目前 phase 無額外 wait，為 8 clocks，並正常提交 A0 postincrement。
5. 本切片保存 palette state供未來 framebuffer 消費，但不產生像素，也不保存 scanline
   中途的 palette history。

## 驗收與停止線

- synthetic tests 覆蓋 16 indices、mask、readback、reset、alias、邊界、odd／byte、user、
  timed wait，並確認失敗不改相鄰色。
- 固定 ROM 必須以完整 D/A、SSP、SR、prefetch 對上首筆寫入前後，並正常跑完整迴圈；
  `$FC6722` 的 16 words、A0、A1、D1 與 Hatari 一致。
- 完整 corpus、ST tests、固定 ROM、`go vet -stdmethods=false ./...` 與 build 全綠後才升
  **CONFORMED**；再有界續跑至下一 typed gate。
