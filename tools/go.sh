#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

exec docker run --rm --network none \
  --memory 2g --cpus 2 --pids-limit 256 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" \
  -e GOCACHE=/tmp/go-cache \
  -e GOPATH=/tmp/go \
  -e PATH=/usr/local/go/bin:/usr/bin:/bin \
  -v "$ROOT:/src" -w /src \
  golang:1.24-bookworm /usr/local/go/bin/go "$@"
