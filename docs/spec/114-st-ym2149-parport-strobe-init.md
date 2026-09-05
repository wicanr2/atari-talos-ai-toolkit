# 114 — ST YM2149 parallel-port strobe初始化

狀態：**CONFORMED**。

## 範圍與停止線

本切片處理空ACSI target 0–7掃描後，EmuTOS `parport_init()`透過YM2149 port A
將parallel-port strobe bit 5設為1的三筆byte I/O：選register 14、讀目前`$03`、
寫回`$23`。印表機資料輸出、ACK／BUSY、`offgibit()`脈衝、聲音合成、其他port A
bit與後續floppy boot皆排除。

## 證據

- **已確認（EmuTOS固定原始碼）**：EmuTOS `VERSION_1_3` commit
  `95eb9e498e979022dd9626f528d32de861f26c85`。`bios/parport.c:108 parport_init`
  呼叫`ongibit(GI_STROBE)`；檔案SHA-256
  `78bbf8e1683b5cd357d131b1ee52b91e7f26b6cc1c34f6aea7199277e3a7f065`。
  `bios/sound.c:115 ongibit`在critical section內選`PSG_PORT_A`、讀回、以OR加入
  傳入mask再寫回；檔案SHA-256
  `4b671a0f5af921dc793f750e3be8d3f7f4a6c01cbf9d501b151cbecfc1fd139c`。
- **強證據（固定Hatari正常路徑）**：Hatari 2.4.1 image、EmuTOS 1.3 UK ROM
  SHA-256 `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；
  66-VBL `psg_all,fdc,video_vbl` trace SHA-256
  `cdd513ad571b90e33881f39e5c2f902eca9808a10d8b51632f8a01fbe73df7c2`。
  VBL62依序記錄PC `$FC6E58`寫`$FF8800=$0E`、PC `$FC6E5C`讀回`$03`、
  PC `$FC6E64`寫`$FF8802=$23`；三筆指令內I/O phase為12／8／12 clocks。
- **已確認（固定Talos入口）**：規格113完成target 7後，第一個typed gate為
  867,255 instructions／462 interrupts／11,598,096 clocks的supervisor byte write
  `$FF8800`；D0 low byte=`$20`、A0=`$FFFF8800`、PC=`$FC6E5E`、
  prefetch=`$000E,$1410`。

## typed行為

1. ACSI scan stage5、FDC probe stage14、PSG selected register14、R14=`$03`時，
   只接受supervisor byte write`$FF8800=$0E`，進入strobe init read stage。
2. 下一筆只接受supervisor byte read`$FF8800`，回`$03`並進入write stage。
3. 下一筆只接受supervisor byte write`$FF8802=$23`，原子更新R14為`$23`並完成。
   `$23 == $03 | $20`；既有drive／side low三位保持不變。
4. 錯序、錯值、錯位址、user／word access均失敗即關閉且不得改stage或register。
   三筆timed byte access沿用既有PSG 4 wait-clock契約；cold reset清stage與register。
5. 現有`psgDriveStage`名稱只作既有port-A開機序列的導覽識別，不把strobe語意
   冒稱為drive狀態；本切片不建立未使用的印表機裝置模型。

## 驗收

- synthetic測試涵蓋三段正常序列、read value、bit保留、錯序／錯值原子拒絕、
  timed wait與cold reset。
- 固定ROM必須自然完成R14=`$23`，鎖定CPU state／clock，再有界定位第一個
  post-strobe typed gate。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`
  與`go build ./...`通過後才升 **CONFORMED**。

本切片不修改Dungeon Master規則、資料、素材、畫面、存檔或權利邊界。

## 驗收結果

- 固定Talos正常路徑自然完成`$0E→read $03→write $23`，於867,260 instructions、
  462 interrupts、11,598,144 clocks完成；selected register=`$0E`、R14=`$23`，
  PC=`$FC6E6C`、prefetch=`$40C0,$46C1`，完整D/A、stack與SR已鎖入測試。
- 有界續跑至867,320 instructions／11,599,192 clocks；下一gate為IKBD ACIA data
  `$FFFC02` supervisor byte write，PC=`$FC515A`、prefetch=`$FC02,$241F`。
- 完整240,000筆CPU corpus、固定ROM、全測試、`go vet -stdmethods=false ./...`
  與`go build ./...`均通過。
