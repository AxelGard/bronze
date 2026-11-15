// Package editors holds the editor support files that `bronze setup`
// installs: the tree-sitter grammar's C source and highlight queries, and
// the Neovim plugin. They are built into the bronze binary, so a released
// bronze can set up an editor without a clone of the repository.
package editors

import "embed"

// Files are the support files, at their paths in this directory.
//
//go:embed tree-sitter-bronze/src/parser.c tree-sitter-bronze/src/tree_sitter/*.h
//go:embed tree-sitter-bronze/queries/*.scm
//go:embed nvim/plugin/bronze.lua nvim/ftdetect/bronze.lua
var Files embed.FS
