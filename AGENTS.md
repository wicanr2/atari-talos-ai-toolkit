# Atari Talos AI Toolkit 工作規則

- 面向使用者與主要文件預設使用繁體中文。
- 所有分析、建置、測試、轉檔與執行一律在 Docker 內進行；主機只作 Git 與 Docker 控制。
- 一次性容器使用 `--rm`、資源上限與 `--network none`；每輪確認沒有殘留容器。
- Hatari 只作外部 oracle。禁止翻譯、移植、連結或複製 Hatari／Steem 的 GPL 程式碼。
- 依公開硬體文件寫規格；規格未達 READY，不得實作對應硬體行為。
- 每條相容性聲明都要有可重跑證據。單元測試只證明內部自洽，不等於 Atari ST parity。
- 未實作、未知或輸入不完整時必須失敗即關閉，不得以空白畫面或預設值冒充成功。
- TOS ROM、遊戲磁碟、IPF／STX 映像、截圖與其他原版素材不得加入 Git 或公開發行包。
- README 保存用途、穩定入口與現況摘要；歷史寫入 `WORKLOG.md`，研究證據寫入
  `RESEARCH-LOG.md`，目前真相寫入 `CONTEXT.md`。
- 正式版號使用 `v.<major>.<minor>.<patch>-YYYYMMDD`。
