package check

import (
	"fmt"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/lexer"
)

// callName names the function of a call for messages: "`find`", or "this
// call".
func callName(e ast.Expr) string {
	if call, ok := e.(*ast.Call); ok {
		switch fn := call.Fn.(type) {
		case *ast.Ident:
			return "`" + fn.Name + "`"
		case *ast.Member:
			return "`" + fn.Name.Name + "`"
		}
	}
	return "this call"
}

// unhandled reports a call that can fail used without `else`.
func (c *checker) unhandled(e ast.Expr) {
	d := c.errorf(e.Span(), "%s can fail, and the error is not handled", callName(e))
	d.Help = []string{"add `else return`, `else <fallback>` or `else (e) { ... }`"}
}

// elseExpr checks `x else handler`, where x can fail or be null.
func (c *checker) elseExpr(e *ast.Else) Type {
	switch t := c.expr(e.X).(type) {
	case *Failable:
		e.Kind = ast.ElseError
		return c.handler(e, t.Value, t.Err)
	case *Optional:
		e.Kind = ast.ElseNull
		return c.handler(e, t.Inner, nil)
	default:
		switch {
		case t == NoValue:
			c.errorf(e.X.Span(), "%s can't fail, so there is nothing for `else` to handle", callName(e.X))
		case !silent(t):
			d := c.errorf(e.X.Span(), "`else` handles errors and null, but this is %s", t)
			d.Notes = []string{"only a call that can fail (`-> T ! E`) or a `T?` value needs `else`"}
		}
		c.handler(e, Invalid, nil)
		return Invalid
	}
}

// handler checks the part after `else`. value is the type the handled
// expression has when it doesn't fail and isn't null; errType is the error
// type, or nil when the else handles null.
func (c *checker) handler(e *ast.Else, value Type, errType *Class) Type {
	c.scope = &scope{parent: c.scope, vars: map[string]*variable{}}
	defer func() { c.scope = c.scope.parent }()

	if e.Err != nil {
		bound := Type(errType)
		if errType == nil {
			bound = Invalid
			if value != Invalid {
				d := c.errorf(e.Err.Sp, "there is no error to bind: `%s` can be null, but it can't fail", c.text(e.X))
				d.Help = []string{"remove the `(" + e.Err.Name + ")`"}
			}
		}
		c.declare(e.Err, &variable{kind: Con, typ: bound, decl: e.Err.Sp})
	}

	switch {
	case e.Body != nil:
		c.blockStmts(e.Body)
		if !terminates(e.Body) {
			d := c.errorf(e.Body.Rbrace, "this block must leave the function")
			d.Notes = []string{"end it with `return` or `panic(...)`: there is no value to continue with"}
		}

	case e.Return && e.Value == nil:
		c.bareElseReturn(e, errType, value)

	case e.Return:
		c.checkReturn(e.Value.Span(), e.Value)

	default:
		var ft Type
		if value == NoValue {
			ft = c.expr(e.Value) // the message below beats "returns nothing"
		} else {
			ft = c.value(e.Value)
		}
		switch {
		case value == NoValue:
			if ft != Never && !silent(ft) {
				d := c.errorf(e.Value.Span(), "%s returns nothing, so there is no value for a fallback to replace", callName(e.X))
				d.Help = []string{"write `else return`, `else (e) { ... }` or `else panic(...)`"}
			}
		case !assignable(ft, value) && !c.nullMismatch(e.Value, ft, value):
			c.errorf(e.Value.Span(), "the fallback is %s, but the value it replaces is %s", ft, value)
		}
	}
	return value
}

// bareElseReturn checks `else return` without a value: on an error it
// passes the error on, on null it returns nothing.
func (c *checker) bareElseReturn(e *ast.Else, errType *Class, value Type) {
	if c.fn == nil {
		c.errorf(e.Sp, "`return` can't be used in the initial value of a field")
		return
	}
	name := c.fn.name
	if errType == nil {
		if value != Invalid && hasValue(c.fn.result) {
			d := c.errorf(e.Sp, "`else return` needs a value here: `%s` returns %s", name, resultValue(c.fn.result))
			d.Help = []string{"write `else return <value>`, or give a fallback: `else <value>`"}
		}
		return
	}
	f, ok := c.fn.result.(*Failable)
	if !ok {
		d := c.errorf(e.Sp, "`else return` passes the error on, but `%s` can't fail", name)
		if hasValue(c.fn.result) {
			d.Help = []string{fmt.Sprintf("handle it here with `else <fallback>` or `else (e) { ... }`, or let `%s` fail: `-> %s ! %s`", name, c.fn.result, errType.Name)}
		} else {
			d.Help = []string{fmt.Sprintf("handle it here with `else (e) { ... }`, or let `%s` fail: `-> ! %s`", name, errType.Name)}
		}
		return
	}
	if !assignable(errType, f.Err) {
		d := c.errorf(e.Sp, "`else return` would fail with %s, but `%s` fails with %s", errType.Name, name, f.Err.Name)
		d.Notes = []string{fmt.Sprintf("%s doesn't use %s", errType.Name, f.Err.Name)}
	}
}

func resultValue(t Type) Type {
	if f, ok := t.(*Failable); ok {
		return f.Value
	}
	return t
}

// isExpr checks `x is C`.
func (c *checker) isExpr(e *ast.Is) Type {
	t := c.value(e.X)
	switch {
	case c.mayBeNull(e.X, t):
	case !silent(t) && !c.isError(t):
		c.errorf(e.X.Span(), "`is` checks which kind of error this is, but it is %s", t)
	}
	name := e.Class.Name
	switch cls := c.classes[name]; {
	case cls != nil && c.isError(cls):
	case cls != nil:
		d := c.errorf(e.Class.Sp, "`%s` is not an error class", name)
		d.Notes = []string{"`is` works only with `Error` and classes that use it"}
	case c.broken[name]:
	default:
		c.errorf(e.Class.Sp, "unknown class `%s`", name)
	}
	return Bool
}

// ---------------------------------------------------------------------
// Null

// mayBeNull reports "`x` may be null" if t is a `T?`, and says whether it
// did.
func (c *checker) mayBeNull(e ast.Expr, t Type) bool {
	o, ok := t.(*Optional)
	if !ok {
		return false
	}
	text := c.text(e)
	d := c.errorf(e.Span(), "`%s` may be null", text)
	switch x := e.(type) {
	case *ast.Ident:
		if v := c.scope.lookup(x.Name); v != nil && v.kind == Mut {
			d.Notes = []string{fmt.Sprintf("`%s` is `mut`, so `if %s != null` doesn't make it non-null", text, text)}
			d.Help = []string{fmt.Sprintf("copy it into a `con` first, or give a fallback: `%s else ...`", text)}
		} else {
			d.Help = []string{fmt.Sprintf("check it first with `if %s != null { ... }`, or give a fallback: `%s else ...`", text, text)}
		}
	case *ast.Member:
		d.Notes = []string{"fields are not narrowed by `if`"}
		d.Help = []string{fmt.Sprintf("copy it into a `con` first: `con value = %s`, then check that", text)}
	default:
		d.Help = []string{fmt.Sprintf("give a fallback with `else`, or check it first; its type is %s", o)}
	}
	return true
}

// nullMismatch reports "may be null" when a `T?` is used where a T is
// needed, and says whether it did.
func (c *checker) nullMismatch(e ast.Expr, t, want Type) bool {
	o, ok := t.(*Optional)
	if !ok || !assignable(o.Inner, want) {
		return false
	}
	return c.mayBeNull(e, t)
}

// nullNeedsType reports a variable or field whose only clue to its type
// is `null`.
func (c *checker) nullNeedsType(name *ast.Ident, value ast.Expr, mut bool) {
	kw := "con"
	if mut {
		kw = "mut"
	}
	d := c.errorf(value.Span(), "`null` needs a type here")
	d.Help = []string{fmt.Sprintf("write the type: `%s %s: Type? = null`", kw, name.Name)}
}

// ifStmt checks an if statement, and narrows `x` from `T?` to T where a
// `x != null` or `x == null` condition proves it is not null.
func (c *checker) ifStmt(s *ast.If) {
	c.cond(s.Cond)
	name, narrowed, notNull := c.nullCheck(s.Cond)

	in := func(when bool, check func()) {
		if narrowed == nil || notNull != when {
			check()
			return
		}
		c.scope = &scope{parent: c.scope, vars: map[string]*variable{name: narrowed}}
		check()
		c.scope = c.scope.parent
	}
	in(true, func() { c.block(s.Then) })
	in(false, func() {
		switch e := s.Else.(type) {
		case *ast.Block:
			c.block(e)
		case *ast.If:
			c.stmt(e)
		}
	})

	// `if x == null { return }`: x is not null for the rest of the block.
	if narrowed != nil && !notNull && terminates(s.Then) {
		c.scope.vars[name] = narrowed
	}
}

// nullCheck looks for `x != null` or `x == null`, where x is a `con`
// variable or a parameter of type `T?`. It returns x's name, a copy of x
// with type T, and whether the condition is true when x is not null.
func (c *checker) nullCheck(cond ast.Expr) (string, *variable, bool) {
	b, ok := cond.(*ast.Binary)
	if !ok || (b.Op != lexer.EQ && b.Op != lexer.NE) {
		return "", nil, false
	}
	x, y := b.X, b.Y
	if _, ok := x.(*ast.NullLit); ok {
		x, y = y, x
	}
	id, ok := x.(*ast.Ident)
	if _, isNull := y.(*ast.NullLit); !ok || !isNull {
		return "", nil, false
	}
	v := c.scope.lookup(id.Name)
	if v == nil || v.kind == Mut {
		return "", nil, false
	}
	o, ok := v.typ.(*Optional)
	if !ok {
		return "", nil, false
	}
	narrowed := *v
	narrowed.typ = o.Inner
	return id.Name, &narrowed, b.Op == lexer.NE
}
