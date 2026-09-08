# Dungeon Master 私人磁片驗證

日期：2026-09-08；分支：`feat/dm-raw-disk-verification-20260908`。
基底 `9e3b8bb` 已整合 main `90d94f7`。

## 後續現況：DM12EN 原片資料

Automation 合輯已透過規格 148 明示 82/2/10 幾何載入，但版本未明且 crack intro
需要額外硬體；改採使用者接受的版本可追溯無防拷測試盤，不改原始資料。
STX 內 START.PAK SHA-256 與 ReDMCSB `ORIGINAL/START.PAK DM12EN` 相同：
`0c09079cb84bc5bcde7bba77a364b14100f0d18e43953f8c73934768117ea549`。

來源：<https://atari.8bitchip.info/ASTGA/D/DungeonMaster1v2E.zip>。
PRIVATE `dm12en-nocp-10spt.st` 為 80/2/10、819200 bytes，SHA-256
`a1e6c4e3573babf633afe51dc321eb6279e915137897ccef3b9c7f274cf04873`。
內容：ReDMCSB `BUILD/ENGINE/NOCP/DM12EN.PRG` 放 `AUTO/DM.PRG`，
同片 GRAPHICS.DAT／DUNGEON.DAT 放根目錄。不是原始磁片逐位元映像，無防拷驗證
不能拿滿原版證據分數。完整輸入來源、雜湊與 Hatari 結果記在 Dungeon Master 專案
`docs/verification/atari-st12-20260908.md`；私人收據在該專案
`workplace/verification/st12-20260908/`，不進本 public repo。

新增規格 148–152 支援已通過全套 Go 測試及 vet；MOVEP 5,000 筆外部語料
比較 state／RAM／clock／transactions。舊「未支援即拒絕」測試僅移除本輪已證實
支援的值，保留未支援值的拒絕與無副作用檢查。

重跑：Docker 唯讀掛載私人盤與 EmuTOS 192k 1.3 ROM，設定
`TALOS_BOOT_DISK`、`TALOS_TOS_ROM`、`TALOS_BOOT_STEPS=50000000`，執行
`go test ./internal/st -run TestPrivateDiskBoot -v`。此測試自 reset，不改 PC／RAM；
步數到限不等於遊戲通過。截圖只用當下整幅 palette，非 raster palette 精確證據。
舊收據在 step 20344161、clocks 307257568、ROM PC `$FC6266` 阻塞於
Timer A `$FFFA19` byte write（FC=5）；此阻塞已由規格 153 解決。
目前 Talos 自 reset 五千萬步後自然顯示 ENTER／RESUME，framebuffer SHA-256
`e38e319d828abb6e3302ac45692fc104c2e8c99af14a8af822b9b864961a14c7`。
Timer A mode=1、data=112、timeouts=11579728、IACK=0；該路徑未啟用 A 中斷。
設定 `TALOS_BOOT_ENTER=1` 追加正常滑鼠輸入時，回報
`st: ikbd is not in relative mouse mode`，為下一個獨立阻塞。
私人新收據在 `workplace/verification/st12-20260908/timera/`。
Hatari 可自然進地城並到 ELIJA 候選面板，**不代表 Talos 已完成對拍**。

以下為最初盤點收據；BPB 阻塞已由明示幾何 API 解決，不再當成目前待辦。

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
