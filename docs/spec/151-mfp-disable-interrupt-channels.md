# 151 — MFP 中斷通道停用

狀態：**READY**

已確認：Motorola MC68901 手冊 §4.3.1，IER bit 寫零停用對應通道、
清除同 bit 的 pending，不清除 in-service。
<https://www.nxp.com/docs/en/reference-manual/MC68901UM.pdf>。
已確認（DM12EN NOCP 自 reset 實跑）：20374232 steps 的 ROM `$FC61A4`
嘗試寫 IERB `$40`，先前 `$60`；這是停用 Timer C，保留 ACIA。

契約：沿用既有初始化通道設定；新增任意既有啟用集合的子集寫入。
即 `new & ~old == 0`，套用 `pending &= new`，in-service 不變。
不擴充未證實的通道啟用與計時器模式。相同值亦可重入。
既有 EmuTOS 初始化 stage 判斷仍優先，防止初始化收據漏記。

驗收：A/B 兩組的已啟用集合停用／重入、pending 清除、in-service 保留；
非既有支援路徑的新通道啟用仍拒絕；DM 正常啟動前移。
