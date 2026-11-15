# Bronze for Neovim

A small plugin: `.brz` files get the `bronze` filetype, tree-sitter highlighting, and the language server (`bronze lsp`: diagnostics, go to definition, hover, completion, rename).

Needs Neovim 0.11 or later, and [nvim-treesitter](https://github.com/nvim-treesitter/nvim-treesitter) (master branch) for highlighting.

## Install

1. Add this directory as a plugin. With packer:

   ```lua
   use "~/Programs/repositories/bronze/editors/nvim"
   ```

   (Use the path of your clone.) Restart Neovim.

2. From the repository root, run:

   ```
   make install
   ```

   This builds `bronze` into `~/.local/bin` (make sure it is on your `PATH`), and builds the tree-sitter parser with `:TSInstallSync bronze`. Run it again after pulling changes to the compiler or the grammar.

Check it in a `.brz` file: `:InspectTree` shows the syntax tree, `:checkhealth vim.lsp` the language server, and `:LspLog` its log.

## What the plugin does

| File | What |
|---|---|
| `ftdetect/bronze.lua` | `.brz` files are `bronze` |
| `plugin/bronze.lua` | registers the grammar in `editors/tree-sitter-bronze` with nvim-treesitter, starts highlighting for `bronze` buffers, and sets up `bronze lsp` if `bronze` is on the `PATH`, with completion as you type (unless nvim-cmp or blink.cmp is loaded, or `vim.g.bronze_autocomplete = false`) |
| `queries/bronze/highlights.scm` | a link to the grammar's highlight queries |

On macOS, building the parser needs a working C compiler. If `make install` reports `ld: tapi error` or `unknown architecture`, the selected Xcode is older than the macOS SDK: `sudo xcode-select -s /Library/Developer/CommandLineTools`, or update Xcode.
