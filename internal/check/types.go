package check

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
