# 074 — MC68000 第 4 級自動向量中斷接受

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理外部機器已完成仲裁並提供自動向量回應後，MC68000 接受第 4 級
中斷的 CPU 狀態轉換。GLUE 的 VBL 產生時點、實體 interrupt acknowledge bus cycle、
MFP 向量中斷及第 7 級不可遮罩語意不在本切片。

- **已確認（Motorola／NXP 一手規格）**：NXP 官方《M68000 Family Programmer’s
  Reference Manual》，PDF SHA-256
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
  `STOP` 先把 PC 推進到下一指令；只有高於新 SR interrupt mask 的請求會產生
  interrupt exception。附錄 B 將 level 4 autovector 定為 vector 28、offset `$070`。
- **已確認（固定 EmuTOS 1.3／Hatari oracle）**：ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。
  首個 `$FCD09A: STOP #$2300` 後，RAM `$70` 為 `$00FC0446`。下一個 VBL 在
  `$FC0446` 入口顯示 SR mask 4、SSP 由 `$F70` 降至 `$F6A`；6-byte frame 是
  `$2300,$00FC,$D09E`，也就是原 SR 與 STOP 後下一指令 PC。handler 預取為
  `$52B8,$0466`（`ADDQ.L #1,$466`）。入口 FrameCycles=`124`。
- **強證據（MC68000 公開 timing 與 Hatari 對拍）**：autovector interrupt exception
  採 44 clocks。Hatari 在固定 color ST profile 的 VBL handler 入口位於 frame cycle
  124，與 VBL interrupt 採樣／接受點及 44-clock exception sequence 相符；目前不宣稱
  interrupt acknowledge 的逐 pin 波形一致。

## typed 行為

1. API 只接受 level 1–6；其餘值失敗即關閉。若 `level <= (SR>>8)&7`，回報未接受、
   0 clocks、無 transaction，CPU state 與 stopped latch 均不變。
2. 接受時以目前 pipeline PC 作 saved PC。這使 STOP 後保存其下一指令，而一般指令
   邊界也保存下一個待執行位置。
3. 以原 SR 建立 MC68000 6-byte frame（SR、PC high、PC low），切 supervisor、清 trace，
   並把 SR interrupt mask 設為 level；從 `(24+level)*4` 讀 handler 並預取兩 words。
4. 成功接受後才清 stopped latch，回報 44 clocks。現有 transaction 契約只描述記憶體
   frame、vector 與 prefetch；外部自動向量 acknowledge 尚無 Bus 型別，44 clocks 中未顯示
   的部分保留為 idle phase，不冒稱已建模實體 IACK。

## 驗收與停止線

- typed test 覆蓋 fixed level 4 frame、SR、prefetch、STOP 喚醒、遮罩拒絕、非法 level
  失敗即關閉及 timed-bus 44 clocks。
- 固定 ROM 的下一階段必須由真正執行 `$FC0446` handler 令 `$466` 遞增；本切片不直接
  寫 `$466`，也不在尚無 READY GLUE 排程規格時猜一個 VBL deadline。
- 完整 232,500 筆外部 CPU corpus、固定 ROM STOP gate、Go 測試、靜態檢查與建置全綠後，
  才可改為 **CONFORMED**。
