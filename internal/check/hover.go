package check

import (
	"strings"

	"github.com/AxelGard/bronze/internal/ast"
)

// The hover text of a symbol says what it is, as Bronze code:
//
//	con total: Int
//	func withdraw(amount: Int) -> Int ! Error
//	class Account
//
// with a line of notes below it, such as "in `Account`" for a field.

// varHover is the hover text of a variable, parameter or loop variable.
func varHover(name string, v *variable) string {
	s := name
	if t := typeText(v.typ); t != "" {
		s += ": " + t
	}
	switch v.kind {
	case Con:
		return "con " + s
	case Mut:
		return "mut " + s
	}
	return s
}

// typeText writes a type for hover text, or "" when it is not known.
func typeText(t Type) string {
	if t == nil || silent(t) || t == Null {
		return ""
	}
	return t.String()
}

// sigHover is the signature of a function or method:
// `func withdraw(amount: Int) -> Int ! Error`.
func sigHover(sig *funcSig) string {
	var ps []string
	for i, p := range sig.decl.Params {
		s := p.Name.Name
		if t := typeText(sig.params[i]); t != "" {
			s += ": " + t
		}
		ps = append(ps, s)
	}
	s := "func " + sig.name + "(" + strings.Join(ps, ", ") + ")"
	switch r := sig.result.(type) {
	case *Failable:
		if r.Value == NoValue {
			s += " -> ! " + r.Err.Name
		} else {
			s += " -> " + typeText(r.Value) + " ! " + r.Err.Name
		}
	default:
		if t := typeText(r); t != "" && r != NoValue {
			s += " -> " + t
		}
	}
	return s
}

// setHovers fills in the hover text of every declaration, once their types
// are known.
func (c *checker) setHovers() {
	for _, sig := range c.funcs {
		if sig.sym != nil {
			sig.sym.Hover = sigHover(sig)
		}
	}
	for _, cls := range c.classes {
		cls.Sym.Hover = "class " + cls.Name
		for _, f := range cls.Fields {
			if f.sym == nil || f.sym.Hover != "" {
				continue
			}
			kw := "con "
			if f.Mut {
				kw = "mut "
			}
			f.sym.Hover = kw + f.Name
			if t := typeText(f.typ); t != "" {
				f.sym.Hover += ": " + t
			}
			owner := cls.Name
			if cls.Super != nil && cls.Super.Fields[f.Name] == f {
				owner = cls.Super.Name
			}
			f.sym.In = owner
		}
		for _, sig := range cls.Methods {
			if sig.sym != nil {
				sig.sym.Hover = sigHover(sig)
				sig.sym.In = cls.Name
			}
		}
	}
	for _, en := range c.enums {
		en.Sym.Hover = "enum " + en.Name
		for name, sym := range en.CaseSym {
			sym.Hover = en.Name + "." + name
		}
		for _, sig := range en.Methods {
			if sig.sym != nil {
				sig.sym.Hover = sigHover(sig)
				sig.sym.In = en.Name
			}
		}
	}
}

// builtinFuncHovers are the signatures and descriptions of the built-in
// functions.
var builtinFuncHovers = map[string][2]string{
	"print":  {"func print(x: T)", "prints `x` and a newline; `x` is an Int, Float, Bool, String or enum, or a list or map of those"},
	"String": {"func String(x: T) -> String", "the text `print` would show for `x`"},
	"Float":  {"func Float(n: Int) -> Float", "the same number as a Float"},
	"panic":  {"func panic(message: String)", "stops the program with a stack trace, and exit code 101"},
	"assert": {"func assert(condition: Bool, message: String)", "panics with `assertion failed: message` if the condition is false"},
}

// refBuiltinFunc records a use of a built-in function, for hover.
func (c *checker) refBuiltinFunc(id *ast.Ident) {
	h, ok := builtinFuncHovers[id.Name]
	if !ok {
		return
	}
	c.ref(id, &Symbol{Name: id.Name, Kind: SymFunction, Builtin: true, Hover: h[0], Doc: h[1]})
}

// refBuiltinMethod records a use of a built-in method, for hover.
func (c *checker) refBuiltinMethod(name *ast.Ident, m *method, recv Type) {
	c.ref(name, &Symbol{Name: name.Name, Kind: SymMethod, Builtin: true, Hover: m.Signature(), In: recv.String()})
}
