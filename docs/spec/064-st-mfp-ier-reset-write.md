# 064 — ST MFP IERA／IERB reset-state zero writes

狀態：**CONFORMED**。

## 範圍與證據

本切片一起處理固定 EmuTOS 初始化迴圈對 MC68901 Interrupt Enable Register A／B
（IERA `$FFFA07`、IERB `$FFFA09`）的 reset-state `$00→$00` 寫入。實際啟用 channel、
pending／IRQ、in-service、mask、timer／USART／GPIP interrupt sources 仍失敗即關閉。

- **已確認（MC68901 一手規格）**：NXP《MC68901 Multi-Function Peripheral》
  user manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`，
  §4.3.1。IERA／IERB reset 都為 `$00`；bit=1 enable 對應 channel，bit=0 disable。
  對任一 enable bit 寫 0 會清除相應 pending bit、終止該 channel request，卻不清
  in-service bit。
- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/ioMemTabST.c` SHA-256
  `c0214b586bdd32a1f3d50f91827ce6b84f1fd6411b417838193b03eadde4f631`
  將 `$FFFA07/$FFFA09` 映至 IER byte handlers；`src/mfp.c` SHA-256
  `610e30dc75acf0d0f802b0712e899be83f0926b8d1e54d1c0bea85466bcfc69b`
  reset IERA／IERB／IPRA／IPRB 為 0；access 增加 4 wait clocks；write IER 後以
  `pending &= enable` 並重新評估 IRQ。此來源只作外部 oracle。
- **已確認（固定 EmuTOS 1.3）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/machine.c` SHA-256
  `21a9733139f223f8781b61340c8ded45b98a51fc3aa31f92bbb10abeeaa8fe0c`。
  ROM `$FC614A` 的第四、第五次迭代分別以 A0=`$FFFFFA07/$FFFFFA09` 寫 `$00`。
- **已確認（Hatari 外部 oracle）**：image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
  IERA iteration `$FC614A→$FC614E` 為 FrameCycles `44254→44270`；IERB 為
  `44298→44314`，兩者都是 **16 clocks**，寫前寫後 IERA／IERB 均 `$00`，
  register／flags／prefetch 變化相同。

## typed 行為

1. cold reset 與 MC68000 `RESET` 將 IERA、IERB 設為 `$00`。
2. 兩位址的 supervisor data byte read 回各自 latch；user access 沿用 I/O
   protection fault，word access仍不合法。
3. 只接受 current=`$00` 且 write=`$00`。其他 write 回 typed
   `unsupported_device_state`，避免在未建模 sources／pending／IRQ 時假裝 channel
   enable 或 disable side effects 已完整。
4. 兩位址 byte access各增加 4 wait clocks；已驗證的 `MOVE.B #$00,(An)` 各 16 clocks。
5. 完成 IERB 後下一次迴圈在 `$FFFA0B` IPRA write 維持 reserved-I/O fault。

## 驗收與停止線

- memory table test涵蓋兩個 latch 的 reset、zero read/write、alias、wait、user
  protection、nonzero write fail-closed、word access與 IPRA 未映射。
- 固定 EmuTOS 完成第 7,491 條、累計 clocks／state／prefetch／IERA／IERB 對上
  Hatari；再三條到 `$FFFA0B` write 明確停止。
- 完整 230,000 筆 CPU corpus、固定 ROM、Go 測試、靜態檢查與建置通過後才升
  **CONFORMED**。

## CONFORMED 收據

- table test確認 IERA／IERB 的 reset／zero read-write、24-bit alias、4 wait clocks、
  user protection與word access；nonzero write 回 `unsupported_device_state`，IPRA未映射。
- 固定 EmuTOS 第 7,487 條 IERA 與第 7,491 條 IERB 都為 16 clocks；完成後累計
  176,814 clocks，兩 latch=`$00`，D/A、USP、SSP=`$0F8C`、SR=`$2714`、內部
  prefetch 游標 PC=`$FC6152`、prefetch=`$5488,$B0FC` 均符合 Hatari 邊界。
- 再完成三條迴圈控制指令後，第 7,495 條嘗試停在 `$FFFA0B` IPRA reserved-I/O
  write，未泛化 pending register 語意。
- 完整 230,000 筆 CPU corpus、固定 ROM 全測試、`go vet -stdmethods=false ./...`
  與 `go build ./...` 全部通過。
