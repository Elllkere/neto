#!/bin/sh

set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
OUT="${OUT:-$ROOT_DIR/dist/netod-dev}"

mkdir -p "$(dirname "$OUT")"
cd "$ROOT_DIR"
go build -buildvcs=false -trimpath -o "$OUT" ./cmd/netod
echo "$OUT"
