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

// FuncDecl is `func name(params) -> result ! fail { body }`.
type FuncDecl struct {
	Name   *Ident
	Params []*Param
	Result *TypeExpr // nil if the function returns no value
	Fail   *TypeExpr // the error type after `!`, or nil
	Body   *Block
	Bad    bool // the body had a syntax error
	Sp     diag.Span
}

// Param is `name: Type`.
type Param struct {
	Name *Ident
	Type *TypeExpr
	Sp   diag.Span
}

// TypeExpr is a written type: `Int`, `Account?`, `List<Int>`.
type TypeExpr struct {
	Name     *Ident
	Args     []*TypeExpr // the type arguments, as in `Map<String, Int>`
	Optional bool        // T?
	Sp       diag.Span
}

// ClassDecl is `class Name use Error { fields and methods }`.
type ClassDecl struct {
	Name     *Ident
	UseError bool // `use Error`
	Fields   []*Field
	Methods  []*FuncDecl
	Bad      bool // some member had a syntax error
	Builtin  bool // declared by bronze itself, like Error
	Sp       diag.Span
}

// Field is `con name: Type = value` or `mut name = value` in a class.
type Field struct {
	Mut   bool
	Name  *Ident
	Type  *TypeExpr // nil if not written
	Value Expr
	Sp    diag.Span
}

// EnumDecl is `enum Name { cases and methods }`.
type EnumDecl struct {
	Name    *Ident
	Cases   []*Ident
	Methods []*FuncDecl
	Bad     bool // some member had a syntax error
	Sp      diag.Span
}

// BadDecl is a declaration with a syntax error.
type BadDecl struct {
	Name *Ident // the declared name, if it could be read
	Sp   diag.Span
}

func (d *FuncDecl) Span() diag.Span  { return d.Sp }
func (d *BadDecl) Span() diag.Span   { return d.Sp }
func (d *ClassDecl) Span() diag.Span { return d.Sp }
func (d *EnumDecl) Span() diag.Span  { return d.Sp }
func (f *Field) Span() diag.Span     { return f.Sp }
func (p *Param) Span() diag.Span     { return p.Sp }
func (t *TypeExpr) Span() diag.Span  { return t.Sp }

func (*FuncDecl) declNode()  {}
func (*BadDecl) declNode()   {}
func (*ClassDecl) declNode() {}
func (*EnumDecl) declNode()  {}

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
	Target Expr // an *Ident, a *Member or an *Index
	Value  Expr
	Sp     diag.Span
}

// Compound is `target op= value`. Op is the arithmetic operator:
// lexer.PLUS for `+=`, and so on.
type Compound struct {
	Op     lexer.Kind
	Target Expr // an *Ident, a *Member or an *Index
	Value  Expr
	OpSp   diag.Span // the `op=`
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

// For is `for x in xs { body }` or `for i, x in xs { body }`. A loop
// variable written `_` is an Ident named "_".
type For struct {
	Vars []*Ident // one or two
	X    Expr
	Body *Block
	Sp   diag.Span
}

// Break is `break`.
type Break struct {
	Sp diag.Span
}

// Continue is `continue`.
type Continue struct {
	Sp diag.Span
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
func (s *Compound) Span() diag.Span { return s.Sp }
func (s *For) Span() diag.Span      { return s.Sp }
func (s *Break) Span() diag.Span    { return s.Sp }
func (s *Continue) Span() diag.Span { return s.Sp }
func (s *If) Span() diag.Span       { return s.Sp }
func (s *While) Span() diag.Span    { return s.Sp }
func (s *Return) Span() diag.Span   { return s.Sp }
func (s *ExprStmt) Span() diag.Span { return s.X.Span() }
func (s *BadStmt) Span() diag.Span  { return s.Sp }

func (*Block) stmtNode()    {}
func (*VarDecl) stmtNode()  {}
func (*Assign) stmtNode()   {}
func (*Compound) stmtNode() {}
func (*For) stmtNode()      {}
func (*Break) stmtNode()    {}
func (*Continue) stmtNode() {}
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

// Interp is a string with interpolations: `"a {x} b"` has Strs "a " and
// " b", and Exprs x. There is always one more string than expressions.
type Interp struct {
	Strs  []string
	Exprs []Expr
	Sp    diag.Span
}

// BoolLit is `true` or `false`.
type BoolLit struct {
	Value bool
	Sp    diag.Span
}

// NullLit is `null`.
type NullLit struct {
	Sp diag.Span
}

// Is is `x is Class`.
type Is struct {
	X     Expr
	Class *Ident
}

// ElseKind says what an `else` handles. The checker sets it.
type ElseKind int

const (
	ElseError ElseKind = iota // a failure of a call that can fail
	ElseNull                  // a null
)

// Else is `x else handler`:
//
//	x else return            Return, no Value
//	x else return v          Return, Value
//	x else (e) return v      Err, Return, Value
//	x else (e) { ... }       Err, Body
//	x else fallback          Value
type Else struct {
	X      Expr
	Err    *Ident // the bound error, or nil
	Return bool
	Value  Expr   // the returned value or the fallback; nil for a bare return
	Body   *Block // the block, or nil
	Kind   ElseKind
	Sp     diag.Span
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

// Range is `lo..hi`, or `lo..=hi` when Inclusive. Hi is nil only after a
// syntax error the checker reports: `a..` outside `[ ]`.
type Range struct {
	Lo, Hi    Expr
	Inclusive bool
	OpSp      diag.Span
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

// Match is `match x { pattern => value ... }`.
type Match struct {
	X    Expr
	Arms []*Arm
	Bad  bool // some arm had a syntax error
	Sp   diag.Span
}

// Arm is `pattern => value`, or `pattern => { block }` when the match is a
// statement. Exactly one of Value and Body is set.
type Arm struct {
	Pattern *Pattern
	Value   Expr
	Body    *Block
}

// PatternKind is the kind of a match pattern.
type PatternKind int

const (
	PatInt    PatternKind = iota // 42, -1
	PatString                    // "text"
	PatBool                      // true, false
	PatCase                      // Enum.Case
	PatAny                       // _
)

// Pattern is a match pattern.
type Pattern struct {
	Kind PatternKind
	Int  int64
	Str  string
	Bool bool
	Enum *Ident // for PatCase
	Case *Ident // for PatCase
	Sp   diag.Span
}

// ListLit is `[a, b, c]`.
type ListLit struct {
	Elems []Expr
	Sp    diag.Span
}

// MapLit is `{k: v, ...}`.
type MapLit struct {
	Entries []*Entry
	Sp      diag.Span
}

// Entry is `key: value` in a map literal.
type Entry struct {
	Key, Value Expr
}

// Index is `x[i]`.
type Index struct {
	X     Expr
	Index Expr
	Sp    diag.Span
}

// Slice is `x[lo..hi]` or `x[lo..=hi]`. Lo and Hi are nil when left out.
type Slice struct {
	X         Expr
	Lo, Hi    Expr
	Inclusive bool
	Sp        diag.Span
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
func (e *Interp) Span() diag.Span    { return e.Sp }
func (e *Unary) Span() diag.Span     { return e.Sp }
func (e *NullLit) Span() diag.Span   { return e.Sp }
func (e *Is) Span() diag.Span        { return e.X.Span().To(e.Class.Sp) }
func (e *Else) Span() diag.Span      { return e.Sp }
func (e *Binary) Span() diag.Span    { return e.X.Span().To(e.Y.Span()) }
func (e *Range) Span() diag.Span {
	if e.Hi == nil {
		return e.Lo.Span().To(e.OpSp)
	}
	return e.Lo.Span().To(e.Hi.Span())
}
func (e *ListLit) Span() diag.Span { return e.Sp }
func (e *MapLit) Span() diag.Span  { return e.Sp }
func (e *Index) Span() diag.Span   { return e.Sp }
func (e *Slice) Span() diag.Span   { return e.Sp }
func (e *Call) Span() diag.Span    { return e.Sp }
func (e *Member) Span() diag.Span  { return e.X.Span().To(e.Name.Sp) }
func (e *This) Span() diag.Span    { return e.Sp }
func (e *Match) Span() diag.Span   { return e.Sp }
func (p *Pattern) Span() diag.Span { return p.Sp }
func (e *Paren) Span() diag.Span   { return e.Sp }
func (e *BadExpr) Span() diag.Span { return e.Sp }

func (*Ident) exprNode()     {}
func (*IntLit) exprNode()    {}
func (*FloatLit) exprNode()  {}
func (*StringLit) exprNode() {}
func (*BoolLit) exprNode()   {}
func (*Interp) exprNode()    {}
func (*Unary) exprNode()     {}
func (*NullLit) exprNode()   {}
func (*Is) exprNode()        {}
func (*Else) exprNode()      {}
func (*Binary) exprNode()    {}
func (*Range) exprNode()     {}
func (*ListLit) exprNode()   {}
func (*MapLit) exprNode()    {}
func (*Index) exprNode()     {}
func (*Slice) exprNode()     {}
func (*Call) exprNode()      {}
func (*Member) exprNode()    {}
func (*This) exprNode()      {}
func (*Match) exprNode()     {}
func (*Paren) exprNode()     {}
func (*BadExpr) exprNode()   {}
