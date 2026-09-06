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
- 固定語料同時有 `LINK.json.bin` 與命名為完整單字的 `UNLINK.json.bin`，各 2,500 筆；
  先前只尋找助記碼 `UNLK` 而漏列後者。UNLINK 語料含正常 1,385、odd-frame vector-3
  1,115 筆；確認 odd frame 不提交 active SP、fault address／saved PC／原始 data FC
  的例外契約，以及 A7 alias 最終以讀出的 long 覆蓋中間 frame+4。
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
- DM12EN 的 ReDMCSB 產生組語中，SUB.W 265 次、SUB.B 4 次、SUB.L 32 次，
  SUBI.W 10 次、SUBI.B 3 次、SUBI.L 1 次，SUBQ.W 287 次、SUBQ.B 16 次、
  SUBQ.L 17 次，合計 635 個靜態使用點；此計數只用於模擬器實作優先序。
- `SUB.b/w/l.json.bin` 共 7,500 筆，其中一般 SUB 4,608 筆、SUBI 337 筆、
  SUBQ 2,555 筆。語料確認其 EA、clock、prefetch、讀改寫及 vector-3 微時序可與
  ADD 族共用，運算與 X／NZVC 必須使用減法及 borrow 規則。
- DM12EN 的 ReDMCSB 產生組語中，ASL.W 257 次、ASL.L 547 次，合計 804 個靜態
  使用點；此計數只用於模擬器實作優先序。
- `ASL.b/w/l.json.bin` 共 7,500 筆確認立即值 count 的 0→8、Dn count 只取低六位、
  count 0 時 C 清除而 X 保留、ASL 的逐步符號變化 V、動態 clocks，以及 word memory
  型的 read／prefetch／write 與 vector-3 微時序。
- DM12EN 的 ReDMCSB 產生組語中，ASR.W 125 次、ASR.B 4 次、ASR.L 25 次，合計
  154；LSR.W 593 次、LSR.B 4 次、LSR.L 8 次，合計 605 個靜態使用點。
- `ASR.b/w/l.json.bin` 與 `LSR.b/w/l.json.bin` 共 15,000 筆確認兩族共用 count、
  clocks、C／X、NZV、memory RMW 與 vector-3 微時序；ASR 以原符號位填入，LSR
  以零填入，count 大於 operand 寬度時仍依 68000 的低六位完整推進。
- DM12EN 的 ReDMCSB 產生組語中，MULS 324 次、MULU 77 次，合計 401 個靜態
  使用點；此計數只用於模擬器實作優先序。
- `MULS.json.bin`／`MULU.json.bin` 共 5,000 筆確認 16×16→32、word data EA、
  NZVC 與 vector-3 微時序。MULU clocks 為 38 加來源 popcount×2；MULS clocks 為
  38 加 Booth 相鄰位元轉換數×2，且必須把 bit 0 與虛擬前一位 0 的邊界算入。
- DM12EN 的 ReDMCSB 產生組語中，NOT.W 54 次、NOT.L 2 次，NEG.W 28 次、
  NEG.L 6 次，合計 90 個靜態使用點。
- `NOT.b/w/l.json.bin` 與 `NEG.b/w/l.json.bin` 共 15,000 筆確認兩族的 Dn／完整
  memory destination EA、RMW bus 次序、long low-word-first 寫回與 vector-3 微時序。
  NOT 保留 X 並清 V／C；NEG 對非零 operand 同時設定 X／C，最小負值設定 V。
- DM12EN 的 ReDMCSB 產生組語中，Scc 族合計 244 個靜態使用點；其中 SEQ 142、
  SNE 54，其餘分布於 SCC、SGE、SGT、SHI、SLE、SLS、SLT、SMI、SPL、ST、SVC、SVS。
- `Scc.json.bin` 2,500 筆確認 Dn 成立為 6 clocks、不成立為 4 clocks；memory 型
  無論結果都先讀原 byte，再 prefetch 並寫 `ff`／`00`，且完全不改 SR。
- DM12EN 的 ReDMCSB 產生組語中，DBF 有 14 個靜態使用點，未見其他 DBcc 條件；
  此計數只用於模擬器實作優先序。
- `DBcc.json.bin` 2,500 筆確認 condition 成立、計數到期與成功分支分別為 12、14、
  10 clocks。奇數分支目標進入 vector 3 時不提交 Dn 遞減；fault address 是計算出的
  奇數目標，frame saved PC 則是 extension 之後的順序 PC，兩者不可混用。
- DM12EN 重建組語有 `BTST` 76、`BCLR` 10、`BSET` 10 個靜態使用點，未見 `BCHG`；
  四份固定語料仍完整納入 10,000 筆，確認 Dn modulo 32、memory modulo 8、Z、byte
  RMW、PC-relative FC，以及修改型在 bit 16–31 多 2 clocks 的資料相依時序。
- DM12EN 重建組語有 `DIVU` 70、`DIVS` 22 個靜態使用點。兩份固定語料共 5,000 筆
  確認成功、overflow、word EA、vector 3 與資料相依迭代 clocks；語料未含 divisor=0。
  兩個 Hatari 最小探針補證 DIVU／DIVS register divisor=0 均為 40 clocks，先設 Z，
  Dn 不變，再建立 SR+PC 的 6-byte vector 5 frame。
- DM12EN 重建組語有 `TRAP` 8、`RTE` 6 個靜態使用點。TRAP 語料 2,500 筆固定
  34 clocks；RTE 2,500 筆分成正常 600、odd-target vector 3 共 614、user privilege
  vector 8 共 1,286。RTE 先提交 SSP+6 與 masked restored SR，才嘗試奇數 target。
- DM12EN 重建組語有 EOR／EORI 共 35 個靜態使用點，涵蓋 Dn、memory destination
  與 immediate。三份固定語料共 7,500 筆，確認三種寬度的 XOR、X／NZVC、EA、
  read-modify-write、long low-word-first bus 次序與 vector 3；CCR／SR 特例另有獨立語料，
  本切片不納入。
- DM12EN 重建組語有 LSL 16 個靜態使用點，包含 immediate／Dn count 與 word／long。
  三份固定語料共 7,500 筆確認低六位 count、零 count、X／NZVC、動態 clocks、
  word memory RMW、EA side effect、完整 bus 次序與 vector 3。
- DM12EN 重建組語有 ROR 10 個靜態使用點，包含 immediate／Dn count 與 word／long。
  三份固定語料共 7,500 筆確認低六位 count、零 count 時 C 清除且 X 保留、NZVC、
  動態 clocks、word memory RMW、EA side effect、完整 bus 次序與 vector 3。
- DM12EN 重建組語有 SUBA 8、CMPA 8 個靜態使用點。四份固定語料共 10,000 筆，
  確認 word source 符號延伸、long source、全部 data EA、CCR／X、clocks、bus 與
  vector 3；`CMPA.L An,An` 必須先於較寬鬆的 CMPM mask 解碼。
- DM12EN 重建組語有 EXG 6 個靜態使用點，全部是 Dn↔Dn。固定語料 2,500 筆另完整
  覆蓋 Dn↔Dn 834、An↔An 828、Dn↔An 838，確認 A7 stack bank、SR 不變、固定
  6 clocks 與 prefetch bus。
- `MOVEtoUSP`／`MOVEfromUSP` 固定語料共 5,000 筆，其中 supervisor 正常 2,567、
  user privilege 2,433；確認 A7 使用目前 SSP bank、正常 4 clocks，以及 user mode
  以指令起始 PC 建立 34-clock vector-8 format-0 frame。
- `MOVEtoCCR`／`MOVEtoSR` 固定語料共 5,000 筆，確認 CCR 低五位、SR `0xa71f` mask、
  word data EA、1,440 筆 vector 3、1,290 筆 user privilege vector 8；正常路徑在狀態
  載入後以新 program FC 重讀 `PC-2`，再預取 `PC`，總 clocks 不額外增加。
- `MOVEfromSR.json.bin` 固定語料 2,500 筆分為 Dn 404、正常 memory 1,118、
  odd destination vector 3 共 978 筆；確認 MC68000 user mode 合法、SR 不變，memory
  型依序讀舊目的 word、完成 prefetch、再寫入完整 SR。
- DM12EN 31 份重建組語的剩餘特殊指令盤點中只有 TAS 有 1 個靜態使用點：
  `COMMAND.S:198`；`COMMAND.C:119` 的原始註解確認它以舊值 Z 判斷命令佇列鎖，
  再設定 byte bit 7。
- `TAS.json.bin` 共 2,500 筆，Dn 392、memory 2,108；上游明載特殊 5-cycle RMW
  timing 不正確。48-byte Hatari PRG 連續執行兩次 `TAS (A0)`，FrameCycles 每次皆
  增加 16，分別確認 `01→81, N/Z=0` 與 `81→81, N=1/Z=0`，因此 memory 語料總
  clocks 逐筆加 2；pin-level 波形未由 debugger 觀測，維持未建模。
- Atari Corporation《Engineering Hardware Specification of the Atari ST Computer
  System》（1986-01-07）保存掃描 SHA-256 為
  `eb3a001ed636123f94c9c612ab33b6de2b1b118177ea01cfb971bf3ae17e6044`。第 25–27 頁
  確認低 2 KiB 與 I/O supervisor protection、reset 前 8 bytes ROM shadow、512 KiB／
  1 MiB RAM、`FC0000–FEFFFF` 192 KiB ROM 及 `FF0000–FFFFFF` I/O space。
- NXP 官方《M68000 Family Programmer's Reference Manual》附錄 B 確認 reset
  vector 0／1 分別載入 ISP 與 PC；PDF SHA-256 為
  `06e4864b78da0e815054cead9326b7ec9914661f240fd39a455f2061ff47c4e8`。
- EmuTOS 1.3 UK 192 KiB ROM SHA-256 為
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；前 8 bytes
  `60 2e 01 04 00 fc 00 30` 對應 `SSP=602e0104`、`PC=fc0030`。Hatari 2.4.1
  於首條 `BRA.W $001c` 後觀測 `PC=fc004e`、FrameCycles=10、`SR=2700`、
  `A7=602e0104`；因此初始 SSP 即使看似非法位址也不得代換。
- Atari 一手硬體規格第 27 頁確認 `$FF8001` 是 supervisor-only byte R/W
  MMU 組態；bits 3–2／1–0 分別選 bank0／bank1 的 128 KiB、512 KiB、2 MiB。
  Hatari 2.4.1 同 ROM／1 MiB ST 第一 VBL I/O trace 為 read `$00` @`FC0052`、
  write `$0A` @`FC0188`、write `$05` @`FC0218`。
  表格的高四位不可臆測為固定零：Hatari debugger 寫 `$FA` 後讀回 `$FA`，
  因此只在 bank 解碼時取低四位，latch 本身保留完整 byte。
- 加入 MMU 後，EmuTOS 從 reset 連續 7 條指令到 `$FC0070`；Atari Talos
  與 Hatari 均為 92 clocks、`SSP=00001000`、`SR=2704`、prefetch=`4e7b,0801`。
  `$4E7B` 是 68010+ `MOVEC D0,VBR` CPU 型號探測；MC68000 應走 vector 4，
  該次因 illegal-instruction exception 尚未建立而成為第一停點。
- Hatari 對 `$FC0070` `$4E7B` 的 vector 4 收據：vector long=`$00FC0074`；
  進入 handler 後 FrameCycles `92→128`，`SSP=$0FFA`、`SR=$2704`、
  prefetch=`$21FC,$00FC`。MMU cold `$00` 下的 physical `$1FFA` frame 為
  `$2704,$00FC,$0070`，確認 saved PC 是 opcode 位址且本例外為 36 clocks。
- 繼續探測時，兩引擎均以 10 條完成指令、220 clocks 到 `$FC0088`
  `RESET`，但對 `$FC0080` `TST.W $8006` 產生的 vector 2 frame 不同：
  Hatari 保存 fault address `$FFFF8006`，Atari Talos 保存 `$00FF8006`。
  修正後 CPU frame 保留 effective address 的 32-bit 符號延伸表示，只有
  bus call／transaction 使用低 24-bit；完整 frame 已一致。
- Hatari 在 `$FC008A` breakpoint 的 FrameCycles=352；相對 `RESET` handler
  `$FC0088` 的 220 clocks 邊界，確認 supervisor `$4E70` 為 132 clocks。
  RESET 後 D0–D7／A0–A6 全零、`SSP=$0FEC`、`USP=0`、`SR=$2700`，
  prefetch=`$0CB9,$FA52`；Atari Talos 第 11 條已全狀態一致。
- 繼續執行的新第一失敗點是 `$FC008A` `CMPI.L #$FA52235F,$00FA0000`：
  Hatari debugger 顯示 operand `$FFFFFFFF`，Atari Talos 對 `$FA0000`
  回 `unmapped` typed bus fault。下一切片需確認 Atari ST cartridge
  `$FA0000–$FBFFFF` 空槽與載入映像的正式契約。
- Hatari v2.4.1 commit `4371dcd647fc85d31c0629400adaeaa4212040d9`：
  `src/cpu/memory.c:1798–1803` 將 `$FA0000–$FBFFFF` 映射為 ROM；
  `src/cart.c:108–139` reset 時以 `$FF` 填滿 `$20000` bytes；ROM write backend
  對 byte／word／long 均產生 bus error。Atari Talos 依此建立空槽，不複製 GPL 程式碼。
- 空 cartridge 後，兩引擎在第 12 條 `CMPI.L` 完成時皆為 380 clocks；第 13 條
  `BNE` 仍同為 10。第 14 條 `MOVE.L #$FC00B2,$0010` 首次分歧：Hatari
  FrameCycles `390→416`（26），Atari Talos `390→414`（24）。同形狀指令較早
  為 24 clocks，故這 2 clocks 是依全機週期位置產生的 RAM／Shifter bus wait，
  不能寫死在 MOVE timing。
- Hatari `$FC00BE` `$F010` 會讀 vector 11=`$FC00D4`，以 36 clocks 進 handler；
  但該邊界 Hatari=496、Atari Talos=494 clocks。line-F 實作必須等待前置
  bus arbitration clocks 對齊，避免以 exception timing 掩蓋較早差異。
- 查讀 Hatari v2.4.1 commit `4371dcd647fc85d31c0629400adaeaa4212040d9`
  的 cycle-exact memory access：`src/cpu/custom.c:217–223,360–366` 在實際
  read／write 前，以全機 clock 加指令內 clock 對四 clock bus slot 對齊；phase bit 1
  為 1 時等待 `4-phase`。因此第 14 條差異應收窄為 ST 共用 memory bus slot alignment，
  而非已證實的即時 Shifter 搶占。固定 MOVE.L 語料沒有 opcode `$21FC`，其 access
  offset 與地址分類仍缺證據，先建立 DRAFT 規格 056，不進 production timing。
- `$21FC` phase 探針以 12-clock `JMP`／10-clock `BRA.W` 到同一 `$FC0040`，保持
  CPU state、prefetch、operand 與 destination 相同；Hatari cycle-exact 分別得到
  24／26 clocks。MAME microcoded 語料內相同 immediate source＋單一 destination
  extension 的 `$2B7C`／`$257C` 為六個連續 4-clock phases，與 phase 0 的 24 clocks
  共同確認 `$21FC` offsets 0／4／8／12／16／20。規格 056 升 READY。
- 新第一停點 `$FF860F` 不是一般 ST 的 DMA register：固定 EmuTOS 1.3
  `bios/machine.c:87-93 detect_modectl` 以 `check_read_byte()` 探測 `dma.h` 標示僅
  Falcon 存在的 `modectl`。固定 Hatari 2.4.1 `src/ioMem.c:139-148,315-322,867-881`
  對普通 ST／Ricoh chipset 將該位址設為不產生 bus error、read 回 `$FF`。
- Hatari tracepoint 在 `$FC0636` 記錄 `TST.B (A0)` 前 A0=`$FFFF860F`、SR=`$2704`、
  prefetch=`$4A10,$4E71`、FrameCycles=34720；到 `$FC0638` 為 34728、SR=`$2708`、
  prefetch=`$4E71,$7001`，確認 8 clocks 且沒有 vector 2。這是 ST chipset 的 void
  byte read 特例，不可擴張成 Falcon register 或整區 I/O fallback。
- 下一停點 `$FFFC21` 是 Mega-ST RP5C15 RTC 的 seconds-units 位址，但固定 Hatari
  `src/ioMem.c:360-371` 在普通 ST／STE 將 `$FFFC21–$FFFC3F` 整段改成 read `$FF`、
  write discard 的 void handlers。固定 EmuTOS `bios/clock.c:642-665 detect_megartc`
  會用多個 byte read／write 驗證 RTC，不可只放行第一個探針位址。
- Hatari tracepoint 以 A0=`$FFFFFC21` 捕捉同一 `$FC0636` `TST.B (A0)`；
  FrameCycles 35088→35096，SR `$2704→$2708`、prefetch `$4A10,$4E71→$4E71,$7001`，
  確認普通 ST 讀得 `$FF` 且 8 clocks，不取用主機 wall-clock。
- 固定 Hatari 在 `--machine st --blitter false` 下將 `$FF8A00–$FF8A3F` 設為
  bus-error region；EmuTOS `bios/machine.c:303-320 detect_blitter` 以
  `check_read_byte($FFFF8A3C)` 判斷裝置是否存在。這個停點不應映射假 Blitter。
- `$FC0636` `TST.B (A0)` fault 前為 A0=`$FFFF8A3C`、SSP=`$0F84`、SR=`$2704`、
  prefetch=`$4A10,$4E71`、FrameCycles=35524；vector 2 handler `$FC063C` 入口為
  35588、SSP=`$0F76`、prefetch=`$21C9,$0008`。14-byte frame words 是
  `$4A15,$FFFF,$8A3C,$4A10,$2704,$00FC,$0638`，確認 64 clocks、byte read SSW
  與 next-instruction saved PC。
- NXP 官方 MC68901 user manual（PDF SHA-256
  `b24db5d20694016364b83dfe7ff444ca37b42dca560e4d98fd217e4e2e3a85a0`）確認 GPIP
  的 DDR=0 為 input／high impedance、DDR=1 為 push-pull output，GPIP write 只改
  output bits。固定 Hatari `mfp.c` reset GPIP／DDR 為 0，write 採 DDR mask 並加
  4 wait clocks；`ioMemTabST.c` 將 `$FFFA01` 映至該 byte handler。
- 固定 M68000 corpus 的同形 `MOVE.B #imm,(An)` 是 extension read、byte write、
  refill 三個 4-clock phases，共 12 clocks。Hatari 固定 EmuTOS tracepoint 實測
  `$FC614A→$FC614E` 為 FrameCycles `44122→44138`，加上 MFP wait 後共 16 clocks；
  GPIP／DDR 皆維持 `$00`，SR condition codes 由 N=1/Z=0/V=0/C=1 變為
  N=0/Z=1/V=0/C=0。
- NXP MC68901 manual §5.1.2 確認 AER reset 八 bits 全為 0；0 選 falling edge、
  1 選 rising edge。edge bit 與 input buffer 經 XOR transition detector，因此改寫
  AER 可能產生 interrupt transition，不能在未建模 pending interrupt 時開放任意 latch。
- 固定 Hatari A0=`$FFFFFA03` tracepoint 實測 `$FC614A→$FC614E` 的 FrameCycles
  `44166→44182`，仍為 16 clocks；GPIP／AER／DDR 前後均 `$00`，registers、flags、
  prefetch 變化與第一輪 GPIP clear 相同。固定 EmuTOS 第 7,479 條對拍後，下一輪
  停在 `$FFFA05` DDR write。
- NXP MC68901 manual §5.1.3 確認 DDR reset 八 bits 全為 0；0 是 high-impedance
  input，1 是 push-pull output。固定 Hatari DDR write 會以 old/new DDR 重新評估 GPIP
  interrupt，因此在尚未建模 pin transition 時不能泛化任意非零值。
- 固定 Hatari A0=`$FFFFFA05` tracepoint 實測 `$FC614A→$FC614E` 的 FrameCycles
  `44210→44226`，共 16 clocks；GPIP／AER／DDR 前後皆 `$00`。固定 EmuTOS
  第 7,483 條對拍後，下一輪停在 `$FFFA07` IERA write。
- NXP MC68901 manual §4.3.1 確認 IERA／IERB reset=`$00`；bit=1 enable、bit=0
  disable，寫 0 會清相應 pending request但不影響 in-service bit。固定 Hatari handlers
  亦採 `pending &= enable` 後重新評估 IRQ，因此非零 enable 不可脫離 interrupt sources
  單獨冒稱完整。
- 固定 Hatari IERA trace為 FrameCycles `44254→44270`，IERB 為
  `44298→44314`，各 16 clocks；兩 latch 前後皆 `$00`。固定 EmuTOS 第 7,491 條
  對拍後，下一輪停在 `$FFFA0B` IPRA write。
- NXP MC68901 manual §4.3.2 確認 IPRA／IPRB reset=`$00`，bit 1/0 表示
  pending/cleared；固定 Hatari handlers 進一步確認 software write 採
  `pending &= written`，只能以 0 清除、不能設 pending，且每次 access 加 4 wait clocks。
- 固定 Hatari IPRA trace為 FrameCycles `44342→44358`，IPRB 為
  `44386→44402`，各 16 clocks；兩 register 前後皆 `$00`。固定 EmuTOS 第 7,499 條／
  176,902 clocks 對拍後，下一輪停在 `$FFFA0F` ISRA write。
- NXP MC68901 manual §4.3.4、§4.4.1–§4.4.3 確認 ISRA／ISRB reset=`$00`；
  automatic EOI 強制 bits 為 0，software EOI 在 IACK 時設 bit，processor 只能
  寫 0 清除、寫 1 保留。固定 Hatari handlers 亦採 `in_service &= written`，
  並在 access 加 4 wait clocks後重新評估 IRQ。
- 固定 Hatari ISRA trace為 FrameCycles `44430→44446`，ISRB 為
  `44474→44490`，各 16 clocks；兩 register 前後皆 `$00`。固定 EmuTOS 第 7,507 條／
  176,990 clocks 對拍後，下一輪停在 `$FFFA13` IMRA write。
- NXP MC68901 manual §4.3.3 確認 IMRA／IMRB reset=`$00`；0 mask、1 unmask，
  mask 不清 pending，但會即時撤除該 channel 的 IRQ，重新 unmask 時既有 pending
  依 priority 再請求服務。固定 Hatari handlers 完整保存 byte、加 4 wait clocks，
  再重新評估 IRQ。
- 固定 Hatari IMRA trace為 FrameCycles `44518→44534`，IMRB 為
  `44562→44578`，各 16 clocks；兩 register 前後皆 `$00`。固定 EmuTOS 第 7,515 條／
  177,078 clocks 對拍後，下一輪停在 `$FFFA17` Vector Register write。
- NXP MC68901 manual §4.1.3、§4.4.1–§4.4.3 確認 VR reset=`$00`；bits 7–4
  是 vector base、bit 3 選 EOI mode、bits 2–0 unused 且 read zero。automatic EOI
  強制 ISR bits 為 0。固定 Hatari write handler 在 software→automatic 時清雙 ISR
  並重算 IRQ，但會保存 unused bits；Talos 依一手規格採 `$F8` mask，將差異明列。
- 固定 Hatari VR trace為 FrameCycles `44606→44622`，共 16 clocks；VR、ISRA、
  ISRB 前後皆 `$00`。固定 EmuTOS 第 7,519 條／177,122 clocks 對拍後，下一輪
  停在 `$FFFA19` Timer A Control Register write。
- NXP MC68901 manual §6.2.2 確認 TACR／TBCR／TCDCR reset=`$00`，control 0
  表示停止，main counter 保留而 prescaler residual 丟失；非零 control 會進入
  delay／event-count／pulse-width mode，TACR／TBCR bit 4 另會拉低 output。
- 固定 Hatari TACR trace為 FrameCycles `44650→44666`、TBCR `44694→44710`、
  TCDCR `44738→44754`，各 16 clocks；三 register 前後皆 `$00`。固定 EmuTOS
  第 7,531 條／177,254 clocks 對拍後，下一輪停在 `$FFFA1F` Timer A Data Register。
- NXP MC68901 manual §6.2.1 確認四個 TDR/main counter reset=`$00`；timer stopped
  時 write 同時載入 TDR 與 main counter，read 捕捉 main counter。active write 延後
  到 count-through-01 reload，臨界 write 可能載入不定值，故留待完整 timer state machine。
- 固定 Hatari TADR trace為 FrameCycles `44782→44798`、TBDR `44826→44842`、
  TCDR `44870→44886`、TDDR `44914→44930`，各 16 clocks；四 register 前後皆
  `$00`。固定 EmuTOS 第 7,547 條／177,430 clocks 對拍後，下一輪停在 `$FFFA27` SCR。
- NXP MC68901 manual §3.3、§7.1.2、§7.1.3、§7.2.2、§7.3.2 確認 SCR／UCR／RSR
  硬體 reset 為 `$00`，但 TSR／UDR 不由硬體 reset 清除；RSR/TSR 的 bit 0 分別控制
  receiver/transmitter enable。固定 EmuTOS `bios/mfp.c:25-36 reset_mfp_regs` 明確由
  GPIP 每隔一 byte 寫零至 TSR，刻意排除 UDR。
- 固定 Hatari SCR trace為 FrameCycles `44958→44974`、UCR `45002→45018`、RSR
  `45046→45062`、TSR `45090→45106`，各 16 clocks，四 register 前後皆 `$00`。
  固定 EmuTOS 第 7,563 條／177,606 clocks 全狀態對拍；有界續跑至第 7,598 條／
  178,092 clocks，下一次嘗試在 PC `$FCD09E` 遇到 `STOP` `$4E72`。
- NXP M68000 Programmer’s Reference Manual §6 確認 STOP 以執行前 SR 判 supervisor，
  合法時載入完整 immediate SR 並停止 fetch／execute；trace、較高優先 interrupt 或
  external reset 才喚醒。固定 `STOP.json.bin` 2,500 筆確認 supervisor 路徑 4 clocks、
  SR mask `$A71F`，user 路徑為 vector 8／34 clocks。
- 固定 ROM bytes 與 Hatari 均將首個相關 STOP 定位在 `$FCD09A: STOP #$2300`；Talos
  的 pipeline PC `$FCD09E` 是下一指令位址，不是 opcode 位址。Hatari 該點
  FrameCycles=`45752`、SR/prefetch 與 Talos一致；D2／D3 分別為 `$2710/$1`，Talos
  為 `$2704/$0`，列為後續開機差異，不以 STOP 實作遮蔽。
- 固定 Hatari 在 `$FC67B8` GPIP read 前 cached GPIP=`$20`；read handler 依 DDR 取樣
  color monitor bit 7 與 no-printer bit 0 後，連同既有 FDC-idle bit 5 得 `$A1`。
  `$FC67B8→$FC67C2` FrameCycles `45300→45360`，D0 最終為 `$1`。Talos 加入固定
  color profile sample 後，STOP 前 D2 從 `$2704` 收斂為 Hatari `$2710`。
- 剩餘 D3 差異不是 GPIP：固定 ROM `$FC6904` 的 bytes
  `26 39 00 00 04 66` 是 `MOVE.L $00000466,D3`；EmuTOS `tosvars.ld:62` 證實
  `$466=_frclock`，`bios/vectors.S:323-324 _int_vbl` 每次 VBL 加 1，`screen.c:1204-1207`
  的 `vsync` 讀它後 STOP 等待中斷。Hatari 已有一個 VBL故 D3=1，Talos 尚無 VBL故 0。
- NXP MC68000 Programmer’s Reference Manual 的 STOP 契約確認 PC 先推進到下一指令，
  中斷 level 必須高於 SR mask；附錄 B 確認 level 4 autovector 是 vector 28／offset `$70`。
  固定 Hatari 在 `$FCD09A: STOP #$2300` 讀得 `$70=$00FC0446`；第二個 VBL handler
  入口 SSP `$F70→$F6A`、SR mask 4、frame bytes 對應 `$2300,$00FC,$D09E`，prefetch
  `$52B8,$0466`。handler 第一條即 `ADDQ.L #1,$466`；入口時 `$466` 尚為 1，證明
  frclock 應由 guest handler 執行增加，而不是由主機事件直接寫入。
- Hatari 第二個 VBL handler 入口 FrameCycles=`124`，支援 44-clock autovector exception
  sequence；實體 IACK pin／VPA 波形未由 debugger 觀測，維持未建模。Talos CPU API
  只接受已由外部仲裁的 level 1–6 autovector，第 7 級特殊語意失敗即關閉。
- 固定 Hatari 若使用 `--fast-boot true`，第一個 `$FC0446` 的 D4／D7／A5 為 0；這不是
  raw-ROM Talos 的同 profile。改用 `--fast-boot false` 後三者為
  `$00080000/$1/$00FC01F4`，其餘 D/A、SSP `$F6E`、SR `$2400`、saved PC `$FC6904`
  及 prefetch `$52B8,$0466` 全與 Talos 一致。後續對拍凡比較 raw ROM 開機狀態，禁止
  混用 fast-boot patch。
- Hatari fixed source commit `4371dcd647fc85d31c0629400adaeaa4212040d9` archive
  SHA-256 `ed3861b10b05283d0a97df0a9070cef5ae71293ddf4c797a82174ae50ea8877c`：
  color ST reset 將 sync register 清 0，初始 60 Hz 為 263×508 clocks，STF VBL offset
  64，第一 deadline 133,668；被 SR mask 擋住的 pending 不消失。Talos 依此 raise
  第一 event，在 `$FC6904` 前接受並由 guest `ADDQ.L #1,$466` 寫出 frclock=1。
- 同狀態測試另揭露 autovector saved-PC 必須區分 running 與 stopped：running pipeline
  保存 `State.PC-4`，STOP 已將 PC 推進則保存 `State.PC`。修正後第一 VBL frame
  `$2300,$00FC,$6904` 與第二 VBL frame `$2300,$00FC,$D09E` 各自成立。
- Atari hardware map 與固定 Hatari source 確認 `$FF8201/$FF8203` 是 Shifter display
  base high／middle bytes。`DMA_MaskAddressHigh()` 對普通、最多 4 MiB 的 ST／STE 回
  `$3F`；ST 組址忽略 low byte，因此 programmed base 固定 256-byte aligned。
- 固定 Hatari 在 `$FC67FA: MOVE.B D1,$8201` 前 CycleCounter=403,924、D1=`$0F`；
  12 clocks 後 high=`$0F`，但 `info video` 的 active base 仍為 0。`$FC67FE`
  `LSR.L #8,D0` 用 24 clocks 將 `$000F8000` 變 `$00000F80`；`$FC6800`
  `MOVE.B D0,$8203` 再用 12 clocks 得 programmed `$0F8000`，active base 仍為 0。
  因此暫存器 write 與掃描基址 reload 必須是兩個切片。
- 固定 Hatari `video_hbl` trace 顯示前三次 `VideoBase` reload 都在 HBL 260 且為 0；
  第三幀於 line 0 改成 50 Hz 後仍在 HBL 262 結束，沒有 HBL 310 reload。debugger 在
  CycleCounter=535,524、VBL=3、HBL=263 仍讀 active=0；`$FC299E LEA $1C(A7),A4`
  跨 deadline 535,528，535,532／VBL=4 時 active=`$0F8000`。這是 Hatari
  `Video_ClearOnVBL` 的保底重載，不是一般 50 Hz 提前三線時序。
- Hatari／EmuTOS VBL4、5、6 的 `$0F8000` 32,000-byte screen RAM全零；VBL7／
  CycleCounter=1,016,352 首次非黑，raw SHA-256 `98dcbfd3…a0570f`、368 bytes非零。
  依 4 個 big-endian plane words、bit15→左、plane0→index bit0 解碼後 SHA-256
  `6157070b…10444`；64,000 pixels僅 color0=63,679、color15=321，首非零 `(1,0)`。
  Shifter DMA 必須繞過 CPU reset ROM shadow；base0 讀 RAM，不能重用 CPU `ReadByte`。
- 固定Hatari 2.4.1／EmuTOS 1.3 UK的680-VBL FDC trace SHA-256
  `9cb1d1ac50082934c7296b3b06fe53eada0ce9d8d20d9d9cea5bb4212924a9c0`證實：VBL235
  送出Type-II `$80`後，無磁片不產生IRQ且command保持busy；EmuTOS到VBL310才送
  `$D0` force-interrupt，恰為motor-on `MOTORON_TIMEOUT`的75個50 Hz VBL／1.5秒。
  Hatari隨後清busy與IRQ並完成command；下一筆FDC transaction是`$0086` data register。
- 固定Hatari 2.4.1／EmuTOS 1.3 UK的330-VBL FDC trace SHA-256
  `a0e4a318dfbe98d21788d1f56071827104beba085fedf7f130ba715bf19b2251`證實VBL310
  force-interrupt後的`$0086/$0000/$0080/$0013` dummy seek、command complete／IRQ及
  `$0080` status `$E4` read-clear。固定Talos跨裝置繼續執行後，先遇到YM2149
  `$FF8800`，因此FDC trace所示的下一筆`$0084`只代表「下一筆FDC transaction」，
  不能當成整台機器的下一個typed gate。
- 固定Hatari 2.4.1／EmuTOS 1.3 UK的330-VBL PSG＋FDC trace SHA-256
  `aa08a1b7743650950ad8e489659fc830efb9ca37383865777444a561b717c7dd`補出VBL310
  status `$E4` read後的R14 select／read `$25`／同值write `$25`，其後才是FDC
  `$0084`。因此該PSG transaction是新一輪`select(0,0)`的可觀察bus行為，但不代表
  drive或side狀態改變。
- EmuTOS官方SourceForge 1.3發行包確認既有固定ROM是`emutos-192k-1.3.zip`內的
  `etos192uk.img`，SHA-256 `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`；
  512K UK image雜湊不同，不可混用。以固定Hatari 2.4.1重生的330-VBL PSG＋FDC
  trace SHA-256為`af37cf3ecec5c31ea86650a6ef7f40ac8dcdd3b99bd60132f2fe7603c13849be`。
- 該trace在VBL310的R14 `$25`同值寫回後，完整重送`$0084/$0001`、DMA address
  `$04/$10/$00`、direction toggle `$0190/$0090`、count `$0001`與Type-II
  `$0080/$0080`；Hatari明記track 0／sector 1／side 0／drive 0／address `$001004`，
  隨後為`no disk/drive`。Talos固定ROM收據定位第二次command clock 118,371,398；
  後續130,385,952 clocks才抵達第二次timeout selector，期間無成功傳輸證據。
