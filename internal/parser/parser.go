package parser

import (
	"fmt"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/diag"
	"github.com/AxelGard/bronze/internal/lexer"
)

// Parse lexes and parses a source file. It always returns a tree, with
// Bad nodes where there were syntax errors, and all lexer and parser errors.
func Parse(src *diag.Source) (*ast.File, diag.List) {
	toks, diags := lexer.Lex(src)
	p := &parser{toks: toks, diags: diags}
	f := p.file()
	return f, p.diags
}

type parser struct {
	toks  []lexer.Token
	pos   int
	diags diag.List
}

// bailout is panicked after a syntax error, and recovered at the statement
// or declaration level, so each statement reports at most one error.
type bailout struct{}

func (p *parser) tok() lexer.Token     { return p.toks[p.pos] }
func (p *parser) kind() lexer.Kind     { return p.toks[p.pos].Kind }
func (p *parser) at(k lexer.Kind) bool { return p.kind() == k }

func (p *parser) advance() lexer.Token {
	t := p.toks[p.pos]
	if t.Kind != lexer.EOF {
		p.pos++
	}
	return t
}

// prev is the span of the token before the current one.
func (p *parser) prev() diag.Span {
	if p.pos == 0 {
		return p.tok().Span
	}
	return p.toks[p.pos-1].Span
}

func (p *parser) skipNewlines() {
	for p.at(lexer.NEWLINE) {
		p.advance()
	}
}

// fail reports an error at span and bails out of the current statement.
func (p *parser) fail(span diag.Span, format string, args ...any) {
	p.diags.Errorf(span, format, args...)
	panic(bailout{})
}

// failHere reports an error at the current token. If the token is ILLEGAL,
// the lexer has already reported it, so it only bails out.
func (p *parser) failHere(format string, args ...any) {
	if p.at(lexer.ILLEGAL) {
		panic(bailout{})
	}
	p.fail(p.tok().Span, format, args...)
}

func (p *parser) expect(k lexer.Kind) lexer.Token {
	if !p.at(k) {
		p.failHere("expected %s, found %s", k, describe(p.tok()))
	}
	return p.advance()
}

func describe(t lexer.Token) string {
	switch t.Kind {
	case lexer.EOF:
		return "end of file"
	case lexer.NEWLINE:
		return "end of line"
	case lexer.IDENT, lexer.INT, lexer.FLOAT, lexer.RESERVED:
		return "`" + t.Text + "`"
	case lexer.STRING:
		return "a string"
	}
	return t.Kind.String()
}

// notYet reports a feature that is in Bronze but not in this version.
func (p *parser) notYet(span diag.Span, what string) {
	p.fail(span, "%s not supported yet", what)
}

func (p *parser) ident() *ast.Ident {
	t := p.tok()
	switch {
	case t.Kind == lexer.RESERVED:
		p.fail(t.Span, "`%s` is reserved for a later version, and can't be used as a name", t.Text)
	case t.Kind != lexer.IDENT:
		p.failHere("expected a name, found %s", describe(t))
	case t.Text == "_":
		p.fail(t.Span, "`_` can only be used as a match pattern")
	}
	p.advance()
	return &ast.Ident{Name: t.Text, Sp: t.Span}
}

// ---------------------------------------------------------------------
// Declarations

func (p *parser) file() *ast.File {
	f := &ast.File{}
	p.skipNewlines()
	for !p.at(lexer.EOF) {
		f.Decls = append(f.Decls, p.decl())
		p.skipNewlines()
	}
	f.Sp = diag.Span{Start: p.toks[0].Span.Start, End: p.tok().Span.End}
	return f
}

func (p *parser) decl() (d ast.Decl) {
	start := p.pos
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(bailout); !ok {
				panic(r)
			}
			p.syncDecl()
			d = &ast.BadDecl{Sp: p.toks[start].Span.To(p.prev())}
		}
	}()

	switch p.kind() {
	case lexer.FUNC:
		fn := p.funcDecl()
		p.endOfLine("declaration")
		return fn
	case lexer.CLASS:
		p.notYet(p.tok().Span, "classes are")
	case lexer.ENUM:
		p.notYet(p.tok().Span, "enums are")
	}
	p.failHere("expected `func`, `class` or `enum`, found %s", describe(p.tok()))
	return nil
}

// syncDecl skips to the next line that starts with `func`, `class` or
// `enum` outside of any braces.
func (p *parser) syncDecl() {
	depth := 0
	for !p.at(lexer.EOF) {
		switch p.kind() {
		case lexer.LBRACE:
			depth++
		case lexer.RBRACE:
			depth = max(depth-1, 0)
		case lexer.FUNC, lexer.CLASS, lexer.ENUM:
			if depth == 0 && p.pos > 0 && p.toks[p.pos-1].Kind == lexer.NEWLINE {
				return
			}
		}
		p.advance()
	}
}

func (p *parser) funcDecl() *ast.FuncDecl {
	start := p.expect(lexer.FUNC).Span
	fn := &ast.FuncDecl{Name: p.ident()}
	p.expect(lexer.LPAREN)
	if !p.at(lexer.RPAREN) {
		for {
			fn.Params = append(fn.Params, p.param())
			if !p.at(lexer.COMMA) {
				break
			}
			p.advance()
		}
	}
	p.expect(lexer.RPAREN)
	if p.at(lexer.ARROW) {
		p.advance()
		if p.at(lexer.BANG) {
			p.notYet(p.tok().Span, "functions that can fail are")
		}
		fn.Result = p.typeExpr()
		if p.at(lexer.BANG) {
			p.notYet(p.tok().Span, "functions that can fail are")
		}
	}
	fn.Body = p.block()
	fn.Sp = start.To(fn.Body.Sp)
	return fn
}

func (p *parser) param() *ast.Param {
	name := p.ident()
	if !p.at(lexer.COLON) {
		p.failHere("expected `:` and a type after parameter `%s`, found %s", name.Name, describe(p.tok()))
	}
	p.advance()
	t := p.typeExpr()
	return &ast.Param{Name: name, Type: t, Sp: name.Sp.To(t.Sp)}
}

func (p *parser) typeExpr() *ast.TypeExpr {
	if !p.at(lexer.IDENT) && !p.at(lexer.RESERVED) {
		p.failHere("expected a type, found %s", describe(p.tok()))
	}
	name := p.ident()
	t := &ast.TypeExpr{Name: name, Sp: name.Sp}
	if p.at(lexer.QUEST) {
		p.notYet(p.tok().Span, "`?` types are")
	}
	return t
}

// endOfLine requires the end of a statement or declaration: a newline, or
// a `}` that closes the enclosing block (not consumed).
func (p *parser) endOfLine(what string) {
	switch p.kind() {
	case lexer.NEWLINE:
		p.advance()
	case lexer.RBRACE, lexer.EOF:
	default:
		p.failHere("expected the end of the %s, found %s", what, describe(p.tok()))
	}
}

// ---------------------------------------------------------------------
// Statements

func (p *parser) block() *ast.Block {
	p.skipNewlines()
	start := p.expect(lexer.LBRACE).Span
	b := &ast.Block{}
	p.skipNewlines()
	for !p.at(lexer.RBRACE) {
		if p.at(lexer.EOF) {
			p.diags.Errorf(p.tok().Span, "expected `}`, found end of file").Notes = []string{
				fmt.Sprintf("the block starts at line %d", start.Start.Line),
			}
			b.Sp = start.To(p.tok().Span)
			b.Rbrace = p.tok().Span
			return b
		}
		b.Stmts = append(b.Stmts, p.stmt())
		p.skipNewlines()
	}
	b.Rbrace = p.advance().Span
	b.Sp = start.To(b.Rbrace)
	return b
}

// stmt parses one statement and the newline after it. On a syntax error it
// skips to the end of the statement and returns a BadStmt.
func (p *parser) stmt() (s ast.Stmt) {
	start := p.pos
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(bailout); !ok {
				panic(r)
			}
			p.syncStmt()
			s = &ast.BadStmt{Sp: p.toks[start].Span.To(p.prev())}
		}
	}()
	s = p.simpleStmt()
	p.endOfLine("statement")
	return s
}

// syncStmt skips to the end of the current statement: the next newline
// outside of braces, or the `}` that closes the enclosing block.
func (p *parser) syncStmt() {
	depth := 0
	for !p.at(lexer.EOF) {
		switch p.kind() {
		case lexer.LBRACE:
			depth++
		case lexer.RBRACE:
			if depth == 0 {
				return
			}
			depth--
		case lexer.NEWLINE:
			if depth == 0 {
				p.advance()
				return
			}
		}
		p.advance()
	}
}

func (p *parser) simpleStmt() ast.Stmt {
	switch p.kind() {
	case lexer.CON, lexer.MUT:
		return p.varDecl()
	case lexer.IF:
		return p.ifStmt()
	case lexer.WHILE:
		start := p.advance().Span
		cond := p.expr()
		body := p.block()
		return &ast.While{Cond: cond, Body: body, Sp: start.To(body.Sp)}
	case lexer.RETURN:
		t := p.advance()
		r := &ast.Return{Sp: t.Span}
		if !p.at(lexer.NEWLINE) && !p.at(lexer.RBRACE) && !p.at(lexer.EOF) {
			r.Value = p.expr()
			r.Sp = t.Span.To(r.Value.Span())
		}
		return r
	case lexer.LBRACE:
		p.failHere("unexpected `{`")
	case lexer.RESERVED:
		p.fail(p.tok().Span, "`%s` is reserved for a later version", p.tok().Text)
	case lexer.FUNC:
		p.failHere("functions can only be declared at the top level")
	}

	x := p.expr()
	if p.at(lexer.ASSIGN) {
		p.advance()
		if _, ok := x.(*ast.Ident); !ok {
			p.fail(x.Span(), "cannot assign to this expression")
		}
		v := p.expr()
		return &ast.Assign{Target: x, Value: v, Sp: x.Span().To(v.Span())}
	}
	if _, ok := x.(*ast.Call); !ok {
		if _, bad := x.(*ast.BadExpr); !bad {
			d := p.diags.Errorf(x.Span(), "this expression is not used")
			d.Notes = []string{"only a call can be used as a statement"}
			panic(bailout{})
		}
	}
	return &ast.ExprStmt{X: x}
}

func (p *parser) varDecl() *ast.VarDecl {
	t := p.advance()
	d := &ast.VarDecl{Mut: t.Kind == lexer.MUT, Name: p.ident()}
	if p.at(lexer.COLON) {
		p.advance()
		d.Type = p.typeExpr()
	}
	if !p.at(lexer.ASSIGN) {
		p.failHere("expected `=` and a value for `%s`, found %s", d.Name.Name, describe(p.tok()))
	}
	p.advance()
	d.Value = p.expr()
	d.Sp = t.Span.To(d.Value.Span())
	return d
}

func (p *parser) ifStmt() *ast.If {
	start := p.expect(lexer.IF).Span
	s := &ast.If{Cond: p.expr()}
	s.Then = p.block()
	s.Sp = start.To(s.Then.Sp)

	// `else` may be on the line after the `}`.
	save := p.pos
	p.skipNewlines()
	if !p.at(lexer.ELSE) {
		p.pos = save
		return s
	}
	p.advance()
	if p.at(lexer.IF) {
		s.Else = p.ifStmt()
	} else {
		s.Else = p.block()
	}
	s.Sp = start.To(s.Else.Span())
	return s
}

// ---------------------------------------------------------------------
// Expressions

func (p *parser) expr() ast.Expr {
	x := p.orExpr()
	if p.at(lexer.ELSE) {
		p.notYet(p.tok().Span, "`else` for errors and null is")
	}
	return x
}

func (p *parser) orExpr() ast.Expr {
	x := p.andExpr()
	for p.at(lexer.OR) {
		op := p.advance()
		y := p.andExpr()
		x = &ast.Binary{Op: op.Kind, X: x, Y: y, OpSp: op.Span}
	}
	return x
}

func (p *parser) andExpr() ast.Expr {
	x := p.cmpExpr()
	for p.at(lexer.AND) {
		op := p.advance()
		y := p.cmpExpr()
		x = &ast.Binary{Op: op.Kind, X: x, Y: y, OpSp: op.Span}
	}
	return x
}

func isCmp(k lexer.Kind) bool {
	switch k {
	case lexer.EQ, lexer.NE, lexer.LT, lexer.LE, lexer.GT, lexer.GE:
		return true
	}
	return false
}

func (p *parser) cmpExpr() ast.Expr {
	x := p.addExpr()
	if p.at(lexer.IS) {
		p.notYet(p.tok().Span, "`is` is")
	}
	if !isCmp(p.kind()) {
		return x
	}
	op := p.advance()
	y := p.addExpr()
	if isCmp(p.kind()) {
		d := p.diags.Errorf(p.tok().Span, "comparisons can't be chained, use `and`")
		d.Help = []string{"write `a < b and b < c` instead of `a < b < c`"}
		panic(bailout{})
	}
	return &ast.Binary{Op: op.Kind, X: x, Y: y, OpSp: op.Span}
}

func (p *parser) addExpr() ast.Expr {
	x := p.mulExpr()
	for p.at(lexer.PLUS) || p.at(lexer.MINUS) {
		op := p.advance()
		y := p.mulExpr()
		x = &ast.Binary{Op: op.Kind, X: x, Y: y, OpSp: op.Span}
	}
	return x
}

func (p *parser) mulExpr() ast.Expr {
	x := p.unary()
	for p.at(lexer.STAR) || p.at(lexer.SLASH) || p.at(lexer.PERCENT) {
		op := p.advance()
		y := p.unary()
		x = &ast.Binary{Op: op.Kind, X: x, Y: y, OpSp: op.Span}
	}
	return x
}

func (p *parser) unary() ast.Expr {
	if p.at(lexer.NOT) || p.at(lexer.MINUS) {
		op := p.advance()
		x := p.unary()
		return &ast.Unary{Op: op.Kind, X: x, Sp: op.Span.To(x.Span())}
	}
	return p.postfix()
}

func (p *parser) postfix() ast.Expr {
	x := p.primary()
	for {
		switch p.kind() {
		case lexer.LPAREN:
			open := p.advance().Span
			call := &ast.Call{Fn: x}
			if !p.at(lexer.RPAREN) {
				for {
					call.Args = append(call.Args, p.expr())
					if !p.at(lexer.COMMA) {
						break
					}
					p.advance()
				}
			}
			if !p.at(lexer.RPAREN) {
				p.closeParen(open, "`,` or `)` in the argument list")
			}
			call.Sp = x.Span().To(p.advance().Span)
			x = call
		case lexer.DOT:
			p.notYet(p.tok().Span, "fields and methods are")
		default:
			return x
		}
	}
}

// closeParen reports a missing `)`. Newlines don't end statements inside
// ( ), so if the next token is on a later line, the `(` was most likely
// never closed: point at it.
func (p *parser) closeParen(open diag.Span, expected string) {
	if p.at(lexer.EOF) || p.tok().Span.Start.Line > p.prev().End.Line {
		p.fail(open, "`(` is not closed")
	}
	p.failHere("expected %s, found %s", expected, describe(p.tok()))
}

func (p *parser) primary() ast.Expr {
	t := p.tok()
	switch t.Kind {
	case lexer.INT:
		p.advance()
		return &ast.IntLit{Value: t.Int, Sp: t.Span}
	case lexer.FLOAT:
		p.advance()
		return &ast.FloatLit{Value: t.Float, Sp: t.Span}
	case lexer.STRING:
		p.advance()
		return &ast.StringLit{Value: t.Str, Sp: t.Span}
	case lexer.TRUE, lexer.FALSE:
		p.advance()
		return &ast.BoolLit{Value: t.Kind == lexer.TRUE, Sp: t.Span}
	case lexer.IDENT, lexer.RESERVED:
		return p.ident()
	case lexer.LPAREN:
		p.advance()
		x := p.expr()
		if !p.at(lexer.RPAREN) {
			p.closeParen(t.Span, "`)`")
		}
		end := p.advance().Span
		return &ast.Paren{X: x, Sp: t.Span.To(end)}
	case lexer.NULL:
		p.notYet(t.Span, "`null` is")
	case lexer.THIS:
		p.notYet(t.Span, "`this` is")
	case lexer.MATCH:
		p.notYet(t.Span, "`match` is")
	}
	p.failHere("expected an expression, found %s", describe(t))
	return nil
}
