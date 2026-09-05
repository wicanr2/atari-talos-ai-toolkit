# 061 — ST MFP GPIP reset-state byte write

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理普通 Atari ST 開機初始化時對 MC68901 MFP GPIP `$FFFA01` 的
supervisor byte read／write，以及同形 `MOVE.B #imm,(An)` 的 timed write phase。
相鄰 MFP 暫存器、外部腳位、interrupt、timer、USART 與完整 E-clock 波形仍失敗即關閉。

- **已確認（MC68901 一手規格）**：NXP《MC68901 Multi-Function Peripheral》
  user manual，下載檔 SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`。
  GPIP 的 DDR bit=0 時腳位為 input／high impedance，bit=1 時為 push-pull output；
  寫 GPIP 只改變被 DDR 設為 output 的 bits，input bits 保留外部輸入值。
- **強證據（固定 Hatari 2.4.1 原始碼）**：commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`；`src/ioMemTabST.c` SHA-256
  `c0214b586bdd32a1f3d50f91827ce6b84f1fd6411b417838193b03eadde4f631`
  將 `$FFFA01` 映至 GPIP byte handler；`src/mfp.c` SHA-256
  `610e30dc75acf0d0f802b0712e899be83f0926b8d1e54d1c0bea85466bcfc69b`
  的 reset 將 GPIP／DDR 設為 0，GPIP write 採
  `(old & ^DDR) | (written & DDR)`，並為 MFP access 加 4 wait clocks。
  此來源只作外部 oracle，production 不翻譯或移植 GPL 程式碼。
- **已確認（固定 EmuTOS 1.3）**：commit
  `95eb9e498e979022dd9626f528d32de861f26c85`；`bios/machine.c` SHA-256
  `21a9733139f223f8781b61340c8ded45b98a51fc3aa31f92bbb10abeeaa8fe0c`。
  ROM `$FC6144–$FC6154` 先把 A0 設為符號延伸 `$FFFFFA01`，再以
  `MOVE.B #$00,(A0)`／`ADDQ.L #2,A0` 迴圈清 MFP odd-address register bank。
- **已確認（固定 M68000 microcoded corpus）**：同形 immediate source → `(An)`
  樣本 `MOVE.b #,(A2)` 為三個連續 4-clock bus phases：讀 extension、byte write、
  refill，共 12 clocks。
- **已確認（Hatari 外部 oracle）**：image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
  `$FC614A` 到 `$FC614E` 的 FrameCycles `44122→44138`，共 **16 clocks**；
  GPIP／DDR 寫前寫後均為 `$00`。D/A／USP／SSP 不變，SR condition codes
  `N=1,Z=0,V=0,C=1 → N=0,Z=1,V=0,C=0`，prefetch
  `$10BC,$0000 → $5488,$B0FC`。

## typed 行為

1. `$FFFA01` 只接受 supervisor data FC byte access；user data access依既有 I/O
   protection 產生 typed bus fault。word access與相鄰位址仍不映射。
2. cold reset 與 MC68000 `RESET` 都令本切片的 GPIP、DDR 為 `$00`；RAM 不受影響。
3. GPIP read 回目前 pin value；write 採
   `(oldGPIP & ^DDR) | (value & DDR)`。本切片不提供外部 pin injection，且不開放
   DDR register，因此可驗證 reset slice 中任何 write 都維持 GPIP=`$00`。
4. `MOVE.B #imm,(An)` 的 CPU core 基礎時序仍為 12 clocks；MFP data write phase
   額外等待 4 clocks，總計 16。本切片不把 MFP wait 泛化到 RAM／ROM bus-slot
   alignment，也不宣稱 pin-level E-clock parity。
5. 完成 `$FFFA01` 後下一個 `$FFFA03` AER write 必須維持 reserved-I/O fault，作為
   切片停止線。

## 驗收與停止線

- memory 測試確認 reset、DDR=0 masking、24-bit alias、user protection、word／相鄰
  位址 fail-closed，以及 timed MFP write 固定 4 wait clocks。
- synthetic CPU 測試確認三個 bus phases、16 clocks、write FC/lane、flags 與 prefetch。
- 固定 EmuTOS 完成第 7,475 條，state／prefetch／GPIP／累計 clocks 對上 Hatari
  對應邊界；向後只探測到 `$FFFA03` typed fault。
- 完整 230,000 筆 CPU 語料、固定 ROM、Go 測試、靜態檢查與建置都通過後才升
  **CONFORMED**。

## CONFORMED 收據

- memory 測試確認 GPIP／DDR reset、DDR mask、24-bit alias、supervisor protection、
  4 wait clocks，以及 `$FFFA03` 與 word access 保持 fail-closed。
- synthetic timed CPU 測試確認 extension read、MFP write、refill 三 phases；MFP wait
  插在 data write 前，總計 16 clocks，flags、prefetch、FC 與 byte lane 均符合契約。
- 固定 EmuTOS 第 7,475 條完成後為 176,638 累計 clocks；GPIP=`$00`，D/A、USP、
  SSP=`$0F8C`、SR=`$2714`、prefetch=`$5488,$B0FC` 對上 Hatari。Talos 的內部 PC
  `$FC6152` 是預取游標；Hatari debugger 顯示的架構下一指令位址為 `$FC614E`，
  不把兩種 PC 表示混稱相同欄位。
- 再完成 `ADDQ.L`、`CMPA.W`、`BLS` 三條後，第 7,479 條嘗試明確停在
  `$FFFA03` AER reserved-I/O write；沒有把其餘 MFP bank 當 RAM 或 void register。
- 完整 230,000 筆 CPU corpus、固定 ROM 全測試、`go vet -stdmethods=false ./...`
  與 `go build ./...` 全部通過。
