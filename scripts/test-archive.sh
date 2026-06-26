#!/bin/sh

set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
ARCHIVE="${1:-$ROOT_DIR/dist/neto-openwrt-embedded.tar.gz}"
TMP="${TMPDIR:-/tmp}/neto-archive-test.$$"

cleanup() {
	rm -rf "$TMP"
}
trap cleanup EXIT INT TERM

[ -f "$ARCHIVE" ] || {
	echo "archive not found: $ARCHIVE" >&2
	exit 1
}

mkdir -p "$TMP"
tar -xzf "$ARCHIVE" -C "$TMP"
if [ -d "$TMP/neto" ]; then
	TMP="$TMP/neto"
fi

for target in \
	linux-amd64 \
	linux-arm64 \
	linux-armv7 \
	linux-mips-softfloat \
	linux-mipsle-softfloat
do
	[ -x "$TMP/bin/$target/netod" ] || {
		echo "missing netod for $target" >&2
		exit 1
	}
done

[ -f "$TMP/files/etc/config/neto" ] || {
	echo "missing default UCI config" >&2
	exit 1
}
[ -x "$TMP/install.sh" ] || {
	echo "missing install.sh" >&2
	exit 1
}
[ -x "$TMP/uninstall.sh" ] || {
	echo "missing uninstall.sh" >&2
	exit 1
}
[ -x "$TMP/upgrade.sh" ] || {
	echo "missing upgrade.sh" >&2
	exit 1
}

echo "archive ok"
