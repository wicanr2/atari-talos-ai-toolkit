# 035 — Atari ST／STF 基礎記憶體映射

狀態：**CONFORMED**（2026-09-05）。

## 範圍與證據

本切片只建立 TOS reset 與後續 CPU 存取所需的 RAM、ROM shadow、主 TOS ROM、保護區
及未映射區。Shifter、DMA、PSG、MFP、ACIA、MMU configuration register 與 bus-error
exception 的 CPU 微時序另立規格，不在此處猜補。

- **已確認（Atari 一手硬體規格）**：Atari Corporation《Engineering Hardware
  Specification of the Atari ST Computer System》，1986-01-07，第 25–27 頁。
- 保存掃描：`GEM_0904.pdf`，SHA-256
  `eb3a001ed636123f94c9c612ab33b6de2b1b118177ea01cfb971bf3ae17e6044`，來源：
  <https://bitsavers.trailing-edge.com/pdf/atari/ST/Atari_ST_GEM_Programming_1986/GEM_0904.pdf>。
- 規格明列 24-bit address、512 KiB／1 MiB RAM、192 KiB ROM；`0x000000–0x000007`
  是主 ROM 的 reset SSP／PC shadow，RAM 自 `0x000008` 起；主 ROM 位於
  `0xFC0000–0xFEFFFF`，I/O space 位於 `0xFF0000–0xFFFFFF`。
- 第一個 2 KiB 與 I/O space 只允許 supervisor reference。user reference、ROM／shadow
  write、保留 I/O 與未映射位址必須失敗並保留存取 metadata。

## typed 行為

1. 建構器只接受 512 KiB 或 1 MiB RAM，以及恰好 192 KiB 的 TOS ROM；輸入 ROM
   必須複製，呼叫端後續修改不得改變機器內容。
2. 所有位址先截為 24 bit。`0x000000–0x000007` read 對應 ROM offset 0–7；其後直到
   配置容量末端為 RAM；`0xFC0000–0xFEFFFF` 對應完整 ROM。
3. FC 1／2 是 user data／program，FC 5／6 是 supervisor data／program；其他 FC
   在本切片失敗即關閉。user 對 `0x000000–0x0007FF` 或 I/O 的存取回傳 typed bus fault。
4. byte／word 採 big-endian；word 必須偶數對齊。word write 在驗證兩個 byte 均可寫後
   才提交，避免跨界 fault 留下半次寫入。
5. bus fault 記錄 24-bit address、FC、read／write、byte／word size 與穩定原因碼。

## 驗收與停止線

- 測試兩種 RAM 容量、reset shadow、ROM mirror、ROM immutable、RAM 首尾、低 2 KiB
  權限、24-bit mask、big-endian word、odd address、未映射 gap 與保留 I/O。
- 本切片通過只能宣稱 memory-map routing 自洽；CPU 尚未把 backend bus fault 轉成
  MC68000 vector 2，也尚未宣稱 TOS 可開機或與 Hatari parity。
- 不納入 TOS ROM 或任何原版素材；測試使用自行建立的小型 pattern。
- 上述建構、映射、權限、read-only、未映射、word 與原子失敗測試均已通過，故本規格
  升為 CONFORMED。
