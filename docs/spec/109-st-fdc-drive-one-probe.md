# 109 — ST FDC drive 1探測鏈重用

狀態：**CONFORMED**。

## 範圍與證據

本切片處理規格108後，固定EmuTOS對drive 1重跑與drive 0相同的
FDC存在性探測：force interrupt、restore、status read、data register、
seek track zero與第二次status read。實際磁碟映像、sector DMA、跨track移動、
write protect與磁碟機差異不在範圍。

- **已確認（EmuTOS原始碼）**：EmuTOS 1.3 `bios/floppy.c`
  SHA-256 `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`
  的`flop_detect_drive()`對device 0與1依次呼叫同一套select、force interrupt、
  restore、status與seek流程；drive參數只影響被選的磁碟機。
- **強證據（固定Hatari 2.4.1／EmuTOS正常路徑）**：ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；
  CPU／FDC trace SHA-256
  `476c4bf3cbfa8f1798cb6f510b5efcdba68b632f617c0363e7efddf7fcdb061f`。
  YM2149 port A由`$05`轉為`$03`後，trace再出現`$0080→$00D0→$0080→
  $000B`，restore期間九次GPIP bit 5 inactive poll，完成後status read為`$E4`；
  接著`$0086→data $00→$0080→seek $13`，seek亦有九次inactive poll，
  完成後第二次status read為`$E4`。
- **已確認（Hatari實作契約）**：Hatari source commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`，`src/fdc.c` SHA-256
  `7de0d37a0972d6de43d21dc6f653ee6e3b736b611b978c9de221bb9b938f66f7`；
  本切片沿用規格105–107已驗證的type-I command狀態、IRQ與read-clear契約。

## typed行為

1. PSG drive stage6、R14=`$03`且前一顆drive的FDC stage14時，只接受
   supervisor word write `$FF8606=$0080`；將當前probe drive設為1，
   清除前一次probe的restore／seek收據，並回到FDC stage1。
2. drive 0初次進入同一鏈時，將probe drive設為0。cold reset為-1，
   表示尚未選定探測對象。
3. drive 1完整重用已驗證的stage1→14：force `$D0`；reselect
   mode `$0080`；restore `$0B`；IRQ後status `$E4`並read-clear；mode `$0086`；
   data `$00`；mode `$0080`；seek `$13`；IRQ後status `$E4`並read-clear。
4. restore與seek各在command write後729 CPU clocks完成，期間各觀測九次
   GPIP bit 5 inactive poll；完成後bit 5 active，status read再清IRQ。
5. 切換probe時只清除每顆drive的收據與pending狀態，不重置CPU、MFP、
   YM2149或全局clock。錯序、錯值、user／byte access失敗即關閉且不部分改狀態。
6. 本切片只建模EmuTOS在無磁碟時實際觀測到的探測路徑，不將固定
   `$E4`推廣為已實作的完整WD1772。

## 驗收與停止線

- synthetic測試從已完成drive 0的狀態開始，驗證drive 1重開鏈、
  舊收據清除、兩段deadline／各九次poll、兩次IRQ read-clear、錯序原子拒絕與reset。
- 固定ROM必須自然完成drive 1的stage1→14，鎖定CPU state、clock、
  command／status與poll收據，再有界定位下一typed gate。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過後才升 **CONFORMED**。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

## 驗收收據

- 固定Talos自然完成drive 1全探測鏈，於290,970 instructions、234 interrupts、
  2,997,708 clocks抵達stage14；restore command／status-read clock為
  2,992,662／2,994,002，seek為2,996,378／2,997,694，兩段各有九次
  inactive poll，IRQ均由status read清除。
- 該邊界PC=`$FC38A0`、prefetch=`$4E75,$2F0A`，D/A、stack、SR、command=`$13`、
  status=`$E4`與全部probe收據均鎖入固定ROM測試。
- 有界續跑至291,291 instructions、3,001,516 clocks；下一gate為
  `$FF860D`的supervisor byte write，pipeline PC=`$FC3600`、prefetch=`$860D,$11EF`。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`與
  `go build ./...`通過。
