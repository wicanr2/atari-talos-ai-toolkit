# 111 — ST DMA toggle reset與sector count zero

狀態：**CONFORMED**。

## 範圍與停止線

本切片處理規格110後固定EmuTOS對`$FF8606`依次寫入`$0190`、
`$0090`，以DMA write-direction bit 8的兩次toggle重設DMA，再對已選定的
sector-count register `$FF8604`寫0。

下一個`$0088` ACSI mode、HDC command/status、DMA FIFO內容、sector transfer、DRQ、
DMA status read與RAM傳輸不在範圍。內部FIFO尚未存在時，本切片只保存
可被後續消費的sector count與reset收據，不用虛構FIFO欄位冒充已實作傳輸。

## 證據

- **已確認（Hatari固定實作）**：Hatari 2.4.1 commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/fdc.c` SHA-256
  `7de0d37a0972d6de43d21dc6f653ee6e3b736b611b978c9de221bb9b938f66f7`。
  `FDC_DmaModeControl_WriteWord()`先保存方向bit之前值，寫入新mode；
  `(old^new)&$0100`非零時呼叫`FDC_ResetDMA()`，清FIFO、將bytes-in-sector
  回復512、sector count清為0，並清內部transfer進度。
- **已確認（Hatari固定實作）**：`FDC_DiskController_WriteWord()`
  在mode bit 4為1時路由到`FDC_WriteSectorCountRegister()`；普通ST只保留
  `$FF8604`寫入值的low byte。兩個word-write handler各有額外4 wait clocks。
- **已確認（EmuTOS原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/floppy.c` SHA-256
  `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
  `fdc_start_dma_read()`以`DMA_SCREG|DMA_FLOPPY|DMA_WRBIT` (`$0190`)、
  `DMA_SCREG|DMA_FLOPPY` (`$0090`)依次toggle，之後才寫sector count。
- **強證據（固定Hatari正常路徑）**：ROM與trace雜湊同規格110。
  VBL19 video cycle 84,778在PC `$FC1224`寫`$0190`並reset DMA；84,940在
  `$FC1232`寫`$0090`並再reset DMA；85,100在`$FC1240`寫
  `$FF8604=$0000`，trace明記sector count=`$00`。下一mode write為
  video cycle 85,226、PC `$FC126E`的`$0088`。
- **已確認（固定Talos入口）**：規格110後291,343 instructions、
  3,002,130 clocks停在`$FF8606=$0190`；pipeline PC=`$FC122A`、
  prefetch=`$8606,$2239`。

## typed行為

1. DMA位址write stage3、drive-1 FDC probe stage14且mode=`$0080`時，接受
   supervisor word write `$FF8606=$0190`；保存mode，因bit 8由0→1而將
   sector count清為0，記錄第一次DMA reset。
2. 第一次reset後只接受`$FF8606=$0090`；保存mode，因bit 8由1→0
   再將sector count清為0，記錄第二次DMA reset。
3. mode bit 4已選sector count且完成兩次reset後，接受
   supervisor word write `$FF8604=$0000`；保存low-byte sector count 0並進入完成stage。
4. 三筆word access沿用shared-bus slot alignment且各增加4 wait clocks。錯序、
   錯值、user／byte access失敗即關閉，不部分修改mode、count或reset收據。
5. cold reset將mode、sector count、reset count與本序列stage清為0；DMA address亦依
   規格110清為0。

## 驗收

- synthetic測試從非零sector count開始，驗證`$0190→$0090→data $0000`、
  兩次reset、mode／count，錯序與錯值原子拒絕、timed wait與cold reset。
- 固定ROM必須自然完成三筆write，鎖定CPU state、clock、mode、count與
  reset收據，再有界定位下一typed gate。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

## 驗收收據

- 固定Talos自然完成`$0190→$0090→$FF8604=$0000`，記錄兩次bit-8
  toggle reset，mode=`$0090`、sector count=0；於291,376 instructions、
  234 interrupts、3,002,468 clocks完成。
- 完成邊界PC=`$FC1248`、prefetch=`$0045,$0008`，D/A、SSP、SR、mode、
  sector count、reset count與三段收據均鎖入固定ROM測試。
- 有界續跑至291,386 instructions、3,002,576 clocks；下一gate為
  `$FF8606=$0088`的supervisor word write，pipeline PC=`$FC1274`、
  prefetch=`$8606,$0045`，將由floppy DMA切至ACSI command mode。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過。
