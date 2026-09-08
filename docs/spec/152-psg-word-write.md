# 152 — PSG 偶數位址 word 寫入

狀態：**READY**

已確認：Steven Tattersall 的 ST 實機實驗使用 `move.l #$00cc02cc,$ffff8800.w`，
其中偶數位址的高 byte 選擇暫存器與寫入資料，低 byte 不作用。
來源：<https://clarets.org/steve/projects/2021_ym2149_sync_square.html>，
「A quick Introduction」與「Doing some tests」。這是實驗作者的一手記錄；
只採存取寬度，不延伸聲波或取樣時序結論。

契約：`WriteWord($FF8800/$FF8802, value)` 使用 `value >> 8`，
沿用既有 byte 寫入的權限、暫存器與磁片控制閘門。一次 word 是一次
裝置存取，不拆成兩個 byte；計時沿用既有 PSG byte 的 4 wait clocks。
此處不新增聲音合成、不支援鏡射／奇數 byte、不宣稱硬體音訊精確。

驗收：高 byte 生效、低 byte 不作用、非特權仍拒絕、定時存取一致；
DM12EN NOCP 自 reset 重跑，不改 PC 或遊戲記憶體。
