# 113 — ST空ACSI bus target 1–7掃描

狀態：**CONFORMED**。

## 範圍與停止線

本切片將規格112已驗證的空ACSI target-0 attempt參數化到target 1–7。
每個target都由EmuTOS重寫DMA address、toggle reset、sector count 0，送出
`target<<5`的首個command byte；無裝置時不有IRQ，guest依自己的Timer C／
`hz_200`逐一timeout。

有裝置ACSI、完整六個byte command packet、status／DMA transfer、target 7後的
floppy boot-sector讀取與非空匯流排除。本切片不以單一target通過推論全掃描；
固定trace必須直接出現八個target。

## 證據

- **已確認（EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`，`bios/acsi.c` SHA-256
  `e9065d864bd99aaf7ef5a4302cf767d195ecbb4891159c08c0ea85c86ed53243`。
  `send_command()`以`cdb[0] |= dev<<5`嵌入device target；每次attempt又完整呼叫
  `set_dma_addr()`、`hdc_start_dma()`與`timeout_gpip()`。
- **已確認（Hatari固定實作）**：Hatari 2.4.1 commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`，`src/hdc.c` SHA-256
  `1bcb763bbbf5eedb473b08ba353493aed7c2499852b5fd640e3f1af41cd52660`。
  無ACSI image時`bAcsiEmuOn=false`，所有target的`HDC_WriteCommandByte()`均無動作，
  不設HDC IRQ。
- **強證據（固定Hatari完整正常路徑）**：EmuTOS 1.3 UK ROM
  SHA-256 `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；
  65-VBL FDC／SCSI trace SHA-256
  `35742ed2a2d78ae16382ffc045fa6fea347f25b8cb0b9594dc07335889d8aea1`。
  trace直接記錄target command `$00/$20/$40/$60/$80/$A0/$C0/$E0`，分別始於
  VBL19/24/30/35/41/46/52/57；每次均依序出現`$0190→$0090→count 0→
  $0088→command→$008A`，並於VBL24/29/35/40/46/51/57/62寫`$0080`離開等待。
- **已確認（固定Talos入口）**：規格112完成target 0 timeout與第二次
  DMA setup後，於361,268 instructions、263 interrupts、4,062,736 clocks停在
  target-1 `$FF8606=$0088`，pipeline PC=`$FC1274`、prefetch=`$8606,$0045`。

## typed行為

1. 上一target已timeout、DMA setup再次抵達init stage3且mode=`$0090`時，
   `$FF8606=$0088`開始下一target；target由前值加1，且不得超過7。
2. `$FF8604`的首個command word必須等於`target<<5`，保存low byte後，
   同條long write的`$FF8606=$008A`進入等待。無裝置時IRQ與GPIP5均保持inactive。
3. 每個target的guest timeout後，`$FF8606=$0080`結束attempt；target 1–6清除
   每次DMA init/reset收據供下一輪重用，target 7則進入全掃描完成stage。
   FDC probe stage在八次ACSI timeout全程保持14。
4. target、command、mode與stage必須一致；跳target、重送、錯command、錯mode、
   user／byte access均失敗即關閉且原子拒絕。
5. 每筆word handler沿用shared-bus alignment與4 wait-clock契約。cold reset回到
   target=-1、ACSI stage0，並清timeout-return clock收據。

## 驗收

- table-driven synthetic測試覆蓋target 0–7的command計算、無IRQ、每輪timeout
  return、DMA receipt reset、最後完成stage與錯command原子拒絕。
- 固定ROM必須自然掃完target 7，鎖定各target的command與timeout clock、
  最終CPU state／clock，再有界定位第一個post-ACSI typed gate。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

## 驗收結果

- Talos自然送出八筆command `$00/$20/$40/$60/$80/$A0/$C0/$E0`；timeout-return
  clocks依序為3,771,064、4,853,396、5,976,920、7,099,340、8,222,868、
  9,345,296、10,468,836、11,591,272，期間FDC保持stage14且沒有HDC／FDC IRQ。
- target 7完成邊界為866,723 instructions／461 interrupts／11,591,284 clocks，
  CPU PC=`$FC12A8`、prefetch=`$33FC,$0000`；八筆command與clock均由typed receipt
  鎖定，並拒絕錯command與第九個target。
- 第一個post-ACSI typed gate為867,255 instructions／462 interrupts／
  11,598,096 clocks的YM2149 `$FF8800` byte write；本規格不外推其語意。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`
  與`go build ./...`均通過。
