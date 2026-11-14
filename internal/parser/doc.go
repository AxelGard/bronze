// Package parser builds an ast from tokens.
//
// It is a hand-written recursive-descent parser with precedence climbing for
// expressions. It must recover from errors (skip to the next statement or
// closing brace) so that it can report several errors at once and still
// produce a tree: the LSP depends on this.
package parser
