package interp

import (
	"bufio"
	"cmp"
	"fmt"
	"io"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/diag"
	"github.com/AxelGard/bronze/internal/lexer"
)

// Value is a Bronze value: int64 (Int), float64 (Float), bool (Bool),
// string (String) or *Object (an object of a class).
type Value any

// Object is an object of a class. Objects are shared: a Value holding an
// *Object is a reference, and `==` compares identity.
type Object struct {
	Class  *ast.ClassDecl
	Fields map[string]Value
}

// MaxDepth is the number of nested calls after which the program panics
// with "stack overflow".
const MaxDepth = 10_000

// Panic is a Bronze panic: a runtime failure with a Bronze stack trace.
type Panic struct {
	Message string
	Trace   []Frame // innermost first
}

// Frame is one function in a stack trace, and the line it was running.
type Frame struct {
	Func string
	File string
	Line int
}

// Format writes the panic as it is shown to the user:
//
//	panic: division by zero
//	  at divide (core.brz:12)
//	  at main   (core.brz:40)
//
// Very deep traces show only their first and last frames.
func (p *Panic) Format(w io.Writer) {
	fmt.Fprintf(w, "panic: %s\n", p.Message)
	const keep = 10
	frames := p.Trace
	width := 0
	for _, f := range frames {
		width = max(width, len(f.Func))
	}
	for i, f := range frames {
		if len(frames) > 2*keep+1 && i >= keep && i < len(frames)-keep {
			if i == keep {
				fmt.Fprintf(w, "  ... %d more\n", len(frames)-2*keep)
			}
			continue
		}
		fmt.Fprintf(w, "  at %-*s (%s:%d)\n", width, f.Func, f.File, f.Line)
	}
}

// Run runs a checked program: it calls main and returns main's result as
// the exit code. If the program panics, it returns the panic instead.
// Output goes to stdout, which is flushed before Run returns.
func Run(file *ast.File, src *diag.Source, stdout io.Writer) (code int, p *Panic) {
	out := bufio.NewWriter(stdout)
	defer out.Flush()
	in := &interp{
		src:     src,
		out:     out,
		funcs:   map[string]*ast.FuncDecl{},
		classes: map[string]*class{},
	}
	for _, d := range file.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			in.funcs[d.Name.Name] = d
		case *ast.ClassDecl:
			cls := &class{decl: d, inits: map[int]*ast.FuncDecl{}, methods: map[string]*ast.FuncDecl{}}
			for _, m := range d.Methods {
				if m.Name.Name == "init" {
					cls.inits[len(m.Params)] = m
				} else {
					cls.methods[m.Name.Name] = m
				}
			}
			in.classes[d.Name.Name] = cls
		}
	}

	defer func() {
		if r := recover(); r != nil {
			bp, ok := r.(*Panic)
			if !ok {
				panic(r) // a bug in bronze, not in the program
			}
			p = bp
		}
	}()
	main := in.funcs["main"]
	result := in.call("main", main, nil, nil, main.Name.Sp)
	return int(result.(int64)), nil
}

type interp struct {
	src     *diag.Source
	out     *bufio.Writer
	funcs   map[string]*ast.FuncDecl
	classes map[string]*class
	stack   []*frame
}

type class struct {
	decl    *ast.ClassDecl
	inits   map[int]*ast.FuncDecl // by number of parameters
	methods map[string]*ast.FuncDecl
}

type frame struct {
	name string // `add`, or `Counter.increment` for a method
	line int    // the line being run
	this *Object
}

type env struct {
	parent *env
	vars   map[string]Value
}

func (e *env) lookup(name string) Value {
	for ; e != nil; e = e.parent {
		if v, ok := e.vars[name]; ok {
			return v
		}
	}
	panic("interp: unknown variable " + name)
}

func (e *env) set(name string, v Value) {
	for ; e != nil; e = e.parent {
		if _, ok := e.vars[name]; ok {
			e.vars[name] = v
			return
		}
	}
	panic("interp: unknown variable " + name)
}

// fail raises a Bronze panic at span.
func (in *interp) fail(span diag.Span, format string, args ...any) {
	in.stack[len(in.stack)-1].line = span.Start.Line
	p := &Panic{Message: fmt.Sprintf(format, args...)}
	for i := len(in.stack) - 1; i >= 0; i-- {
		f := in.stack[i]
		p.Trace = append(p.Trace, Frame{Func: f.name, File: in.src.Name, Line: f.line})
	}
	panic(p)
}

// ---------------------------------------------------------------------
// Calls

// enter pushes a frame for a call made at span at.
func (in *interp) enter(name string, line int, this *Object, at diag.Span) {
	if len(in.stack) > 0 {
		in.stack[len(in.stack)-1].line = at.Start.Line
	}
	if len(in.stack) >= MaxDepth {
		in.fail(at, "stack overflow")
	}
	in.stack = append(in.stack, &frame{name: name, line: line, this: this})
}

func (in *interp) leave() { in.stack = in.stack[:len(in.stack)-1] }

// call calls a function, or a method when this is not nil.
func (in *interp) call(name string, fn *ast.FuncDecl, this *Object, args []Value, at diag.Span) Value {
	in.enter(name, fn.Name.Sp.Start.Line, this, at)
	result := in.body(fn, args)
	in.leave()
	return result
}

// body runs the body of fn in the current frame.
func (in *interp) body(fn *ast.FuncDecl, args []Value) Value {
	e := &env{vars: make(map[string]Value, len(fn.Params))}
	for i, p := range fn.Params {
		e.vars[p.Name.Name] = args[i]
	}
	_, result := in.block(fn.Body, e)
	return result
}

// construct creates an object: the field values are evaluated first, then
// the init with the matching number of parameters runs.
func (in *interp) construct(cls *class, args []Value, at diag.Span) *Object {
	obj := &Object{Class: cls.decl, Fields: make(map[string]Value, len(cls.decl.Fields))}
	init := cls.inits[len(args)]
	line := cls.decl.Name.Sp.Start.Line
	if init != nil {
		line = init.Name.Sp.Start.Line
	}
	in.enter(cls.decl.Name.Name+".init", line, obj, at)
	empty := &env{vars: map[string]Value{}}
	for _, f := range cls.decl.Fields {
		obj.Fields[f.Name.Name] = in.expr(f.Value, empty)
	}
	if init != nil {
		in.body(init, args)
	}
	in.leave()
	return obj
}

// ---------------------------------------------------------------------
// Statements

// block runs a block in a new scope inside e. It reports whether a
// `return` ran, and the returned value.
func (in *interp) block(b *ast.Block, parent *env) (bool, Value) {
	e := &env{parent: parent, vars: map[string]Value{}}
	for _, s := range b.Stmts {
		if ret, v := in.stmt(s, e); ret {
			return true, v
		}
	}
	return false, nil
}

func (in *interp) stmt(s ast.Stmt, e *env) (bool, Value) {
	switch s := s.(type) {
	case *ast.VarDecl:
		e.vars[s.Name.Name] = in.expr(s.Value, e)
	case *ast.Assign:
		switch t := s.Target.(type) {
		case *ast.Ident:
			e.set(t.Name, in.expr(s.Value, e))
		case *ast.Member:
			obj := in.expr(t.X, e).(*Object)
			obj.Fields[t.Name.Name] = in.expr(s.Value, e)
		}
	case *ast.ExprStmt:
		in.expr(s.X, e)
	case *ast.If:
		if in.expr(s.Cond, e).(bool) {
			return in.block(s.Then, e)
		}
		switch els := s.Else.(type) {
		case *ast.Block:
			return in.block(els, e)
		case *ast.If:
			return in.stmt(els, e)
		}
	case *ast.While:
		for in.expr(s.Cond, e).(bool) {
			if ret, v := in.block(s.Body, e); ret {
				return true, v
			}
		}
	case *ast.Return:
		if s.Value == nil {
			return true, nil
		}
		return true, in.expr(s.Value, e)
	case *ast.Block:
		return in.block(s, e)
	default:
		panic(fmt.Sprintf("interp: unexpected statement %T", s))
	}
	return false, nil
}

// ---------------------------------------------------------------------
// Expressions

func (in *interp) expr(x ast.Expr, e *env) Value {
	switch x := x.(type) {
	case *ast.IntLit:
		return x.Value
	case *ast.FloatLit:
		return x.Value
	case *ast.StringLit:
		return x.Value
	case *ast.BoolLit:
		return x.Value
	case *ast.Ident:
		return e.lookup(x.Name)
	case *ast.Paren:
		return in.expr(x.X, e)
	case *ast.This:
		return in.stack[len(in.stack)-1].this
	case *ast.Member:
		return in.expr(x.X, e).(*Object).Fields[x.Name.Name]
	case *ast.Unary:
		v := in.expr(x.X, e)
		switch x.Op {
		case lexer.NOT:
			return !v.(bool)
		case lexer.MINUS:
			switch v := v.(type) {
			case int64:
				return -v
			case float64:
				return -v
			}
		}
	case *ast.Binary:
		return in.binary(x, e)
	case *ast.Call:
		return in.callExpr(x, e)
	}
	panic(fmt.Sprintf("interp: unexpected expression %T", x))
}

func (in *interp) binary(x *ast.Binary, e *env) Value {
	switch x.Op {
	case lexer.AND:
		return in.expr(x.X, e).(bool) && in.expr(x.Y, e).(bool)
	case lexer.OR:
		return in.expr(x.X, e).(bool) || in.expr(x.Y, e).(bool)
	}
	a, b := in.expr(x.X, e), in.expr(x.Y, e)
	switch x.Op {
	case lexer.EQ:
		return a == b
	case lexer.NE:
		return a != b
	}
	switch a := a.(type) {
	case int64:
		b := b.(int64)
		switch x.Op {
		case lexer.PLUS:
			return a + b
		case lexer.MINUS:
			return a - b
		case lexer.STAR:
			return a * b
		case lexer.SLASH, lexer.PERCENT:
			if b == 0 {
				in.fail(x.Span(), "division by zero")
			}
			if x.Op == lexer.SLASH {
				return a / b
			}
			return a % b
		}
		return order(x.Op, a, b)
	case float64:
		b := b.(float64)
		switch x.Op {
		case lexer.PLUS:
			return a + b
		case lexer.MINUS:
			return a - b
		case lexer.STAR:
			return a * b
		case lexer.SLASH:
			return a / b
		}
		return order(x.Op, a, b)
	case string:
		b := b.(string)
		if x.Op == lexer.PLUS {
			return a + b
		}
		return order(x.Op, a, b)
	}
	panic(fmt.Sprintf("interp: bad operands %T %s %T", a, x.Op, b))
}

// order applies < <= > >=. For Floats, every comparison with NaN is false.
func order[T cmp.Ordered](op lexer.Kind, a, b T) bool {
	switch op {
	case lexer.LT:
		return a < b
	case lexer.LE:
		return a <= b
	case lexer.GT:
		return a > b
	case lexer.GE:
		return a >= b
	}
	panic("interp: unknown operator " + op.String())
}

func (in *interp) args(x *ast.Call, e *env) []Value {
	args := make([]Value, len(x.Args))
	for i, a := range x.Args {
		args[i] = in.expr(a, e)
	}
	return args
}

func (in *interp) callExpr(x *ast.Call, e *env) Value {
	if m, ok := x.Fn.(*ast.Member); ok {
		obj := in.expr(m.X, e).(*Object)
		args := in.args(x, e)
		cls := in.classes[obj.Class.Name.Name]
		fn := cls.methods[m.Name.Name]
		return in.call(obj.Class.Name.Name+"."+fn.Name.Name, fn, obj, args, x.Sp)
	}
	name := x.Fn.(*ast.Ident).Name
	args := in.args(x, e)
	if fn, ok := in.funcs[name]; ok {
		return in.call(name, fn, nil, args, x.Sp)
	}
	if cls, ok := in.classes[name]; ok {
		return in.construct(cls, args, x.Sp)
	}
	switch name {
	case "print":
		in.out.WriteString(Format(args[0]))
		in.out.WriteByte('\n')
		return nil
	case "Float":
		return float64(args[0].(int64))
	case "String":
		return Format(args[0])
	case "panic":
		in.fail(x.Sp, "%s", args[0].(string))
	case "assert":
		if !args[0].(bool) {
			in.fail(x.Sp, "assertion failed: %s", args[1].(string))
		}
		return nil
	}
	panic("interp: unknown function " + name)
}
