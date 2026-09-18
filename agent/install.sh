#!/bin/sh
set -eu

UPSTREAM_URL="https://raw.githubusercontent.com/komari-monitor/komari-agent/refs/heads/main/install.sh"
TMP_SCRIPT="${TMPDIR:-/tmp}/komarix-agent-install-$$.sh"
TMP_BRANDED="${TMP_SCRIPT}.branded"

cleanup() {
    rm -f "$TMP_SCRIPT" "$TMP_BRANDED"
}
trap cleanup EXIT INT TERM

if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$UPSTREAM_URL" -o "$TMP_SCRIPT"
elif command -v wget >/dev/null 2>&1; then
    wget -qO "$TMP_SCRIPT" "$UPSTREAM_URL"
else
    echo "[ERROR] KomariX Agent installer requires curl or wget." >&2
    exit 1
fi

# Keep upstream installation logic and compatibility names unchanged.
# Only replace user-visible Komari Agent branding.
sed \
    -e 's/Komari Agent/KomariX Agent/g' \
    -e 's/Komari-agent/KomariX Agent/g' \
    -e 's/KOMARI Agent/KomariX Agent/g' \
    "$TMP_SCRIPT" > "$TMP_BRANDED"
mv "$TMP_BRANDED" "$TMP_SCRIPT"

set +e
sh "$TMP_SCRIPT" "$@"
status=$?
set -e
cleanup
trap - EXIT INT TERM
exit "$status"
