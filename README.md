# Atari Talos AI Toolkit

**給大型語言模型與自動化測試使用的無頭 Atari ST／STF 對拍工具。**

Atari Talos 誕生的目的，是讓大型語言模型（LLM）能更可靠地執行、觀察與比較
Atari ST 遊戲。它將提供可重現輸入、逐幀執行、狀態擷取與機器可讀的對拍證據，
協助復古遊戲重製專案更準確地還原原版行為。

傳統模擬器主要讓人透過視窗操作；Codex、Claude Code 與 `go test` 需要的則是
「問得到、量得到、可重播」：精確推進指定畫格、讀取 CPU／記憶體狀態、保存快照，
並以 JSON Lines 交換命令與結果，而不是送按鍵後等待一段牆上時間再猜畫面是否穩定。

## 現況

專案處於 **M0 控制契約**：`ataritalos` 已提供版本化 JSON Lines 協定、能力查詢，
並對尚未實作的開機與畫格推進命令明確失敗。它目前**還不是可啟動遊戲的模擬器**。

首版範圍固定為 Atari ST／STF：Motorola 68000、RAM／ROM 位址空間，以及遊戲需要的
GLUE、MMU、Shifter、MFP、PSG、ACIA 與 FDC 路徑。STE、TT、Falcon、DSP 與 Videl
不在第一階段。

Hatari 是外部相容性 oracle，不是程式碼來源。Atari Talos 依公開硬體規格獨立以 Go
重寫；不翻譯、移植或連結 Hatari 的 GPL 程式碼。

## JSON Lines 控制介面

每行是一個 JSON 物件；回覆沿用呼叫端提供的 `id`。未知命令、未知欄位與尚未具備的
能力一律失敗即關閉（fail-closed）。

```console
$ printf '%s\n' '{"id":"1","op":"hello"}' '{"id":"2","op":"capabilities"}' | ataritalos
{"id":"1","ok":true,"result":{"name":"Atari Talos AI Toolkit","protocol":"talos-jsonl/1","version":"0.0.1-dev"}}
{"id":"2","ok":true,"result":{"commands":["hello","capabilities","quit"],"emulation_ready":false,"machine":"atari-stf"}}
```

未完成的命令不會假裝成功：

```console
$ printf '%s\n' '{"id":"3","op":"run_frames","frames":1}' | ataritalos
{"id":"3","ok":false,"error":{"code":"not_implemented","message":"run_frames requires the Atari ST machine core"}}
```

正式控制協定見 [`docs/spec/002-jsonl-control.md`](docs/spec/002-jsonl-control.md)。Go 套件
與 CLI 將使用同一組 request／response 型別，讓 remake 可以直接在 `go test` 內嵌，
也能讓 Codex、Claude Code、shell 或其他語言透過程序介面控制。

## 建置與測試

所有建置與測試都在固定 Go Docker image 中執行：

```sh
tools/go.sh test ./...
tools/go.sh build -o bin/ataritalos ./cmd/ataritalos

# 下載固定版本的外部 CPU 語料，再跑逐 clock／bus 驗收
tools/fetch-m68000-tests.sh
TALOS_M68000_TESTS=workplace/m68000-tests/v1 tools/go.sh test ./internal/m68k
```

## 里程碑

| 里程碑 | 完成條件 | 狀態 |
|---|---|---|
| M0 | JSON Lines 契約、CLI、Docker 測試、文件、授權與 public repo | **完成** |
| M1 | 68000 核心通過公開指令語料與邊界測試 | **基礎／控制流／control EA、完整 MOVE／MOVEA／ADDA／AND／ANDI／CMP／CMPI／CMPM／ADD／ADDI／ADDQ／CLR，共 77,500 筆已通過** |
| M2 | ST／STF 記憶體、TOS 開機與決定性時鐘 | 未開始 |
| M3 | Shifter 畫面、鍵鼠、FDC 與磁碟映像 | 未開始 |
| M4 | breakpoint、watchpoint、trace、快照與畫面輸出 | 未開始 |
| M5 | 《Dungeon Master》與 Hatari 同狀態原版對拍 | 未開始 |

目前真相、限制與下一步以 [`CONTEXT.md`](CONTEXT.md) 為準。

## 原版素材與商標

本 repo 與發行包不包含 TOS ROM、遊戲磁碟、遊戲素材或其他 Atari 原版內容。使用者必須
自備合法取得的檔案。Atari、Atari ST 與相關商標屬於各自權利人；本專案與 Atari、
Hatari 或原版遊戲權利人沒有隸屬、合作、贊助或授權關係。

## 授權

本專案採 [RRSAL-1.0](LICENSE)：非商業使用、修改與再散布免費；實況、影片、評論與
一般平台分潤明示允許；商業使用另洽。這是一份 source-available 授權，不是 OSI
定義的開放原始碼授權。RRSAL-1.0 不涵蓋 TOS ROM、遊戲磁碟、商標或第三方內容。
