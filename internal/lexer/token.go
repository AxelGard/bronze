package lexer

import "github.com/AxelGard/bronze/internal/diag"

// Kind is the kind of a token.
type Kind int

const (
	EOF Kind = iota
	NEWLINE
	ILLEGAL // a character or literal the lexer reported an error for

	IDENT
	INT
	FLOAT
	STRING
	STRHEAD  // `"text{`: a string up to its first interpolation
	STRMID   // `}text{`: between two interpolations
	STREND   // `}text"`: after the last interpolation
	RESERVED // a word reserved for a later version, such as `data`

	// operators
	PLUS    // +
	MINUS   // -
	STAR    // *
	SLASH   // /
	PERCENT // %
	EQ      // ==
	NE      // !=
	LT      // <
	LE      // <=
	GT      // >
	GE      // >=
	ASSIGN  // =
	ARROW   // ->
	FATARR  // =>
	BANG    // !
	QUEST   // ?
	DOT     // .
	DOTDOT  // ..
	DOTDOTE // ..=
	PLUSEQ  // +=
	MINUSEQ // -=
	STAREQ  // *=
	SLASHEQ // /=
	PERCEQ  // %=

	// punctuation
	LPAREN // (
	RPAREN // )
	LBRACE // {
	RBRACE // }
	LBRACK // [
	RBRACK // ]
	COMMA  // ,
	COLON  // :

	// keywords
	FUNC
	RETURN
	CON
	MUT
	IF
	ELSE
	WHILE
	AND
	OR
	NOT
	TRUE
	FALSE
	CLASS
	ENUM
	USE
	THIS
	NULL
	MATCH
	IS
	FOR
	IN
	BREAK
	CONTINUE
	STATIC
	OVERRIDE
	SUPER
	PRIVATE
	PROTECTED
)

var kindNames = [...]string{
	EOF:       "end of file",
	NEWLINE:   "newline",
	ILLEGAL:   "illegal token",
	IDENT:     "identifier",
	INT:       "integer",
	FLOAT:     "float",
	STRING:    "string",
	STRHEAD:   "string",
	STRMID:    "`}`",
	STREND:    "`}`",
	RESERVED:  "reserved word",
	PLUS:      "`+`",
	MINUS:     "`-`",
	STAR:      "`*`",
	SLASH:     "`/`",
	PERCENT:   "`%`",
	EQ:        "`==`",
	NE:        "`!=`",
	LT:        "`<`",
	LE:        "`<=`",
	GT:        "`>`",
	GE:        "`>=`",
	ASSIGN:    "`=`",
	ARROW:     "`->`",
	FATARR:    "`=>`",
	BANG:      "`!`",
	QUEST:     "`?`",
	DOT:       "`.`",
	DOTDOT:    "`..`",
	DOTDOTE:   "`..=`",
	PLUSEQ:    "`+=`",
	MINUSEQ:   "`-=`",
	STAREQ:    "`*=`",
	SLASHEQ:   "`/=`",
	PERCEQ:    "`%=`",
	LPAREN:    "`(`",
	RPAREN:    "`)`",
	LBRACE:    "`{`",
	RBRACE:    "`}`",
	LBRACK:    "`[`",
	RBRACK:    "`]`",
	COMMA:     "`,`",
	COLON:     "`:`",
	FUNC:      "`func`",
	RETURN:    "`return`",
	CON:       "`con`",
	MUT:       "`mut`",
	IF:        "`if`",
	ELSE:      "`else`",
	WHILE:     "`while`",
	AND:       "`and`",
	OR:        "`or`",
	NOT:       "`not`",
	TRUE:      "`true`",
	FALSE:     "`false`",
	CLASS:     "`class`",
	ENUM:      "`enum`",
	USE:       "`use`",
	THIS:      "`this`",
	NULL:      "`null`",
	MATCH:     "`match`",
	IS:        "`is`",
	FOR:       "`for`",
	IN:        "`in`",
	BREAK:     "`break`",
	CONTINUE:  "`continue`",
	STATIC:    "`static`",
	OVERRIDE:  "`override`",
	SUPER:     "`super`",
	PRIVATE:   "`private`",
	PROTECTED: "`protected`",
}

// String returns a description of the kind for error messages, such as
// "`(`" or "identifier".
func (k Kind) String() string { return kindNames[k] }

var keywords = map[string]Kind{
	"func":      FUNC,
	"return":    RETURN,
	"con":       CON,
	"mut":       MUT,
	"if":        IF,
	"else":      ELSE,
	"while":     WHILE,
	"and":       AND,
	"or":        OR,
	"not":       NOT,
	"true":      TRUE,
	"false":     FALSE,
	"class":     CLASS,
	"enum":      ENUM,
	"use":       USE,
	"this":      THIS,
	"null":      NULL,
	"match":     MATCH,
	"is":        IS,
	"for":       FOR,
	"in":        IN,
	"break":     BREAK,
	"continue":  CONTINUE,
	"static":    STATIC,
	"override":  OVERRIDE,
	"super":     SUPER,
	"private":   PRIVATE,
	"protected": PROTECTED,
}

var reserved = map[string]bool{
	"data": true, "interface": true, "namespace": true, "import": true,
	"test": true, "type": true, "copy": true,
}

// Token is one token of source text.
type Token struct {
	Kind Kind
	Text string // the source text; for NEWLINE, "\n"
	Span diag.Span

	Int   int64   // value of an INT
	Float float64 // value of a FLOAT
	Str   string  // value of a STRING, with escapes resolved
}

// continues reports whether a newline after a token of this kind is not
// the end of a statement, because the expression goes on.
func (k Kind) continues() bool {
	switch k {
	case PLUS, MINUS, STAR, SLASH, PERCENT, EQ, NE, LT, LE, GT, GE,
		AND, OR, NOT, IS, COMMA, ARROW, FATARR, ASSIGN, DOT, BANG,
		DOTDOT, DOTDOTE, PLUSEQ, MINUSEQ, STAREQ, SLASHEQ, PERCEQ:
		return true
	}
	return false
}
