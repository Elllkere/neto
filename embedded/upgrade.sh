#!/bin/sh

set -eu

INSTALL_URL="${NETO_INSTALL_URL:-https://raw.githubusercontent.com/elllkere/neto/main/embedded/install.sh}"
TMP="${TMPDIR:-/tmp}/neto-upgrade.$$"

cleanup() {
	rm -f "$TMP" "$TMP.tmp"
}
trap cleanup EXIT INT TERM

curl_usable() {
	command -v curl >/dev/null 2>&1 || return 1
	curl --version >/dev/null 2>&1
}

download() {
	local url="$1"
	local dest="$2"
	local tmp="$dest.tmp"
	local attempts=""

	rm -f "$tmp"
	if command -v wget >/dev/null 2>&1; then
		attempts="$attempts wget"
		if wget -O "$tmp" "$url"; then
			mv "$tmp" "$dest"
			return 0
		fi
		rm -f "$tmp"
	fi
	if curl_usable; then
		attempts="$attempts curl"
		if curl -fsSL "$url" -o "$tmp"; then
			mv "$tmp" "$dest"
			return 0
		fi
		rm -f "$tmp"
	elif command -v curl >/dev/null 2>&1; then
		attempts="$attempts broken-curl"
	fi

	echo "neto upgrade: failed to download $url; attempted:${attempts:- none}" >&2
	exit 1
}

download "$INSTALL_URL" "$TMP"

sh "$TMP"
