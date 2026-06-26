#!/bin/sh

set -eu

BASE_URL="${NETO_BASE_URL:-https://example.com/neto}"
TMP="${TMPDIR:-/tmp}/neto-upgrade.$$"

cleanup() {
	rm -f "$TMP"
}
trap cleanup EXIT INT TERM

if command -v curl >/dev/null 2>&1; then
	curl -fsSL "$BASE_URL/install.sh" -o "$TMP"
elif command -v wget >/dev/null 2>&1; then
	wget -O "$TMP" "$BASE_URL/install.sh"
else
	echo "neto upgrade: curl or wget is required" >&2
	exit 1
fi

sh "$TMP"

