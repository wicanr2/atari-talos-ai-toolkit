#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

run_go() {
  exec docker run --rm --network none \
    --memory 2g --cpus 2 --pids-limit 256 \
    --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" \
    -e GOCACHE=/tmp/go-cache \
    -e GOPATH=/tmp/go \
    -e PATH=/usr/local/go/bin:/usr/bin:/bin \
    "$@"
}

if [ -n "${TALOS_M68000_TESTS:-}" ]; then
  case "$TALOS_M68000_TESTS" in
    /*) CORPUS=$TALOS_M68000_TESTS ;;
    *) CORPUS=$ROOT/$TALOS_M68000_TESTS ;;
  esac
  [ -f "$CORPUS/NOP.json.bin" ] && [ -f "$CORPUS/ILLEGAL_LINEF.json.bin" ] && \
    [ -f "$CORPUS/MOVE.q.json.bin" ] && \
    [ -f "$CORPUS/SWAP.json.bin" ] && [ -f "$CORPUS/EXT.w.json.bin" ] && \
    [ -f "$CORPUS/EXT.l.json.bin" ] && [ -f "$CORPUS/Bcc.json.bin" ] && \
    [ -f "$CORPUS/BSR.json.bin" ] && [ -f "$CORPUS/RTS.json.bin" ] && \
    [ -f "$CORPUS/JMP.json.bin" ] && [ -f "$CORPUS/JSR.json.bin" ] && \
    [ -f "$CORPUS/LEA.json.bin" ] && [ -f "$CORPUS/PEA.json.bin" ] && \
    [ -f "$CORPUS/MOVE.b.json.bin" ] && [ -f "$CORPUS/MOVE.w.json.bin" ] && \
    [ -f "$CORPUS/MOVE.l.json.bin" ] && [ -f "$CORPUS/MOVEA.w.json.bin" ] && \
    [ -f "$CORPUS/MOVEA.l.json.bin" ] && [ -f "$CORPUS/ADDA.w.json.bin" ] && \
    [ -f "$CORPUS/ADDA.l.json.bin" ] && [ -f "$CORPUS/AND.b.json.bin" ] && \
    [ -f "$CORPUS/AND.w.json.bin" ] && [ -f "$CORPUS/AND.l.json.bin" ] && \
    [ -f "$CORPUS/CMP.b.json.bin" ] && [ -f "$CORPUS/CMP.w.json.bin" ] && \
    [ -f "$CORPUS/CMP.l.json.bin" ] && [ -f "$CORPUS/ADD.b.json.bin" ] && \
    [ -f "$CORPUS/ADD.w.json.bin" ] && [ -f "$CORPUS/ADD.l.json.bin" ] && \
    [ -f "$CORPUS/CLR.b.json.bin" ] && [ -f "$CORPUS/CLR.w.json.bin" ] && \
    [ -f "$CORPUS/CLR.l.json.bin" ] && [ -f "$CORPUS/MOVEM.w.json.bin" ] && \
    [ -f "$CORPUS/MOVEM.l.json.bin" ] && [ -f "$CORPUS/LINK.json.bin" ] && \
    [ -f "$CORPUS/TST.b.json.bin" ] && [ -f "$CORPUS/TST.w.json.bin" ] && \
    [ -f "$CORPUS/TST.l.json.bin" ] && [ -f "$CORPUS/OR.b.json.bin" ] && \
    [ -f "$CORPUS/OR.w.json.bin" ] && [ -f "$CORPUS/OR.l.json.bin" ] && \
    [ -f "$CORPUS/SUB.b.json.bin" ] && [ -f "$CORPUS/SUB.w.json.bin" ] && \
    [ -f "$CORPUS/SUB.l.json.bin" ] && [ -f "$CORPUS/ASL.b.json.bin" ] && \
    [ -f "$CORPUS/ASL.w.json.bin" ] && [ -f "$CORPUS/ASL.l.json.bin" ] && \
    [ -f "$CORPUS/ASR.b.json.bin" ] && [ -f "$CORPUS/ASR.w.json.bin" ] && \
    [ -f "$CORPUS/ASR.l.json.bin" ] && [ -f "$CORPUS/LSR.b.json.bin" ] && \
    [ -f "$CORPUS/LSR.w.json.bin" ] && [ -f "$CORPUS/LSR.l.json.bin" ] && \
    [ -f "$CORPUS/MULS.json.bin" ] && [ -f "$CORPUS/MULU.json.bin" ] && \
    [ -f "$CORPUS/NOT.b.json.bin" ] && [ -f "$CORPUS/NOT.w.json.bin" ] && \
    [ -f "$CORPUS/NOT.l.json.bin" ] && [ -f "$CORPUS/NEG.b.json.bin" ] && \
    [ -f "$CORPUS/NEG.w.json.bin" ] && [ -f "$CORPUS/NEG.l.json.bin" ] && \
    [ -f "$CORPUS/Scc.json.bin" ] && [ -f "$CORPUS/DBcc.json.bin" ] && \
    [ -f "$CORPUS/UNLINK.json.bin" ] && [ -f "$CORPUS/TAS.json.bin" ] && \
    [ -f "$CORPUS/ROL.b.json.bin" ] && [ -f "$CORPUS/ROL.w.json.bin" ] && \
    [ -f "$CORPUS/ROL.l.json.bin" ] && \
    [ -f "$CORPUS/STOP.json.bin" ] || {
    echo "m68000 corpus is incomplete: $CORPUS" >&2
    exit 1
  }
fi

if [ -n "${TALOS_UCSD_INTERP:-}" ]; then
  case "$TALOS_UCSD_INTERP" in
    /*) INTERP=$TALOS_UCSD_INTERP ;;
    *) INTERP=$ROOT/$TALOS_UCSD_INTERP ;;
  esac
  [ -f "$INTERP/SYSTEM.INTERP" ] || {
    echo "SYSTEM.INTERP not found under: $INTERP" >&2
    exit 1
  }
fi

if [ -n "${TALOS_TOS_ROM:-}" ]; then
  case "$TALOS_TOS_ROM" in
    /*) ROM=$TALOS_TOS_ROM ;;
    *) ROM=$ROOT/$TALOS_TOS_ROM ;;
  esac
  [ -f "$ROM" ] || {
    echo "TOS ROM not found: $ROM" >&2
    exit 1
  }
fi

# 由內往外疊 docker 參數，避免把使用者傳給 go 的參數重新加引號。
set -- golang:1.24-bookworm /usr/local/go/bin/go "$@"
if [ -n "${ROM:-}" ]; then
  set -- -e TALOS_TOS_ROM=/tos.img -v "$ROM:/tos.img:ro" "$@"
fi
if [ -n "${INTERP:-}" ]; then
  set -- -e TALOS_UCSD_INTERP=/ucsd -v "$INTERP:/ucsd:ro" "$@"
fi
if [ -n "${CORPUS:-}" ]; then
  set -- -e TALOS_M68000_TESTS=/corpus -v "$CORPUS:/corpus:ro" "$@"
fi
set -- -v "$ROOT:/src" -w /src "$@"

run_go "$@"
