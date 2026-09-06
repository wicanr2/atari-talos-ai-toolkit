# 140 — `flopvbl()` 選哪個 drive 由 ROM 決定，收工時把兩個都放掉

狀態：**CONFORMED**。

## 範圍與證據

規格 121 以來，模型用「已經跑完幾輪」推這一輪要檢查哪個 drive
（`flopVBLMediaDrive = flopVBLMediaChecks & 1`）。那個假設撐了 76 輪，第 77 輪破掉：
進場值 `$25`、模型算出 drive 0（目標 `$25`），ROM 卻寫 `$23`。

本切片做兩件事：把 drive 的來源從「模型預測」改成「從 ROM 的寫入觀察」，
並補上一次性的 deselect（`$27`）。範圍是 port A 的 data 那一步；drive 輪替的
節奏、FDC status 讀取、媒體確認與收尾還原都不變。

- **已確認（Hatari 外部 oracle）**：Hatari 2.4.1 image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  EmuTOS 1.3 UK ROM `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`，
  `--run-vbls 1450 --trace psg_write,fdc`。**輪替是 ROM 內部的自由計數，不是
  「已執行輪數」**：VBL 66 起每 8 個 VBL 一輪、drive 0／1 交替；VBL 242–610 之間
  磁碟在忙，**連續 47 個輪詢時槽被跳過**（奇數），而 VBL 618 恢復時仍是 drive 1
  ——若計數只在真的跑完一輪時前進，那裡的奇偶就會翻過來。機器端看不到那個計數器。
- **已確認（同一份 trace）**：deselect 是**獨立的一次 `set_psg_porta`**。
  VBL 1130 的完整形狀是

  ```text
  $25 → $23   選 drive 1（pc=fc36dc）
  $FF8606 ← $0080、讀 $FF8604 = $E4
  $23 → $25   還原進場值（pc=fc36dc）
  $25 → $27   deselect（pc=fc36ca 選 R14、pc=fc36dc 寫 data）
  ```

  最後那一次前後沒有任何 FDC 存取，走完 data 就結束。之後每一輪的進場值都是
  `$27`：VBL 1138 是 `$27 → $25 → $27`、VBL 1146 是 `$27 → $23 → $27`。
- **已確認（EmuTOS ROM）**：`$FC36CA` 選 R14、`$FC36DC` 寫 data，就是規格 139 那支
  `set_psg_porta`；deselect 走的是同一支，只是傳進去的值是 `$7`。

## typed 行為

1. 共用前置（選 R14、讀回進場值）不變，進場值仍限 `$23`／`$25`／`$27`。
2. **data 那一步才知道這一輪選的是哪個 drive**：寫 `$25` 記成 drive 0、寫 `$23`
   記成 drive 1，其餘（除下一條的 `$27`）失敗即關閉。進場時 `flopVBLMediaDrive`
   設成 `-1`（未知），收尾與 status 讀取仍以記下來的值檢查。
3. **寫 `$27` 是 deselect**：把兩個 drive 都放掉，該次 `set_psg_porta` 到此結束
   （stage 退回 8），不讀 status、不計入 `flopVBLMediaChecks`。之後每一輪的進場值
   就是 `$27`，收尾也還原成 `$27`——那部分規格 139 已經支援。
4. 媒體確認借用共用前置的分派（DMA control `$0084`）不變；它一定寫 `$25`，
   所以第 2 條記下的 drive 0 與它一致。
5. 其餘（drive 輪替的 8-VBL 節奏、FDC status 的選擇與讀取、`flopVBLMediaChecks`
   的計數、cold reset）都不變。

## 驗收與停止線

- synthetic：同一個 `flopVBLMediaChecks` 值之下兩個 drive 都走得完整一輪
  （證明不再由 checks 決定）、deselect 之後 stage 回 8 且 checks 不變、
  deselect 之後的下一輪以 `$27` 當進場值並還原成 `$27`、data 那一步寫非 drive
  選擇值失敗即關閉。
- 固定 ROM 必須從 10,544,770 條越過這個 gate，且先前的錨點不變。
- 完整 CPU 語料、全測試、vet 與 build 通過才升 **CONFORMED**。
- **不擴大**：本片**移除**了一個假設而不是加上新的。模型不再宣稱知道
  `flopvbl()` 這一輪要檢查哪個 drive——那是 ROM 內部計數器的結果，機器沒有辦法
  觀察，硬猜只會在下一次跳過輪詢時再錯一次。換來的代價是：data 那一步接受兩個
  合法的 drive 選擇值，不再因為「猜錯邊」而 fault。其餘每一步（進場值、階段順序、
  收尾還原、status 讀取的前置值）都仍然失敗即關閉。
- 也**不宣稱** port A 的其他位元（motor、side、printer strobe）已建模，
  `$27` 只當成「兩個 drive 都不選」。

## CONFORMED 收據

- 2026-09-06：四條 synthetic 測試通過——同一個 checks 值下 `$23` 與 `$25` 兩條路
  都走得完；deselect 之後 stage 回 8、checks 不變、port A 是 `$27`；緊接著的下一輪
  以 `$27` 進場並還原成 `$27`；data 那一步寫 `$21`／`$24`／`$26`／`$00` 一律 fault
  且不改狀態。
- 固定 ROM 從 10,544,770 條推進到**開機完成**：EmuTOS 1.3 走到 GEM 桌面，
  1.2 億條指令（240 億 clocks、343,855 次中斷、18,741 輪 `flopvbl()`）之內
  沒有再撞到任何未建模的存取。畫面在 12,500,000 條之前就畫完並保持不變。
- 桌面畫面的錨點：`$F8000` 起的 32,000 bytes SHA-256
  `1de1eb45e862218844abe07ae05fda4c4a9453817ed0ab348a374bca67768f78`，
  解析度 0（320×200 四平面），非零位元組 8,867。內容是選單列
  `Desk File View Options`、`DISK A`／`DISK B` 兩個磁碟圖示、`TRASH` 垃圾桶
  與滑鼠游標。
- 先前的錨點沿用同一組數字：第一輪 sector-read setup 1,286,164／106,340,824、
  第三次 dummy seek 4,600,755／142,982,988、第五輪媒體確認 6,775,690／167,087,528、
  第二輪檢查點 2,372,056／118,369,888。
- 完整 CPU 語料、EmuTOS 端到端、UCSD 那一組、vet 與 build 全數通過。
