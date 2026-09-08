# Dungeon Master 私人磁片驗證

日期：2026-09-08；分支：`feat/dm-raw-disk-verification-20260908`。
基底 `9e3b8bb` 已整合 main `90d94f7`。

## 第一層至第二層

在 ELIJA 招募基底後設定
`TALOS_BOOT_ACTIONS_TAIL=/src/docs/dm12en-stairs-actions.json`，共 63 動作。
自冷啟動正常走到地圖 0 的 `(6,9)` 踏板，開 `(5,9)` 門，經 `(3,15)` 樓梯
抵達地圖 1；再前進、左轉面向木門。沒有注入 PC、RAM 或座標。
動作 48／58／60／62 的開門、下樓前後及木門畫面加上原有招募三點雜湊，
全部冷啟動重播一致。最終 clocks=18149864946、Timer B events=402399、IACK=3952。

remake 同路線場景一致，正常 GUI 寫出的 ST 存檔為 map 1、(3,0) 面西、
ELIJA health=60。這是同操作與場景對帳，不是同步 RNG／tick 或逐像素證明。
remake 讀檔後三個額外骷髏欄位及場景變暗仍待查，原版遊戲寫盤仍未驗證。
私人圖像與兩輪日誌保存 DM `workplace/verification/st12-20260908/stairs-talos/`，
細節見 DM 的 `docs/verification/atari-st12-stairs-20260908.md`。本輪只加輸入，未改核心。

## 招募後衣袍檢查點

在既有 `dm12en-elija-actions.json` 後接 `dm12en-inventory-actions.json`，
共 59 動作，由冷啟動正常走到 ELIJA 招募後，右鍵開啟物品欄，取下 ROBE、
按住眼睛、放進背包第一槽、取回穿上。四個新增畫面雜湊冷啟動重跑一致。
已目視確認眼睛顯示 ROBE／WEIGHS 0.4 KG、背包與胸甲槽圖示移轉、
穿回後胸甲槽恢復。這些雜湊只證明 Talos 重播，不能當作 remake 像素對拍。

在既有私人開機測試環境，追加設定：

```sh
TALOS_BOOT_ACTIONS=/src/docs/dm12en-elija-actions.json \
TALOS_BOOT_ACTIONS_TAIL=/src/docs/dm12en-inventory-actions.json \
go test ./internal/st -run TestPrivateDiskBoot -v
```

其餘 ROM、raw disk、50M 啟動及 55M ENTER 後等待沿用下文。
眼睛按住 5M clocks；ReDMCSB `INVNTORY.C:1068`
`F352_aszz_INVENTORY_ProcessCommand71_ClickOnEye` 先呼叫 Delay(8) 再畫物品說明。
原始檔 SHA-256：`a202edfa0e8f35d96868b3f26ac8c525e3edbea973fd32b22f3a7a0aacb373dc`。
這僅解釋測試等待，沒有改寫原版程式。私人圖像與日誌不加入公開 repo。
遊戲寫盤、戰鬥及全狀態同步仍未驗收。
remake 同路徑的槽位取放與負重抽樣一致；規格 72 已修正持物眼睛誤顯技能與
放開未恢復的差異。新建置正常路徑顯示 ROBE／0.4 KG，放開回食物水頁，
與本收據語意一致。新圖保存 DM 私人 `eye-fixed/`，原失敗圖不覆寫。
此次未改 Talos 核心，沿用已重播通過的七點收據，不冒稱新跑原片或逐像素一致。

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
舊版設定 `TALOS_BOOT_ENTER=1` 回報 `st: ikbd is not in relative mouse mode`；
已由規格 154 修正開機預設值。同命令重跑按下／放開後顯示開門動畫，
framebuffer SHA-256 `7a092861ba0f516c800bd25cb219549c4b4cb4032c8f45fa5e7af9364e13bb26`，
Timer A IACK=4727。不代表已完成地城／招募驗收；新收據另存 `ikbd/`。
私人新收據在 `workplace/verification/st12-20260908/timera/`。
Hatari 可自然進地城並到 ELIJA 候選面板，**不代表 Talos 已完成對拍**。

## ENTER 後延長載入驗證

以下是規格 155 修正前的阻塞收據；Timer B 與正常招募已由下一節的新收據解決。

新增 `TALOS_BOOT_ENTER_STEPS`（1..100000000；預設仍為 1000000）控制
放開 ENTER 後的有界步數，只影響診斷測試，不改硬體或遊戲時序。
15000000／50000000 步畫面仍是入口走廊；磁頭持續推進，不能誤判為等待輸入。
設 100000000 後在第 51897669 步觸發真實拒絕：CPU PC `$0000F108`，
`write 1-byte bus fault at 0xfffa1b fc=5: unsupported_device_state`。
畫面已出現地城方向按鈕，但沒有完成可操作地城驗收。
framebuffer SHA-256 `15741e67de4ed7ac97707913649bbbbd8ef08cb8eb857c72834e8c922e1fb7b9`。
這是 Timer B 裝置缺口，不是 remake 規則差異；下一步須先完成 Timer B READY 規格。
私人收據另存 Dungeon Master 的 `workplace/verification/st12-20260908/timerb-gate/`。
失敗保留為失敗，不將「到達預期阻塞」改寫成整體測試通過。

## Timer B 修正後：正常移動與 ELIJA 招募

規格 155 完成 50 Hz 正常畫面的 Timer B 事件計數、中斷及 STOP 到期喚醒。
同片與 EmuTOS 1.3，自 reset 50000000 步點 ENTER，放開後再等 55000000 步，
執行 `docs/dm12en-elija-actions.json`。不是座標／角色／PC 注入。
`key` 是 IKBD 掃描碼（72 前進、82 左轉、71 右轉），按住 160000 clocks 後
正常放開；`dx/dy/left/right` 是實體滑鼠相對事件。`clocks` 控制每動作的等待。
三次向左上移動讓游標自然碰到邊界，再移到鏡子與招募按鈕，不直接改遊戲座標。

| 檢查點 | 動作索引 | framebuffer SHA-256 |
|---|---:|---|
| ELIJA 鏡子 | 32 | `37a7a237a8128849f9559dbd42cebf85966960a144ab7320fb50037d7398474c` |
| ELIJA 候選面板 | 38 | `85702d833c6e0782cf4bbc68bd5a653773547557a3ffcad7ee9df72a25541bb7` |
| RESURRECT 後隊伍列 | 41 | `9b55df6be46de173c40b849ed9ef328710e99176652456111bad0b3e22242a51` |

目視確認 `ELIJA RESURRECTED`、鏡子清空及角色加入隊伍。
候選姓名 ELIJA LION OF YAITOPYA、生命 60/60、體力 58/58、魔力 22/22、
負重 2.0/44 kg 與既有 Hatari／remake 收據一致。最終 Timer B events=237661、
IACK=2305，確認實際送入 CPU。各動作 PNG、輸入及日誌保存 Dungeon Master 的
`workplace/verification/st12-20260908/timerb-elija/`，不公開原片或遊戲畫面。

重跑：沿用前述 ROM／磁片環境，額外設定：

```sh
TALOS_BOOT_ENTER=1 TALOS_BOOT_ENTER_STEPS=55000000 \
TALOS_BOOT_ACTIONS=/src/docs/dm12en-elija-actions.json \
go test ./internal/st -run TestPrivateDiskBoot -v
```

JSON 的三個 `frame_sha256` 會使畫面不一致時測試失敗。這是本條固定路徑的
可重現檢查，不能取代新機制的原版證據。逐像素差異、raster palette、
物品取放、遊戲存讀檔、戰鬥／法術及全遊戲對拍仍未驗收，remake 分數不提高。

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
