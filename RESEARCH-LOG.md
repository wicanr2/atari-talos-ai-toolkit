# Atari Talos 研究紀錄

## 基準模擬器

- Hatari 2.6.1 是 Atari ST／STE／TT／Falcon 模擬器，目標包含遊戲與 demo 的硬體相容性。
- Hatari 採 GPL-2.0-or-later；Atari Talos 不複製、翻譯、移植或連結其程式碼。
- Hatari 的角色只限外部 oracle：相同 TOS、磁碟、機型、輸入與檢查點下產生獨立收據。
- 上述版本與授權資訊在開始實作硬體前仍須固定上游 commit 與檔案雜湊。

## Motorola 68000 語料

- 固定 `SingleStepTests/m68000` commit
  `64b253116a3de04aaac4346c43680960dc9b67e5`；LICENSE 為 MIT。
- 上游 README SHA-256：
  `7f36eed9cb93f061d13bfd18dc310c9bda7b5f3ac96b6d0dfe292e00612c46b6`。
- 語料由 MAME microcoded core 產生，含暫存器、RAM、prefetch、clock 與 bus transaction。
- 上游明列 TAS 的 5-cycle RMW timing 與 TRAPV 有疑義；這兩組目前不得作 confirmed oracle。
- NOP 2,500 筆顯示 `PC` 是下一次預取位址：執行 `prefetch[0]` 後，舊
  `prefetch[1]` 左移，在第 4 clock 從舊 PC 取 word，PC 再加 2。
- MOVEQ 2,500 筆確認 8-bit 符號延伸、D0–D7 選擇，以及 X 保留、N/Z 更新、V/C 清除；
  其預取與 program bus read 與 NOP 共用同一個 4-clock 路徑。
- Motorola／NXP《M68000 Family Programmer's Reference Manual》官方 PDF
  <https://www.nxp.com/docs/en/reference-manual/M68000PRM.pdf>，SHA-256 為
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`；Bcc 見
  4-25～4-26，BRA 見 4-55。分支位移基準為指令字位址加 2，且不改 condition codes。
- `Bcc.json.bin` 共 2,500 筆；其中 1,830 筆為正常偶數目標並已通過，670 筆包含
  address-error transaction；兩者目前均已通過。
- 670 筆 address error 全數確認 14-byte frame、SSW 低 bits、saved PC、fault address、
  user／supervisor 切換、trace 清除、寫入順序、vector 3 fetch 與 60 clocks。語料的
  `re` data bus 是未 assert AS 時的未定義殘值，且 fault 位址刻意不在 RAM map；驗收
  必須正規化該欄位，不能虛構一次 aligned RAM read。
- BSR 2,500 筆分成 1,229 筆正常與 1,271 筆 address error；確認 return PC 會先寫入
  原模式 active stack，奇數目標時 push 保留，例外 saved PC 為 fault target。
- RTS 2,500 筆分成 1,263 筆正常與 1,237 筆 address error；確認先從 active stack
  讀 long 並令 SP+4，奇數 return 時該更新保留，例外 saved PC 為 RTS opcode 後位址。
- JMP／JSR 各 2,500 筆涵蓋七種 68000 control effective address。JMP 為 1,272 筆
  正常、1,228 筆 address error；JSR 為 1,341 筆正常、1,159 筆 address error。
- 官方 brief-extension 相容性文字與語料共同確認：68000 忽略 extension bits 10–8，
  不把後代 CPU 使用的 scale／format 編碼解成例外；初版「bit 8 非零應拒絕」已撤回。
- JSR 會先嘗試讀 target 第一個 word，成功後才 push return PC，再讀 target+2；因此
  奇數 target 不修改 active stack，與 BSR 先 push 再嘗試 target 的順序不同。
- LEA 與 PEA 各 2,500 筆均完整涵蓋七種 control EA；LEA 驗證目的 An／A7 與
  extension 後的順序預取，PEA 驗證 effective address high／low 寫入 active stack。
- PEA 的 absolute word／long 會在最後一次預取前完成兩次 stack write；其他 control
  modes 則先完成該 instruction 的預取再寫 stack。最終狀態相同不足以驗證此差異。
- Motorola 手冊 4-116～4-118 確認 MOVE 的旗標與合法 source modes；byte 不允許 An direct。
- `MOVE.b.json.bin` 中 destination mode 為 Dn 的固定子集共 384 筆，涵蓋其餘全部 source
  modes。語料確認 byte 語意位址可為奇數，但 bus address lines 記錄偶數 word base，
  由 UDS／LDS 選取 high／low lane；A7 的 byte predecrement／postincrement delta 為 2。
- `MOVE.B` source prefetch 次序依 extension 數量不同；absolute long 特別是 low extension、
  first refill、data byte、second refill。這些差異已納入 transaction 全序比較。
- 記憶體 destination 的 2,116 筆確認 source An side effect 先於 destination EA；同一
  register 可連續 postincrement／predecrement。destination predecrement 先完成 final
  refill 再 write，與其他 destination modes 的 write → refill 不同。
- absolute-long destination 在 Dn／immediate source 與 memory source 使用不同 bus
  排程；memory source 為 low extension → byte write → 兩次 refill。語料 byte write
  會把未驅動 lane 以顯式零保存，驗收以「absent 與 zero 等價」正規化稀疏 RAM map，
  但非零 bytes 與完整 UDS／LDS transaction 仍逐筆嚴格比較。
- `MOVE.w.json.bin` 固定分為 1,013 筆正常、839 筆 source `re`、648 筆 destination
  `we`。source `(An)+` 在 read fault 前已 An+2，destination `(An)+` 在 write fault
  不遞增；兩者不能共用同一條 postincrement 時序假設。
- destination `-(An)` 在 odd-address fault 前已完成 final refill，因此 exception frame
  的 opcode／SSW 取當時管線中的 `prefetch[0]`。absolute-long destination 的 write
  fault 額外 clock 依 source 是否為 memory 分成 8／4；完整 transaction 全序已驗證。
- `MOVE.l.json.bin` 固定分為正常 1,013、source `re` 869、destination `we` 618。long
  memory access 只要求 word alignment，依 big-endian 拆成 EA／EA+2 兩次 word cycle。
- long destination predecrement 正常時先寫低 word 到 An-2，再寫高 word 到 An-4；若
  第一個 An-2 已是奇數，fault address 為 An-2 且 An-4 不提交。source predecrement
  則先提交 An-4；source／destination postincrement 都只在完整 access 成功後提交。
- `MOVE.L` destination address error 的 CCR 取決於故障所在微操作階段；register／
  immediate 與 memory source、直接／延伸目的模式不可共用單一「先更新完整 flags」假設。
  規格 015 已保存各組合，618 筆完整 exception frame 與 transaction 全序均已通過。
- `MOVEA.w.json.bin` 固定分為正常 1,658、source `re` 842；`MOVEA.l.json.bin` 為
  正常 1,655、source `re` 845。兩者的 source EA、prefetch 與 fault frame 分別和
  MOVE.W／MOVE.L 相同；差異只在目的為 An、word 符號延伸且 CCR 完全不變。
- DM12EN 的 ReDMCSB `OBJECT/ENGINE/FULL` 產生組語中，`ADDA` 出現 1,539 次、
  `ADDA.L` 18 次；這是靜態編譯產物計數，只作模擬器優先序，不等同執行期頻率。
- `ADDA.w.json.bin` 固定分為正常 1,683、source `re` 817；`ADDA.l.json.bin` 為
  正常 1,675、source `re` 825。兩者確認 word 符號延伸、32-bit 回繞、A7 active
  stack 與 CCR 不變。`ADDA.L` memory source 的 clocks 為 `6 + sourceCost`，而
  register direct／immediate 為 `8 + sourceCost`，不可直接沿用 MOVEA 的時序。
- DM12EN 的 ReDMCSB 產生組語中，AND／ANDI 各寬度合計 2,197 個靜態使用點。
  `AND.b/w/l.json.bin` 不只含一般 AND，也分別含 176／158／129 筆 ANDI。
- 7,500 筆確認 memory destination 是 operand read → final prefetch → operand write；
  long 結果固定 low word 先寫、high word 後寫。word `(An)+` 在 odd read fault 前提交
  `An+2`，long 同類 fault 不提交 `An+4`；兩種寬度不可共用一條副作用規則。
- DM12EN 的 ReDMCSB 產生組語中，CMP 各寬度合計 1,025、CMPI 各寬度合計 1,483，
  共 2,508 個靜態使用點；沒有 CMPM 靜態使用點。`CMP.b/w/l.json.bin` 同時包含
  CMP 6,086、CMPI 630、CMPM 784 筆。
- CMPM odd-address fault 具有專用微時序：來源 fault 提交 `An+2`，目的 fault 不提交
  目的增量，兩者 saved PC 都比一般 mode-3 source fault 前進 2。此結論由 327 筆
  CMPM vector-3 語料逐筆確認。
- DM12EN 的 ReDMCSB 產生組語中，一般 ADD 各寬度合計 668、ADDI 27、ADDQ 2,637，
  共 3,332 個靜態使用點。`ADD.b/w/l.json.bin` 同時包含一般 ADD 4,637、ADDI 297、
  ADDQ 2,566 筆。
- ADDQ 寫 An 一律作 32-bit 加法且不改 CCR；寫 Dn 的 long 版本比 byte／word 多
  4 clocks。記憶體目的端沿用 ADD 的讀改寫次序，這三條由固定語料確認。
- DM12EN 的 ReDMCSB 產生組語中，CLR.W 858 次、CLR.B 64 次、CLR.L 80 次，
  合計 1,002 個靜態使用點；此計數只用於模擬器實作優先序。
- `CLR.b/w/l.json.bin` 共 7,500 筆確認 MC68000 的記憶體 CLR 仍會先讀 operand，
  再做最後一次 program prefetch 與 zero write；long 寫回為 low word 先行。
- DM12EN 的 ReDMCSB 產生組語中，`MOVEM.L` 有 660 個靜態使用點，未見未加寬度的
  `MOVEM`；此計數只用於模擬器實作優先序。
- `MOVEM.w/l.json.bin` 共 5,000 筆確認 memory-to-register 在最後一個實際 operand 後
  還有一次 word 虛讀；PC-relative operand 使用 program FC。predecrement long 的
  odd-address write fault 落在先寫 low word 的第一個 `An-2` 微操作，而不是最終 `An-4`。
- 主機已有可重現的 Atari ST oracle image `sundog-atari-st-oracle:20260812`，不可變
  image ID／digest 均為 `sha256:d634e10516572262d0039b51823cd565df73d3ee25b06604dc9ecf337b0c1fca`；
  容器內 Hatari 為 2.4.1，並有 Xvfb、xdotool 與 Python 3。這是工具鏈盤點，尚未證明
  Dungeon Master 可由該 image 正常啟動。
- SunDog 的既有 Hatari 實跑證據 JSON 已證明一組可用欄位：來源 archive／disk／TOS
  SHA-256、image 與 Hatari 版本、顯示與輸入邊界、逐步事件、截圖 SHA-256、證據等級、
  unknown、next gate、artifact retention 及 `supersededBy` 勘誤鏈。Atari Talos 可沿用
  這些資訊需求，但公開契約要採 bundle、JSONL 或兩者並存仍是未定案的架構決策。
- DM12EN 的 ReDMCSB 產生組語中，LINK 與 UNLK 各 445 次，合計 890 個靜態使用點；
  兩者是 Megamax C 產物中成對且高頻的函式框架指令。
- 固定語料只有 `LINK.json.bin` 2,500 筆，沒有 UNLK 檔。LINK 語料確認 extension
  prefetch → active-stack long push → 最後 program prefetch 的順序及 16 clocks；UNLK
  目前只依官方契約與本地正常路徑測試，odd frame 的 vector-3 微時序維持未實作。
- DM12EN 的 ReDMCSB 產生組語中，TST.W 271 次、TST.B 194 次、TST.L 5 次，
  合計 470 個靜態使用點；此計數只用於模擬器實作優先序。
- `TST.b/w/l.json.bin` 共 7,500 筆確認三種寬度只讀不寫、X 保留、NZVC 更新，以及
  memory EA 的 prefetch、clock 與 vector-3 微時序。
- DM12EN 的 ReDMCSB 產生組語中，OR.W 291 次、OR.B 3 次、OR.L 61 次，ORI.W
  61 次、ORI.B 63 次、ORI.L 1 次，合計 480 個靜態使用點；此計數只用於模擬器
  實作優先序。
- `OR.b/w/l.json.bin` 共 7,500 筆，其中一般 OR `<ea>→Dn` 3,857 筆、
  `Dn→memory` 3,173 筆、ORI 470 筆。語料確認 OR 與 AND 使用相同 EA、clock、
  prefetch、讀改寫及 vector-3 微時序，差異限於逐位元運算結果。
