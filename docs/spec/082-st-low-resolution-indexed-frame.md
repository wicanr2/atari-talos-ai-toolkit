# 082 — ST 低解析度 4-plane 索引畫面快照

狀態：**CONFORMED**。

## 範圍與證據

本切片建立普通 ST low resolution 的 320×200、4-plane big-endian framebuffer 解碼，
輸出每像素 0–15 的 palette index，並從 `Memory.ActiveVideoBase()` 讀出一份靜態 VBL
畫面與當下 16 色 palette。RGB 電壓／色階換算、border、overscan、scanline 中途 palette
變更、HBL 310 提前重載、螢幕截圖編碼與 JSONL 控制命令均未涵蓋。

- **已確認（Atari 一手硬體規格）**：Atari Corporation《Engineering Hardware
  Specification of the Atari ST Computer System》，1986-01-07，既有收據 SHA-256
  `eb3a001ed636123f94c9c612ab33b6de2b1b118177ea01cfb971bf3ae17e6044`；low resolution
  是 320×200、4 bitplanes／16 色，plane words 為 big-endian，bit 15 對應每組最左像素。
- **強證據（固定 Hatari oracle 實作）**：Hatari source commit
  `4371dcd647fc85d31c0629400adaeaa4212040d9`、archive SHA-256
  `ed3861b10b05283d0a97df0a9070cef5ae71293ddf4c797a82174ae50ea8877c`。
  `screen.h` 定義中央 320 pixels 為每行 160 bytes；`screenConvert.c`
  `Screen_BitplaneToChunky16/32` 以 TOS big-endian bitplane word order將 plane 0–3
  合成 palette index。Hatari 只作程序外 oracle，本專案不複製或連結 GPL 程式碼。
- **已確認（固定 EmuTOS／Hatari 實跑）**：EmuTOS 1.3 UK ROM SHA-256
  `ad64942f5b0f468a08b909827f6cfa2c38e786f853fab407011dc7d6f9c52135`。Hatari
  `--fast-boot false` 在 VBL 4、5、6 的 `$0F8000..$0FFCFF` 32,000 bytes 均全零；
  VBL 7／CycleCounter=1,016,352 首次非黑，原始 dump SHA-256
  `98dcbfd3bd49a1854a7544d349d1d6dee0a629f66ed976262bdfd9fd72a0570f`、368 個 raw
  bytes非零。獨立直接解碼得到 index SHA-256
  `6157070b2e1adde8ec0cf121ee72133824c34edc5b133ff0064632a73e910444`；色號 0
  共 63,679 pixels、色號 15 共 321 pixels，其餘為 0，第一個非零像素為 `(1,0)`。

## typed 行為

1. 固定常數：width=320、height=200、16 pixels/group、4 big-endian plane words/group、
   160 bytes/line、32,000 input bytes、64,000 output indices。
2. 對每組第 `x=0..15` 像素，取四個 plane word 的 bit `15-x`；plane 0 是 index bit 0，
   plane 3 是 index bit 3。輸入長度不是精確 32,000 bytes 時回 error，不輸出部分畫面。
3. `Memory.LowResolutionFrame()` 只接受目前 resolution=0；由 active base開始經 ST RAM
   MMU topology作 Shifter DMA 讀取，明確繞過 MC68000 reset ROM shadow、ROM、cartridge
   與 I/O。32,000-byte window若超出 22-bit DMA range或任一位址不對應實體 RAM即整體
   失敗。成功回固定尺寸、
   64,000-byte indices與當下 palette value copy；呼叫後修改 RAM／palette不得回寫快照。
4. 本切片是 VBL 靜態 snapshot，不套 border、不重播 scanline palette history，也不把
   `$0777` 色碼猜成 RGB。未來 PNG／RGBA 必須另立 READY 規格決定數位色階契約。
5. 外部 Hatari dump、ROM 與任何衍生二進位不進 Git；測試只在環境變數
   `TALOS_HATARI_FRAMEBUFFER` 指定合法本機 fixture 時執行 oracle 檢查。

## 驗收與停止線

- synthetic tests 覆蓋 16 個 index、四 plane bit權重、bit 15／0、group／line／frame
  邊界、錯誤長度、active/programmed 分離、base 0 不讀 ROM shadow、RAM fault、非 low-res、
  palette copy與 snapshot isolation。
- 外部 fixture test 必須先驗原始 32,000-byte SHA-256，再驗 64,000 indices SHA-256、
  index histogram及第一個非零座標；不能只拿 decoder 自己產生的 expected data互比。
- 固定 ROM 正常 Talos 路徑目前在 VBL 7 前的 962,832 clocks 因 `$FFFA1D` 非零 timer
  control 失敗即關閉，故本切片不宣稱 Talos 已自行跑出非黑畫面。完整 corpus、全測試、
  oracle fixture、`go vet -stdmethods=false ./...` 與 build 全綠後才升 **CONFORMED**。
