#!/bin/sh

set -eu

INSTALL_URL="${NETO_INSTALL_URL:-https://raw.githubusercontent.com/elllkere/neto/main/embedded/install.sh}"
TMP="${TMPDIR:-/tmp}/neto-upgrade.$$"

cleanup() {
	rm -f "$TMP"
}
trap cleanup EXIT INT TERM

if command -v curl >/dev/null 2>&1; then
	curl -fsSL "$INSTALL_URL" -o "$TMP"
elif command -v wget >/dev/null 2>&1; then
	wget -O "$TMP" "$INSTALL_URL"
else
	echo "neto upgrade: curl or wget is required" >&2
	exit 1
fi

sh "$TMP"
