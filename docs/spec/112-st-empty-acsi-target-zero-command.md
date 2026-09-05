# 112 — ST空ACSI bus的target 0 command開始

狀態：**CONFORMED**。

## 範圍與停止線

本切片處理規格111後，固定EmuTOS寫`$FF8606=$0088`選擇ACSI、
以同一條long write對`$FF8604/$FF8606`送出data `$0000`與next-control `$008A`。
Hatari未掛任何ACSI image，因此target 0不接受command、不產生HDC IRQ；
EmuTOS依自己的GPIP timeout離開等待。

本切片不實作有裝置的ACSI command packet、SCSI opcode、DMA transfer、status，
也不包含target 1–7掃描。本切片包含timeout後`$0080`回復，因為它與
先前drive-1 FDC探測使用同值，必須以ACSI與FDC狀態明確分流。無裝置不是「立即回錯誤」；
必須保持IRQ inactive，讓guest的真實timeout路徑決定後續。

## 證據

- **已確認（Hatari固定實作）**：Hatari 2.4.1 commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/hdc.c` SHA-256
  `1bcb763bbbf5eedb473b08ba353493aed7c2499852b5fd640e3f1af41cd52660`。
  `HDC_Init()`只在至少一個ACSI image成功啟用時設`bAcsiEmuOn`；
  `HDC_WriteCommandByte()`在普通ST且`bAcsiEmuOn=false`時不呼叫
  `Acsi_WriteCommandByte()`，因此不設status與IRQ。
- **已確認（EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/acsi.c` SHA-256
  `e9065d864bd99aaf7ef5a4302cf767d195ecbb4891159c08c0ea85c86ed53243`。
  `send_command()`將`DMA_CS_ACSI`加到control，`dma_send_byte()`以
  `ACSIDMA->datacontrol = MAKE_ULONG(data, control)`同時寫data與下一control；
  送完每個byte後呼叫`timeout_gpip()`，無IRQ時走timeout而非讀取status。
- **強證據（固定Hatari空ACSI正常路徑）**：EmuTOS 1.3 UK ROM
  SHA-256 `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；
  30-VBL FDC／SCSI trace SHA-256
  `844013f18cea6eb1fd69e8faa7b3015af4a6ab987267e32e2f981ea4ac7c6342`。
  VBL19 video cycle 85,226在PC `$FC126E`寫mode `$0088`；85,336在
  PC `$FC128A`同時記錄`$FF8604 data=$0000`、ACSI addr0 command `$00`與
  `$FF8606=$008A`。期間沒有HDC IRQ；直到VBL24 video cycle 47,736才寫
  `$FF8606=$0080`，與guest timeout相符。
- **已確認（固定Talos入口）**：規格111後291,386 instructions、
  3,002,576 clocks停在`$FF8606=$0088`；pipeline PC=`$FC1274`、
  prefetch=`$8606,$0045`。

## typed行為

1. DMA init stage3、mode=`$0090`時，接受supervisor word write
   `$FF8606=$0088`；保存mode，記錄ACSI stage1與target 0。bit 8沒有toggle，
   不新增DMA reset。
2. ACSI stage1、mode=`$0088`時，接受`$FF8604=$0000`作為addr0的
   第一個command byte `$00`；因target 0未啟用，只保存command收據，
   不設status、DMA error或IRQ，GPIP bit 5維持inactive high。
3. 緊接的long-write low word寫`$FF8606=$008A`；保存next-control並進
   ACSI stage3。同一條CPU long write的high-word→low-word順序由68000已驗證契約保證。
4. IRQ維持inactive，guest以Timer C／`hz_200`自然進入timeout後，接受
   `$FF8606=$0080`回到floppy mode並完成target-0 attempt；清除每次attempt的
   DMA reset/init收據，但不改變已完成的drive-1 FDC stage。drive-1初次探測的
   `$0080`只能在probe drive仍為0時命中，不可誤收此ACSI return。
5. 四筆word handler各沿用shared-bus alignment並增加4 wait clocks；這些wait不代表
   ACSI裝置已回應。
6. 錯序、錯值、user／byte access失敗即關閉且不部分改狀態。cold reset
   將ACSI stage、target與command收據清除，target回到-1。

## 驗收

- synthetic測試覆蓋`$0088→data $0000→$008A`、無IRQ、timeout後`$0080`回復、
  FDC stage不回退、錯序／錯值原子拒絕、bit-8不觸發reset、timed wait與cold reset。
- 固定ROM必須自然完成ACSI stage3，鎖定CPU state、clock、mode、target、
  command與IRQ，再保留無IRQ等待以定位guest timeout後的下一typed gate。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

## 驗收收據

- 固定Talos自然完成`$0088→data $0000→$008A`，於291,404 instructions、
  234 interrupts、3,002,700 clocks抵達ACSI stage3；target=0、command=`$00`、
  IRQ維持inactive。
- 該邊界PC=`$FC1292`、prefetch=`$4878,$0014`，D/A、SSP、SR、mode、target、
  command與IRQ均鎖入固定ROM測試。
- guest依Timer C正常進入timeout，於clock 3,771,064寫`$0080`；Talos完成
  target-0 attempt且保持FDC stage14，沒有誤重開drive-1 probe。
- 後續第二次DMA address／reset／sector-count setup自然完成；於
  361,268 instructions、263 interrupts、4,062,736 clocks抵達下一gate
  `$FF8606=$0088`，pipeline PC=`$FC1274`、prefetch=`$8606,$0045`，這是target 1
  attempt的ACSI mode。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過。
