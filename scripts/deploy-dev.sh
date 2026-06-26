#!/bin/sh

set -eu

: "${NETO_ROUTER:?set NETO_ROUTER to root@router-address}"

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
BIN="$ROOT_DIR/dist/netod-dev"

if [ ! -x "$BIN" ]; then
	"$ROOT_DIR/scripts/build-dev.sh" >/dev/null
fi

scp "$BIN" "$NETO_ROUTER:/usr/bin/netod"
ssh "$NETO_ROUTER" "chmod 0755 /usr/bin/netod && /etc/init.d/neto restart"

