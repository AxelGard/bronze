# Editors

Bronze comes with two things editors can use:

- a **language server**, `bronze lsp`, for errors as you type, go to definition, hover, completion, and rename
- a **tree-sitter grammar**, in [`editors/tree-sitter-bronze`](https://github.com/AxelGard/bronze/tree/master/editors/tree-sitter-bronze), for highlighting

There is a ready-made plugin for Neovim. Other editors that speak LSP or tree-sitter can use the same two pieces.

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

The plugin is in [`editors/nvim`](https://github.com/AxelGard/bronze/tree/master/editors/nvim). It gives `.brz` files the `bronze` filetype, tree-sitter highlighting, and starts `bronze lsp`.

It needs Neovim 0.11 or later, and [nvim-treesitter](https://github.com/nvim-treesitter/nvim-treesitter) (master branch) for highlighting.

1. Clone the Bronze repository, and add its `editors/nvim` directory as a plugin. With packer:

    ```lua
    use "~/path/to/bronze/editors/nvim"
    ```

    Restart Neovim.

2. Make sure `bronze` is on your `PATH` (see [Install](../install.md)). The plugin only starts the language server if it finds it.

3. Build the parser, in Neovim:

    ```vim
    :TSInstallSync bronze
    ```

    Or run `make install` in the clone: it builds `bronze` into `~/.local/bin` and the parser in one go. Run it again after pulling changes.

To check that it works, open a `.brz` file: `:InspectTree` shows the syntax tree, `:checkhealth vim.lsp` the language server, and `:LspLog` its log.

!!! note "Building the parser on macOS"
    It needs a working C compiler. If it fails with `ld: tapi error` or `unknown architecture`, the selected Xcode is older than the macOS SDK: run `sudo xcode-select -s /Library/Developer/CommandLineTools`, or update Xcode.
