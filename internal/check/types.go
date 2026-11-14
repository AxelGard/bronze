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
	// Null is the type of `null`. It fits every `T?`.
	Null Type = &special{"null"}
)

// Optional is `T?`: a T, or null.
type Optional struct {
	Inner Type
}

func (o *Optional) String() string { return o.Inner.String() + "?" }

// List is `List<T>`. Lists are shared by reference.
type List struct {
	Elem Type
}

func (l *List) String() string { return "List<" + l.Elem.String() + ">" }

// Failable is the result of calling a function that can fail: `T ! E`, or
// `! E` when Value is NoValue. It must be handled with `else`.
type Failable struct {
	Value Type
	Err   *Class
}

func (f *Failable) String() string {
	if f.Value == NoValue {
		return "! " + f.Err.Name
	}
	return f.Value.String() + " ! " + f.Err.Name
}

// silent reports whether t is a type that never causes an error.
func silent(t Type) bool { return t == Invalid || t == Never }

// identical reports whether a and b are the same type.
func identical(a, b Type) bool {
	if a == b {
		return true
	}
	switch a := a.(type) {
	case *Optional:
		b, ok := b.(*Optional)
		return ok && identical(a.Inner, b.Inner)
	case *List:
		b, ok := b.(*List)
		return ok && identical(a.Elem, b.Elem)
	}
	return false
}

// assignable reports whether a value of type from can be used as a to: the
// same type, a class that uses `Error` as an Error, or a T or null as a T?.
func assignable(from, to Type) bool {
	if identical(from, to) || silent(from) || to == Invalid {
		return true
	}
	switch to := to.(type) {
	case *Class:
		if k, ok := from.(*Class); ok {
			for ; k != nil; k = k.Super {
				if k == to {
					return true
				}
			}
		}
	case *Optional:
		if from == Null {
			return true
		}
		if f, ok := from.(*Optional); ok {
			return assignable(f.Inner, to.Inner)
		}
		return assignable(from, to.Inner)
	}
	return false
}

// hasValue reports whether a function result has a value: not nothing and
// not `! E`.
func hasValue(t Type) bool {
	if f, ok := t.(*Failable); ok {
		return f.Value != NoValue
	}
	return t != NoValue
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
	Super   *Class           // Error, for a class that uses Error
	Sym     *Symbol
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
	sym   *Symbol
}

// Enum is an enum type. Its values are its cases.
type Enum struct {
	Name    string
	Decl    *ast.EnumDecl
	Cases   map[string]int // case name to its position
	Methods map[string]*funcSig
	Sym     *Symbol
	CaseSym map[string]*Symbol
}

func (e *Enum) String() string { return e.Name }

// someCase returns the name of the first case, for examples in messages.
func (e *Enum) someCase() string {
	if len(e.Decl.Cases) == 0 {
		return "Case"
	}
	return e.Decl.Cases[0].Name
}
