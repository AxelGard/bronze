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

	recovered int // how many times the parser recovered from an error
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

// failNote is failHere with a note.
func (p *parser) failNote(note, format string, args ...any) {
	if !p.at(lexer.ILLEGAL) {
		p.diags.Errorf(p.tok().Span, format, args...).Notes = []string{note}
	}
	panic(bailout{})
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
			p.recovered++
			bad := &ast.BadDecl{}
			// Keep the name, so that uses of it don't cause more errors.
			if k := p.toks[start].Kind; (k == lexer.FUNC || k == lexer.CLASS || k == lexer.ENUM) &&
				p.toks[start+1].Kind == lexer.IDENT {
				t := p.toks[start+1]
				bad.Name = &ast.Ident{Name: t.Text, Sp: t.Span}
			}
			p.syncDecl()
			bad.Sp = p.toks[start].Span.To(p.prev())
			d = bad
		}
	}()

	switch p.kind() {
	case lexer.FUNC:
		fn := p.funcDecl()
		p.endOfLine("declaration")
		return fn
	case lexer.CLASS:
		c := p.classDecl()
		p.endOfLine("declaration")
		return c
	case lexer.ENUM:
		e := p.enumDecl()
		p.endOfLine("declaration")
		return e
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
		if !p.at(lexer.BANG) {
			fn.Result = p.typeExpr()
		}
		if p.at(lexer.BANG) {
			p.advance()
			fn.Fail = p.typeExpr()
		}
	}
	before := p.recovered
	fn.Body = p.block()
	fn.Bad = p.recovered > before
	fn.Sp = start.To(fn.Body.Sp)
	return fn
}

func (p *parser) classDecl() *ast.ClassDecl {
	start := p.expect(lexer.CLASS).Span
	c := &ast.ClassDecl{Name: p.ident()}
	if p.at(lexer.USE) {
		p.advance()
		t := p.tok()
		if t.Kind != lexer.IDENT || t.Text != "Error" {
			p.failNote("`use` only works with `Error` for now",
				"expected `Error` after `use`, found %s", describe(t))
		}
		p.advance()
		c.UseError = true
	}
	p.skipNewlines()
	p.expect(lexer.LBRACE)
	p.skipNewlines()
	for !p.at(lexer.RBRACE) {
		if p.at(lexer.EOF) {
			p.diags.Errorf(p.tok().Span, "expected `}`, found end of file").Notes = []string{
				fmt.Sprintf("the class starts at line %d", start.Start.Line),
			}
			c.Bad = true
			c.Sp = start.To(p.tok().Span)
			return c
		}
		p.member(c)
		p.skipNewlines()
	}
	c.Sp = start.To(p.advance().Span)
	return c
}

// member parses one field or method into c. On a syntax error it skips to
// the end of the member and marks the class as bad.
func (p *parser) member(c *ast.ClassDecl) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(bailout); !ok {
				panic(r)
			}
			p.recovered++
			c.Bad = true
			p.syncStmt()
		}
	}()
	switch p.kind() {
	case lexer.CON, lexer.MUT:
		c.Fields = append(c.Fields, p.field())
	case lexer.FUNC:
		c.Methods = append(c.Methods, p.funcDecl())
	default:
		p.failHere("expected a field (`con` or `mut`) or a method (`func`), found %s", describe(p.tok()))
	}
	p.endOfLine("member")
}

func (p *parser) enumDecl() *ast.EnumDecl {
	start := p.expect(lexer.ENUM).Span
	e := &ast.EnumDecl{Name: p.ident()}
	p.skipNewlines()
	p.expect(lexer.LBRACE)
	p.skipNewlines()
	for !p.at(lexer.RBRACE) {
		if p.at(lexer.EOF) {
			p.diags.Errorf(p.tok().Span, "expected `}`, found end of file").Notes = []string{
				fmt.Sprintf("the enum starts at line %d", start.Start.Line),
			}
			e.Bad = true
			e.Sp = start.To(p.tok().Span)
			return e
		}
		p.enumMember(e)
		p.skipNewlines()
	}
	e.Sp = start.To(p.advance().Span)
	return e
}

// enumMember parses one case or method into e, recovering like member.
func (p *parser) enumMember(e *ast.EnumDecl) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(bailout); !ok {
				panic(r)
			}
			p.recovered++
			e.Bad = true
			p.syncStmt()
		}
	}()
	switch p.kind() {
	case lexer.IDENT, lexer.RESERVED:
		e.Cases = append(e.Cases, p.ident())
		p.endOfLine("case")
	case lexer.FUNC:
		e.Methods = append(e.Methods, p.funcDecl())
		p.endOfLine("method")
	default:
		p.failNote("cases are written one per line, as plain names",
			"expected a case or a method (`func`), found %s", describe(p.tok()))
	}
}

func (p *parser) field() *ast.Field {
	t := p.advance()
	f := &ast.Field{Mut: t.Kind == lexer.MUT, Name: p.ident()}
	if p.at(lexer.COLON) {
		p.advance()
		f.Type = p.typeExpr()
	}
	if !p.at(lexer.ASSIGN) {
		d := p.diags.Errorf(p.tok().Span, "field `%s` needs an initial value", f.Name.Name)
		d.Help = []string{fmt.Sprintf("give it a value: `%s %s = ...`", t.Text, f.Name.Name)}
		panic(bailout{})
	}
	p.advance()
	f.Value = p.expr()
	f.Sp = t.Span.To(f.Value.Span())
	return f
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
		t.Optional = true
		t.Sp = name.Sp.To(p.advance().Span)
		if p.at(lexer.QUEST) {
			p.fail(p.tok().Span, "`%s??` is not allowed: a `?` type is already nullable", name.Name)
		}
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
			p.recovered++
			p.syncStmt()
			s = &ast.BadStmt{Sp: p.toks[start].Span.To(p.prev())}
		}
	}()
	s = p.simpleStmt()
	p.endOfLine("statement")
	return s
}

// syncStmt skips to the end of the current statement: the next newline
// outside of braces, or the `}` that closes the enclosing block. A block
// that starts on the next line, as in
//
//	if x is Foo
//	{
//
// belongs to the statement and is skipped too.
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
				if !p.nextLineIsBlock() {
					return
				}
				continue
			}
		}
		p.advance()
	}
}

// nextLineIsBlock reports whether the next token after any newlines is `{`,
// and if so moves to it.
func (p *parser) nextLineIsBlock() bool {
	i := p.pos
	for p.toks[i].Kind == lexer.NEWLINE {
		i++
	}
	if p.toks[i].Kind != lexer.LBRACE {
		return false
	}
	p.pos = i
	return true
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
	case lexer.ELSE:
		d := p.diags.Errorf(p.tok().Span, "`else` must be on the same line as the expression it handles")
		d.Notes = []string{"only the `else` of an `if` can be on the line after the `}`"}
		panic(bailout{})
	}

	x := p.expr()
	if p.at(lexer.ASSIGN) {
		p.advance()
		switch x.(type) {
		case *ast.Ident, *ast.Member:
		case *ast.This:
			p.fail(x.Span(), "cannot assign to `this`")
		default:
			p.fail(x.Span(), "cannot assign to this expression")
		}
		v := p.expr()
		return &ast.Assign{Target: x, Value: v, Sp: x.Span().To(v.Span())}
	}
	switch x.(type) {
	case *ast.Call, *ast.Match, *ast.Else, *ast.BadExpr:
	default:
		d := p.diags.Errorf(x.Span(), "this expression is not used")
		d.Notes = []string{"only a call or a match can be used as a statement"}
		panic(bailout{})
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
	if !p.at(lexer.ELSE) {
		return x
	}
	p.advance()
	e := &ast.Else{X: x}
	if p.at(lexer.LPAREN) && p.peekKind(1) == lexer.IDENT && p.peekKind(2) == lexer.RPAREN {
		// `(e)` binds the error if `return` or a block follows.
		after := p.pos + 3
		for p.toks[after].Kind == lexer.NEWLINE {
			after++
		}
		if k := p.toks[after].Kind; k == lexer.RETURN || k == lexer.LBRACE {
			p.advance()
			e.Err = p.ident()
			p.advance()
		}
	}
	switch {
	case e.Err != nil && !p.at(lexer.RETURN):
		e.Body = p.block()
	case p.at(lexer.RETURN):
		p.advance()
		e.Return = true
		if canStartExpr(p.kind()) {
			e.Value = p.orExpr()
		}
	default:
		e.Value = p.orExpr()
	}
	e.Sp = x.Span()
	switch {
	case e.Body != nil:
		e.Sp = e.Sp.To(e.Body.Sp)
	case e.Value != nil:
		e.Sp = e.Sp.To(e.Value.Span())
	default:
		e.Sp = e.Sp.To(p.prev())
	}
	if p.at(lexer.ELSE) {
		d := p.diags.Errorf(p.tok().Span, "an expression can only have one `else`")
		d.Help = []string{"handle one thing at a time: `con a = f() else return`, then `con b = a else ...`"}
		panic(bailout{})
	}
	return e
}

func (p *parser) peekKind(n int) lexer.Kind {
	if p.pos+n < len(p.toks) {
		return p.toks[p.pos+n].Kind
	}
	return lexer.EOF
}

// canStartExpr reports whether a token can start an expression.
func canStartExpr(k lexer.Kind) bool {
	switch k {
	case lexer.INT, lexer.FLOAT, lexer.STRING, lexer.TRUE, lexer.FALSE, lexer.NULL,
		lexer.THIS, lexer.IDENT, lexer.LPAREN, lexer.MINUS, lexer.NOT, lexer.MATCH:
		return true
	}
	return false
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
		p.advance()
		x = &ast.Is{X: x, Class: p.ident()}
		if isCmp(p.kind()) || p.at(lexer.IS) {
			p.failHere("`is` can't be combined with another comparison; use `and`")
		}
		return x
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
			p.advance()
			x = &ast.Member{X: x, Name: p.ident()}
		default:
			return x
		}
	}
}

func (p *parser) match() *ast.Match {
	start := p.expect(lexer.MATCH).Span
	m := &ast.Match{X: p.expr()}
	p.skipNewlines()
	p.expect(lexer.LBRACE)
	p.skipNewlines()
	for !p.at(lexer.RBRACE) {
		if p.at(lexer.EOF) {
			p.fail(p.tok().Span, "expected `}` to end the match that starts at line %d, found end of file", start.Start.Line)
		}
		p.arm(m)
		p.skipNewlines()
	}
	m.Sp = start.To(p.advance().Span)
	return m
}

// arm parses `pattern => value` into m. On a syntax error it skips to the
// next arm and marks the match as bad.
func (p *parser) arm(m *ast.Match) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(bailout); !ok {
				panic(r)
			}
			p.recovered++
			m.Bad = true
			p.syncStmt()
		}
	}()
	pat := p.pattern()
	if !p.at(lexer.FATARR) {
		p.failHere("expected `=>` after the pattern, found %s", describe(p.tok()))
	}
	p.advance()
	arm := &ast.Arm{Pattern: pat}
	if p.at(lexer.LBRACE) {
		arm.Body = p.block()
	} else {
		arm.Value = p.expr()
	}
	m.Arms = append(m.Arms, arm)
	p.endOfLine("arm")
}

func (p *parser) pattern() *ast.Pattern {
	t := p.tok()
	switch t.Kind {
	case lexer.INT:
		p.advance()
		return &ast.Pattern{Kind: ast.PatInt, Int: t.Int, Sp: t.Span}
	case lexer.MINUS:
		p.advance()
		n := p.tok()
		if n.Kind != lexer.INT {
			p.failHere("expected a number after `-` in a pattern, found %s", describe(n))
		}
		p.advance()
		return &ast.Pattern{Kind: ast.PatInt, Int: -n.Int, Sp: t.Span.To(n.Span)}
	case lexer.STRING:
		p.advance()
		return &ast.Pattern{Kind: ast.PatString, Str: t.Str, Sp: t.Span}
	case lexer.TRUE, lexer.FALSE:
		p.advance()
		return &ast.Pattern{Kind: ast.PatBool, Bool: t.Kind == lexer.TRUE, Sp: t.Span}
	case lexer.IDENT:
		p.advance()
		if t.Text == "_" {
			return &ast.Pattern{Kind: ast.PatAny, Sp: t.Span}
		}
		if p.at(lexer.DOT) {
			p.advance()
			c := p.ident()
			return &ast.Pattern{Kind: ast.PatCase, Enum: &ast.Ident{Name: t.Text, Sp: t.Span}, Case: c, Sp: t.Span.To(c.Sp)}
		}
		p.diags.Errorf(t.Span, "`%s` is not a pattern", t.Text).Notes = []string{patternNote}
		panic(bailout{})
	case lexer.FLOAT:
		p.fail(t.Span, "Float values can't be used as patterns")
	}
	p.failNote(patternNote, "expected a pattern, found %s", describe(t))
	return nil
}

const patternNote = "a pattern is an Int, String or Bool literal, a case like `Color.Red`, or `_`"

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
		p.advance()
		return &ast.NullLit{Sp: t.Span}
	case lexer.THIS:
		p.advance()
		return &ast.This{Sp: t.Span}
	case lexer.MATCH:
		return p.match()
	}
	p.failHere("expected an expression, found %s", describe(t))
	return nil
}
