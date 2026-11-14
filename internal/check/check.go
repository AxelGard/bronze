package check

import (
	"fmt"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/diag"
	"github.com/AxelGard/bronze/internal/lexer"
)

// Info is what the checker learned about the program.
type Info struct {
	Types map[ast.Expr]Type // the type of every checked expression
}

// Check checks a parsed file and returns all errors it finds. Bodies of
// functions with syntax errors are not checked, to avoid follow-up errors.
func Check(file *ast.File, src *diag.Source) (*Info, diag.List) {
	c := &checker{
		src:   src,
		info:  &Info{Types: map[ast.Expr]Type{}},
		funcs: map[string]*funcSig{},
	}
	c.collect(file)
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if sig := c.funcs[fn.Name.Name]; sig != nil && sig.decl == fn && !hasSyntaxError(fn) {
			c.funcBody(sig)
		}
	}
	return c.info, c.diags
}

type checker struct {
	src   *diag.Source
	diags diag.List
	info  *Info
	funcs map[string]*funcSig

	fn    *funcSig // the function being checked
	scope *scope
}

type funcSig struct {
	decl   *ast.FuncDecl
	params []Type
	result Type // NoValue if the function has no `->`
}

// VarKind is how a variable was declared.
type VarKind int

const (
	Con VarKind = iota
	Mut
	Param
)

type variable struct {
	kind VarKind
	typ  Type
	decl diag.Span
}

type scope struct {
	parent *scope
	vars   map[string]*variable
}

func (s *scope) lookup(name string) *variable {
	for ; s != nil; s = s.parent {
		if v, ok := s.vars[name]; ok {
			return v
		}
	}
	return nil
}

// builtinFuncs are the built-in functions. `Float` and `String` are also
// type names; they are callable as conversions.
var builtinFuncs = map[string]bool{
	"print": true, "Float": true, "String": true, "Int": true, "Bool": true,
	"panic": true, "assert": true, "Error": true,
}

func (c *checker) errorf(span diag.Span, format string, args ...any) *diag.Diagnostic {
	return c.diags.Errorf(span, format, args...)
}

// text returns the source text of a node.
func (c *checker) text(n ast.Node) string {
	s := n.Span()
	return string(c.src.Text[s.Start.Offset:s.End.Offset])
}

// ---------------------------------------------------------------------
// Declarations

// collect records every top-level declaration, so they can be used before
// they are declared.
func (c *checker) collect(file *ast.File) {
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := fn.Name.Name
		if prev, ok := c.funcs[name]; ok {
			c.errorf(fn.Name.Sp, "`%s` is declared twice", name).Notes = []string{
				fmt.Sprintf("the first declaration is at line %d", prev.decl.Name.Sp.Start.Line),
			}
			continue
		}
		if builtinFuncs[name] || builtinTypes[name] != nil {
			c.errorf(fn.Name.Sp, "`%s` is a built-in name and can't be declared again", name)
			continue
		}
		sig := &funcSig{decl: fn, result: NoValue}
		for _, p := range fn.Params {
			sig.params = append(sig.params, c.resolveType(p.Type))
		}
		if fn.Result != nil {
			sig.result = c.resolveType(fn.Result)
		}
		c.funcs[name] = sig
	}

	main, ok := c.funcs["main"]
	switch {
	case !ok:
		d := c.errorf(diag.Span{Start: file.Sp.Start, End: file.Sp.Start}, "the program has no `main` function")
		d.Help = []string{"add `func main() -> Int`; its return value is the exit code"}
	case len(main.params) > 0 || main.result != Int:
		d := c.errorf(main.decl.Name.Sp, "`main` must be declared as `func main() -> Int`")
		d.Notes = []string{"`main` takes no parameters, and its return value is the exit code"}
	}
}

func (c *checker) resolveType(t *ast.TypeExpr) Type {
	if bt, ok := builtinTypes[t.Name.Name]; ok {
		return bt
	}
	if t.Name.Name == "Error" {
		c.errorf(t.Sp, "the `Error` type is not supported yet")
		return Invalid
	}
	c.errorf(t.Sp, "unknown type `%s`", t.Name.Name)
	return Invalid
}

func (c *checker) funcBody(sig *funcSig) {
	c.fn = sig
	c.scope = &scope{vars: map[string]*variable{}}
	for i, p := range sig.decl.Params {
		c.declare(p.Name, &variable{kind: Param, typ: sig.params[i], decl: p.Name.Sp})
	}
	c.blockStmts(sig.decl.Body)
	if sig.result != NoValue && !terminates(sig.decl.Body) {
		d := c.errorf(sig.decl.Body.Rbrace, "missing return at the end of `%s`", sig.decl.Name.Name)
		d.Notes = []string{fmt.Sprintf("`%s` returns %s, so every path must end with `return`", sig.decl.Name.Name, sig.result)}
	}
	c.fn, c.scope = nil, nil
}

// declare adds a variable to the current scope. Names must not hide
// anything: not a variable in this or an outer scope, a parameter, a
// function or a built-in.
func (c *checker) declare(name *ast.Ident, v *variable) {
	n := name.Name
	if prev := c.scope.lookup(n); prev != nil {
		what := "a variable"
		if prev.kind == Param {
			what = "a parameter"
		}
		d := c.errorf(name.Sp, "`%s` is already %s", n, what)
		d.Notes = []string{fmt.Sprintf("it is declared at line %d", prev.decl.Start.Line)}
		d.Help = []string{"choose a different name"}
		return
	}
	if f, ok := c.funcs[n]; ok {
		d := c.errorf(name.Sp, "`%s` is already the name of a function", n)
		d.Notes = []string{fmt.Sprintf("the function is declared at line %d", f.decl.Name.Sp.Start.Line)}
		return
	}
	if builtinFuncs[n] || builtinTypes[n] != nil {
		c.errorf(name.Sp, "`%s` is a built-in name and can't be used for a variable", n)
		return
	}
	c.scope.vars[n] = v
}

// terminates reports whether a statement always leaves the function: it is
// a `return`, a `panic(...)`, a block ending in one, or an `if` with an
// `else` where every branch does.
func terminates(s ast.Stmt) bool {
	switch s := s.(type) {
	case *ast.Return:
		return true
	case *ast.ExprStmt:
		return isPanicCall(s.X)
	case *ast.Block:
		return len(s.Stmts) > 0 && terminates(s.Stmts[len(s.Stmts)-1])
	case *ast.If:
		return s.Else != nil && terminates(s.Then) && terminates(s.Else)
	case *ast.BadStmt:
		return true
	}
	return false
}

func isPanicCall(e ast.Expr) bool {
	call, ok := e.(*ast.Call)
	if !ok {
		return false
	}
	id, ok := call.Fn.(*ast.Ident)
	return ok && id.Name == "panic"
}

func hasSyntaxError(fn *ast.FuncDecl) bool {
	found := false
	var walkStmt func(s ast.Stmt)
	walkStmt = func(s ast.Stmt) {
		switch s := s.(type) {
		case *ast.BadStmt:
			found = true
		case *ast.Block:
			for _, s := range s.Stmts {
				walkStmt(s)
			}
		case *ast.If:
			walkStmt(s.Then)
			if s.Else != nil {
				walkStmt(s.Else)
			}
		case *ast.While:
			walkStmt(s.Body)
		}
	}
	walkStmt(fn.Body)
	return found
}

// ---------------------------------------------------------------------
// Statements

func (c *checker) block(b *ast.Block) {
	c.scope = &scope{parent: c.scope, vars: map[string]*variable{}}
	c.blockStmts(b)
	c.scope = c.scope.parent
}

func (c *checker) blockStmts(b *ast.Block) {
	for _, s := range b.Stmts {
		c.stmt(s)
	}
}

func (c *checker) stmt(s ast.Stmt) {
	switch s := s.(type) {
	case *ast.VarDecl:
		t := c.value(s.Value)
		if s.Type != nil {
			want := c.resolveType(s.Type)
			if !assignable(t, want) {
				c.errorf(s.Value.Span(), "`%s` is declared as %s, but the value is %s", s.Name.Name, want, t)
			}
			t = want
		}
		if t == Never {
			t = Invalid
		}
		kind := Con
		if s.Mut {
			kind = Mut
		}
		c.declare(s.Name, &variable{kind: kind, typ: t, decl: s.Name.Sp})

	case *ast.Assign:
		c.assign(s)

	case *ast.If:
		c.cond(s.Cond)
		c.block(s.Then)
		switch e := s.Else.(type) {
		case *ast.Block:
			c.block(e)
		case *ast.If:
			c.stmt(e)
		}

	case *ast.While:
		c.cond(s.Cond)
		c.block(s.Body)

	case *ast.Return:
		c.returnStmt(s)

	case *ast.ExprStmt:
		c.expr(s.X)

	case *ast.Block:
		c.block(s)

	case *ast.BadStmt:
	}
}

func (c *checker) assign(s *ast.Assign) {
	id := s.Target.(*ast.Ident)
	t := c.value(s.Value)
	v := c.scope.lookup(id.Name)
	if v == nil {
		if _, ok := c.funcs[id.Name]; ok {
			c.errorf(id.Sp, "cannot assign to `%s`: it is a function", id.Name)
			return
		}
		d := c.errorf(id.Sp, "cannot assign to `%s`: it is not declared", id.Name)
		d.Help = []string{fmt.Sprintf("declare it with `mut %s = ...`", id.Name)}
		return
	}
	switch v.kind {
	case Con:
		d := c.errorf(id.Sp, "cannot assign to `%s`: it is declared with `con`", id.Name)
		d.Notes = []string{fmt.Sprintf("`%s` is declared at line %d", id.Name, v.decl.Start.Line)}
		d.Help = []string{"declare it with `mut` if it needs to change"}
		return
	case Param:
		d := c.errorf(id.Sp, "cannot assign to `%s`: parameters can't be changed", id.Name)
		d.Help = []string{fmt.Sprintf("copy it into a variable first: `mut %s_ = %s`", id.Name, id.Name)}
		return
	}
	if !assignable(t, v.typ) {
		c.errorf(s.Value.Span(), "cannot assign %s to `%s`, which is %s", t, id.Name, v.typ)
	}
}

func (c *checker) cond(e ast.Expr) {
	t := c.value(e)
	if !assignable(t, Bool) {
		c.errorf(e.Span(), "condition must be Bool, found %s", t)
	}
}

func (c *checker) returnStmt(s *ast.Return) {
	name := c.fn.decl.Name.Name
	want := c.fn.result
	if s.Value == nil {
		if want != NoValue {
			c.errorf(s.Sp, "`%s` must return %s, but this `return` has no value", name, want)
		}
		return
	}
	t := c.value(s.Value)
	if want == NoValue {
		if !silent(t) {
			d := c.errorf(s.Value.Span(), "`%s` returns nothing, so `return` can't have a value", name)
			d.Help = []string{fmt.Sprintf("add a return type: `func %s(...) -> %s`", name, t)}
		}
		return
	}
	if !assignable(t, want) {
		c.errorf(s.Value.Span(), "`%s` returns %s, but this is %s", name, want, t)
	}
}

// ---------------------------------------------------------------------
// Expressions

// value checks an expression whose value is used.
func (c *checker) value(e ast.Expr) Type {
	t := c.expr(e)
	if t == NoValue {
		name := "this call"
		if call, ok := e.(*ast.Call); ok {
			if id, ok := call.Fn.(*ast.Ident); ok {
				name = "`" + id.Name + "`"
			}
		}
		c.errorf(e.Span(), "%s returns nothing, so it can't be used as a value", name)
		return Invalid
	}
	return t
}

func (c *checker) expr(e ast.Expr) Type {
	t := c.expr1(e)
	c.info.Types[e] = t
	return t
}

func (c *checker) expr1(e ast.Expr) Type {
	switch e := e.(type) {
	case *ast.IntLit:
		return Int
	case *ast.FloatLit:
		return Float
	case *ast.StringLit:
		return String
	case *ast.BoolLit:
		return Bool
	case *ast.Paren:
		return c.value(e.X)
	case *ast.Ident:
		return c.ident(e)
	case *ast.Unary:
		return c.unary(e)
	case *ast.Binary:
		return c.binary(e)
	case *ast.Call:
		return c.call(e)
	case *ast.BadExpr:
		return Invalid
	}
	panic(fmt.Sprintf("check: unexpected expression %T", e))
}

func (c *checker) ident(e *ast.Ident) Type {
	if v := c.scope.lookup(e.Name); v != nil {
		return v.typ
	}
	if _, ok := c.funcs[e.Name]; ok {
		d := c.errorf(e.Sp, "`%s` is a function, and functions can't be used as values yet", e.Name)
		d.Help = []string{fmt.Sprintf("to call it, write `%s(...)`", e.Name)}
		return Invalid
	}
	if builtinTypes[e.Name] != nil {
		c.errorf(e.Sp, "`%s` is a type, not a value", e.Name)
		return Invalid
	}
	if builtinFuncs[e.Name] {
		c.errorf(e.Sp, "`%s` is a built-in function, and functions can't be used as values yet", e.Name)
		return Invalid
	}
	c.errorf(e.Sp, "unknown name `%s`", e.Name)
	return Invalid
}

func (c *checker) unary(e *ast.Unary) Type {
	t := c.value(e.X)
	if silent(t) {
		return Invalid
	}
	switch e.Op {
	case lexer.MINUS:
		if isNumeric(t) {
			return t
		}
		c.errorf(e.Sp, "cannot negate %s", t)
	case lexer.NOT:
		if t == Bool {
			return Bool
		}
		c.errorf(e.Sp, "`not` needs a Bool, found %s", t)
	}
	return Invalid
}

var arithVerbs = map[lexer.Kind]string{
	lexer.PLUS:    "add %s and %s",
	lexer.MINUS:   "subtract %[2]s from %[1]s",
	lexer.STAR:    "multiply %s by %s",
	lexer.SLASH:   "divide %s by %s",
	lexer.PERCENT: "take the remainder of %s by %s",
}

func opText(k lexer.Kind) string {
	s := k.String()
	return s[1 : len(s)-1]
}

func (c *checker) binary(e *ast.Binary) Type {
	if e.Op == lexer.AND || e.Op == lexer.OR {
		x, y := c.value(e.X), c.value(e.Y)
		for _, side := range []struct {
			t Type
			e ast.Expr
		}{{x, e.X}, {y, e.Y}} {
			if !assignable(side.t, Bool) {
				c.errorf(side.e.Span(), "`%s` needs Bool on both sides, found %s", opText(e.Op), side.t)
			}
		}
		return Bool
	}

	x, y := c.value(e.X), c.value(e.Y)
	if silent(x) || silent(y) {
		if isCmp(e.Op) {
			return Bool
		}
		return Invalid
	}
	label := fmt.Sprintf("%s %s %s", x, opText(e.Op), y)

	switch e.Op {
	case lexer.EQ, lexer.NE:
		if x == y {
			return Bool
		}
		d := c.errorf(e.Span(), "cannot compare %s and %s with `%s`", x, y, opText(e.Op))
		d.Label = label
		c.mixedHelp(d, e, x, y)
		return Bool

	case lexer.LT, lexer.LE, lexer.GT, lexer.GE:
		if x == y && (isNumeric(x) || x == String) {
			return Bool
		}
		var d *diag.Diagnostic
		if x == y {
			d = c.errorf(e.Span(), "cannot compare %s values with `%s`", x, opText(e.Op))
		} else {
			d = c.errorf(e.Span(), "cannot compare %s and %s with `%s`", x, y, opText(e.Op))
		}
		d.Label = label
		c.mixedHelp(d, e, x, y)
		return Bool
	}

	// arithmetic
	switch {
	case x == y && x == Int:
		return Int
	case x == y && x == Float && e.Op != lexer.PERCENT:
		return Float
	case x == y && x == String && e.Op == lexer.PLUS:
		return String
	}
	var d *diag.Diagnostic
	if x == Float && y == Float {
		d = c.errorf(e.Span(), "cannot use `%%` on Float")
	} else {
		d = c.errorf(e.Span(), "cannot "+arithVerbs[e.Op], x, y)
	}
	d.Label = label
	c.mixedHelp(d, e, x, y)
	if e.Op == lexer.PLUS && (x == String) != (y == String) {
		other := e.X
		if x == String {
			other = e.Y
		}
		if t := c.info.Types[other]; t == Int || t == Float || t == Bool {
			d.Help = append(d.Help, "convert explicitly: "+c.replaced(e, other, "String"))
		}
	}
	return Invalid
}

// mixedHelp adds "convert explicitly: Float(a) / 2.0" when one side of e is
// an Int and the other a Float.
func (c *checker) mixedHelp(d *diag.Diagnostic, e *ast.Binary, x, y Type) {
	if !(isNumeric(x) && isNumeric(y) && x != y) {
		return
	}
	intSide := e.X
	if y == Int {
		intSide = e.Y
	}
	d.Help = append(d.Help, "convert explicitly: "+c.replaced(e, intSide, "Float"))
}

// replaced returns the source text of e with part wrapped in a call to fn.
func (c *checker) replaced(e, part ast.Node, fn string) string {
	text := c.src.Text
	s, p := e.Span(), part.Span()
	return string(text[s.Start.Offset:p.Start.Offset]) + fn + "(" + c.text(part) + ")" +
		string(text[p.End.Offset:s.End.Offset])
}

func isCmp(k lexer.Kind) bool {
	switch k {
	case lexer.EQ, lexer.NE, lexer.LT, lexer.LE, lexer.GT, lexer.GE:
		return true
	}
	return false
}

// ---------------------------------------------------------------------
// Calls

func (c *checker) call(e *ast.Call) Type {
	id, ok := e.Fn.(*ast.Ident)
	if !ok {
		c.value(e.Fn)
		c.args(e)
		c.errorf(e.Fn.Span(), "this expression can't be called")
		return Invalid
	}
	if v := c.scope.lookup(id.Name); v != nil {
		c.args(e)
		c.errorf(id.Sp, "`%s` is a variable, not a function", id.Name)
		return Invalid
	}
	if sig, ok := c.funcs[id.Name]; ok {
		types := c.args(e)
		if len(types) != len(sig.params) {
			c.arityError(e, id.Name, len(sig.params), sig.decl.Name.Sp.Start.Line)
			return sig.result
		}
		for i, t := range types {
			if !assignable(t, sig.params[i]) {
				p := sig.decl.Params[i]
				c.errorf(e.Args[i].Span(), "argument `%s` of `%s` must be %s, found %s", p.Name.Name, id.Name, sig.params[i], t).Notes = []string{
					fmt.Sprintf("`%s` is declared at line %d", id.Name, sig.decl.Name.Sp.Start.Line),
				}
			}
		}
		return sig.result
	}
	if builtinFuncs[id.Name] {
		return c.builtin(e, id.Name)
	}
	c.args(e)
	c.errorf(id.Sp, "unknown function `%s`", id.Name)
	return Invalid
}

func (c *checker) args(e *ast.Call) []Type {
	var types []Type
	for _, a := range e.Args {
		types = append(types, c.value(a))
	}
	return types
}

func (c *checker) arityError(e *ast.Call, name string, want, line int) {
	plural := "s"
	if want == 1 {
		plural = ""
	}
	verb := "were"
	if len(e.Args) == 1 {
		verb = "was"
	}
	d := c.errorf(e.Sp, "`%s` takes %d argument%s, but %d %s given", name, want, plural, len(e.Args), verb)
	if line > 0 {
		d.Notes = []string{fmt.Sprintf("`%s` is declared at line %d", name, line)}
	}
}

// builtin checks a call to a built-in function.
func (c *checker) builtin(e *ast.Call, name string) Type {
	types := c.args(e)

	// expect checks the argument count and types; it reports whether they
	// are fine.
	expect := func(want ...Type) bool {
		if len(types) != len(want) {
			c.arityError(e, name, len(want), 0)
			return false
		}
		ok := true
		for i, t := range types {
			if !assignable(t, want[i]) {
				c.errorf(e.Args[i].Span(), "`%s` needs %s here, found %s", name, want[i], t)
				ok = false
			}
		}
		return ok
	}

	switch name {
	case "print":
		if len(types) != 1 {
			c.arityError(e, name, 1, 0)
			return NoValue
		}
		if t := types[0]; !silent(t) && t != Int && t != Float && t != Bool && t != String {
			c.errorf(e.Args[0].Span(), "cannot print a value of type %s", t)
		}
		return NoValue
	case "Float":
		if len(types) == 1 && types[0] == Float {
			c.errorf(e.Args[0].Span(), "this is already a Float")
			return Float
		}
		expect(Int)
		return Float
	case "String":
		if len(types) != 1 {
			c.arityError(e, name, 1, 0)
			return String
		}
		switch t := types[0]; {
		case t == String:
			c.errorf(e.Args[0].Span(), "this is already a String")
		case t != Int && t != Float && t != Bool && !silent(t):
			c.errorf(e.Args[0].Span(), "cannot convert %s to String", t)
		}
		return String
	case "Int":
		d := c.errorf(e.Fn.Span(), "there is no `Int(...)` conversion yet")
		d.Notes = []string{"the rounding rules for Float to Int are not decided"}
		return Invalid
	case "Bool":
		c.errorf(e.Fn.Span(), "there is no `Bool(...)` conversion")
		return Invalid
	case "Error":
		c.errorf(e.Fn.Span(), "`Error` is not supported yet")
		return Invalid
	case "panic":
		expect(String)
		return Never
	case "assert":
		expect(Bool, String)
		return NoValue
	}
	panic("check: unknown built-in " + name)
}
