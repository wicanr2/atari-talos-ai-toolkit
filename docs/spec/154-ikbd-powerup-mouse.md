# 154 — IKBD 開機滑鼠預設值

狀態：**CONFORMED**（先完成 READY 契約再實作）。

已確認：Atari IKBD 協定 Power-Up Mode、RESET、SET RELATIVE MOUSE POSITION
REPORTING 與 SET MOUSE THRESHOLD 定義：開機及控制器 reset 後為相對模式、
門檻 1/1、Y 原點在上、button action=0。
來源：https://www.kernel.org/doc/html/v4.12/input/devices/atarikbd.html

修正規格 138 的「cold reset 清設定值」：清除命令組裝及輸入佇列仍成立，
但設定值必須恢復硬體預設而非 Go 零值。既有 ACIA reset 不等同控制器 reset。
控制器已建模的 80/01 完成時也恢復滑鼠預設；不改時鐘與既有回覆期限。
不擴張未知命令、搖桿、絕對模式或輸入封包範圍。

驗收：cold reset 不需 08 命令即可排相對封包；非預設設定恢復預設；
參數組裝仍不誤執行命令；完整回歸；DM12EN 同片正常按下／放開 ENTER。
原片收據沿用規格 153 的 ROM／磁片雜湊，不修改遊戲記憶體或 PC。

驗收結果：完整 Go 測試（含外部 68000 語料）、vet 通過；
`TestIKBDPowerupMousePacketWithoutInitmous` 與
`TestIKBDControllerResetRestoresMouseDefaults` 通過。
`TALOS_BOOT_ENTER=1` 同片乾淨重跑通過，目視確認開門動畫；
不是已進地城或完整遊戲對拍的聲明。截圖與日誌留在私人證據目錄。
