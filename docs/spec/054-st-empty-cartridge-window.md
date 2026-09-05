# 054 — Atari ST 空 cartridge ROM window

狀態：**CONFORMED**。

## 範圍與證據

本切片建立未安裝 cartridge image 時的 ST／STF cartridge ROM window，直接解開
EmuTOS 1.3 在 `$FC008A` 對 `$FA0000` 的 cartridge magic probe。

- **強證據（Hatari v2.4.1 固定原始碼）**：官方 GitHub mirror tag `v2.4.1`，
  commit `4371dcd647fc85d31c0629400adaeaa4212040d9`。`src/cpu/memory.c:1798–1803`
  將 `$FA0000` 起的兩個 64 KiB banks 映射到 ROM backend；
  `src/cart.c:108–139 Cart_ResetImage` 以 `$FF` 填滿 `$20000` bytes；
  `src/cpu/memory.c:1018–1058` 提供 big-endian byte／word／long read，三種 write
  均產生 bus error。Hatari 註解記錄 cartridge 讀取 timing 已在實體 STF 測試。
- **已確認（Hatari 外部 oracle）**：Hatari 2.4.1 image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`，
  EmuTOS 1.3 UK 192 KiB ROM
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`，
  `--machine st --memsize 1 --fast-boot false`。debugger 在 `$FC008A` 將
  `CMPI.L #$FA52235F,$00FA0000` 的 operand 顯示為 `$FFFFFFFF`。

## typed 行為

1. cartridge window 固定為 `$FA0000–$FBFFFF`，共 `$20000` bytes；兩端皆包含。
2. 未載入 cartridge 時每個 byte 為 `$FF`。`ReadByte`／`ReadWord` 依既有
   big-endian 與 odd-word 契約讀取，且不受 RAM MMU bank translation 影響。
3. 既有合法 user／supervisor data／program FC 均可讀 cartridge；非法 FC
   仍由共用權限檢查失敗即關閉。
4. cartridge 是 ROM：byte／word write 產生 typed `FaultReadOnly`；word write
   維持原子性，窗口最後一個 byte 的跨界 word 仍先依 odd-address 規則失敗。
5. cold reset 與 MC68000 external reset 不改變空槽 `$FF` 內容。

## 驗收與停止線

- ST memory 測試驗 base／中間／end byte、base／end-1 word、user／supervisor FC、
  MMU 組態獨立、byte／word read-only fault、odd boundary 與鄰接未映射區。
- 固定 EmuTOS ROM 從 reset 完成第 12 條 cartridge comparison，對拍 Hatari 的
  clocks、CPU state、prefetch，然後找出新的第一失敗點。
- 通過後升為 **CONFORMED**；cartridge image 載入、`.STC` 四 byte header、
  Hatari 內建 GEMDOS cartridge、hot swap、cartridge 檔案權利與發行不在本切片。
  未建立正式 image API 前不接受或偷偷保存外部 cartridge bytes。

## CONFORMED 收據

- 2026-09-05：ST memory 測試通過 `$FA0000–$FBFFFF` 尺寸、base／middle／end
  byte、兩端 word、user／supervisor FC、MMU 獨立、read-only byte／word write、
  odd end address 與 pre-window gap。
- 固定 EmuTOS ROM 完成第 12 條 `CMPI.L #$FA52235F,$FA0000` 後，Atari Talos
  與 Hatari 均為 380 clocks；D／A 全零、`SSP=$0FEC`、`SR=$2700`，
  prefetch=`$660A,$4DFA`，空槽 operand 為 `$FFFFFFFF`。
- 繼續逐條對拍時，第 13 條仍同為 10 clocks；第 14 條
  `MOVE.L #$FC00B2,$0010` Hatari 為 26 clocks、Atari Talos 為 24 clocks。
  同 opcode 形狀先前曾是 24 clocks，故新缺口屬全機 RAM／Shifter bus
  arbitration 的動態 wait state，不回寫成本 cartridge 固定 timing。
