# Hatari 外部 oracle 工具

本目錄提供可重現的 Hatari 2.4.1 Docker 建置、無頭 trace 與私人 GEMDOS 測試磁片工具。
Hatari 只作外部 oracle；Atari Talos 不連結、移植或複製 Hatari 程式碼。

## 建置映像

```sh
docker build -t atari-talos-hatari:2.4.1 tools/hatari-oracle
```

Dockerfile 釘住 Hatari tarball 版本與 SHA-256，並包含建立 FAT12 磁片所需的 `mtools`。

## 建立私人 Dungeon Master bootstrap 磁片

把 repo 唯讀掛入 oracle image，並只讓指定的私人輸出目錄可寫：

```sh
docker run --rm --network none --entrypoint /repo/tools/hatari-oracle/build-gemdos-disk.sh \
  -u "$(id -u):$(id -g)" -v "$PWD:/repo:ro" -v "$PRIVATE_DIR:/private" \
  atari-talos-hatari:2.4.1 \
  /private/OUTPUT.st /private/START.PRG /private/START.PAK \
  /private/GRAPHICS.DAT /private/DUNGEON.DAT /private/MANIFEST.txt
```

腳本建立 720 KiB raw `.st`，把 `START.PRG` 放入 `AUTO/`，其餘資料放在根目錄，
並輸出所有輸入與映像的 SHA-256 和 FAT 目錄。原版輸入、輸出映像與 manifest 應放在
私人工作目錄，禁止加入本 repo 或公開發行包。

混用不同版本的檔案只適合揭露模擬器 I/O 缺口，不是資料相容、可玩性或原版 parity
證據。正式同狀態對拍仍須使用者自備、版本可辨識的合法 Atari ST 原版磁片。
