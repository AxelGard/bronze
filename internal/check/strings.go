package check

import "github.com/AxelGard/bronze/internal/ast"

// slice checks `x[a..b]` and the other slices, which give a new list or
// String.
func (c *checker) slice(e *ast.Slice) Type {
	t := c.value(e.X)
	if c.mayBeNull(e.X, t) {
		t = t.(*Optional).Inner
	}
	for _, b := range []ast.Expr{e.Lo, e.Hi} {
		if b == nil {
			continue
		}
		if bt := c.value(b); !assignable(bt, Int) && !c.nullMismatch(b, bt, Int) {
			c.errorf(b.Span(), "a slice needs Int bounds, found %s", bt)
		}
	}
	switch t := t.(type) {
	case *List:
		return t
	}
	if t == String {
		return String
	}
	if !silent(t) {
		d := c.errorf(e.X.Span(), "cannot slice %s", t)
		d.Notes = []string{"only lists and strings can be sliced"}
	}
	return Invalid
}

func stringMethods() map[string]*method {
	str := []Type{String}
	ms := []*method{
		{name: "length", result: Int},
		{name: "contains", params: []string{"s"}, types: str, result: Bool},
		{name: "starts_with", params: []string{"s"}, types: str, result: Bool},
		{name: "ends_with", params: []string{"s"}, types: str, result: Bool},
		{name: "index_of", params: []string{"s"}, types: str, result: &Optional{Inner: Int}},
		{name: "split", params: []string{"sep"}, types: str, result: &List{Elem: String}},
		{name: "trim", result: String},
		{name: "to_upper", result: String},
		{name: "to_lower", result: String},
		{name: "replace", params: []string{"old", "new"}, types: []Type{String, String}, result: String},
	}
	byName := map[string]*method{}
	for _, m := range ms {
		m.recv = "String"
		byName[m.name] = m
	}
	return byName
}

// stringMethod checks a call to a method of a String.
func (c *checker) stringMethod(e *ast.Call, m *ast.Member) Type {
	bm := stringMethods()[m.Name.Name]
	if bm == nil {
		c.args(e)
		c.errorf(m.Name.Sp, "String has no method `%s`", m.Name.Name)
		return Invalid
	}
	c.refBuiltinMethod(m.Name, bm, String)
	c.methodArgs(e, m, bm)
	return bm.result
}
