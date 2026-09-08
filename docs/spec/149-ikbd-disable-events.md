# 149 — IKBD 停止事件回報

狀態：**READY**

已確認：Atari IKBD 協定 §8.13、§8.21 定義 `$12` 停止滑鼠事件，
`$1A` 停止搖桿事件，均無參數；有效模式設定命令可重新啟用。
來源：<https://www.kernel.org/doc/html/v4.12/input/devices/atarikbd.html>。
Talos Automation 097 自 reset 走至 4097505 steps，在 ROM `$FC5154`
`move.b d2,$fffc02`（D2=$12）遭拒；輸入雜湊見規格 148。

契約：增加 disabled 狀態，`$12/$1A` 走既有 TX／TDRE 與命令組裝器，
不建立回覆。滑鼠 disabled 時外部動作不排上行封包，`$08` 解除 disabled。
不清除先前已排程封包，不改動鍵盤。cold reset 清除新增狀態。
搖桿仍沒有輸入 API；接受停用命令不代表已有搖桿模擬。

驗收：停用後滑鼠不排封包，鍵盤仍可排；恢復相對模式後滑鼠再次可排。
既有未知命令、TDRE、cold reset 檢查仍通過。
