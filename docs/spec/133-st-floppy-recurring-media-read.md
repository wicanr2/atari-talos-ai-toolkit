# 133 — ST floppy可重入媒體確認讀取循環

狀態：**CONFORMED**。

## 決策與範圍

使用者於2026-09-08確認：媒體確認改為可重入循環模型，排除為每次呼叫增加固定stage。
使用者隨後補充：現有約50個逐輪軟碟收據欄位也必須一併移除，不保留永久相容層。
本切片將規格122–132前三次精確數值遷入同一種receipt作回歸錨點，所有高階
`flop_mediach()`皆以一份目前交易狀態、單調attempt count及8筆有界receipt ring處理。
有磁片DMA成功、資料解碼及Dungeon Master載入另立規格。

## 證據

- **已確認（EmuTOS原始碼）**：固定commit
  `95eb9e498e979022dd9626f528d32de861f26c85`，`bios/floppy.c:455–550
  flop_mediach`會在write-protect latch條件下呼叫`flopio()`；`1081–1185 flopio`中
  `flopcmd()` timeout立刻設`EDRVNR`並break，`1322–1341 flopunlk`仍呼叫
  `1481–1501 dummy_seek`。`IO_RETRIES=2`不會在timeout分支內重試。`floppy.c`
  SHA-256 `cb6f8e38a0fc29f7321ded51892ec47ae11d7a84f327c0563a3ecb85d6ffa37b`。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1 oracle image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。700-VBL
  `fdc,psg_write,psg_read` trace SHA-256
  `267942b9e9f5a9d530253220484b9c36b959804d080a3808d33ca0600dfe18dd`顯示媒體讀取
  command於VBL235、310、389、464、539、620持續出現，700 VBL仍未終止；固定次數
  stage因此與原版正常路徑矛盾。
- **已確認（Talos入口）**：規格132於stage 68完成第三次dummy seek，下一gate為
  4,601,570 instructions／2,903 interrupts／142,994,602 clocks的YM2149 R14 select。

## typed行為

1. `mediaReadPhase`描述一輪固定序列：PSG select/read/write；sector selector/data；DMA
   address low/mid/high；direction reset兩筆；count；Type-II `$80`；guest timeout後
   `$0080/$D0`；data 0與Type-I `$13`；scheduler完成；status `$E4` read-clear。
2. 每輪開始遞增attempt count並清current receipt；完成時寫入容量8的ring。ring覆寫最舊
   項，不改變單調總數。原有逐輪欄位全部移除，前三輪測試改查同型receipt。
3. 所有register、值、寬度、權限與順序仍失敗即關閉；DMA buffer不因無磁片而改變。
   PSG未定時`MOVE.B`沿用Machine instruction epoch fallback，其餘保存bus clock。
4. seek重用728-FDC-clock scheduler；timeout由guest Timer C／`hz_200`形成，Talos不造
   額外計時器。cold reset清phase、counter、current與ring。

## 驗收與停止線

- 新資料模型先以獨立測試連續完成至少12輪，證明ring wrap、attempt單調、查詢與reset；
  再遷移所有舊分支並以完整synthetic序列證明錯序原子性及RAM不變。
- 固定ROM越過第四、第五輪，至少抵達700-VBL oracle涵蓋範圍，且前三輪CPU／clock錨點不變。
- 固定ROM、完整240,000筆CPU corpus、全測試、vet與build通過才升 **CONFORMED**。
- 本切片只泛化既有硬體錯誤路徑，不改遊戲、存檔、素材、protocol或發行權利。

## CONFORMED 收據

- 2026-09-06：前三輪的遷移層拆掉了。`floppyReadStage`（0–68 的固定 stage 機）與
  `floppyMediaLegacy[3]`（約 50 個逐輪欄位）整組移除，三輪與之後的每一輪都走同一條
  `mediaReadPhase` 循環。`internal/st` 淨減 499 行，其中 `memory.go` 少了約 550 行。
- 第一輪與後續輪的兩個差異收進 receipt 的 `LockedTrack`：只有第一次呼叫
  `flop_mediach()` 會鎖 track（DMA `$0082` ＋ track 0 兩筆交易），因此那一輪的
  drive 重選讀回的是 `flopvbl()` 留下的 `$23`、sector selector 寫在 `$0082` 上；
  之後每一輪都是 `$25` 同值重選、`$0080`。`floppyMediaLocked` 記住整台機器有沒有
  鎖過，決定 `Idle` 那一步走 track lock 還是直接重選 drive。
- **失敗即關閉的範圍變嚴了**：DMA 位址的三個位元組，先前只有第二輪起會拒絕亂序寫入，
  第一輪靜靜地不推進；現在只要循環在跑（phase 不是 `Idle`），任何不合當下 phase 的
  位址寫入都 fault。這是規格「順序仍失敗即關閉」的直接後果，測試也改成期望 fault。
- 固定 ROM 的錨點一個都沒動：第一輪 sector-read setup 仍在 1,286,164 instructions／
  106,340,824 clocks，第三次 dummy seek 仍在 4,600,755／142,982,988，第五輪完成仍在
  6,775,690 instructions／3,655 interrupts／167,087,528 clocks。
  （那之後原本是 IKBD `$FFFC02` 的 gate，規格 138 已經打通，所以回歸點改記在第五輪
  完成的那一刻而不是 gate 上。）
  前三輪的 receipt 逐欄位與遷移前相同（`DriveWriteClock`、`ReadCommandClock`、
  `TimeoutSelectorClock`、`ForceInterruptClock`、`SeekStartClock`、`StatusReadClock`、
  `InactivePolls`、`IRQObserved`）。
- 測試改查同型 receipt：完成的一輪從 ring 取（`mediaReceipt(m, n)`），還在跑的那一輪
  取 `floppyMediaCurrent`；驅動迴圈的「跑到 stage N」改成「第 n 輪的 phase p」
  （`reachedMedia`），因為 phase 每一輪都會重來。
- 完整 CPU 語料、EmuTOS 端到端、UCSD 那一組、vet 與 build 全數通過。
