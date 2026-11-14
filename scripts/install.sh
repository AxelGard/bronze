#!/bin/sh
# Installs Bronze for this user:
#   1. builds `bronze` into ~/.local/bin (or $BRONZE_BIN)
#   2. if Neovim is set up with editors/nvim, builds the tree-sitter parser
#      through nvim-treesitter (:TSInstallSync bronze)
#
# Run it from anywhere: scripts/install.sh, or `make install`.
set -eu

repo=$(cd "$(dirname "$0")/.." && pwd)
bindir=${BRONZE_BIN:-$HOME/.local/bin}

say() { printf '%s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }
fail() { printf 'error: %s\n' "$*" >&2; exit 1; }

# --- bronze -----------------------------------------------------------

command -v go >/dev/null 2>&1 || fail "Go is not installed (https://go.dev/dl/)"
mkdir -p "$bindir"
say "building bronze into $bindir"
(cd "$repo" && go build -o "$bindir/bronze" ./cmd/bronze)
"$bindir/bronze" version

case ":$PATH:" in
*":$bindir:"*) ;;
*) warn "$bindir is not on your PATH; add it, or Neovim won't find \`bronze lsp\`:
    export PATH=\"$bindir:\$PATH\"" ;;
esac

# --- Neovim -----------------------------------------------------------

if ! command -v nvim >/dev/null 2>&1; then
    say "Neovim not found; skipping editor setup"
    exit 0
fi

# Is the plugin (editors/nvim) loaded, and does it see nvim-treesitter?
state=$(nvim --headless \
    -c 'lua local ok, p = pcall(require, "nvim-treesitter.parsers"); io.stdout:write(vim.g.loaded_bronze and (ok and p.get_parser_configs and p.get_parser_configs().bronze and "ready" or "no-treesitter") or "no-plugin")' \
    -c 'qa!' 2>/dev/null || true)

case "$state" in
ready) ;;
no-treesitter)
    warn "the Bronze plugin is loaded, but nvim-treesitter (master branch) is not; skipping the parser"
    exit 0 ;;
*)
    say ""
    say "To set up Neovim, add the plugin in editors/nvim. With packer:"
    say "    use \"$repo/editors/nvim\""
    say "Then restart Neovim, and run this script again to build the parser."
    exit 0 ;;
esac

# nvim-treesitter compiles src/parser.c with the system C compiler. Make
# sure that works first, with a message that says what is wrong if not.
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
printf 'int bronze_check(void) { return 0; }\n' > "$tmp/check.c"
if ! cc -shared -fPIC -o "$tmp/check.so" "$tmp/check.c" 2> "$tmp/cc.log"; then
    cat "$tmp/cc.log" >&2
    if [ "$(uname)" = Darwin ] && grep -q "tapi\|unknown architecture" "$tmp/cc.log"; then
        fail "the C compiler can't link: the selected Xcode is older than the macOS SDK.
Select the Command Line Tools, or update Xcode:
    sudo xcode-select -s /Library/Developer/CommandLineTools"
    fi
    fail "the C compiler (cc) doesn't work, so the parser can't be built"
fi

say "building the tree-sitter parser (:TSInstallSync bronze)"
nvim --headless -c 'TSInstallSync! bronze' -c 'qa!'
say ""
say "done. Open a .brz file; :InspectTree shows the tree, and :LspInfo the language server."
