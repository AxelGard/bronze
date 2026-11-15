# Editors

Bronze comes with two things editors can use:

- a **language server**, `bronze lsp`, for errors as you type, go to definition, hover, completion, and rename
- a **tree-sitter grammar**, in [`editors/tree-sitter-bronze`](https://github.com/AxelGard/bronze/tree/master/editors/tree-sitter-bronze), for highlighting

For Neovim, `bronze setup nvim` sets up both. Other editors that speak LSP or tree-sitter can use the same two pieces.

## The language server

`bronze lsp` reads and writes the Language Server Protocol on stdin and stdout. It supports:

| Feature | What it does |
|---|---|
| Diagnostics | every open `.brz` file is checked on each change, and its errors shown, the same ones `bronze check` reports |
| Go to definition | from a use of a variable, parameter, function, class, enum, enum case, field or method to where it is declared |
| Hover | shows what the name under the cursor is: `con total: Int` for a variable, the signature for a function or method, `class Account`, `Status.Frozen`. Built-ins like `print` and `List.append` show their signature too |
| Completion | after a `.`, the fields and methods of the value (or the cases of an enum, or the methods of a list, map or `String`); elsewhere, the variables, parameters, functions, classes, enums and built-ins visible at the cursor; in a written type, the type names |
| Rename | renames such a name everywhere in the file |

Rename refuses names that aren't valid identifiers, keywords and reserved words, and built-in names like `print`, `Error` or `init`. It needs a file without syntax errors.

Each file is checked on its own, since a Bronze program is one file for now.

## Neovim

With `bronze` installed and on your `PATH`, one command sets up Neovim (0.11 or later):

```sh
bronze setup nvim
```

It installs a package into Neovim's data directory (`~/.local/share/nvim/site/pack/bronze/start/bronze`), which Neovim loads by itself, and compiles the tree-sitter parser there with your C compiler. You get the `bronze` filetype, highlighting, `//` comments, and the language server with completion as you type. No plugin manager and no nvim-treesitter are needed.

Open a `.brz` file to check it: `:InspectTree` shows the syntax tree, and `:checkhealth vim.lsp` the language server.

Run `bronze setup nvim` again after updating `bronze`; Neovim warns when they don't match. To remove it, delete the package directory. Completion pops up as you type, except when nvim-cmp or blink.cmp is loaded (they ask the server themselves); set `vim.g.bronze_autocomplete = false` to turn it off.

If you set it up the old way, with `editors/nvim` in a plugin manager and the parser from nvim-treesitter, remove that plugin entry and run `:TSUninstall bronze`. `bronze setup nvim` tells you if Neovim still finds the old parser.

!!! note "Compiling the parser"
    It needs a C compiler: `cc`, or the one in `$CC`. On macOS, install one with `xcode-select --install`. If it fails with `ld: tapi error` or `unknown architecture`, the selected Xcode is older than the macOS SDK: run `sudo xcode-select -s /Library/Developer/CommandLineTools`, or update Xcode.

## Other editors

The language server works with any editor that speaks LSP: run `bronze lsp`, which talks over stdin and stdout. The tree-sitter grammar is in [`editors/tree-sitter-bronze`](https://github.com/AxelGard/bronze/tree/master/editors/tree-sitter-bronze), with highlight queries in `queries/highlights.scm`.
