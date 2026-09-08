# Dungeon Master 私人磁片驗證

日期：2026-09-08；分支：`feat/dm-raw-disk-verification-20260908`。
基底 `9e3b8bb` 已整合 main `90d94f7`。

## 已確認

輸入 `automation_a_097.st` 為 839680 bytes，SHA-256：
`afab0a6d2b41eec7420a76e174e96a906687cb959ea563f9343672f0b3febc23`。
開機區檔案偏移 `0x0B` 的 little-endian word 為 512 bytes/sector，
`0x13` 為 1600 sectors，`0x18` 為 10 sectors/track，`0x1A` 為 2 sides。
BPB 共宣告 819200 bytes；檔案另有 20480 bytes。

實跑 `NewRawFloppy` 拒絕：
`st: raw floppy length 839680, BPB requires 819200`。
尚未抵達遊戲啟動，遊戲版本亦未確認。
來源目錄：<https://ataristdb.sidecartridge.com/db/d.csv>，
項目 `Dungeon Master`／`AUTOMATION/A_097.ST`。

## 重跑與下一步

在 Go Docker 工具鏈中唯讀掛載磁片為 `/disk.st`，設定
`TALOS_RAW_FLOPPY=/disk.st`，執行
`go test ./internal/st -run '^TestPrivateRawFloppy$' -v`。
測試輸出大小與 SHA-256，拒絕時回報理由；成功時逐 CHS 核對全部 bytes。
未設定私人輸入時明確跳過。

下一步先確認額外 20480 bytes 的用途及媒體幾何，對照格式文件與 Hatari
外部行為；若需要支援實體磁軌多於 FAT 宣告容量，先完成 READY 規格。
不截短輸入、不改寫 BPB 來繞過現行驗證。

整合後完整 `go test ./...` 通過，但未提供 ROM／外部語料的條件測試
不計入動態驗收。規格 142–145 在兩個分支各有不同主題，引用須附完整
檔名。取得版本、啟動及正常玩家路徑收據前，維持 remake 原有分數。
