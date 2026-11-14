package check

import (
	"fmt"
	"strconv"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/lexer"
)

// validKey reports whether t can be the key type of a map.
func validKey(t Type) bool {
	_, isEnum := t.(*Enum)
	return t == Int || t == String || t == Bool || isEnum
}

// mapType resolves `Map<K, V>`.
func (c *checker) mapType(t *ast.TypeExpr) Type {
	if len(t.Args) != 2 {
		d := c.errorf(t.Sp, "`Map` needs two type arguments, the types of its keys and values")
		d.Help = []string{"write it like `Map<String, Int>`"}
		return Invalid
	}
	key, val := c.resolveType(t.Args[0]), c.resolveType(t.Args[1])
	okKey := c.elemType(t.Args[0], key, "a map's keys")
	if okKey && !validKey(key) {
		c.badKey(t.Args[0], key)
		okKey = false
	}
	if !c.elemType(t.Args[1], val, "a map's values") || !okKey {
		return Invalid
	}
	return &Map{Key: key, Val: val}
}

func (c *checker) badKey(at ast.Node, t Type) {
	d := c.errorf(at.Span(), "a map key can be an Int, String, Bool or enum, not %s", t)
	d.Notes = []string{"keys are compared by value, so they can only be simple values"}
}

func (c *checker) mapLit(m *ast.MapLit, want Type) Type {
	if wm, ok := nonNull(want).(*Map); ok {
		for _, en := range m.Entries {
			if t := c.valueFor(en.Key, wm.Key); !assignable(t, wm.Key) && !c.nullMismatch(en.Key, t, wm.Key) {
				c.errorf(en.Key.Span(), "this key is %s, but the map's keys are %s", t, wm.Key)
			}
			if t := c.valueFor(en.Value, wm.Val); !assignable(t, wm.Val) && !c.nullMismatch(en.Value, t, wm.Val) {
				c.errorf(en.Value.Span(), "this value is %s, but the map's values are %s", t, wm.Val)
			}
		}
		c.duplicateKeys(m)
		return wm
	}
	if len(m.Entries) == 0 {
		d := c.errorf(m.Sp, "`{}` needs a type here")
		d.Help = []string{"write the type where it is declared, like `con ages: Map<String, Int> = {}`"}
		return Invalid
	}
	var keys, vals []ast.Expr
	for _, en := range m.Entries {
		keys = append(keys, en.Key)
		vals = append(vals, en.Value)
	}
	key := c.sameTypes(keys, "map keys", "a map")
	val := c.sameTypes(vals, "map values", "a map")
	if key != nil && !validKey(key) {
		c.badKey(m.Entries[0].Key, key)
		key = nil
	}
	c.duplicateKeys(m)
	if key == nil || val == nil {
		return Invalid
	}
	return &Map{Key: key, Val: val}
}

// sameTypes checks the elements of a list literal, or the keys or values
// of a map literal, without an expected type: they must all have the same
// type, which it returns, or nil if there is none.
func (c *checker) sameTypes(es []ast.Expr, what, holder string) Type {
	var typ Type // the type so far
	mixed := false
	for _, e := range es {
		t := c.valueFor(e, typ)
		switch {
		case silent(t) || c.mayBeNull(e, t):
			continue
		case t == Null:
			d := c.errorf(e.Span(), "%s can't hold null", holder)
			if holder == "a list" {
				d.Notes = []string{"its elements can't be a `?` type"}
			} else {
				d.Notes = []string{"its keys and values can't be `?` types"}
			}
			continue
		case typ == nil:
			typ = t
		case !identical(t, typ) && !mixed:
			d := c.errorf(e.Span(), "%s have different types: %s and %s", what, typ, t)
			d.Help = []string{"if they share a type, write it where the value is declared, like `con xs: List<Error> = [...]`"}
			mixed = true
		}
	}
	return typ
}

// duplicateKeys reports a key that is written twice in a map literal,
// when both are literals or enum cases.
func (c *checker) duplicateKeys(m *ast.MapLit) {
	seen := map[string]ast.Expr{}
	for _, en := range m.Entries {
		k, ok := c.constKey(en.Key)
		if !ok {
			continue
		}
		if prev, ok := seen[k]; ok {
			d := c.errorf(en.Key.Span(), "the key %s is written twice", c.text(en.Key))
			d.Notes = []string{fmt.Sprintf("it is first written at line %d", prev.Span().Start.Line)}
			continue
		}
		seen[k] = en.Key
	}
}

// constKey returns a key that is a literal or an enum case as a string
// that identifies it.
func (c *checker) constKey(e ast.Expr) (string, bool) {
	switch e := e.(type) {
	case *ast.IntLit:
		return "i" + strconv.FormatInt(e.Value, 10), true
	case *ast.StringLit:
		return "s" + e.Value, true
	case *ast.BoolLit:
		return "b" + strconv.FormatBool(e.Value), true
	case *ast.Unary:
		if n, ok := e.X.(*ast.IntLit); ok && e.Op == lexer.MINUS {
			return "i-" + strconv.FormatInt(n.Value, 10), true
		}
	case *ast.Member:
		if en := c.enumName(e.X); en != nil {
			return "e" + en.Name + "." + e.Name.Name, true
		}
	case *ast.Paren:
		return c.constKey(e.X)
	}
	return "", false
}

// mapKey checks the key in `m[k]`.
func (c *checker) mapKey(m *Map, k ast.Expr) {
	t := c.valueFor(k, m.Key)
	if !assignable(t, m.Key) && !c.nullMismatch(k, t, m.Key) {
		c.errorf(k.Span(), "the keys of %s are %s, but this is %s", m, m.Key, t)
	}
}

func mapMethods(m *Map) map[string]*method {
	ms := []*method{
		{name: "length", result: Int},
		{name: "contains", params: []string{"key"}, types: []Type{m.Key}, result: Bool},
		{name: "remove", params: []string{"key"}, types: []Type{m.Key}, result: &Optional{Inner: m.Val}},
		{name: "keys", result: &List{Elem: m.Key}},
		{name: "values", result: &List{Elem: m.Val}},
	}
	byName := map[string]*method{}
	for _, bm := range ms {
		bm.recv = "Map"
		byName[bm.name] = bm
	}
	return byName
}

// mapMethod checks a call to a method of a map.
func (c *checker) mapMethod(e *ast.Call, m *ast.Member, t *Map) Type {
	bm := mapMethods(t)[m.Name.Name]
	if bm == nil {
		c.args(e)
		c.errorf(m.Name.Sp, "%s has no method `%s`", t, m.Name.Name)
		return Invalid
	}
	c.methodArgs(e, bm)
	return bm.result
}
