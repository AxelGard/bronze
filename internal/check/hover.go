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
		if v.kind == Param {
			t = v.mode + t
		}
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
			if p.Mut {
				t = "mut " + t
			}
			if p.Copy {
				t = "copy " + t
			}
			s += ": " + t
		}
		ps = append(ps, s)
	}
	s := "func " + sig.name + "(" + strings.Join(ps, ", ") + ")"
	if sig.mut {
		s = "mut " + s
	}
	if sig.static {
		s = "static " + s
	}
	s = visPrefix(sigVis(sig)) + s
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
		switch {
		case cls.Data && cls.Decl.Short:
			cls.Sym.Hover = "data " + cls.Name + dataParamsText(cls.Decl)
		case cls.Data:
			cls.Sym.Hover = "data " + cls.Name
		}
		var uses []string
		if cls.Super != nil && !cls.Decl.Builtin {
			uses = append(uses, cls.Super.Name)
		}
		for _, i := range cls.Ifaces {
			uses = append(uses, i.Name)
		}
		if len(uses) > 0 {
			cls.Sym.Hover += " use " + strings.Join(uses, ", ")
		}
		for _, f := range cls.Fields {
			if f.sym == nil || f.sym.Hover != "" {
				continue
			}
			kw := "con "
			if f.Mut {
				kw = "mut "
			}
			f.sym.Hover = visPrefix(f.Decl.Vis) + kw + f.Name
			if t := typeText(f.typ); t != "" {
				f.sym.Hover += ": " + t
			}
			f.sym.In = f.Owner.Name
		}
		for _, sig := range cls.Methods {
			if sig.sym != nil {
				sig.sym.Hover = sigHover(sig)
				sig.sym.In = sig.class.Name
			}
		}
	}
	for _, i := range c.ifaces {
		i.Sym.Hover = "interface " + i.Name
		for j, p := range i.Parents {
			if j == 0 {
				i.Sym.Hover += " use "
			} else {
				i.Sym.Hover += ", "
			}
			i.Sym.Hover += p.Name
		}
		for _, sig := range i.Methods {
			if sig.sym != nil {
				sig.sym.Hover = sigHover(sig)
				sig.sym.In = sig.iface.Name
			}
		}
	}
	for _, en := range c.enums {
		en.Sym.Hover = "enum " + en.Name
		if en.Value != nil {
			en.Sym.Hover += ": " + en.Value.String()
		}
		for name, sym := range en.CaseSym {
			sym.Hover = caseHover(en, name)
		}
		for _, sig := range en.Methods {
			if sig.sym != nil {
				sig.sym.Hover = sigHover(sig)
				sig.sym.In = en.Name
			}
		}
	}
}

// caseHover is the hover text of an enum case: `Status.Frozen`, or
// `Role.Admin = "admin"`.
func caseHover(en *Enum, name string) string {
	s := en.Name + "." + name
	if i, ok := en.Cases[name]; ok && en.Value != nil && i < len(en.Decl.Values) && en.Decl.Values[i] != nil {
		if text, t := literalValue(en.Decl.Values[i]); t != nil {
			s += " = " + text
		}
	}
	return s
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
