# Atari Talos 研究紀錄

## 基準模擬器

- Hatari 2.6.1 是 Atari ST／STE／TT／Falcon 模擬器，目標包含遊戲與 demo 的硬體相容性。
- Hatari 採 GPL-2.0-or-later；Atari Talos 不複製、翻譯、移植或連結其程式碼。
- Hatari 的角色只限外部 oracle：相同 TOS、磁碟、機型、輸入與檢查點下產生獨立收據。
- 上述版本與授權資訊在開始實作硬體前仍須固定上游 commit 與檔案雜湊。

## Motorola 68000 語料

- 固定 `SingleStepTests/m68000` commit
  `64b253116a3de04aaac4346c43680960dc9b67e5`；LICENSE 為 MIT。
- 上游 README SHA-256：
  `7f36eed9cb93f061d13bfd18dc310c9bda7b5f3ac96b6d0dfe292e00612c46b6`。
- 語料由 MAME microcoded core 產生，含暫存器、RAM、prefetch、clock 與 bus transaction。
- 上游明列 TAS 的 5-cycle RMW timing 與 TRAPV 有疑義；這兩組目前不得作 confirmed oracle。
- NOP 2,500 筆顯示 `PC` 是下一次預取位址：執行 `prefetch[0]` 後，舊
  `prefetch[1]` 左移，在第 4 clock 從舊 PC 取 word，PC 再加 2。
- MOVEQ 2,500 筆確認 8-bit 符號延伸、D0–D7 選擇，以及 X 保留、N/Z 更新、V/C 清除；
  其預取與 program bus read 與 NOP 共用同一個 4-clock 路徑。
