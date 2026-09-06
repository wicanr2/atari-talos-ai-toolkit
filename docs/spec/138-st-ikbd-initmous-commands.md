# 138 — ST IKBD `Initmous` 的四條主機命令

狀態：**CONFORMED**。

## 範圍與證據

本切片處理 EmuTOS 1.3 在讀完時鐘之後送給 IKBD 的四條滑鼠設定命令，也就是
目前開機路徑的第一失敗點（`$FFFC02` write）。鍵盤與滑鼠的**封包上行**、
joystick、memory load／read、controller execute 與其餘 IKBD 命令不在範圍。

- **已確認（Talos 探針，一次看完）**：暫時讓 IKBD data write 接受一切並記錄，
  固定 ROM 從 6,779,282 條一路跑到 8,058,248 條才撞到下一個 gate（PSG `$FF8802`），
  中間主機送出的位元組恰好是 `08 0b 01 01 10 07 00`，**七個位元組、四條命令**。
  這是「跳到事件本身」而不是逐條撞 gate 的做法——一次拿到整串，才知道這一片
  要涵蓋多少。
- **已確認（Hatari 外部 oracle，獨立解出同一串）**：Hatari 2.4.1 image
  `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`、
  同一顆 ROM `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`，
  `--run-vbls 1450 --trace ikbd_acia,ikbd_cmds`，在 **VBL 615 的 HBL 33–155**
  依序解出：

  ```text
  IKBD_Cmd_RelMouseMode            ← $08
  IKBD_Cmd_SetMouseThreshold 1,1   ← $0B 01 01
  IKBD_Cmd_SetYAxisUp              ← $10
  IKBD_Cmd_MouseAction 0           ← $07 00
  ```

  1450 個 VBL 裡除了既有的 reset／read clock／set clock 之外**沒有其他命令**，
  兩邊對這一串的切法完全一致。切法本身因此不是猜的：`$0B` 吃兩個參數、
  `$07` 吃一個、`$08` 與 `$10` 沒有參數，否則兩邊不可能切出同樣的四條。
- **已確認（同一份 trace）**：這四條命令**都不回應**。送出期間 `ikbd acia
  tx_state` 一直是 0，只有 `rx_state` 在收；`$1C` 那種 interrogate 才會讓
  IKBD 回傳。所以本切片不建立任何 response deadline。
- **已確認（EmuTOS ROM）**：`$FC5142` 是 `ikbd_writeb`——`$FC5132` 讀
  `$FFFC00` 測 TDRE（`btst #1`），不成立就重試，成立才 `move.b d2,$fffc02`
  （`$FC5154`）。所以每一個位元組都各自等 TDRE，這一片不改既有的 TX 時序。

## typed 行為

1. IKBD 主機命令是「命令位元組 ＋ 固定長度參數」。本切片認得四個命令碼，
   長度分別是：`$07` 一個、`$08` 零個、`$0B` 兩個、`$10` 零個。
   **其餘命令碼一律失敗即關閉**——不把未知命令當成無參數命令吞掉。
2. 命令位元組寫入時開始組裝：長度為零就當場完成，否則後續的位元組依序當參數
   收滿為止。參數期間再收到的位元組**不重新解讀成命令**。
3. 完成時把效果記進裝置狀態：`$08` 設相對滑鼠回報、`$0B` 記下兩個門檻值、
   `$10` 設 Y 軸原點在上、`$07` 記下 button action。這些值在開機路徑上沒有
   讀回端，記錄是為了讓後續切片有東西可查，不是為了對拍。
4. TX 時序沿用既有機制：`ikbdACIATDR` ＋ `TXPending` ＋ TDRE 清除，由
   `advanceIKBDACIAClock` 推進。本切片**不建立 response deadline**，因為這四條
   都不回應。
5. 既有的 `$80,$01` reset、`$1B` set clock、`$1C` interrogate 分支不變，
   優先於本切片的命令組裝器。
6. cold reset 清掉組裝器與四個設定值。

## 驗收與停止線

- synthetic：四條命令逐條送完、參數收滿才生效、參數期間的位元組不被當成命令、
  未知命令碼 fault、TDRE 未設時 fault、cold reset 清空。
- 固定 ROM 必須從 6,779,282 條越過這四條命令，抵達新的第一失敗點，且先前的
  錨點（第五輪媒體確認的 6,779,282 instructions／3,656 interrupts／
  167,143,396 clocks）不變。
- 完整 CPU 語料、全測試、vet 與 build 通過才升 **CONFORMED**。
- **不宣稱 IKBD 已建模**：滑鼠與鍵盤的上行封包、joystick、`$0D`／`$0E` 這類
  會回應的滑鼠命令、以及這四條設定值對後續封包的實際效果，都不在本切片。
  這裡做的只是「主機送得出去、送完 ROM 繼續跑」。

## CONFORMED 收據

- 2026-09-06：五條 synthetic 測試通過——七個位元組的完整序列、參數期間的
  `$08`／`$10` 被當成門檻值而不是命令、七個未知命令碼（`$09`／`$0C`／`$0D`／
  `$0E`／`$14`／`$20`／`$FF`）全部 fault 且不留狀態、TDRE 未空時拒絕第二個位元組、
  cold reset 清乾淨。
- 固定 ROM 從先前停住的 6,779,282 條走完這四條命令，抵達新的第一失敗點
  **8,058,248 instructions／4,088 interrupts／180,984,736 clocks**，gate 是
  `$FC36DE` 的 PSG `$FF8802` write。四個設定值在那一刻都已套用。
- 完整 CPU 語料、EmuTOS 端到端、UCSD 那一組、vet 與 build 全數通過。
- **驗收的重心在 synthetic**：固定 ROM 那八百萬條只證明開機路徑走得過去，
  行為本身是由上面五條釘住的。這是使用者 2026-09-06 給的對拍方法——
  把狀態設到事件觸發的那一刻，不從 reset 長跑去等它發生。
