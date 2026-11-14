package check

import (
	"fmt"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/diag"
	"github.com/AxelGard/bronze/internal/parser"
)

// Info is what the checker learned about the program.
type Info struct {
	Types map[ast.Expr]Type // the type of every checked expression
	Refs  []Ref             // every name that refers to a symbol
}

// Check checks a parsed file and returns all errors it finds. Bodies of
// functions with syntax errors are not checked, to avoid follow-up errors.
func Check(file *ast.File, src *diag.Source) (*Info, diag.List) {
	c := &checker{
		src:     src,
		info:    &Info{Types: map[ast.Expr]Type{}},
		funcs:   map[string]*funcSig{},
		classes: map[string]*Class{},
		enums:   map[string]*Enum{},
		broken:  map[string]bool{},
	}
	decls := append(append([]ast.Decl{}, parser.Prelude()...), file.Decls...)
	c.collect(file, decls)
	for _, d := range decls {
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
			c.inPrelude = d.Builtin
			for _, f := range d.Fields {
				if fd := cls.Fields[f.Name.Name]; fd != nil && fd.Decl == f {
					c.fieldType(fd)
				}
			}
			for _, sig := range c.methodsInOrder(cls) {
				c.funcBody(sig)
			}
			c.inPrelude = false
		case *ast.EnumDecl:
			en := c.enums[d.Name.Name]
			if en == nil || en.Decl != d {
				continue
			}
			for _, m := range d.Methods {
				if sig := en.Methods[m.Name.Name]; sig != nil && sig.decl == m {
					c.funcBody(sig)
				}
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
	enums   map[string]*Enum
	broken  map[string]bool // names of declarations with syntax errors
	errCls  *Class          // the built-in Error class

	inPrelude bool // checking the prelude: don't record references

	fn    *funcSig // the function being checked; nil in a field's value
	scope *scope
	loops int // how many loops the statement being checked is in
}

type funcSig struct {
	name   string // `add`, or `init` / `describe` for methods
	decl   *ast.FuncDecl
	params []Type
	result Type   // NoValue if the function has no `->`; *Failable with `!`
	class  *Class // for methods of a class: the class of `this`
	enum   *Enum  // for methods of an enum: the enum of `this`
	isInit bool
	line   int     // where it is declared
	sym    *Symbol // nil for init
}

// VarKind is how a variable was declared.
type VarKind int

const (
	Con VarKind = iota
	Mut
	Param
	Loop // a loop variable
)

type variable struct {
	kind VarKind
	typ  Type
	decl diag.Span
	sym  *Symbol
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
func (c *checker) collect(file *ast.File, decls []ast.Decl) {
	// Names first, so that types can refer to any class.
	declared := map[string]*ast.Ident{}
	var funcs []*ast.FuncDecl
	var classes []*ast.ClassDecl
	var enums []*ast.EnumDecl
	for _, d := range decls {
		var name *ast.Ident
		switch d := d.(type) {
		case *ast.FuncDecl:
			name = d.Name
		case *ast.ClassDecl:
			name = d.Name
		case *ast.EnumDecl:
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
		if cd, ok := d.(*ast.ClassDecl); (!ok || !cd.Builtin) && (builtinFuncs[name.Name] || builtinTypes[name.Name] != nil || genericTypes[name.Name]) {
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
				Sym:     c.newSym(name, SymClass, d.Builtin),
			}
		case *ast.EnumDecl:
			enums = append(enums, d)
			c.enums[name.Name] = &Enum{
				Name:    name.Name,
				Decl:    d,
				Cases:   map[string]int{},
				Methods: map[string]*funcSig{},
				Sym:     c.newSym(name, SymEnum, false),
				CaseSym: map[string]*Symbol{},
			}
		}
	}

	c.errCls = c.classes["Error"]
	for _, d := range classes {
		if d.UseError {
			c.classes[d.Name.Name].Super = c.errCls
		}
	}
	for _, fn := range funcs {
		sig := c.signature(fn, nil)
		sig.sym = c.newSym(fn.Name, SymFunction, false)
		c.funcs[fn.Name.Name] = sig
	}
	for _, d := range classes {
		c.classMembers(c.classes[d.Name.Name])
	}
	for _, d := range enums {
		c.enumMembers(c.enums[d.Name.Name])
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
	if fn.Fail != nil {
		c.failType(fn, sig)
	}
	return sig
}

// failType resolves the `! E` of a signature and makes its result a
// *Failable. If E is not an error type, the function still counts as one
// that can fail (with Error), so that its callers don't get more errors.
func (c *checker) failType(fn *ast.FuncDecl, sig *funcSig) {
	t := c.resolveType(fn.Fail)
	if sig.isInit {
		c.errorf(fn.Fail.Sp, "`init` can't fail")
		return
	}
	cls, ok := t.(*Class)
	switch {
	case silent(t):
		cls = c.errCls
	case fn.Fail.Optional:
		c.errorf(fn.Fail.Sp, "the error type after `!` can't be a `?` type")
		cls = c.errCls
	case !ok || !c.isError(cls):
		d := c.errorf(fn.Fail.Sp, "`%s` is not an error type", t)
		d.Help = []string{"use `Error`, or a class that uses `Error`: `class NotFound use Error { ... }`"}
		cls = c.errCls
	}
	value := sig.result
	inner := value
	if o, ok := value.(*Optional); ok {
		inner = o.Inner
	}
	if k, ok := inner.(*Class); ok && c.isError(k) {
		d := c.errorf(fn.Result.Sp, "the value of `-> T ! E` can't be an error")
		d.Notes = []string{fmt.Sprintf("a function that fails with `%s` returns it with `return`, it isn't the value", k.Name)}
		return
	}
	sig.result = &Failable{Value: value, Err: cls}
}

// isError reports whether t is Error or a class that uses it.
func (c *checker) isError(t Type) bool {
	return c.errCls != nil && t != Invalid && assignable(t, c.errCls) && !silent(t)
}

// classMembers records the fields, methods and inits of a class. The types
// of fields are found later, from their initial values.
func (c *checker) classMembers(cls *Class) {
	d := cls.Decl
	if cls.Super != nil {
		for name, f := range cls.Super.Fields {
			cls.Fields[name] = f
		}
	}
	for _, f := range d.Fields {
		if cls.Super != nil && cls.Super.Fields[f.Name.Name] != nil {
			d := c.errorf(f.Name.Sp, "`%s` already has a field `%s`, from `use %s`", cls.Name, f.Name.Name, cls.Super.Name)
			d.Help = []string{fmt.Sprintf("set it in `init` instead: `this.%s = ...`", f.Name.Name)}
			continue
		}
		if prev, ok := cls.Fields[f.Name.Name]; ok {
			c.errorf(f.Name.Sp, "`%s` already has a field `%s`", cls.Name, f.Name.Name).Notes = []string{
				fmt.Sprintf("the first one is at line %d", prev.Decl.Name.Sp.Start.Line),
			}
			continue
		}
		cls.Fields[f.Name.Name] = &Field{Name: f.Name.Name, Mut: f.Mut, Decl: f, sym: c.newSym(f.Name, SymField, d.Builtin)}
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
		sig.sym = c.newSym(m.Name, SymMethod, d.Builtin)
		cls.Methods[name] = sig
	}
	if len(cls.Inits) == 0 {
		// A class without an init gets an empty one.
		cls.Inits[0] = &funcSig{name: "init", result: NoValue, class: cls, isInit: true, line: d.Name.Sp.Start.Line}
	}
}

// enumMembers records the cases and methods of an enum.
func (c *checker) enumMembers(en *Enum) {
	d := en.Decl
	for i, cs := range d.Cases {
		if prev, ok := en.Cases[cs.Name]; ok {
			c.errorf(cs.Sp, "`%s` already has a case `%s`", en.Name, cs.Name).Notes = []string{
				fmt.Sprintf("the first one is at line %d", d.Cases[prev].Sp.Start.Line),
			}
			continue
		}
		en.Cases[cs.Name] = i
		en.CaseSym[cs.Name] = c.newSym(cs, SymCase, false)
	}
	for _, m := range d.Methods {
		name := m.Name.Name
		if name == "init" {
			d := c.errorf(m.Name.Sp, "an enum can't have an `init`")
			d.Notes = []string{fmt.Sprintf("its values are its cases, like `%s.%s`", en.Name, en.someCase())}
			continue
		}
		if _, ok := en.Cases[name]; ok {
			c.errorf(m.Name.Sp, "`%s` already has a case `%s`; a method can't have the same name", en.Name, name)
			continue
		}
		if prev, ok := en.Methods[name]; ok {
			c.errorf(m.Name.Sp, "`%s` already has a method `%s`", en.Name, name).Notes = []string{
				fmt.Sprintf("the first one is at line %d", prev.line),
			}
			continue
		}
		sig := c.signature(m, nil)
		sig.enum = en
		sig.sym = c.newSym(m.Name, SymMethod, false)
		en.Methods[name] = sig
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
	saveFn, saveScope, saveLoops := c.fn, c.scope, c.loops
	c.fn, c.scope, c.loops = nil, &scope{vars: map[string]*variable{}}, 0
	var want Type
	if f.Decl.Type != nil {
		want = c.resolveType(f.Decl.Type)
	}
	t := c.valueFor(f.Decl.Value, want)
	if f.Decl.Type != nil {
		if !assignable(t, want) {
			c.errorf(f.Decl.Value.Span(), "field `%s` is declared as %s, but the value is %s", f.Name, want, t)
		}
		t = want
	}
	if t == Never || t == Null {
		if t == Null {
			c.nullNeedsType(f.Decl.Name, f.Decl.Value, f.Mut)
		}
		t = Invalid
	}
	c.fn, c.scope, c.loops = saveFn, saveScope, saveLoops
	if f.state == 1 {
		f.typ, f.state = t, 2
	}
	return f.typ
}

func (c *checker) resolveType(t *ast.TypeExpr) Type {
	base := c.resolveBase(t)
	if !t.Optional || silent(base) {
		return base
	}
	return &Optional{Inner: base}
}

func (c *checker) resolveName(name *ast.Ident) Type {
	if bt, ok := builtinTypes[name.Name]; ok {
		return bt
	}
	if cls, ok := c.classes[name.Name]; ok {
		c.ref(name, cls.Sym)
		return cls
	}
	if en, ok := c.enums[name.Name]; ok {
		c.ref(name, en.Sym)
		return en
	}
	if c.broken[name.Name] {
		return Invalid
	}
	c.errorf(name.Sp, "unknown type `%s`", name.Name)
	return Invalid
}

func (c *checker) funcBody(sig *funcSig) {
	if sig.decl.Bad {
		return
	}
	c.fn, c.loops = sig, 0
	c.scope = &scope{vars: map[string]*variable{}}
	for i, p := range sig.decl.Params {
		c.declare(p.Name, &variable{kind: Param, typ: sig.params[i], decl: p.Name.Sp})
	}
	c.blockStmts(sig.decl.Body)
	if hasValue(sig.result) && !terminates(sig.decl.Body) {
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
		switch prev.kind {
		case Param:
			what = "a parameter"
		case Loop:
			what = "a loop variable"
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
	if en, ok := c.enums[n]; ok {
		d := c.errorf(name.Sp, "`%s` is already the name of an enum", n)
		d.Notes = []string{fmt.Sprintf("the enum is declared at line %d", en.Decl.Name.Sp.Start.Line)}
		return
	}
	if builtinFuncs[n] || builtinTypes[n] != nil || genericTypes[n] {
		c.errorf(name.Sp, "`%s` is a built-in name and can't be used for a variable", n)
		return
	}
	kind := SymVariable
	if v.kind == Param {
		kind = SymParameter
	}
	v.sym = c.newSym(name, kind, false)
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
		if m, ok := s.X.(*ast.Match); ok {
			return matchTerminates(m)
		}
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

// matchTerminates reports whether every arm of a match statement leaves
// the function. A match always covers every case, so then it always leaves.
func matchTerminates(m *ast.Match) bool {
	if len(m.Arms) == 0 {
		return false
	}
	for _, a := range m.Arms {
		switch {
		case a.Body != nil:
			if !terminates(a.Body) {
				return false
			}
		case !terminates(&ast.ExprStmt{X: a.Value}):
			return false
		}
	}
	return true
}

func isPanicCall(e ast.Expr) bool {
	call, ok := e.(*ast.Call)
	if !ok {
		return false
	}
	id, ok := call.Fn.(*ast.Ident)
	return ok && id.Name == "panic"
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
		var want Type
		if s.Type != nil {
			want = c.resolveType(s.Type)
		}
		t := c.valueFor(s.Value, want)
		if s.Type != nil {
			if !assignable(t, want) && !c.nullMismatch(s.Value, t, want) {
				c.errorf(s.Value.Span(), "`%s` is declared as %s, but the value is %s", s.Name.Name, want, t)
			}
			t = want
		}
		if t == Null {
			c.nullNeedsType(s.Name, s.Value, s.Mut)
			t = Invalid
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
		case *ast.Index:
			c.assignIndex(target, s.Value)
		}

	case *ast.Compound:
		c.compound(s)

	case *ast.Break:
		if c.loops == 0 {
			c.errorf(s.Sp, "`break` outside a loop")
		}

	case *ast.Continue:
		if c.loops == 0 {
			c.errorf(s.Sp, "`continue` outside a loop")
		}

	case *ast.If:
		c.ifStmt(s)

	case *ast.While:
		c.cond(s.Cond)
		c.loops++
		c.block(s.Body)
		c.loops--

	case *ast.For:
		c.forStmt(s)

	case *ast.Return:
		c.returnStmt(s)

	case *ast.ExprStmt:
		if m, ok := s.X.(*ast.Match); ok {
			c.matchStmt(m)
		} else if _, ok := c.expr(s.X).(*Failable); ok {
			c.unhandled(s.X)
		}

	case *ast.Block:
		c.block(s)

	case *ast.BadStmt:
	}
}

func (c *checker) assignVar(id *ast.Ident, value ast.Expr) {
	want := c.varTarget(id)
	t := c.valueFor(value, want)
	if want != nil && !assignable(t, want) && !c.nullMismatch(value, t, want) {
		c.errorf(value.Span(), "cannot assign %s to `%s`, which is %s", t, id.Name, want)
	}
}

// varTarget checks that id is a variable that can be assigned to, and
// returns its type, or nil after reporting an error.
func (c *checker) varTarget(id *ast.Ident) Type {
	v := c.scope.lookup(id.Name)
	if v != nil {
		c.ref(id, v.sym)
	}
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
		return nil
	}
	switch v.kind {
	case Con:
		d := c.errorf(id.Sp, "cannot assign to `%s`: it is declared with `con`", id.Name)
		d.Notes = []string{fmt.Sprintf("`%s` is declared at line %d", id.Name, v.decl.Start.Line)}
		d.Help = []string{"declare it with `mut` if it needs to change"}
		return nil
	case Loop:
		d := c.errorf(id.Sp, "cannot assign to `%s`: it is a loop variable", id.Name)
		d.Help = []string{fmt.Sprintf("copy it into a variable first: `mut %s_ = %s`", id.Name, id.Name)}
		return nil
	case Param:
		d := c.errorf(id.Sp, "cannot assign to `%s`: parameters can't be changed", id.Name)
		if c.thisField(id.Name) != nil {
			d.Help = []string{fmt.Sprintf("to change the field, write `this.%s = ...`", id.Name)}
		} else {
			d.Help = []string{fmt.Sprintf("copy it into a variable first: `mut %s_ = %s`", id.Name, id.Name)}
		}
		return nil
	}
	c.info.Types[id] = v.typ
	return v.typ
}

func (c *checker) assignField(m *ast.Member, value ast.Expr) {
	want := c.fieldTarget(m)
	t := c.valueFor(value, want)
	if want != nil && !assignable(t, want) && !c.nullMismatch(value, t, want) {
		c.errorf(value.Span(), "cannot assign %s to `%s`, which is %s", t, m.Name.Name, want)
	}
}

// fieldTarget checks that m is a field that can be assigned to, and
// returns its type, or nil after reporting an error.
func (c *checker) fieldTarget(m *ast.Member) Type {
	f, cls := c.fieldOf(m)
	if f == nil {
		return nil
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
			return nil
		}
	}
	c.info.Types[m] = ft
	return ft
}

// compound checks `target op= value`: the target must be assignable, and
// `target op value` must be a value of the target's type.
func (c *checker) compound(s *ast.Compound) {
	var want Type
	switch target := s.Target.(type) {
	case *ast.Ident:
		want = c.varTarget(target)
	case *ast.Member:
		want = c.fieldTarget(target)
	case *ast.Index:
		want = c.indexTarget(target)
	}
	t := c.value(s.Value)
	if want == nil || silent(want) || silent(t) {
		return
	}
	if c.mayBeNull(s.Target, want) || c.mayBeNull(s.Value, t) {
		return
	}
	label := fmt.Sprintf("%s %s= %s", want, opText(s.Op), t)
	c.arith(s.Op, s, s.Target, s.Value, want, t, label, true)
}

// forStmt checks a for loop: what it loops over, and the types of the
// loop variables, which are visible only in its block.
func (c *checker) forStmt(s *ast.For) {
	x := s.X
	for {
		p, ok := x.(*ast.Paren)
		if !ok {
			break
		}
		x = p.X
	}
	t := c.value(s.X)
	var types []Type // the types of the variables, for one and for two
	switch tt := t.(type) {
	case *List:
		types = []Type{tt.Elem, Int, tt.Elem}
	case *Map:
		types = []Type{tt.Key, tt.Key, tt.Val}
	default:
		switch {
		case silent(t):
		case c.mayBeNull(s.X, t):
		default:
			d := c.errorf(s.X.Span(), "cannot loop over %s", t)
			d.Notes = []string{"`for` loops over a list, a map, a String or a range"}
		}
		types = []Type{Invalid, Invalid, Invalid}
	}
	if _, isRange := x.(*ast.Range); isRange && len(s.Vars) == 2 {
		d := c.errorf(s.Vars[1].Sp, "a range gives one value per pass, so it takes one loop variable")
		d.Help = []string{fmt.Sprintf("write `for %s in %s`", s.Vars[0].Name, c.text(s.X))}
	}

	c.scope = &scope{parent: c.scope, vars: map[string]*variable{}}
	vars := types[:1]
	if len(s.Vars) == 2 {
		vars = types[1:]
	}
	for i, v := range s.Vars {
		if v.Name != "_" {
			c.declare(v, &variable{kind: Loop, typ: vars[i], decl: v.Sp})
		}
	}
	c.loops++
	c.block(s.Body)
	c.loops--
	c.scope = c.scope.parent
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
	if !assignable(t, Bool) && !c.nullMismatch(e, t, Bool) {
		c.errorf(e.Span(), "condition must be Bool, found %s", t)
	}
}

func (c *checker) returnStmt(s *ast.Return) {
	c.checkReturn(s.Sp, s.Value)
}

// checkReturn checks `return value` (value may be nil), as a statement or
// in an `else return`.
func (c *checker) checkReturn(at diag.Span, value ast.Expr) {
	if c.fn == nil {
		c.errorf(at, "`return` can't be used in the initial value of a field")
		if value != nil {
			c.value(value)
		}
		return
	}
	name := c.fn.name
	want, fails := c.fn.result, (*Class)(nil)
	if f, ok := want.(*Failable); ok {
		want, fails = f.Value, f.Err
	}
	if value == nil {
		if want != NoValue {
			c.errorf(at, "`%s` must return %s, but this `return` has no value", name, want)
		}
		return
	}
	var expect Type
	if want != NoValue {
		expect = want
	}
	t := c.valueFor(value, expect)
	switch {
	case silent(t):
	case c.fn.isInit:
		d := c.errorf(value.Span(), "`return` in `init` can't have a value")
		d.Notes = []string{fmt.Sprintf("`init` returns the new `%s` by itself", c.fn.class.Name)}
	case fails != nil && assignable(t, fails):
		// fails with the error
	case fails != nil && c.isError(t):
		d := c.errorf(value.Span(), "`%s` fails with %s, but this is %s", name, fails, t)
		d.Notes = []string{fmt.Sprintf("%s doesn't use %s", t, fails)}
	case want == NoValue && fails != nil:
		c.errorf(value.Span(), "`%s` returns nothing, so `return` can only fail with an error (%s), but this is %s", name, fails, t)
	case want == NoValue:
		d := c.errorf(value.Span(), "`%s` returns nothing, so `return` can't have a value", name)
		if c.isError(t) {
			d.Help = []string{fmt.Sprintf("to fail with it, declare `func %s(...) -> ! %s`", name, t)}
		} else {
			d.Help = []string{fmt.Sprintf("add a return type: `func %s(...) -> %s`", name, t)}
		}
	case !assignable(t, want) && !c.nullMismatch(value, t, want):
		d := c.errorf(value.Span(), "`%s` returns %s, but this is %s", name, want, t)
		if c.isError(t) && fails == nil {
			d.Help = []string{fmt.Sprintf("to fail with it, declare `func %s(...) -> %s ! %s`", name, want, t)}
		}
	}
}
