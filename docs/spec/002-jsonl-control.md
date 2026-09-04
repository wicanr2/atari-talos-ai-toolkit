# 002 — JSON Lines 控制契約

狀態：**READY**（使用者定案，2026-09-05）。

## 傳輸

- stdin 每行一個 UTF-8 JSON request，stdout 每行一個 JSON response。
- `id` 是非空字串，由呼叫端指定並原樣回覆；不得重排同一程序內的同步回覆。
- protocol 名稱固定為 `talos-jsonl/1`。
- request 使用嚴格解碼：未知欄位、尾隨 JSON、空行、無 `id` 或無 `op` 都回錯。
- 診斷訊息只能寫 stderr，不得污染 stdout 的 JSON Lines。

## M0 命令

| `op` | 結果 |
|---|---|
| `hello` | 名稱、協定與版本 |
| `capabilities` | 實際可用命令、機型與 `emulation_ready` |
| `quit` | 回覆成功後正常結束 |

保留但尚未實作的 `boot`、`reset`、`run_instructions`、`run_frames`、`key`、`mouse`、
`read_memory`、`write_memory`、`breakpoint`、`watchpoint`、`snapshot`、`restore`、
`framebuffer` 與 `trace` 必須回 `not_implemented`，不可回空成功值。

## 錯誤

```json
{"id":"3","ok":false,"error":{"code":"not_implemented","message":"run_frames requires the Atari ST machine core"}}
```

已定義錯誤碼：`invalid_json`、`invalid_request`、`unknown_operation`、
`not_implemented`、`internal_error`。同一輸入與版本必須產生相同回覆。
