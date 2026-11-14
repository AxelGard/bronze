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

// Value is a Bronze value: int64 (Int), float64 (Float), bool (Bool) or
// string (String).
type Value any

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
		src:   src,
		out:   out,
		funcs: map[string]*ast.FuncDecl{},
	}
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			in.funcs[fn.Name.Name] = fn
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
	result := in.call(main, nil, main.Name.Sp)
	return int(result.(int64)), nil
}

type interp struct {
	src   *diag.Source
	out   *bufio.Writer
	funcs map[string]*ast.FuncDecl
	stack []*frame
}

type frame struct {
	name string
	line int // the line being run
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

func (in *interp) call(fn *ast.FuncDecl, args []Value, at diag.Span) Value {
	if len(in.stack) > 0 {
		in.stack[len(in.stack)-1].line = at.Start.Line
	}
	if len(in.stack) >= MaxDepth {
		in.fail(at, "stack overflow")
	}
	in.stack = append(in.stack, &frame{name: fn.Name.Name, line: fn.Name.Sp.Start.Line})
	e := &env{vars: make(map[string]Value, len(fn.Params))}
	for i, p := range fn.Params {
		e.vars[p.Name.Name] = args[i]
	}
	_, result := in.block(fn.Body, e)
	in.stack = in.stack[:len(in.stack)-1]
	return result
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
		e.set(s.Target.(*ast.Ident).Name, in.expr(s.Value, e))
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

func (in *interp) callExpr(x *ast.Call, e *env) Value {
	name := x.Fn.(*ast.Ident).Name
	args := make([]Value, len(x.Args))
	for i, a := range x.Args {
		args[i] = in.expr(a, e)
	}
	if fn, ok := in.funcs[name]; ok {
		return in.call(fn, args, x.Sp)
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
