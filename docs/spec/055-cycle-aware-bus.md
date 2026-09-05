# 055 — CPU／machine／bus cycle-aware access 契約

狀態：**READY**（時間軸基礎已接線；runtime access 遷移未完成）。

## 決策與範圍

2026-09-05 使用者定案採用完整、漸進式 cycle-aware Bus，排除「指令完成後補
clocks」與忽略 ST 動態等待的方案。此契約服務 MC68000 微時序、全機單調時鐘，
以及後續 Shifter／MFP／DMA／FDC；既有 JSON Lines 控制協定與遊戲資料格式不變。

本規格只建立時間語意、遷移邊界及失敗即關閉條件。Shifter 仲裁公式須另以 Hatari
外部 oracle 與硬體資料完成規格，不在此處猜補；因此第 14 條 EmuTOS 的動態 `+2`
仍是未完成驗收，不得用 opcode、位址或總時鐘特例通過。

## 已知證據

- **已確認（外部 MC68000 語料）**：固定 v1 binary corpus 的 transaction block
  同時保存 active bus cycle 與 kind 0 idle phase；每個 entry 都有 phase duration，
  所有 phase 依序相加為該指令總 clocks。既有載入器過去只保留 active transaction。
- **已確認（Hatari 外部 oracle）**：固定 EmuTOS 1.3 ROM 執行第 14 條
  `MOVE.L #$FC00B2,$0010` 時，Hatari 由 390 到 416 clocks，Atari Talos 由
  390 到 414；前 13 條的 CPU state、prefetch 與總 clocks 相同。
- **強證據**：相同 opcode shape 在較早全機時相為 24 clocks，因此差異不是
  `MOVE.L` 固定成本，而是依全機時相發生的 ST RAM／Shifter bus arbitration。

## typed 契約

1. `BusPhase.Offset` 是相對本指令 epoch 的零起算 MC68000 clock；`Cycles` 是該
   phase 的持續時間，不是絕對 clock。
2. `BusPhase.Transaction == nil` 明確表示 CPU internal／idle phase；非 nil 時完整
   保留既有 FC、24-bit bus address、資料、UDS／LDS、read／write／fault kind。
3. machine 持有唯一的 64-bit 單調 epoch。遷移完成後，每個同步 bus access 在
   **執行 access 前**收到 `epoch + Offset`，裝置可依當下時相回傳 typed wait clocks
   或 fault；CPU 將 wait 插入後續 phase 並增加該步總 clocks。
4. wait 必須影響同一指令後續 access 的絕對時間，不可在 `Step` 回傳後一次補總數。
   如此 timed I/O read 才能看到正確時相。
5. 遷移按指令族進行；未遷移的 access 可沿用既有 Bus，但不得向需要精確時相的
   裝置發出。machine 的精確模式遇到這種路徑必須回 typed unsupported-timing error，
   不可假設零 wait。
6. `Transaction.Cycle` 暫時維持「active transaction duration」語意，以保持既有
   227,500 筆比較；不得把它改解為 offset。完整遷移後再由 `BusPhase` 成為唯一時間軸。

## 漸進接線

1. **本切片**：語料載入器保留所有 idle／active phases，計算每一 phase 的 offset，
   並對每筆測試驗證 phase duration 總和等於 instruction clocks；既有 active bus
   比較不變。
2. 建立 instruction epoch 與 timed access／wait 回傳介面，先以不競爭的 memory
   證明 state、transaction 與 clocks 完全不變。
3. 以語料時間軸逐族遷移 CPU access；每族同時比對 phase 序列，禁止只比總 clocks。
4. 另寫 READY Shifter arbitration 規格後，接入 ST RAM，對拍 EmuTOS 第 14 條的
   動態 wait；再前進 line-F／vector 11。

## 驗收與停止線

- 全部固定 MC68000 語料可讀取時間軸，且每筆 `sum(phase.Cycles) == Clocks`。
- 既有 227,500 筆 CPU state／RAM／clock／active transaction 比較全綠。
- Go 單元測試、`go vet`、全程式建置通過；JSON Lines golden tests 不變。
- 本切片完成只可稱「時間軸基礎已接線」，不可稱 cycle-aware runtime、Shifter
  arbitration 或 Atari ST 開機完成。
