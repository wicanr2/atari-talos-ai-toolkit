# 063 — ST MFP DDR reset-state zero write

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理固定 EmuTOS 初始化迴圈在 reset state 對 MC68901 MFP Data Direction
Register（DDR）`$FFFA05` 寫 `$00`。任意 input/output 方向改寫、外部 pin drive、
GPIP transition／pending interrupt 與其餘 MFP registers 維持失敗即關閉。

- **已確認（MC68901 一手規格）**：NXP《MC68901 Multi-Function Peripheral》
  user manual，PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`，
  §5.1.3。DDR 八 bits reset 均為 0；0 將對應 GPIP pin 設為 high-impedance input，
  1 設為 push-pull output。GPIP read 對 output 取 data register，對 input 取 input buffer。
- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/ioMemTabST.c` SHA-256
  `c0214b586bdd32a1f3d50f91827ce6b84f1fd6411b417838193b03eadde4f631`
  將 `$FFFA05` 映至 DDR byte handlers；`src/mfp.c` SHA-256
  `610e30dc75acf0d0f802b0712e899be83f0926b8d1e54d1c0bea85466bcfc69b`
  reset DDR=`$00`、access 增加 4 wait clocks，write 後以 old/new DDR 重新評估
  GPIP interrupt。此來源只作外部 oracle。
- **已確認（固定 EmuTOS 1.3）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/machine.c` SHA-256
  `21a9733139f223f8781b61340c8ded45b98a51fc3aa31f92bbb10abeeaa8fe0c`。
  ROM `$FC614A` 清 MFP register bank 的第三次迭代以 A0=`$FFFFFA05` 寫 `$00`。
- **已確認（Hatari 外部 oracle）**：image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
  A0=`$FFFFFA05` 時 `$FC614A→$FC614E` 的 FrameCycles `44210→44226`，共
  **16 clocks**；GPIP／AER／DDR 前後皆 `$00`，registers、flags 與 prefetch
  變化和前兩次 iteration 相同。

## typed 行為

1. cold reset 與 MC68000 `RESET` 將 DDR 設為 `$00`。
2. `$FFFA05` supervisor data byte read 回 DDR；user access 沿用 I/O protection
   fault，word access仍不合法。
3. 只有 DDR current=`$00` 且 write value=`$00` 的 no-direction-change case 可提交；
   其他 write 回 typed `unsupported_device_state` fault，不能漏掉 pin direction、
   GPIP read source及 interrupt transition 副作用。
4. `$FFFA05` byte access固定增加 4 wait clocks；已驗證的
   `MOVE.B #$00,(An)` 總計 16 clocks。
5. 本規格驗收時以 `$FFFA07` IERA write 的 reserved-I/O fault 作為停止線；
   後續規格 064 只取代這條停止線，不改變 DDR 契約。

## 驗收與停止線

- memory 測試涵蓋 reset、zero write/read、24-bit alias、4 wait clocks、user
  protection、nonzero write fail-closed、word access與 `$FFFA07` 未映射。
- 固定 EmuTOS 完成第 7,483 條、累計 clocks／state／prefetch／DDR 對上 Hatari；
  再三條到 `$FFFA07` write 明確停止。
- 完整 230,000 筆 CPU corpus、固定 ROM、Go 測試、靜態檢查與建置通過後才升
  **CONFORMED**。

## CONFORMED 收據

- memory 測試確認 DDR reset／zero read-write、24-bit alias、4 wait clocks、user
  protection、word access；nonzero write 回 `unsupported_device_state`，IERA仍未映射。
- 固定 EmuTOS 完成第 7,483 條後為 176,726 累計 clocks；DDR=`$00`，D/A、USP、
  SSP=`$0F8C`、SR=`$2714`、內部 prefetch 游標 PC=`$FC6152`、prefetch
  `$5488,$B0FC` 均符合 Hatari 對應邊界。
- 本規格驗收時，再完成三條迴圈控制指令後，第 7,487 條嘗試停在 `$FFFA07`
  IERA reserved-I/O write；該停止線其後由規格 064 取代，其他未規格化 register
  仍未泛化成可寫 memory。
- 完整 230,000 筆 CPU corpus、固定 ROM 全測試、`go vet -stdmethods=false ./...`
  與 `go build ./...` 全部通過。
