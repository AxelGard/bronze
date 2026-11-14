package check

import (
	"slices"

	"github.com/AxelGard/bronze/internal/ast"
)

// Type is a Bronze type.
type Type interface {
	String() string
}

// Basic is one of the built-in value types.
type Basic struct{ name string }

func (b *Basic) String() string { return b.name }

var (
	Int    = &Basic{"Int"}
	Float  = &Basic{"Float"}
	Bool   = &Basic{"Bool"}
	String = &Basic{"String"}
)

// special types, which are not written in source.
type special struct{ name string }

func (s *special) String() string { return s.name }

var (
	// NoValue is the result of a function without `->`.
	NoValue Type = &special{"no value"}
	// Never is the result of `panic(...)`: it never returns, so it fits
	// wherever a value is expected.
	Never Type = &special{"Never"}
	// Invalid is the type of an expression that already has an error. It is
	// accepted everywhere, so one mistake gives one error.
	Invalid Type = &special{"invalid"}
)

// silent reports whether t is a type that never causes an error.
func silent(t Type) bool { return t == Invalid || t == Never }

// assignable reports whether a value of type from can be used as a to.
func assignable(from, to Type) bool {
	return from == to || silent(from) || to == Invalid
}

func isNumeric(t Type) bool { return t == Int || t == Float }

// builtinTypes are the type names that can be written in source.
var builtinTypes = map[string]Type{
	"Int":    Int,
	"Float":  Float,
	"Bool":   Bool,
	"String": String,
}

// Class is a class type. Objects are shared by reference.
type Class struct {
	Name    string
	Decl    *ast.ClassDecl
	Fields  map[string]*Field
	Methods map[string]*funcSig
	Inits   map[int]*funcSig // by number of parameters
}

func (c *Class) String() string { return c.Name }

// initArities returns the numbers of parameters of the class's inits, in
// increasing order.
func (c *Class) initArities() []int {
	var ns []int
	for n := range c.Inits {
		ns = append(ns, n)
	}
	slices.Sort(ns)
	return ns
}

// Field is a field of a class.
type Field struct {
	Name  string
	Mut   bool
	Decl  *ast.Field
	typ   Type // nil until the initial value is checked
	state int  // 0: not checked, 1: being checked, 2: done
}

// Enum is an enum type. Its values are its cases.
type Enum struct {
	Name    string
	Decl    *ast.EnumDecl
	Cases   map[string]int // case name to its position
	Methods map[string]*funcSig
}

func (e *Enum) String() string { return e.Name }

// someCase returns the name of the first case, for examples in messages.
func (e *Enum) someCase() string {
	if len(e.Decl.Cases) == 0 {
		return "Case"
	}
	return e.Decl.Cases[0].Name
}
