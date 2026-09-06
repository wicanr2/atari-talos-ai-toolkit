# 145 — IKBD 的鍵盤 make／break 掃描碼

狀態：**CONFORMED**。

## 範圍

鍵盤往主機方向的掃描碼：按下送 make、放開送 break。走的是規格 142 建立的
同一條上行佇列。

**不在本切片**：掃描碼與字元的對應（那是 ROM 的鍵盤表）、修飾鍵的狀態機、
`Caps Lock` 的 LED、joystick 與 button-as-key。

## 證據

- **已確認（Atari IKBD 協定文件）**：「The keyboard always returns key
  make/break scan codes. The key scan make (key closure) codes start at 1 …
  The break code for each key is obtained by ORing 0x80 with the make code.」
  <https://www.kernel.org/doc/Documentation/input/atarikbd.txt>
  所以鍵盤事件是**單一位元組**，沒有表頭。
- **已確認（EmuTOS 1.3 固定 ROM，端到端）**：桌面上用滑鼠開出
  `Desk → Desktop Info...` 的對話框（29,830 個像素變動）之後——
  - 送 `$02`／`$82`（`1` 鍵按下與放開）→ **0 個像素變動**，對話框留著。
  - 送 `$1C`／`$9C`（Return 按下與放開）→ **29,256 個像素變動**，對話框關掉。

  一個負對照加一個正對照，證明掃描碼真的被 ROM 解讀，而不是「送什麼都會關」。

## typed 行為

1. `QueueKey(scanCode, pressed)` 排一個位元組進上行佇列：按下是 `scanCode`，
   放開是 `scanCode | $80`。
2. 掃描碼只接受 `$01`–`$72`。0 與 `$73` 以上**失敗即關閉**——`$74`／`$75` 是
   button-as-key 用的，不在範圍。
3. 佇列滿時失敗即關閉。投遞、RDRF、GPIP4 與 IPRB 的效果與規格 142 相同，
   鍵盤與滑鼠共用同一條線。

## 驗收與停止線

- synthetic：make 與 break 的位元組值、非法掃描碼 fail-closed、佇列滿 fail-closed、
  與滑鼠封包共用佇列時的先進先出。
- 固定 ROM：上面那組正負對照。
- **不宣稱鍵盤已完整建模**：修飾鍵、重複鍵、鍵盤表與 LED 都不在這裡。

## CONFORMED 收據

- 2026-09-06：synthetic 與固定 ROM 都通過，數字如上。這一片能動的前提是
  規格 144——按鍵會讓 EmuTOS 彈一聲，那串 PSG 寫入本來會直接撞牆。
