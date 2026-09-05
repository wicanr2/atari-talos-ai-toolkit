# 110 — ST floppy DMA位址暫存器

狀態：**CONFORMED**。

## 範圍與停止線

本切片建模ST／STF floppy／ACSI DMA共用的24-bit位址暫存器
`$FF8609/$FF860B/$FF860D`的byte read/write、22-bit高位限制、word alignment與
ST ripple-carry write語意；並驗證固定EmuTOS於規格109後依low→middle→high
寫入`$001004`。

sector count、DMA status，FDC／ACSI transfer、FIFO、DRQ、bus arbitration、對RAM的
實際讀寫與STE／TT／Falcon擴充不在範圍；本切片不因位址暫存器可設定
就宣稱DMA transfer已實作。

## 證據

- **已確認（EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/dma.c` SHA-256
  `5792c7e82a9807988eadd731e3a7a501bde67631f3ed5aa0bf5f48fb5b028004`。
  `set_dma_addr()`明確依low、middle、high順序寫`addr[3]`、`addr[2]`、`addr[1]`。
- **已確認（Hatari固定實作）**：Hatari 2.4.1 commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/fdc.c` SHA-256
  `7de0d37a0972d6de43d21dc6f653ee6e3b736b611b978c9de221bb9b938f66f7`，
  `src/ioMemTabST.c` SHA-256
  `c0214b586bdd32a1f3d50f91827ce6b84f1fd6411b417838193b03eadde4f631`。
  I/O table將三個奇數位址各自登記為byte R/W；`FDC_DmaAddress_WriteByte()`
  先由三個register重建24-bit值，ST依bit 7／15的1→0偵測ripple carry，
  `FDC_WriteDMAAddress()`再將高byte限制為`$3F`、將low bit 0清為0。
- **強證據（固定Hatari正常路徑）**：EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；
  20-VBL FDC trace SHA-256
  `62eebc6133c11bfbb2e7d734cf78b752b82c043539d6179289003ce2d1d7bad3`。
  VBL19依次在PC `$FC35F8/$FC35FE/$FC3604`寫`$FF860D=$04`、
  `$FF860B=$10`、`$FF8609=$00`，合成位址`$001004`；三筆trace的video cycle
  為84,144／84,164／84,184。ROM `$FC35F8`起原始bytes為
  `11 EF 00 07 86 0D 11 EF 00 06 86 0B 11 EF 00 05 86 09`。
- **已確認（固定Talos入口）**：規格109後291,291 instructions、
  3,001,516 clocks的下一gate是`$FF860D`的supervisor byte write，
  pipeline PC=`$FC3600`、prefetch=`$860D,$11EF`。

## typed行為

1. `$FF8609/$FF860B/$FF860D`分別對應24-bit DMA address的bits 23–16、15–8、7–0；
   只接受supervisor data byte access，user、word與long不由本切片放行。
2. 每次byte write先用新byte與其餘兩個現值重建候選24-bit位址。ST profile下：
   - 若舊low bit 7=1且候選low bit 7=0，候選值加`$000100`；
   - 否則，若舊middle bit 7=1且候選middle bit 7=0，候選值加`$010000`。
3. 將候選值以`$3FFFFE`遮罩：high只保留六位，low bit 0永遠為0；
   三個read均回傳遮罩後的當前byte。
4. cold reset將DMA address清為0。位址暫存器寫入不改變dma mode、FDC command／
   status、IRQ、probe drive，也不自動讀寫RAM。
5. 三個handler本身不增加WD1772額外4-clock wait；仍使用現有ST shared-bus
   slot alignment契約。固定ROM的三條`MOVE.B d16(A7),abs.w`各為20 clocks。

## 驗收

- synthetic測試覆蓋三個byte的獨立read/write、`$3F`高位mask、even-address mask、
  low與middle ripple carry、不相關FDC狀態保留、user／word access拒絕與reset。
- 固定ROM必須自然依low→middle→high寫成`$001004`，鎖定三條指令後的
  CPU state、clock與DMA address，再有界定位下一typed gate。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

## 驗收收據

- 固定Talos自然完成`$FF860D=$04→$FF860B=$10→$FF8609=$00`，三條
  `MOVE.B d16(A7),abs.w`各為20 clocks；於291,294 instructions、234 interrupts、
  3,001,576 clocks形成DMA address `$001004`。
- 完成邊界PC=`$FC360E`、prefetch=`$4E75,$326F`，D/A、SSP、SR與三段write
  收據均鎖入固定ROM測試。
- 有界續跑至291,343 instructions、3,002,130 clocks；下一gate為
  `$FF8606=$0190`的supervisor word write，pipeline PC=`$FC122A`、
  prefetch=`$8606,$2239`。固定Hatari trace證實此寫入會reset DMA。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過。
