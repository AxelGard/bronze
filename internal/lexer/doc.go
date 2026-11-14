// Package lexer turns Bronze source text into tokens.
//
// Every token records its position (file, line, column, byte offset) so that
// later stages and the LSP can point at exact source locations.
// Newlines are significant: they end statements (see doc/spec-0.1.md).
package lexer
