package check

import (
	"fmt"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/diag"
)

// Info is what the checker learned about the program.
type Info struct {
	Types map[ast.Expr]Type // the type of every checked expression
}

// Check checks a parsed file and returns all errors it finds. Bodies of
// functions with syntax errors are not checked, to avoid follow-up errors.
func Check(file *ast.File, src *diag.Source) (*Info, diag.List) {
	c := &checker{
		src:     src,
		info:    &Info{Types: map[ast.Expr]Type{}},
		funcs:   map[string]*funcSig{},
		classes: map[string]*Class{},
		broken:  map[string]bool{},
	}
	c.collect(file)
	for _, d := range file.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if sig := c.funcs[d.Name.Name]; sig != nil && sig.decl == d {
				c.funcBody(sig)
			}
		case *ast.ClassDecl:
			cls := c.classes[d.Name.Name]
			if cls == nil || cls.Decl != d {
				continue
			}
			for _, f := range d.Fields {
				if fd := cls.Fields[f.Name.Name]; fd != nil && fd.Decl == f {
					c.fieldType(fd)
				}
			}
			for _, sig := range c.methodsInOrder(cls) {
				c.funcBody(sig)
			}
		}
	}
	return c.info, c.diags
}

type checker struct {
	src     *diag.Source
	diags   diag.List
	info    *Info
	funcs   map[string]*funcSig
	classes map[string]*Class
	broken  map[string]bool // names of declarations with syntax errors

	fn    *funcSig // the function being checked; nil in a field's value
	scope *scope
}

type funcSig struct {
	name   string // `add`, or `init` / `describe` for methods
	decl   *ast.FuncDecl
	params []Type
	result Type   // NoValue if the function has no `->`
	class  *Class // for methods: the class of `this`
	isInit bool
	line   int // where it is declared
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

// collect records every top-level declaration and its signature, so they
// can be used before they are declared.
func (c *checker) collect(file *ast.File) {
	// Names first, so that types can refer to any class.
	declared := map[string]*ast.Ident{}
	var funcs []*ast.FuncDecl
	var classes []*ast.ClassDecl
	for _, d := range file.Decls {
		var name *ast.Ident
		switch d := d.(type) {
		case *ast.FuncDecl:
			name = d.Name
		case *ast.ClassDecl:
			name = d.Name
		case *ast.BadDecl:
			if d.Name != nil {
				c.broken[d.Name.Name] = true
			}
			continue
		default:
			continue
		}
		if prev, ok := declared[name.Name]; ok {
			c.errorf(name.Sp, "`%s` is declared twice", name.Name).Notes = []string{
				fmt.Sprintf("the first declaration is at line %d", prev.Sp.Start.Line),
			}
			continue
		}
		if builtinFuncs[name.Name] || builtinTypes[name.Name] != nil {
			c.errorf(name.Sp, "`%s` is a built-in name and can't be declared again", name.Name)
			continue
		}
		declared[name.Name] = name
		switch d := d.(type) {
		case *ast.FuncDecl:
			funcs = append(funcs, d)
		case *ast.ClassDecl:
			classes = append(classes, d)
			c.classes[name.Name] = &Class{
				Name:    name.Name,
				Decl:    d,
				Fields:  map[string]*Field{},
				Methods: map[string]*funcSig{},
				Inits:   map[int]*funcSig{},
			}
		}
	}

	for _, fn := range funcs {
		c.funcs[fn.Name.Name] = c.signature(fn, nil)
	}
	for _, d := range classes {
		c.classMembers(c.classes[d.Name.Name])
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

// signature resolves the parameter and result types of a function or method.
func (c *checker) signature(fn *ast.FuncDecl, class *Class) *funcSig {
	sig := &funcSig{name: fn.Name.Name, decl: fn, result: NoValue, class: class, line: fn.Name.Sp.Start.Line}
	sig.isInit = class != nil && fn.Name.Name == "init"
	seen := map[string]bool{}
	for _, p := range fn.Params {
		if seen[p.Name.Name] {
			c.errorf(p.Name.Sp, "parameter `%s` is declared twice", p.Name.Name)
		}
		seen[p.Name.Name] = true
		sig.params = append(sig.params, c.resolveType(p.Type))
	}
	if fn.Result != nil {
		t := c.resolveType(fn.Result)
		switch {
		case !sig.isInit:
			sig.result = t
		case t != class && t != Invalid:
			d := c.errorf(fn.Result.Sp, "`init` returns the new `%s`, so its return type can only be `%s`", class.Name, class.Name)
			d.Help = []string{"remove the return type"}
		}
	}
	return sig
}

// classMembers records the fields, methods and inits of a class. The types
// of fields are found later, from their initial values.
func (c *checker) classMembers(cls *Class) {
	d := cls.Decl
	for _, f := range d.Fields {
		if prev, ok := cls.Fields[f.Name.Name]; ok {
			c.errorf(f.Name.Sp, "`%s` already has a field `%s`", cls.Name, f.Name.Name).Notes = []string{
				fmt.Sprintf("the first one is at line %d", prev.Decl.Name.Sp.Start.Line),
			}
			continue
		}
		cls.Fields[f.Name.Name] = &Field{Name: f.Name.Name, Mut: f.Mut, Decl: f}
	}
	for _, m := range d.Methods {
		name := m.Name.Name
		sig := c.signature(m, cls)
		if f, ok := cls.Fields[name]; ok {
			c.errorf(m.Name.Sp, "`%s` already has a field `%s`", cls.Name, name).Notes = []string{
				fmt.Sprintf("the field is at line %d; a method can't have the same name", f.Decl.Name.Sp.Start.Line),
			}
			continue
		}
		if sig.isInit {
			if prev, ok := cls.Inits[len(sig.params)]; ok {
				d := c.errorf(m.Name.Sp, "`%s` already has an `init` with %s", cls.Name, plural(len(sig.params), "parameter"))
				d.Notes = []string{
					fmt.Sprintf("the other one is at line %d", prev.line),
					"several `init`s must have different numbers of parameters",
				}
				continue
			}
			cls.Inits[len(sig.params)] = sig
			continue
		}
		if prev, ok := cls.Methods[name]; ok {
			c.errorf(m.Name.Sp, "`%s` already has a method `%s`", cls.Name, name).Notes = []string{
				fmt.Sprintf("the first one is at line %d", prev.line),
				"only `init` can be declared several times",
			}
			continue
		}
		cls.Methods[name] = sig
	}
	if len(cls.Inits) == 0 {
		// A class without an init gets an empty one.
		cls.Inits[0] = &funcSig{name: "init", result: NoValue, class: cls, isInit: true, line: d.Name.Sp.Start.Line}
	}
}

// methodsInOrder returns the methods and inits of a class declared in its
// body, in source order.
func (c *checker) methodsInOrder(cls *Class) []*funcSig {
	var sigs []*funcSig
	for _, m := range cls.Decl.Methods {
		if sig := cls.Methods[m.Name.Name]; sig != nil && sig.decl == m {
			sigs = append(sigs, sig)
		}
		if sig := cls.Inits[len(m.Params)]; sig != nil && sig.decl == m {
			sigs = append(sigs, sig)
		}
	}
	return sigs
}

// fieldType returns the type of a field, checking its initial value the
// first time.
func (c *checker) fieldType(f *Field) Type {
	switch f.state {
	case 2:
		return f.typ
	case 1:
		// The initial value depends on the field itself.
		c.errorf(f.Decl.Name.Sp, "the type of field `%s` depends on itself", f.Name).Help = []string{
			fmt.Sprintf("write its type: `%s: Type = ...`", f.Name),
		}
		f.typ, f.state = Invalid, 2
		return Invalid
	}
	f.state = 1
	saveFn, saveScope := c.fn, c.scope
	c.fn, c.scope = nil, &scope{vars: map[string]*variable{}}
	t := c.value(f.Decl.Value)
	if f.Decl.Type != nil {
		want := c.resolveType(f.Decl.Type)
		if !assignable(t, want) {
			c.errorf(f.Decl.Value.Span(), "field `%s` is declared as %s, but the value is %s", f.Name, want, t)
		}
		t = want
	}
	if t == Never {
		t = Invalid
	}
	c.fn, c.scope = saveFn, saveScope
	if f.state == 1 {
		f.typ, f.state = t, 2
	}
	return f.typ
}

func (c *checker) resolveType(t *ast.TypeExpr) Type {
	if bt, ok := builtinTypes[t.Name.Name]; ok {
		return bt
	}
	if cls, ok := c.classes[t.Name.Name]; ok {
		return cls
	}
	if t.Name.Name == "Error" {
		c.errorf(t.Sp, "the `Error` type is not supported yet")
		return Invalid
	}
	if c.broken[t.Name.Name] {
		return Invalid
	}
	c.errorf(t.Sp, "unknown type `%s`", t.Name.Name)
	return Invalid
}

func (c *checker) funcBody(sig *funcSig) {
	if hasSyntaxError(sig.decl) {
		return
	}
	c.fn = sig
	c.scope = &scope{vars: map[string]*variable{}}
	for i, p := range sig.decl.Params {
		c.declare(p.Name, &variable{kind: Param, typ: sig.params[i], decl: p.Name.Sp})
	}
	c.blockStmts(sig.decl.Body)
	if sig.result != NoValue && !terminates(sig.decl.Body) {
		d := c.errorf(sig.decl.Body.Rbrace, "missing return at the end of `%s`", sig.name)
		d.Notes = []string{fmt.Sprintf("`%s` returns %s, so every path must end with `return`", sig.name, sig.result)}
	}
	c.fn, c.scope = nil, nil
}

// declare adds a variable to the current scope. Names must not hide
// anything: not a variable in this or an outer scope, a parameter, a
// function, a class or a built-in.
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
		d.Notes = []string{fmt.Sprintf("the function is declared at line %d", f.line)}
		return
	}
	if cls, ok := c.classes[n]; ok {
		d := c.errorf(name.Sp, "`%s` is already the name of a class", n)
		d.Notes = []string{fmt.Sprintf("the class is declared at line %d", cls.Decl.Name.Sp.Start.Line)}
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

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
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
		switch target := s.Target.(type) {
		case *ast.Ident:
			c.assignVar(target, s.Value)
		case *ast.Member:
			c.assignField(target, s.Value)
		}

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

func (c *checker) assignVar(id *ast.Ident, value ast.Expr) {
	t := c.value(value)
	v := c.scope.lookup(id.Name)
	if v == nil {
		switch {
		case c.funcs[id.Name] != nil:
			c.errorf(id.Sp, "cannot assign to `%s`: it is a function", id.Name)
		case c.classes[id.Name] != nil:
			c.errorf(id.Sp, "cannot assign to `%s`: it is a class", id.Name)
		default:
			d := c.errorf(id.Sp, "cannot assign to `%s`: it is not declared", id.Name)
			if f := c.thisField(id.Name); f != nil {
				d.Help = []string{fmt.Sprintf("to change the field, write `this.%s = ...`", id.Name)}
			} else {
				d.Help = []string{fmt.Sprintf("declare it with `mut %s = ...`", id.Name)}
			}
		}
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
		if c.thisField(id.Name) != nil {
			d.Help = []string{fmt.Sprintf("to change the field, write `this.%s = ...`", id.Name)}
		} else {
			d.Help = []string{fmt.Sprintf("copy it into a variable first: `mut %s_ = %s`", id.Name, id.Name)}
		}
		return
	}
	if !assignable(t, v.typ) {
		c.errorf(value.Span(), "cannot assign %s to `%s`, which is %s", t, id.Name, v.typ)
	}
}

func (c *checker) assignField(m *ast.Member, value ast.Expr) {
	f, cls := c.fieldOf(m)
	t := c.value(value)
	if f == nil {
		return
	}
	ft := c.fieldType(f)
	if !f.Mut {
		_, viaThis := m.X.(*ast.This)
		if !(viaThis && c.fn != nil && c.fn.isInit && c.fn.class == cls) {
			d := c.errorf(m.Name.Sp, "cannot assign to `%s`: it is a `con` field", f.Name)
			d.Notes = []string{
				fmt.Sprintf("`%s.%s` is declared at line %d", cls.Name, f.Name, f.Decl.Name.Sp.Start.Line),
				"a `con` field can only be set in `init`, through `this`",
			}
			d.Help = []string{"declare it with `mut` if it needs to change"}
			return
		}
	}
	if !assignable(t, ft) {
		c.errorf(value.Span(), "cannot assign %s to `%s`, which is %s", t, f.Name, ft)
	}
}

// thisField returns the field of `this` with the given name, if the
// function being checked is a method and its class has one.
func (c *checker) thisField(name string) *Field {
	if c.fn == nil || c.fn.class == nil {
		return nil
	}
	return c.fn.class.Fields[name]
}

func (c *checker) cond(e ast.Expr) {
	t := c.value(e)
	if !assignable(t, Bool) {
		c.errorf(e.Span(), "condition must be Bool, found %s", t)
	}
}

func (c *checker) returnStmt(s *ast.Return) {
	name := c.fn.name
	want := c.fn.result
	if s.Value == nil {
		if want != NoValue {
			c.errorf(s.Sp, "`%s` must return %s, but this `return` has no value", name, want)
		}
		return
	}
	t := c.value(s.Value)
	if c.fn.isInit {
		d := c.errorf(s.Value.Span(), "`return` in `init` can't have a value")
		d.Notes = []string{fmt.Sprintf("`init` returns the new `%s` by itself", c.fn.class.Name)}
		return
	}
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
