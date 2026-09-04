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
  [ -f "$CORPUS/NOP.json.bin" ] && [ -f "$CORPUS/MOVE.q.json.bin" ] && \
    [ -f "$CORPUS/SWAP.json.bin" ] && [ -f "$CORPUS/EXT.w.json.bin" ] && \
    [ -f "$CORPUS/EXT.l.json.bin" ] || {
    echo "m68000 corpus is incomplete: $CORPUS" >&2
    exit 1
  }
  run_go -e TALOS_M68000_TESTS=/corpus -v "$CORPUS:/corpus:ro" \
    -v "$ROOT:/src" -w /src golang:1.24-bookworm /usr/local/go/bin/go "$@"
fi

run_go -v "$ROOT:/src" -w /src \
  golang:1.24-bookworm /usr/local/go/bin/go "$@"
