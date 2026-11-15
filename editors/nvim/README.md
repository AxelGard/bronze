# Bronze for Neovim

Highlighting with the tree-sitter grammar, and the language server (`bronze lsp`: diagnostics, go to definition, hover, completion, rename). Needs Neovim 0.11 or later. nvim-treesitter and a plugin manager are not needed.

## Install

With `bronze` installed (see [Install](https://axelgard.github.io/bronze/install/)) and on your `PATH`:

```
bronze setup nvim
```

That's all. Open a `.brz` file to try it: `:InspectTree` shows the syntax tree, and `:checkhealth vim.lsp` the language server. Run `bronze setup nvim` again after updating `bronze`; Neovim tells you when the two don't match.

`bronze setup nvim` installs a package into Neovim's data directory, `~/.local/share/nvim/site/pack/bronze/start/bronze` (Neovim loads packages there by itself), and compiles the parser there with the C compiler (`cc`, or `$CC`). `--dir <dir>` installs it somewhere else. To remove it, delete that directory.

From a clone of the repository, `make install` builds `bronze` and runs `bronze setup nvim`.

## What it installs

| File | What |
|---|---|
| `ftdetect/bronze.lua` | `.brz` files are `bronze` |
| `plugin/bronze.lua` | starts highlighting in `bronze` buffers and sets `commentstring`; starts `bronze lsp` if `bronze` is on the `PATH`, with completion as you type (unless nvim-cmp or blink.cmp is loaded, or `vim.g.bronze_autocomplete = false`); warns if the parser is missing or was set up by another version of `bronze` |
| `parser/bronze.so` | the parser, compiled from `editors/tree-sitter-bronze/src` |
| `queries/bronze/highlights.scm` | the grammar's highlight queries |
| `version.txt` | the `bronze` that set it up |

The plugin and the grammar are built into the `bronze` binary (`editors/embed.go`), so a released `bronze` can set up Neovim without a clone.

## Moving from the old setup

Before, the plugin was this directory, added to a plugin manager, with the parser built by nvim-treesitter. After running `bronze setup nvim`, remove the `editors/nvim` entry from your plugin manager, and the old parser with `:TSUninstall bronze`. `bronze setup nvim` warns if Neovim still finds another Bronze parser.

## Troubleshooting

- **No colors, and a warning that the parser is not installed:** run `bronze setup nvim`.
- **No language server:** `bronze` must be on the `PATH` that Neovim sees; `:checkhealth vim.lsp` shows what happened, and `:LspLog` the server's log.
- **macOS, `ld: tapi error` or `unknown architecture` while compiling:** the selected Xcode is older than the macOS SDK. Run `sudo xcode-select -s /Library/Developer/CommandLineTools`, or update Xcode. Without any compiler, `xcode-select --install`.
