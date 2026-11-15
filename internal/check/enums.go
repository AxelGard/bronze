package check

import (
	"fmt"
	"strings"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/lexer"
)

// Enums with values: `enum Role: String { Admin = "admin" }`. Each case has
// a value of the enum's value type, `r.value` gives it, and `Role.from(x)`
// gives the case with that value, or null.

// enumValues checks the value type of an enum and the values of its cases.
func (c *checker) enumValues(en *Enum) {
	d := en.Decl
	if d.Type != nil {
		switch t := c.resolveType(d.Type); {
		case silent(t):
		case (t == Int || t == String) && !d.Type.Optional:
			en.Value = t
		default:
			d := c.errorf(d.Type.Sp, "an enum's values can be Int or String, not %s", t)
			d.Notes = []string{"the values are for storing or sending cases, as in a database or JSON"}
		}
	}
	seen := map[string]*ast.Ident{} // a value's text, to the case that has it
	for i, cs := range d.Cases {
		var v ast.Expr
		if i < len(d.Values) {
			v = d.Values[i]
		}
		switch {
		case v == nil && en.Value != nil:
			d := c.errorf(cs.Sp, "case `%s` needs a value, since `%s` is an enum of %s", cs.Name, en.Name, en.Value)
			d.Help = []string{fmt.Sprintf("write it out: `%s = %s`", cs.Name, caseExample(cs.Name, en.Value))}
		case v == nil:
		case d.Type == nil:
			d := c.errorf(v.Span(), "`%s` has no value type, so its cases can't have values", en.Name)
			d.Help = []string{fmt.Sprintf("give it one: `enum %s: Int` or `enum %s: String`", en.Name, en.Name)}
		case en.Value != nil:
			text, t := literalValue(v)
			if t == nil {
				d := c.errorf(v.Span(), "an enum value must be written as a literal, like %s", exampleValue(en.Value))
				d.Notes = []string{"a value can't be computed, so that it can't change by accident"}
				continue
			}
			if t != en.Value {
				c.errorf(v.Span(), "the values of `%s` are %s, but this is %s", en.Name, en.Value, t)
				continue
			}
			if prev, ok := seen[text]; ok {
				c.errorf(v.Span(), "`%s.%s` already has the value %s", en.Name, prev.Name, c.text(v)).Notes = []string{
					"each case must have a value of its own, so that `from` can find it",
				}
				continue
			}
			seen[text] = cs
		}
	}
}

// literalValue returns the text and type of an enum value: an Int (maybe
// negative) or a String literal. It returns a nil type for anything else.
func literalValue(e ast.Expr) (string, Type) {
	switch x := e.(type) {
	case *ast.IntLit:
		return fmt.Sprint(x.Value), Int
	case *ast.Unary:
		if n, ok := x.X.(*ast.IntLit); ok && x.Op == lexer.MINUS {
			return fmt.Sprint(-n.Value), Int
		}
	case *ast.StringLit:
		return fmt.Sprintf("%q", x.Value), String
	}
	return "", nil
}

// caseExample is an example value for a case: `"member"` or `1`.
func caseExample(name string, t Type) string {
	if t == String {
		return fmt.Sprintf("%q", strings.ToLower(name))
	}
	return "1"
}

func exampleValue(t Type) string {
	if t == String {
		return `"admin"`
	}
	return "1"
}

// enumBuiltinNote describes the built-in `value` or `from` of an enum.
func enumBuiltinNote(en *Enum, name string) string {
	if name == "value" {
		return fmt.Sprintf("`%s.%s.value` is its %s", en.Name, en.someCase(), en.Value)
	}
	return fmt.Sprintf("`%s.from(x)` finds the case with value `x`", en.Name)
}

// valueSym is the symbol of the built-in `value` of an enum, for hover.
func valueSym(en *Enum) *Symbol {
	return &Symbol{Name: "value", Kind: SymField, Builtin: true, Hover: "con value: " + en.Value.String(), In: en.Name}
}

// fromMethod is the built-in `from` of an enum with values.
func fromMethod(en *Enum) *method {
	return &method{recv: en.Name, name: "from", params: []string{"value"}, types: []Type{en.Value}, result: &Optional{Inner: en}}
}

// enumValue checks `x.value` on an enum case.
func (c *checker) enumValue(m *ast.Member, en *Enum) Type {
	switch {
	case m.Name.Name != "value":
		c.errorf(m.Name.Sp, "%s has no fields", en.Name)
		return Invalid
	case en.Value == nil:
		d := c.errorf(m.Name.Sp, "`%s` has no values, so its cases have no `value`", en.Name)
		d.Help = []string{fmt.Sprintf("give it values: `enum %s: String { %s = \"...\" }`", en.Name, en.someCase())}
		return Invalid
	}
	c.ref(m.Name, valueSym(en))
	return en.Value
}

// enumFrom checks `Role.from(x)`, or another call on the enum's name.
func (c *checker) enumFrom(e *ast.Call, m *ast.Member, en *Enum) Type {
	name := m.Name.Name
	switch {
	case name == "from" && en.Value != nil:
		bm := fromMethod(en)
		c.refBuiltinMethod(m.Name, bm, en)
		c.methodArgs(e, m, bm)
		return bm.result
	case name == "from":
		c.args(e)
		d := c.errorf(m.Name.Sp, "`%s` has no values, so it has no `from`", en.Name)
		d.Help = []string{fmt.Sprintf("give it values: `enum %s: String { %s = \"...\" }`", en.Name, en.someCase())}
		return Invalid
	case en.Methods[name] != nil:
		c.args(e)
		d := c.errorf(m.Name.Sp, "`%s` is a method of each `%s` case", name, en.Name)
		d.Help = []string{fmt.Sprintf("call it on a case, like `%s.%s.%s()`", en.Name, en.someCase(), name)}
		return Invalid
	}
	c.args(e)
	c.errorf(m.Name.Sp, "`%s` has no `%s`", en.Name, name)
	return Invalid
}
