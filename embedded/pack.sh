#!/bin/sh

set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
OUT_DIR="${OUT_DIR:-$ROOT_DIR/dist}"
WORK_DIR="${TMPDIR:-/tmp}/neto-pack.$$"
ARCHIVE_ROOT="$WORK_DIR/neto"
ARCHIVE="$OUT_DIR/neto-openwrt-embedded.tar.gz"
VERSION="${NETO_VERSION:-$(git -C "$ROOT_DIR" describe --tags --always --dirty 2>/dev/null || echo dev)}"

cleanup() {
	rm -rf "$WORK_DIR"
}
trap cleanup EXIT INT TERM

build_netod() {
	target="$1"
	goarch="$2"
	goarm="$3"
	gomips="$4"
	dest="$ARCHIVE_ROOT/bin/$target/netod"

	mkdir -p "$(dirname "$dest")"
	echo "building $target"
	(
		cd "$ROOT_DIR"
		GOOS=linux \
		GOARCH="$goarch" \
		GOARM="$goarm" \
		GOMIPS="$gomips" \
		CGO_ENABLED=0 \
		go build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "$dest" ./cmd/netod
	)
	chmod 0755 "$dest"
}

copy_managed_singbox() {
	target="$1"
	src="$ROOT_DIR/embedded/bin/$target/sing-box"
	dest="$ARCHIVE_ROOT/bin/$target/sing-box"
	if [ -x "$src" ]; then
		cp "$src" "$dest"
		chmod 0755 "$dest"
	fi
}

rm -rf "$WORK_DIR"
mkdir -p "$ARCHIVE_ROOT" "$OUT_DIR"

cp -R "$ROOT_DIR/embedded/files" "$ARCHIVE_ROOT/files"
cp "$ROOT_DIR/embedded/install.sh" "$ARCHIVE_ROOT/install.sh"
cp "$ROOT_DIR/embedded/uninstall.sh" "$ARCHIVE_ROOT/uninstall.sh"
cp "$ROOT_DIR/embedded/upgrade.sh" "$ARCHIVE_ROOT/upgrade.sh"
chmod 0755 "$ARCHIVE_ROOT/install.sh" "$ARCHIVE_ROOT/uninstall.sh" "$ARCHIVE_ROOT/upgrade.sh"

build_netod "linux-amd64" "amd64" "" ""
build_netod "linux-arm64" "arm64" "" ""
build_netod "linux-armv7" "arm" "7" ""
build_netod "linux-mips-softfloat" "mips" "" "softfloat"
build_netod "linux-mipsle-softfloat" "mipsle" "" "softfloat"

copy_managed_singbox "linux-amd64"
copy_managed_singbox "linux-arm64"
copy_managed_singbox "linux-armv7"
copy_managed_singbox "linux-mips-softfloat"
copy_managed_singbox "linux-mipsle-softfloat"

tar -C "$WORK_DIR" -czf "$ARCHIVE" neto
echo "$ARCHIVE"
