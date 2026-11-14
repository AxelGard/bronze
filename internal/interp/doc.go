// Package interp is a tree-walking interpreter for checked Bronze programs.
//
// It runs the ast directly (no bytecode, no VM). It assumes the program has
// already passed the checker. Runtime failures are panics with a Bronze stack
// trace (function names and .brz line numbers).
package interp
