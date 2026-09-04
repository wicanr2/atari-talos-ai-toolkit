# Atari Talos 工作歷程

## 2026-09-05

- 建立專案骨架、RRSAL-1.0、繁中／英文入口與 Docker Go 工具鏈。
- 建立 `talos-jsonl/1` 控制契約及 CLI；未完成模擬能力採失敗即關閉。
- 固定使用既有 `golang:1.24-bookworm`，明示 `/usr/local/go/bin` 後全套測試與 CLI
  實際往返通過。
- 建立 public GitHub repo，`main` 為預設分支；設定 `atari-st`、`emulator`、
  `ai-tools`、`retrocomputing`、`golang`、`testing` 主題。
- 固定 SingleStepTests/m68000 commit 與授權，建立 NOP READY 規格、24-bit sparse bus、
  預取模型與 corpus loader；以 2,500 筆狀態／clock／bus trace 驗收第一條 CPU 指令。
- 修正外部 corpus 的 Docker 邊界：主機端解析與驗檔，容器內固定唯讀 `/corpus`，
  避免相對路徑在 Codex／Claude Code 工作目錄間漂移。
- 新增 MOVEQ READY 規格與實作；以 2,500 筆外部語料驗 Dn、符號延伸、CCR、預取及 bus。
- 依 Dungeon Master DM12EN 重建組語的實際使用情形，新增 SWAP、EXT.W、EXT.L；三份
  外部語料共 7,500 筆完整狀態、RAM、clock 與 bus transaction 全部通過，CPU 累計
  驗收 12,500 筆。
- 固定 NXP／Motorola 官方 M68000PRM 手冊雜湊，建立 Bcc／BRA 正常控制流規格；
  1,830 筆偶數目標語料全部通過，CPU 累計正常單步驗收 14,330 筆。另確認 670 筆
  奇數目標屬 address error，尚未實作例外時採失敗即關閉且不改狀態。
- 完成 MC68000 address error 的 14-byte frame、模式切換、trace 清除、vector 3 與
  handler 預取；訂正 `re` 未定義 data bus 的初步解讀，不對 fault 位址虛構 RAM read。
  670 筆例外語料全數通過，CPU 累計外部單步驗收 15,000 筆。
- 完成 BSR 與 RTS 的 active-stack call／return 垂直鏈；兩份各 2,500 筆語料均包含
  正常與 address-error 路徑並全數通過，CPU 累計外部單步驗收 20,000 筆。
- 完成 JMP／JSR 共用 control-EA 解碼與各自 bus 次序，七種 68000 control mode 共
  5,000 筆全數通過。依官方相容性文字與語料訂正 extension bits 10–8：68000 忽略，
  不誤判為 full-extension 例外。CPU 累計外部單步驗收 25,000 筆。
- 將 control-EA 接到 LEA／PEA；保留 PEA absolute modes 在最後一次預取前插入 stack
  write 的 microcode 次序。兩份共 5,000 筆全部通過，CPU 累計外部單步驗收 30,000 筆。
- 建立第一個 data-EA 垂直切片：`MOVE.B` 全部合法 source modes 至 Dn。加入 byte bus
  read、UDS／LDS lane、A7 byte delta、立即值與 source prefetch 排程；384 筆固定子集
  全部通過，CPU 累計外部單步驗收 30,384 筆。
- 完成 `MOVE.B` 全部記憶體目的端及 byte write bus，保留 predecrement 與 absolute-long
  的不同 prefetch／write 次序，也驗證 source side effect 後才計算 destination EA。
  新增 2,116 筆全狀態、RAM、clock、bus 驗收，完整 MOVE.B 2,500 筆全通過，CPU
  累計外部單步驗收 32,500 筆。
- 完成 `MOVE.W` 全部合法 source／destination EA 與 data address error。固定分母為
  正常 1,013、source read fault 839、destination write fault 648；2,500 筆完整狀態、
  RAM、clock、bus 與 vector-3 frame 全數通過，CPU 累計外部單步驗收 35,000 筆。
- 完成 `MOVE.L` 全部合法 source／destination EA、兩次 word 組成的 long bus access
  與 data address error。固定分母正常 1,013、source fault 869、destination fault 618；
  2,500 筆全數通過，CPU 累計外部單步驗收 37,500 筆。
- 完成 `MOVEA.W`／`MOVEA.L`：重用已驗證的 word／long source EA 與 address-error
  路徑，新增 word 符號延伸及 An／A7 destination。兩份共 5,000 筆全數通過，CPU
  累計外部單步驗收 42,500 筆。
- 依 DM12EN 產生組語盤點，完成高頻 `ADDA.W`／`ADDA.L`。兩份語料共 5,000 筆
  全數通過；其中 `ADDA.L` 的 memory source 比 register／immediate source 少 2 clocks，
  已分開建模。CPU 累計外部單步驗收 47,500 筆。
- 完成 `AND.B/W/L` 與同組語料涵蓋的 `ANDI.B/W/L`；三種寬度共 7,500 筆全數通過。
  讀改寫固定在最後一次 prefetch 後才 write，long write 為 low word 先行；word／long
  postincrement 的 address-error 副作用分開建模。CPU 累計外部單步驗收 55,000 筆。
- 完成 `CMP.B/W/L`、`CMPI.B/W/L` 與 `CMPM.B/W/L`；三種寬度共 7,500 筆全數通過。
  CMPM 的兩次後遞增、alias、來源／目的 odd-address 副作用與 saved PC 採獨立微時序；
  CPU 累計外部單步驗收 62,500 筆。
- 完成 `ADD.B/W/L`、`ADDI.B/W/L` 與 `ADDQ.B/W/L`；三種寬度共 7,500 筆全數通過。
  ADDQ 的 0→8 immediate、An 特例、long 額外時鐘，以及所有記憶體讀改寫與
  address-error 路徑均逐筆驗收；CPU 累計外部單步驗收 70,000 筆。
