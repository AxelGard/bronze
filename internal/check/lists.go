package check

import (
	"fmt"
	"strings"

	"github.com/AxelGard/bronze/internal/ast"
)

// genericTypes are the built-in types that take type arguments.
var genericTypes = map[string]bool{"List": true, "Map": true}

// resolveBase resolves a written type without its `?`.
func (c *checker) resolveBase(t *ast.TypeExpr) Type {
	switch t.Name.Name {
	case "List":
		if len(t.Args) != 1 {
			d := c.errorf(t.Sp, "`List` needs one type argument, the type of its elements")
			d.Help = []string{"write it like `List<Int>`"}
			return Invalid
		}
		elem := c.resolveType(t.Args[0])
		if !c.elemType(t.Args[0], elem, "a list's elements") {
			return Invalid
		}
		return &List{Elem: elem}
	case "Map":
		return c.mapType(t)
	}
	base := c.resolveName(t.Name)
	if len(t.Args) > 0 && !silent(base) {
		c.errorf(t.Sp, "`%s` takes no type arguments", t.Name.Name)
		return Invalid
	}
	return base
}

// elemType checks that t can be the type of a list's elements or a map's
// keys or values: anything but a `T?`.
func (c *checker) elemType(at *ast.TypeExpr, t Type, what string) bool {
	if silent(t) {
		return false
	}
	if _, ok := t.(*Optional); ok {
		d := c.errorf(at.Sp, "%s can't be a `?` type, like %s", what, t)
		d.Notes = []string{"indexing already gives a `T?`, so the elements themselves can't be null"}
		return false
	}
	return true
}

// valueFor checks an expression whose value is used where a value of type
// want is expected (want may be nil). The expected type gives `[]` its
// type, and lets the elements of a list literal be anything that fits.
func (c *checker) valueFor(e ast.Expr, want Type) Type {
	switch x := e.(type) {
	case *ast.ListLit:
		t := c.listLit(x, want)
		c.info.Types[e] = t
		return t
	case *ast.MapLit:
		t := c.mapLit(x, want)
		c.info.Types[e] = t
		return t
	case *ast.Paren:
		if isLiteral(x.X) {
			t := c.valueFor(x.X, want)
			c.info.Types[e] = t
			return t
		}
	}
	return c.value(e)
}

// isLiteral reports whether e is a list or map literal, which take their
// type from where they are used.
func isLiteral(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.ListLit, *ast.MapLit:
		return true
	case *ast.Paren:
		return isLiteral(x.X)
	}
	return false
}

// argsFor checks the arguments of a call to something with these
// parameter types. With the wrong number of arguments, the types are not
// used.
func (c *checker) argsFor(e *ast.Call, params []Type) []Type {
	if len(e.Args) != len(params) {
		return c.args(e)
	}
	types := make([]Type, len(e.Args))
	for i, a := range e.Args {
		types[i] = c.valueFor(a, params[i])
	}
	return types
}

func (c *checker) listLit(l *ast.ListLit, want Type) Type {
	if wl, ok := nonNull(want).(*List); ok {
		for _, el := range l.Elems {
			t := c.valueFor(el, wl.Elem)
			if !assignable(t, wl.Elem) && !c.nullMismatch(el, t, wl.Elem) {
				c.errorf(el.Span(), "this element is %s, but the list holds %s", t, wl.Elem)
			}
		}
		return wl
	}
	if len(l.Elems) == 0 {
		d := c.errorf(l.Sp, "`[]` needs a type here")
		d.Help = []string{"write the type where it is declared, like `con names: List<String> = []`"}
		return Invalid
	}
	elem := c.sameTypes(l.Elems, "list elements", "a list")
	if elem == nil {
		return Invalid
	}
	return &List{Elem: elem}
}

// index checks `x[i]`, which is a `T?`: null when there is nothing there.
func (c *checker) index(e *ast.Index) Type {
	t := c.value(e.X)
	if c.mayBeNull(e.X, t) {
		t = t.(*Optional).Inner
	}
	switch t := t.(type) {
	case *List:
		c.intIndex(e.Index)
		return &Optional{Inner: t.Elem}
	case *Map:
		c.mapKey(t, e.Index)
		return &Optional{Inner: t.Val}
	}
	if t == String {
		c.intIndex(e.Index)
		return &Optional{Inner: String}
	}
	c.value(e.Index)
	if !silent(t) {
		d := c.errorf(e.X.Span(), "cannot index %s", t)
		d.Notes = []string{"only lists, maps and strings can be indexed with `[ ]`"}
	}
	return Invalid
}

// intIndex checks an index into a list or a string.
func (c *checker) intIndex(i ast.Expr) {
	t := c.value(i)
	if !assignable(t, Int) && !c.nullMismatch(i, t, Int) {
		c.errorf(i.Span(), "an index must be an Int, found %s", t)
	}
}

// indexTarget checks `x[i]` as the target of an assignment, and returns
// the element's type, or nil after reporting an error.
func (c *checker) indexTarget(e *ast.Index) Type {
	t := c.value(e.X)
	if c.mayBeNull(e.X, t) {
		t = t.(*Optional).Inner
	}
	switch t := t.(type) {
	case *List:
		c.intIndex(e.Index)
		c.info.Types[e] = t.Elem
		return t.Elem
	case *Map:
		c.mapKey(t, e.Index)
		c.info.Types[e] = t.Val
		return t.Val
	}
	if t == String {
		c.value(e.Index)
		d := c.errorf(e.Sp, "strings can't be changed")
		d.Help = []string{"build a new String instead, for example with `+`, slices or `replace`"}
		return nil
	}
	c.value(e.Index)
	if !silent(t) {
		d := c.errorf(e.X.Span(), "cannot index %s", t)
		d.Notes = []string{"only lists, maps and strings can be indexed with `[ ]`"}
	}
	return nil
}

// assignIndex checks `x[i] = value`.
func (c *checker) assignIndex(target *ast.Index, value ast.Expr) {
	want := c.indexTarget(target)
	t := c.valueFor(value, want)
	if want != nil && !assignable(t, want) && !c.nullMismatch(value, t, want) {
		what := "an element"
		if _, ok := nonNull(c.info.Types[target.X]).(*Map); ok {
			what = "a value"
		}
		c.errorf(value.Span(), "cannot assign %s to %s of %s, which holds %s", t, what, nonNull(c.info.Types[target.X]), want)
	}
}

// printable reports whether `print` and `String(...)` take a value of
// type t.
func printable(t Type) bool {
	switch t := t.(type) {
	case *Enum:
		return true
	case *List:
		return printable(t.Elem)
	case *Map:
		return printable(t.Key) && printable(t.Val)
	}
	return t == Int || t == Float || t == Bool || t == String
}

// ---------------------------------------------------------------------
// Built-in methods

// method is a built-in method of a list, a map or a String.
type method struct {
	recv   string // `List`, for messages such as "`List.append` is built in"
	name   string
	params []string
	types  []Type
	result Type
}

// Signature returns the method as it would be declared:
// `func insert(i: Int, x: Int)`.
func (m *method) Signature() string {
	var ps []string
	for i, p := range m.params {
		ps = append(ps, p+": "+m.types[i].String())
	}
	s := "func " + m.name + "(" + strings.Join(ps, ", ") + ")"
	if m.result != NoValue {
		s += " -> " + m.result.String()
	}
	return s
}

func listMethods(l *List) map[string]*method {
	t := l.Elem
	ms := []*method{
		{name: "length", result: Int},
		{name: "append", params: []string{"x"}, types: []Type{t}, result: NoValue},
		{name: "insert", params: []string{"i", "x"}, types: []Type{Int, t}, result: NoValue},
		{name: "remove_at", params: []string{"i"}, types: []Type{Int}, result: &Optional{Inner: t}},
		{name: "pop", result: &Optional{Inner: t}},
		{name: "contains", params: []string{"x"}, types: []Type{t}, result: Bool},
		{name: "index_of", params: []string{"x"}, types: []Type{t}, result: &Optional{Inner: Int}},
		{name: "reverse", result: NoValue},
		{name: "sort", result: NoValue},
		{name: "join", params: []string{"sep"}, types: []Type{String}, result: String},
	}
	byName := map[string]*method{}
	for _, m := range ms {
		m.recv = "List"
		byName[m.name] = m
	}
	return byName
}

// listMethod checks a call to a method of a list.
func (c *checker) listMethod(e *ast.Call, m *ast.Member, l *List) Type {
	name := m.Name.Name
	bm := listMethods(l)[name]
	if bm == nil {
		c.args(e)
		c.errorf(m.Name.Sp, "%s has no method `%s`", l, name)
		return Invalid
	}
	c.refBuiltinMethod(m.Name, bm, l)
	switch {
	case name == "sort" && l.Elem != Int && l.Elem != Float && l.Elem != String && !silent(l.Elem):
		d := c.errorf(m.Name.Sp, "`sort` works on lists of Int, Float or String, not on %s", l)
		d.Notes = []string{"it sorts with `<`, which other types don't have"}
	case name == "join" && l.Elem != String && !silent(l.Elem):
		d := c.errorf(m.Name.Sp, "`join` works on a List<String>, not on %s", l)
		d.Help = []string{"convert the elements with `String(...)` first"}
	}
	c.methodArgs(e, bm)
	return bm.result
}

// methodArgs checks the arguments of a call to a built-in method.
func (c *checker) methodArgs(e *ast.Call, bm *method) {
	note := fmt.Sprintf("`%s.%s` is built in: `%s`", bm.recv, bm.name, bm.Signature())
	if len(e.Args) != len(bm.types) {
		c.args(e)
		verb := "were"
		if len(e.Args) == 1 {
			verb = "was"
		}
		d := c.errorf(e.Sp, "`%s` takes %s, but %d %s given", bm.name, plural(len(bm.types), "argument"), len(e.Args), verb)
		d.Notes = []string{note}
		return
	}
	for i, a := range e.Args {
		t := c.valueFor(a, bm.types[i])
		if !assignable(t, bm.types[i]) && !c.nullMismatch(a, t, bm.types[i]) {
			d := c.errorf(a.Span(), "argument `%s` of `%s` must be %s, found %s", bm.params[i], bm.name, bm.types[i], t)
			d.Notes = []string{note}
		}
	}
}
