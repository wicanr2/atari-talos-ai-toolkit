# 052 — MC68000 vector 2 保留 absolute-short 的 32-bit fault address

狀態：**CONFORMED**。

## 範圍與證據

本切片只處理 word 來源 effective address 已由 `(xxx).W` sign-extend 成 32-bit，
再因 24-bit ST bus read fault 進入 vector 2 時，exception frame 應保存哪一種位址。
直接解開 EmuTOS 1.3 在 `$FC0080` 的 `TST.W $8006` CPU／機型探測。

- **已確認（既有 CPU effective-address 契約）**：`(xxx).W` 會將 16-bit extension
  sign-extend；因此 `$8006` 的 CPU effective address 是 `$FFFF8006`，ST bus cycle
  address 則是低 24-bit `$FF8006`。
- **已確認（Hatari 外部 oracle）**：Hatari 2.4.1 image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`，
  EmuTOS 1.3 UK 192 KiB ROM
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`，
  `--machine st --memsize 1 --fast-boot false`。
- `$FC0080` opcode／extension 為 `$4A78,$8006`。進入 vector 2 handler 後，
  physical `$1FEC` 的 14-byte frame 為
  `$4A75,$FFFF,$8006,$4A78,$2700,$00FC,$0080`；其中 fault address 明確是
  `$FFFF8006`，不是 `$00FF8006`。
- 由 reset 繼續後，Hatari 在 10 條完成指令、220 clocks 到 vector 2 handler
  `$FC0088`；`SSP=$0FEC`、`SR=$2700`、prefetch=`$4E70,$0CB9`。

## typed 行為

1. `(xxx).W` 的 CPU effective address 維持完整 `uint32(int32(int16(extension)))`；
   呼叫 24-bit ST bus 時才以 `$00FFFFFF` 遮罩。
2. word-read backend typed fault 的 address 必須等於 effective address 的低 24-bit，
   且必須是 read、size=2；不符合時保留原錯誤並失敗即關閉。
3. vector 2 frame 的 fault-address high／low words 取自未遮罩的 CPU effective address；
   fault bus transaction 仍記錄低 24-bit 並對齊偶數的 `$FF8006`。
4. 本例沿用已驗證的 14-byte access-error frame、SSW、bus 次序與 saved PC 契約；
   `TST.W (xxx).W` 的例外完成時間為 68 clocks（`60 + EA cost 8`）。
5. 修正不得改變一般 absolute-long、address-register 或其他未命中本差異的
   `MOVE.W`／word-source 行為。

## 驗收與停止線

- synthetic CPU 測試以 `$4A78,$8006` 與回報 `$00FF8006` 的 typed bus fault，
  驗證 frame 保存 `$FFFF,$8006`，fault transaction 保存 `$FF8006`，並驗
  SSW、opcode、saved PC、FC、bus 次序與 68 clocks。
- 固定 EmuTOS ROM 從 reset 完成 10 條指令後，必須以 220 clocks 到 `$FC0088`
  handler，且 state、prefetch 與完整七個 frame words均與 Hatari 收據一致。
- 通過後才把本規格升為 **CONFORMED**，並以 `$FC0088` 的 `RESET` 作下一個
  獨立規格切片；本規格不宣稱 `RESET` 已實作或 TOS 已可開機。
- byte／long bus error、destination write、instruction fetch、第二個 longword bus
  cycle、exception 進入期再次 fault、其他 EA 的獨立 timing 不在本切片。

## CONFORMED 收據

- 2026-09-05：synthetic `$4A78,$8006` 測試確認 backend fault address 與低
  24-bit bus cycle 一致，frame 則保存 `$FFFF8006`；SSW、opcode、saved PC、
  FC、13 筆 bus transactions 與 68 clocks 全部通過。
- 固定 EmuTOS ROM 完成第 10 條指令後為 220 clocks、`SSP=$0FEC`、
  `SR=$2700`、prefetch=`$4E70,$0CB9`，完整 frame
  `$4A75,$FFFF,$8006,$4A78,$2700,$00FC,$0080` 與 Hatari 一致。
