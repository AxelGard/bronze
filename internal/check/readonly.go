package check

import (
	"fmt"

	"github.com/AxelGard/bronze/internal/ast"
)

// Read only
//
// A plain parameter is read only: the function may look at the object,
// list or map it was given, but not change it. `a: mut T` may change it,
// and so may a `mut func` change `this`. Read only goes all the way down:
// what is read out of a read-only value is read only too. It is checked
// here only, and costs nothing when the program runs.

// changeable reports whether values of type t can be changed: objects,
// lists and maps. Int, Float, Bool, String and enums can't, so for them
// read only means nothing.
func changeable(t Type) bool {
	switch t := t.(type) {
	case *Class, *Interface, *List, *Map:
		return true
	case *Optional:
		return changeable(t.Inner)
	}
	return false
}

// Where a read-only value comes from. A method that isn't a `mut func`
// may return what it read out of `this`: its result is then read only
// where the object it is called on is. Anything else read only can't be
// returned.
const (
	roThis  = 1 << iota // read out of a read-only `this`
	roOther             // read out of a read-only parameter
)

// readOnly reports whether the value of e, which has been checked, is read
// only and could otherwise be changed.
func (c *checker) readOnly(e ast.Expr) bool { return c.roFlags(e) != 0 }

// roFlags says where the value of e comes from, if it is read only and
// could otherwise be changed; 0 otherwise.
func (c *checker) roFlags(e ast.Expr) int {
	if !changeable(c.info.Types[e]) {
		return 0
	}
	return c.roSource(e)
}

// readOnlySource reports whether e is read out of a read-only value,
// whatever its own type.
func (c *checker) readOnlySource(e ast.Expr) bool { return c.roSource(e) != 0 }

func (c *checker) roSource(e ast.Expr) int {
	switch e := e.(type) {
	case *ast.Paren:
		return c.roSource(e.X)
	case *ast.Ident:
		if v := c.scope.lookup(e.Name); v != nil {
			return v.ro
		}
	case *ast.This:
		if c.thisReadOnly() {
			return roThis
		}
	case *ast.Member:
		return c.roSource(e.X)
	case *ast.Index:
		return c.roSource(e.X)
	case *ast.Slice:
		// A slice is a new list; only its elements are the old list's.
		if l, ok := nonNull(c.info.Types[e]).(*List); ok && changeable(l.Elem) {
			return c.roSource(e.X)
		}
	case *ast.Call:
		return c.roCall(e)
	case *ast.Else:
		ro := c.roSource(e.X)
		if e.Value != nil && !e.Return && e.Body == nil {
			ro |= c.roFlags(e.Value)
		}
		return ro
	case *ast.Match:
		ro := 0
		for _, a := range e.Arms {
			if a.Value != nil {
				ro |= c.roFlags(a.Value)
			}
		}
		return ro
	case *ast.ListLit:
		ro := 0
		for _, x := range e.Elems {
			ro |= c.roFlags(x)
		}
		return ro
	case *ast.MapLit:
		ro := 0
		for _, en := range e.Entries {
			ro |= c.roFlags(en.Key) | c.roFlags(en.Value)
		}
		return ro
	}
	return 0
}

// roCall says whether the result of a call is read only. keys() and
// values() give new lists, but their elements are the map's own. A method
// that isn't a `mut func` gives a result as read only as its object.
func (c *checker) roCall(e *ast.Call) int {
	m, ok := e.Fn.(*ast.Member)
	if !ok {
		return 0
	}
	switch t := nonNull(c.info.Types[m.X]).(type) {
	case *Map:
		if l, ok := c.info.Types[e].(*List); ok && changeable(l.Elem) && (m.Name.Name == "keys" || m.Name.Name == "values") {
			return c.roSource(m.X)
		}
	case *Class:
		if sig := t.Methods[m.Name.Name]; sig != nil && !sig.mut {
			return c.roSource(m.X)
		}
	case *Interface:
		if sig := t.Methods[m.Name.Name]; sig != nil && !sig.mut {
			return c.roSource(m.X)
		}
	}
	return 0
}

// thisReadOnly reports whether `this` is read only in the function being
// checked: in a method of a class that isn't a `mut func`.
func (c *checker) thisReadOnly() bool {
	return c.fn != nil && c.fn.class != nil && !c.fn.isInit && !c.fn.mut && !c.fn.static
}

// readOnlyRoot returns the variable or `this` that a read-only value was
// read out of.
func readOnlyRoot(e ast.Expr) ast.Expr {
	for {
		switch x := e.(type) {
		case *ast.Paren:
			e = x.X
		case *ast.Member:
			e = x.X
		case *ast.Index:
			e = x.X
		case *ast.Slice:
			e = x.X
		case *ast.Call:
			m, ok := x.Fn.(*ast.Member)
			if !ok {
				return e
			}
			e = m.X
		default:
			return e
		}
	}
}

// readOnlyHelp says how to make the value that e was read out of
// changeable.
func (c *checker) readOnlyHelp(e ast.Expr) []string {
	switch root := readOnlyRoot(e).(type) {
	case *ast.This:
		if c.fn != nil && c.fn.decl != nil {
			return []string{fmt.Sprintf("make it a `mut func %s`, so that it may change `this`", c.fn.name)}
		}
	case *ast.Ident:
		v := c.scope.lookup(root.Name)
		if v == nil {
			return nil
		}
		if v.kind == Param {
			return []string{fmt.Sprintf("declare the parameter as `%s: mut %s`", root.Name, v.typ)}
		}
		return []string{fmt.Sprintf("`%s` is read only because the value it got at line %d is", root.Name, v.decl.Start.Line)}
	}
	return nil
}

// changes checks that x, the value a change is made to, is not read only.
// what says what the change is, for the label.
func (c *checker) changes(x ast.Expr, what string) bool {
	if !c.readOnlySource(x) {
		return true
	}
	d := c.errorf(x.Span(), "`%s` is read only", c.text(x))
	if what != "" {
		d.Label = what
	} else if t := c.info.Types[x]; t != nil && !silent(t) {
		d.Label = fmt.Sprintf("%s is a read-only %s", c.text(x), t)
	}
	d.Help = c.readOnlyHelp(x)
	return false
}

// stores checks that a value stored where it could be changed later (a
// field, an element, a changeable variable, a function's result) is not
// read only.
func (c *checker) stores(value ast.Expr, where string) {
	if !c.readOnly(value) {
		return
	}
	c.storeError(value, where)
}

// returns checks that a returned value is not read only, except for what
// a method that isn't a `mut func` read out of `this`.
func (c *checker) returns(value ast.Expr) {
	if ro := c.roFlags(value); ro&roOther != 0 || ro != 0 && !c.thisReadOnly() {
		c.storeError(value, "returned")
	}
}

func (c *checker) storeError(value ast.Expr, where string) {
	d := c.errorf(value.Span(), "`%s` is read only, and can't be %s", c.text(value), where)
	d.Notes = []string{"what is stored or returned could be changed later, through the place it went"}
	d.Help = c.readOnlyHelp(value)
}

// checkModes checks the `mut` of parameters, and of a `mut func`.
func (c *checker) checkModes(sig *funcSig) {
	fn := sig.decl
	for i, p := range fn.Params {
		if p.Mut && !changeable(sig.params[i]) && !silent(sig.params[i]) {
			d := c.errorf(p.MutSp, "`mut` does nothing on %s, which can't be changed", sig.params[i])
			d.Help = []string{fmt.Sprintf("remove it: `%s: %s`", p.Name.Name, c.text(p.Type))}
		}
	}
	c.checkStatic(sig)
	if !fn.Mut || fn.Static {
		return
	}
	switch {
	case sig.isInit:
		d := c.errorf(fn.MutSp, "`init` can't be a `mut func`")
		d.Notes = []string{"`init` may always change `this`"}
	case sig.enum != nil:
		d := c.errorf(fn.MutSp, "an enum's methods can't be `mut func`")
		d.Notes = []string{"an enum case can't change"}
	case sig.class == nil:
		d := c.errorf(fn.MutSp, "only methods can be `mut func`")
		d.Notes = []string{"`mut func` marks a method that changes `this`"}
	}
}

// checkArgModes checks that the arguments for `mut` parameters are not
// read only.
func (c *checker) checkArgModes(e *ast.Call, name string, sig *funcSig) {
	if sig.decl == nil || len(e.Args) != len(sig.decl.Params) {
		return
	}
	for i, p := range sig.decl.Params {
		a := e.Args[i]
		if !p.Mut || !c.readOnly(a) {
			continue
		}
		d := c.errorf(a.Span(), "`%s` is read only, but `%s` may change it", c.text(a), name)
		d.Notes = []string{fmt.Sprintf("parameter `%s` is declared as `mut %s`", p.Name.Name, c.text(p.Type))}
		d.Help = c.readOnlyHelp(a)
	}
}

// checkStatic checks where `static` is written.
func (c *checker) checkStatic(sig *funcSig) {
	fn := sig.decl
	if !fn.Static {
		return
	}
	switch {
	case sig.isInit:
		d := c.errorf(fn.StaticSp, "`init` can't be static")
		d.Notes = []string{"`init` sets up a new object; a static method that creates one calls the class, like `" + sig.class.Name + "(...)`"}
	case sig.enum != nil:
		c.errorf(fn.StaticSp, "an enum's methods can't be static")
	case sig.class == nil:
		d := c.errorf(fn.StaticSp, "only methods can be static")
		d.Notes = []string{"a function outside a class already has no `this`"}
	case fn.Mut:
		d := c.errorf(fn.MutSp, "a static method can't be a `mut func`")
		d.Notes = []string{"it has no `this` to change"}
	}
}
