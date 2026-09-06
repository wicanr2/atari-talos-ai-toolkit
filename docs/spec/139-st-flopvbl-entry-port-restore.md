# 139 — port A 的前置是共用的，`flopvbl()` 還原進場值

狀態：**CONFORMED**。

## 範圍與證據

第 74 輪的 `flopvbl()` 過不去，根因有兩層，兩層都是「既有模型假設太緊」：

1. 規格 121 把 `flopvbl()` 每一輪的前置與收尾值寫死成 `$23`。那在前 73 輪成立，
   因為那之間沒有別的東西改過 port A；第 74 輪不成立——`flop_mediach()` 的媒體
   確認在它之前跑完，留下 `$25`。
2. 更深一層：**`set_psg_porta` 的前兩步（選 R14、讀回舊值）是共用的**，
   `flopvbl()` 與媒體確認都會走。規格 133 讓媒體確認在 **select 那一步**就認領，
   判準是「R14 目前是 `$25`」。第 74 輪的 `flopvbl()` 進場時 R14 剛好也是 `$25`，
   於是被誤認成媒體確認的開始，寫 `$23` 時才 fault。

本切片把前置改成共用，分派點移到 **data 寫入**——那是兩者第一次真的不同的地方。
範圍是 port A 的前置、分派與還原；drive 輪替、FDC status 讀取與媒體確認自身的
後續步驟都不變。

- **已確認（Hatari 外部 oracle）**：Hatari 2.4.1 image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS 1.3 UK ROM `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`，
  `--run-vbls 1450 --trace psg_write,fdc`。`fdc change drive/side` 那一行同時
  印出舊值與新值，整份 trace 裡**每一輪都成對**：

  ```text
  VBL=66    io_porta_old=0x3 io_porta_new=0x5   drive 1->0     ← 前 73 輪
  VBL=66    io_porta_old=0x5 io_porta_new=0x3   drive 0->1        還原
  VBL=618   io_porta_old=0x5 io_porta_new=0x3   drive 0->1     ← 第 74 輪
  VBL=618   io_porta_old=0x3 io_porta_new=0x5   drive 1->0        還原
  VBL=1418  io_porta_old=0x7 io_porta_new=0x3   drive -1->1    ← 更後面
  VBL=1418  io_porta_old=0x3 io_porta_new=0x7   drive 1->-1       還原
  ```

  **三種進場值都出現過**（`$3`、`$5`、`$7`，也就是完整 byte 的 `$23`／`$25`／`$27`），
  而每一輪的形狀都一樣：讀舊值、設成這一輪的目標 drive、做事、寫回舊值。
  寫死 `$23` 只是前 73 輪碰巧成立。
- **已確認（EmuTOS ROM）**：`$FC36B8 set_psg_porta` 讀 R14、`andi.b #$f8` 留高 5 位、
  `andi.b #$07` 取新值低 3 位、`or.b` 合併後寫回，並**回傳舊值的低 3 位**
  （`$FC36E4` 的 `andi.b #$07,d0`）。呼叫端拿那個回傳值在收尾時還原——
  這就是 trace 裡那組成對變化的來源。
- **已確認（Talos 探針）**：暫時放行未建模的 PSG 與 FDC 存取之後，第 74 輪的
  五筆 I/O 依序是 `R14=$23`、`$FF8606=$0080`、`$FF8604` 讀回 `$E4`、`R14=$25`、
  再選 R14——與 Hatari 的 VBL 618 完全一致。

## typed 行為

1. `set_psg_porta` 的前兩步（寫 `$FF8800` = 14、讀 `$FF8800`）是**共用前置**，
   由 `flopvbl()` 的 stage 機接手，讀回時記下進場值（`flopVBLMediaEntryPort`）。
   合法的進場值只有 `$23`、`$25`、`$27`——其餘失敗即關閉。
2. **分派在 data 寫入那一步**：進場值與寫入值都是 `$25`（同一個 drive 重選）
   就是媒體確認的開始，其餘是 `flopvbl()` 這一輪要切到的目標 drive。
   兩者第一次不同的地方就在這裡，前面分不出來也不該分。
3. `flopvbl()`：選目標 drive 時前置值必須等於進場值，寫入值是
   `flopVBLTargetPort()`（drive 0 → `$25`、drive 1 → `$23`）；收尾時**寫回進場值**，
   前置值必須是目標 drive 的值。
4. 媒體確認：由 DMA control `$0084` 那一步直接進 `floppyMediaSectorData`。
   借用的那一輪 `flopvbl()` stage 退回 8、不計入 `flopVBLMediaChecks`（那一輪沒有
   真的跑）。drive 寫入的 clock 先存在 `flopVBLMediaDriveWriteClock`，分派時才寫進
   收據——因為那一步發生時還不知道是誰要用。
   第一輪（track lock 之後）仍走自己的 `DriveSelector`／`DriveRead`／`DriveWrite`：
   它的進場值是 `$23`，與第二輪起的 `$25` 分得開。
5. drive 輪替、FDC status 的選擇與讀取、`flopVBLMediaChecks` 的計數都不變。
6. cold reset 清掉進場值。

## 驗收與停止線

- synthetic：三種進場值（`$23`／`$25`／`$27`）各跑一輪完整的輪詢，證明還原的是
  進場值而不是 `$23`；前置值不符時失敗即關閉；非法進場值失敗即關閉；
  同值 `$25` 走到媒體確認、不同值走到 `flopvbl()`。
- 固定 ROM 必須從 8,058,248 條越過第 74 輪的輪詢，抵達新的第一失敗點，
  且第 73 輪之前的錨點不變。
- 完整 CPU 語料、全測試、vet 與 build 通過才升 **CONFORMED**。
- **不擴大**：媒體確認自身的後續步驟（規格 133 的 sector selector 之後）與 drive
  輪替的計數（規格 121）都不變，也不宣稱 port A 的其他位元（motor、side、
  printer strobe）已建模。規格 133 的 phase 列表因為本片而少用三個，
  那份規格的 typed 行為第 1 條同步更新。

## CONFORMED 收據

- 2026-09-06：四條 synthetic 測試通過——三種進場值（`$23`／`$25`／`$27`）各跑一輪
  完整的輪詢並還原進場值、把 `$25` 的進場值還原成 `$23` 失敗即關閉、六個非法進場值
  在讀回那一步就失敗且不留下記錄、媒體確認借用共用前置時 data 那一步不分派而
  `$0084` 才分派（收據帶著暫存的 clock，借用的那一輪不計入 checks）。
- 固定 ROM 從 8,058,248 條推進到 **10,544,770 instructions／4,967 interrupts／
  209,189,796 clocks**：`flopvbl()` 多走了三輪（checks 73 → 76），媒體確認多走了
  兩輪（ring 6 → 8）。新的第一失敗點是同一支 `set_psg_porta` 寫一個還沒建模的
  port A 值——從 Hatari 的 VBL 1418 看，那是 `$27`（兩個 drive 都不選）。
- 先前的錨點沿用同一組數字：第一輪 sector-read setup 1,286,164／106,340,824、
  第三次 dummy seek 4,600,755／142,982,988、第五輪媒體確認 6,775,690／167,087,528。
  第二輪的檢查點因為分派移到 DMA control 而往後挪了一步（2,372,056／118,369,888），
  收據的 `DrivePort` 與 `DriveWriteClock` 仍是 `$25`／118,369,158。
- 完整 CPU 語料、EmuTOS 端到端、UCSD 那一組、vet 與 build 全數通過。
