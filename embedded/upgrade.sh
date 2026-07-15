#!/bin/sh

set -eu

INSTALL_URL="${NETO_INSTALL_URL:-https://raw.githubusercontent.com/elllkere/neto/main/embedded/install.sh}"
VERSION_URL="${NETO_VERSION_URL:-https://github.com/elllkere/neto/releases/latest/download/neto-version.txt}"
RELEASE_API_URL="${NETO_RELEASE_API_URL:-https://api.github.com/repos/elllkere/neto/releases/latest}"
NETOD_BIN="${NETO_NETOD_BIN:-/usr/bin/netod}"
TMP="${TMPDIR:-/tmp}/neto-upgrade.$$"
MODE="upgrade"

usage() {
	echo "usage: upgrade.sh [--check|--luci]" >&2
}

case "${1:-}" in
	"") ;;
	--check) MODE="check" ;;
	--luci) MODE="luci" ;;
	-h|--help)
		usage
		exit 0
		;;
	*)
		usage
		exit 1
		;;
esac

cleanup() {
	rm -f "$TMP" "$TMP.tmp"
}
trap cleanup EXIT INT TERM

curl_usable() {
	command -v curl >/dev/null 2>&1 || return 1
	curl --version >/dev/null 2>&1
}

download_text() {
	local url="$1"

	if curl_usable; then
		curl -fsSL --connect-timeout 5 --max-time 12 "$url" 2>/dev/null && return 0
	fi
	if command -v wget >/dev/null 2>&1; then
		wget -q -T 12 -t 1 -O- "$url" 2>/dev/null && return 0
	fi

	return 1
}

latest_version() {
	local value=""
	local json=""

	value="$(download_text "$VERSION_URL" || true)"
	value="$(printf '%s\n' "$value" | sed -n '1{s/[[:space:]]//g;p;}')"
	if [ -z "$value" ]; then
		json="$(download_text "$RELEASE_API_URL" || true)"
		value="$(printf '%s\n' "$json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
	fi

	[ -n "$value" ] || return 1
	release_version "$(normalize_version "$value")" || return 1
	printf '%s\n' "$value"
}

normalize_version() {
	printf '%s\n' "$1" | sed 's/^netod[[:space:]]*//; s/^v//; s/[-+].*$//'
}

release_version() {
	printf '%s\n' "$1" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'
}

version_ge() {
	awk -v a="$1" -v b="$2" 'BEGIN {
		split(a, av, ".");
		split(b, bv, ".");
		for (i = 1; i <= 3; i++) {
			ai = av[i] + 0;
			bi = bv[i] + 0;
			if (ai > bi) exit 0;
			if (ai < bi) exit 1;
		}
		exit 0;
	}'
}

check_version() {
	local current=""
	local latest=""
	local current_normalized=""
	local latest_normalized=""
	local status="available"

	current="$("$NETOD_BIN" version 2>/dev/null | awk '{ print $2; exit }')"
	[ -n "$current" ] || current="unknown"
	latest="$(latest_version)" || {
		echo "neto upgrade: failed to query the latest release" >&2
		exit 1
	}

	current_normalized="$(normalize_version "$current")"
	latest_normalized="$(normalize_version "$latest")"
	if [ "$current_normalized" = "$latest_normalized" ]; then
		status="current"
	elif release_version "$current_normalized" &&
		release_version "$latest_normalized" &&
		version_ge "$current_normalized" "$latest_normalized"; then
		status="current"
	fi

	printf 'current=%s\nlatest=%s\nstatus=%s\n' "$current" "$latest" "$status"
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

if [ "$MODE" = "check" ]; then
	check_version
	exit 0
fi

download "$INSTALL_URL" "$TMP"

sh "$TMP"
