package ast

import (
	"github.com/AxelGard/bronze/internal/diag"
	"github.com/AxelGard/bronze/internal/lexer"
)

// Node is any node of the tree.
type Node interface {
	Span() diag.Span
}

// Decl is a top-level declaration.
type Decl interface {
	Node
	declNode()
}

// Stmt is a statement.
type Stmt interface {
	Node
	stmtNode()
}

// Expr is an expression.
type Expr interface {
	Node
	exprNode()
}

// File is a whole source file.
type File struct {
	Decls []Decl
	Sp    diag.Span
}

func (f *File) Span() diag.Span { return f.Sp }

// ---------------------------------------------------------------------
// Declarations

// FuncDecl is `func name(params) -> result { body }`.
type FuncDecl struct {
	Name   *Ident
	Params []*Param
	Result *TypeExpr // nil if the function returns nothing
	Body   *Block
	Sp     diag.Span
}

// Param is `name: Type`.
type Param struct {
	Name *Ident
	Type *TypeExpr
	Sp   diag.Span
}

// TypeExpr is a written type: `Int`, `Account?`.
type TypeExpr struct {
	Name     *Ident
	Optional bool // T?
	Sp       diag.Span
}

// ClassDecl is `class Name { fields and methods }`.
type ClassDecl struct {
	Name    *Ident
	Fields  []*Field
	Methods []*FuncDecl
	Bad     bool // some member had a syntax error
	Sp      diag.Span
}

// Field is `con name: Type = value` or `mut name = value` in a class.
type Field struct {
	Mut   bool
	Name  *Ident
	Type  *TypeExpr // nil if not written
	Value Expr
	Sp    diag.Span
}

// BadDecl is a declaration with a syntax error.
type BadDecl struct {
	Name *Ident // the declared name, if it could be read
	Sp   diag.Span
}

func (d *FuncDecl) Span() diag.Span  { return d.Sp }
func (d *BadDecl) Span() diag.Span   { return d.Sp }
func (d *ClassDecl) Span() diag.Span { return d.Sp }
func (f *Field) Span() diag.Span     { return f.Sp }
func (p *Param) Span() diag.Span     { return p.Sp }
func (t *TypeExpr) Span() diag.Span  { return t.Sp }

func (*FuncDecl) declNode()  {}
func (*BadDecl) declNode()   {}
func (*ClassDecl) declNode() {}

// ---------------------------------------------------------------------
// Statements

// Block is `{ stmts }`.
type Block struct {
	Stmts  []Stmt
	Sp     diag.Span
	Rbrace diag.Span
}

// VarDecl is `con name: Type = value` or `mut name = value`.
type VarDecl struct {
	Mut   bool
	Name  *Ident
	Type  *TypeExpr // nil if not written
	Value Expr
	Sp    diag.Span
}

// Assign is `target = value`.
type Assign struct {
	Target Expr // an *Ident or a *Member
	Value  Expr
	Sp     diag.Span
}

// If is `if cond { then } else ...`. Else is nil, a *Block, or an *If.
type If struct {
	Cond Expr
	Then *Block
	Else Stmt
	Sp   diag.Span
}

// While is `while cond { body }`.
type While struct {
	Cond Expr
	Body *Block
	Sp   diag.Span
}

// Return is `return` or `return value`.
type Return struct {
	Value Expr // nil for a bare return
	Sp    diag.Span
}

// ExprStmt is an expression used as a statement (a call).
type ExprStmt struct {
	X Expr
}

// BadStmt is a statement with a syntax error.
type BadStmt struct {
	Sp diag.Span
}

func (s *Block) Span() diag.Span    { return s.Sp }
func (s *VarDecl) Span() diag.Span  { return s.Sp }
func (s *Assign) Span() diag.Span   { return s.Sp }
func (s *If) Span() diag.Span       { return s.Sp }
func (s *While) Span() diag.Span    { return s.Sp }
func (s *Return) Span() diag.Span   { return s.Sp }
func (s *ExprStmt) Span() diag.Span { return s.X.Span() }
func (s *BadStmt) Span() diag.Span  { return s.Sp }

func (*Block) stmtNode()    {}
func (*VarDecl) stmtNode()  {}
func (*Assign) stmtNode()   {}
func (*If) stmtNode()       {}
func (*While) stmtNode()    {}
func (*Return) stmtNode()   {}
func (*ExprStmt) stmtNode() {}
func (*BadStmt) stmtNode()  {}

// ---------------------------------------------------------------------
// Expressions

// Ident is a name.
type Ident struct {
	Name string
	Sp   diag.Span
}

// IntLit is an Int literal.
type IntLit struct {
	Value int64
	Sp    diag.Span
}

// FloatLit is a Float literal.
type FloatLit struct {
	Value float64
	Sp    diag.Span
}

// StringLit is a String literal, with escapes resolved.
type StringLit struct {
	Value string
	Sp    diag.Span
}

// BoolLit is `true` or `false`.
type BoolLit struct {
	Value bool
	Sp    diag.Span
}

// Unary is `-x` or `not x`. Op is lexer.MINUS or lexer.NOT.
type Unary struct {
	Op lexer.Kind
	X  Expr
	Sp diag.Span
}

// Binary is `x op y`, for arithmetic, comparisons, `and` and `or`.
type Binary struct {
	Op   lexer.Kind
	X, Y Expr
	OpSp diag.Span
}

// Member is `x.name`: a field, or a method when it is called.
type Member struct {
	X    Expr
	Name *Ident
}

// This is `this`.
type This struct {
	Sp diag.Span
}

// Call is `fn(args)`.
type Call struct {
	Fn   Expr
	Args []Expr
	Sp   diag.Span
}

// Paren is `(x)`.
type Paren struct {
	X  Expr
	Sp diag.Span
}

// BadExpr is an expression with a syntax error.
type BadExpr struct {
	Sp diag.Span
}

func (e *Ident) Span() diag.Span     { return e.Sp }
func (e *IntLit) Span() diag.Span    { return e.Sp }
func (e *FloatLit) Span() diag.Span  { return e.Sp }
func (e *StringLit) Span() diag.Span { return e.Sp }
func (e *BoolLit) Span() diag.Span   { return e.Sp }
func (e *Unary) Span() diag.Span     { return e.Sp }
func (e *Binary) Span() diag.Span    { return e.X.Span().To(e.Y.Span()) }
func (e *Call) Span() diag.Span      { return e.Sp }
func (e *Member) Span() diag.Span    { return e.X.Span().To(e.Name.Sp) }
func (e *This) Span() diag.Span      { return e.Sp }
func (e *Paren) Span() diag.Span     { return e.Sp }
func (e *BadExpr) Span() diag.Span   { return e.Sp }

func (*Ident) exprNode()     {}
func (*IntLit) exprNode()    {}
func (*FloatLit) exprNode()  {}
func (*StringLit) exprNode() {}
func (*BoolLit) exprNode()   {}
func (*Unary) exprNode()     {}
func (*Binary) exprNode()    {}
func (*Call) exprNode()      {}
func (*Member) exprNode()    {}
func (*This) exprNode()      {}
func (*Paren) exprNode()     {}
func (*BadExpr) exprNode()   {}
