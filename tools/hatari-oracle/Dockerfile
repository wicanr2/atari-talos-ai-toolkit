# Hatari 只作外部 oracle。這個映像檔只是「怎麼取得並建置上游 Hatari」的說明，
# 不含任何來自 Hatari／Steem 的程式碼，Atari Talos 也不連結它——兩邊各自跑，
# 只比對輸出的狀態數字（AGENTS.md 的界線）。
FROM debian:bookworm-slim

ARG HATARI_VERSION=2.4.1
ARG HATARI_SHA256=2a5da1932804167141de4bee6c1c5d8d53030260fe7fe7e31e5e71a4c00e0547

RUN apt-get update && apt-get install -y --no-install-recommends \
      build-essential cmake pkg-config zlib1g-dev libpng-dev \
      libsdl2-dev ca-certificates curl bzip2 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build
RUN curl -fsSL -o hatari.tar.bz2 \
      "https://download.tuxfamily.org/hatari/${HATARI_VERSION}/hatari-${HATARI_VERSION}.tar.bz2" \
    && echo "${HATARI_SHA256}  hatari.tar.bz2" | sha256sum -c - \
    && tar xf hatari.tar.bz2 \
    && cd "hatari-${HATARI_VERSION}" \
    && cmake -B build -DCMAKE_BUILD_TYPE=Release -DENABLE_SDL2=1 \
    && cmake --build build -j"$(nproc)" \
    && cp build/src/hatari /usr/local/bin/hatari \
    && cd / && rm -rf /build

ENTRYPOINT ["/usr/local/bin/hatari"]
