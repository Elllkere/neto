#!/bin/sh

bin="${1:-/usr/libexec/neto/sing-box}"
config="${2:-/tmp/neto/sing-box.json}"
log_file="${NETO_SINGBOX_LOG:-/tmp/neto/sing-box.log}"
log_max_bytes="${NETO_SINGBOX_LOG_MAX_BYTES:-524288}"
log_keep_bytes="${NETO_SINGBOX_LOG_KEEP_BYTES:-262144}"
log_dir="${log_file%/*}"

rotate_log() {
	case "$log_max_bytes" in
		""|*[!0-9]*)
			log_max_bytes="524288"
			;;
	esac
	case "$log_keep_bytes" in
		""|*[!0-9]*)
			log_keep_bytes="262144"
			;;
	esac

	size="$(wc -c < "$log_file" 2>/dev/null || printf 0)"
	case "$size" in
		""|*[!0-9]*)
			return 0
			;;
	esac
	[ "$size" -gt "$log_max_bytes" ] || return 0

	tmp="${log_file}.$$"
	if tail -c "$log_keep_bytes" "$log_file" > "$tmp" 2>/dev/null; then
		mv "$tmp" "$log_file"
	else
		rm -f "$tmp"
	fi
}

if mkdir -p "$log_dir" 2>/dev/null && touch "$log_file" 2>/dev/null; then
	chmod 0644 "$log_file" 2>/dev/null || true
	rotate_log
	printf "%s neto: starting sing-box\n" "$(date '+%Y-%m-%d %H:%M:%S')" >> "$log_file"
	exec "$bin" run -c "$config" >> "$log_file" 2>&1
fi

exec "$bin" run -c "$config" >/dev/null 2>&1
