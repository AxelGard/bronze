// Package ast defines the Bronze syntax tree.
//
// Every node carries its source span. The parser produces this tree, the
// checker annotates it with types, and the interpreter runs it.
package ast
