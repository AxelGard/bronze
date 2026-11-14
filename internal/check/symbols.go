package check

import (
	"slices"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/diag"
)

// SymbolKind is what kind of thing a symbol names.
type SymbolKind int

const (
	SymVariable SymbolKind = iota
	SymParameter
	SymFunction
	SymClass
	SymEnum
	SymCase
	SymField
	SymMethod
)

var symbolKindNames = [...]string{
	SymVariable:  "variable",
	SymParameter: "parameter",
	SymFunction:  "function",
	SymClass:     "class",
	SymEnum:      "enum",
	SymCase:      "case",
	SymField:     "field",
	SymMethod:    "method",
}

func (k SymbolKind) String() string { return symbolKindNames[k] }

// Symbol is something declared with a name: a variable, a function, a
// field and so on. The LSP uses symbols for go to definition and rename.
type Symbol struct {
	Name    string
	Kind    SymbolKind
	Decl    diag.Span // the name in its declaration
	Builtin bool      // declared by bronze itself, like Error.message

	// Hover is what the symbol is, as Bronze code: `con total: Int`,
	// `func add(a: Int, b: Int) -> Int`, `class Account`.
	Hover string
	In    string // the class, enum or type it belongs to, for fields and methods
	Doc   string // a description, for built-ins
}

// Ref is one place where a symbol's name is written, including its
// declaration.
type Ref struct {
	Span diag.Span
	Sym  *Symbol
}

// RefAt returns the reference at a byte offset, or nil. An offset just
// after a name still counts, so a cursor at the end of a word finds it.
func (info *Info) RefAt(offset int) *Ref {
	for i := range info.Refs {
		r := &info.Refs[i]
		if r.Span.Start.Offset <= offset && offset <= r.Span.End.Offset {
			return r
		}
	}
	return nil
}

// RefsTo returns every reference to sym, in source order.
func (info *Info) RefsTo(sym *Symbol) []Ref {
	var refs []Ref
	for _, r := range info.Refs {
		if r.Sym == sym {
			refs = append(refs, r)
		}
	}
	slices.SortFunc(refs, func(a, b Ref) int { return a.Span.Start.Offset - b.Span.Start.Offset })
	return refs
}

// newSym makes a symbol for a declaration and records the declaration as a
// reference.
func (c *checker) newSym(name *ast.Ident, kind SymbolKind, builtin bool) *Symbol {
	s := &Symbol{Name: name.Name, Kind: kind, Decl: name.Sp, Builtin: builtin || c.inPrelude}
	if !s.Builtin {
		c.ref(name, s)
	}
	return s
}

// ref records that name refers to sym. Names inside the prelude are not
// recorded: their spans are in another source.
func (c *checker) ref(name *ast.Ident, sym *Symbol) {
	if sym == nil || c.inPrelude {
		return
	}
	c.info.Refs = append(c.info.Refs, Ref{Span: name.Sp, Sym: sym})
}
