#!/bin/sh
# Installs Bronze for this user, from this clone:
#   1. builds `bronze` into ~/.local/bin (or $BRONZE_BIN)
#   2. if Neovim is installed, sets it up with `bronze setup nvim`
#      (highlighting and the language server)
#
# Run it from anywhere: scripts/install.sh, or `make install`.
set -eu

repo=$(cd "$(dirname "$0")/.." && pwd)
bindir=${BRONZE_BIN:-$HOME/.local/bin}

fail() { printf 'error: %s\n' "$*" >&2; exit 1; }

command -v go >/dev/null 2>&1 || fail "Go is not installed (https://go.dev/dl/)"
mkdir -p "$bindir"
printf 'building bronze into %s\n' "$bindir"
(cd "$repo" && go build -o "$bindir/bronze" ./cmd/bronze)
"$bindir/bronze" version

case ":$PATH:" in
*":$bindir:"*) ;;
*) printf '\nwarning: %s is not on your PATH; add it, or Neovim won'"'"'t find `bronze lsp`:\n    export PATH="%s:$PATH"\n' "$bindir" "$bindir" >&2 ;;
esac

if command -v nvim >/dev/null 2>&1; then
    printf '\n'
    "$bindir/bronze" setup nvim
fi
