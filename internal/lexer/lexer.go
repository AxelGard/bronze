package lexer

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/AxelGard/bronze/internal/diag"
)

// Lex turns a source file into tokens. The last token is always EOF, and
// the token before it is a NEWLINE unless the file has no tokens.
func Lex(src *diag.Source) ([]Token, diag.List) {
	l := &lexer{src: src, text: src.Text}
	l.run()
	return l.tokens, l.diags
}

type lexer struct {
	src    *diag.Source
	text   []byte
	pos    int // current byte offset
	tokens []Token
	diags  diag.List

	brackets []Kind // open (, [ and {, innermost last
	newline  bool   // a newline was seen since the last token
	nlPos    int    // offset of that newline
}

func (l *lexer) span(start, end int) diag.Span {
	return diag.Span{Start: l.src.PosAt(start), End: l.src.PosAt(end)}
}

func (l *lexer) errorf(start, end int, format string, args ...any) *diag.Diagnostic {
	return l.diags.Errorf(l.span(start, end), format, args...)
}

func (l *lexer) peek(n int) byte {
	if l.pos+n < len(l.text) {
		return l.text[l.pos+n]
	}
	return 0
}

// emit adds a token, first adding a NEWLINE if one is pending and it ends a
// statement here.
func (l *lexer) emit(t Token) {
	if l.newline {
		l.newline = false
		if l.wantNewline() {
			l.tokens = append(l.tokens, Token{Kind: NEWLINE, Text: "\n", Span: l.span(l.nlPos, l.nlPos+1)})
		}
	}
	l.tokens = append(l.tokens, t)
}

// wantNewline reports whether a newline at this point ends a statement.
func (l *lexer) wantNewline() bool {
	if len(l.tokens) == 0 {
		return false
	}
	if n := len(l.brackets); n > 0 && l.brackets[n-1] != LBRACE {
		return false
	}
	last := l.tokens[len(l.tokens)-1].Kind
	return last != NEWLINE && !last.continues()
}

func (l *lexer) run() {
	for {
		l.skipSpaceAndComments()
		if l.pos >= len(l.text) {
			break
		}
		l.next()
	}
	// End the last statement, then EOF.
	if len(l.tokens) > 0 && l.tokens[len(l.tokens)-1].Kind != NEWLINE {
		l.tokens = append(l.tokens, Token{Kind: NEWLINE, Text: "\n", Span: l.span(len(l.text), len(l.text))})
	}
	l.tokens = append(l.tokens, Token{Kind: EOF, Span: l.span(len(l.text), len(l.text))})
}

func (l *lexer) skipSpaceAndComments() {
	for l.pos < len(l.text) {
		c := l.text[l.pos]
		switch {
		case c == '\n':
			if !l.newline {
				l.newline, l.nlPos = true, l.pos
			}
			l.pos++
		case c == ' ' || c == '\t' || c == '\r':
			l.pos++
		case c == '/' && l.peek(1) == '/':
			for l.pos < len(l.text) && l.text[l.pos] != '\n' {
				l.pos++
			}
		case c == '/' && l.peek(1) == '*':
			start := l.pos
			end := strings.Index(string(l.text[l.pos+2:]), "*/")
			if end < 0 {
				l.errorf(start, start+2, "block comment is not closed").Help = []string{"add `*/` to end it"}
				l.pos = len(l.text)
				return
			}
			comment := l.text[l.pos : l.pos+2+end+2]
			if i := strings.IndexByte(string(comment), '\n'); i >= 0 && !l.newline {
				l.newline, l.nlPos = true, l.pos+i
			}
			l.pos += len(comment)
		default:
			return
		}
	}
}

var ops = []struct {
	text string
	kind Kind
}{
	// longest first
	{"..=", DOTDOTE},
	{"==", EQ}, {"!=", NE}, {"<=", LE}, {">=", GE}, {"->", ARROW}, {"=>", FATARR},
	{"..", DOTDOT}, {"+=", PLUSEQ}, {"-=", MINUSEQ}, {"*=", STAREQ}, {"/=", SLASHEQ}, {"%=", PERCEQ},
	{"+", PLUS}, {"-", MINUS}, {"*", STAR}, {"/", SLASH}, {"%", PERCENT},
	{"<", LT}, {">", GT}, {"=", ASSIGN}, {"!", BANG}, {"?", QUEST}, {".", DOT},
	{"(", LPAREN}, {")", RPAREN}, {"{", LBRACE}, {"}", RBRACE}, {"[", LBRACK}, {"]", RBRACK}, {",", COMMA}, {":", COLON},
}

func (l *lexer) next() {
	start := l.pos
	c := l.text[l.pos]
	r, size := utf8.DecodeRune(l.text[l.pos:])

	switch {
	case isLetter(r):
		l.word()
		return
	case c >= '0' && c <= '9':
		l.number()
		return
	case c == '"':
		l.string()
		return
	}

	for _, op := range ops {
		if strings.HasPrefix(string(l.text[l.pos:min(l.pos+3, len(l.text))]), op.text) {
			l.pos += len(op.text)
			// Emit first: a newline before `)` is still inside the ( ).
			l.emit(Token{Kind: op.kind, Text: op.text, Span: l.span(start, l.pos)})
			l.bracket(op.kind)
			return
		}
	}

	l.pos += size
	l.errorf(start, l.pos, "unexpected character %q", r)
}

// bracket tracks open brackets, for the newline rule inside ( ) and [ ].
func (l *lexer) bracket(k Kind) {
	switch k {
	case LPAREN, LBRACK, LBRACE:
		l.brackets = append(l.brackets, k)
	case RPAREN, RBRACK, RBRACE:
		// Pop to the matching bracket; mismatches are the parser's problem.
		open := map[Kind]Kind{RPAREN: LPAREN, RBRACK: LBRACK, RBRACE: LBRACE}[k]
		for i := len(l.brackets) - 1; i >= 0; i-- {
			if l.brackets[i] == open {
				l.brackets = l.brackets[:i]
				break
			}
		}
	}
}

func isLetter(r rune) bool { return r == '_' || unicode.IsLetter(r) }

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func (l *lexer) word() {
	start := l.pos
	for l.pos < len(l.text) {
		r, size := utf8.DecodeRune(l.text[l.pos:])
		if !isLetter(r) && !unicode.IsDigit(r) {
			break
		}
		l.pos += size
	}
	text := string(l.text[start:l.pos])
	kind := IDENT
	if k, ok := keywords[text]; ok {
		kind = k
	} else if reserved[text] {
		kind = RESERVED
	}
	l.emit(Token{Kind: kind, Text: text, Span: l.span(start, l.pos)})
}

// digits reads digits with single underscores between them.
func (l *lexer) digits() {
	for l.pos < len(l.text) {
		c := l.text[l.pos]
		if isDigit(c) {
			l.pos++
		} else if c == '_' && isDigit(l.peek(1)) && l.pos > 0 && isDigit(l.text[l.pos-1]) {
			l.pos++
		} else {
			break
		}
	}
}

func (l *lexer) number() {
	start := l.pos
	l.digits()
	isFloat := false
	if l.peek(0) == '.' && isDigit(l.peek(1)) {
		isFloat = true
		l.pos++
		l.digits()
	}
	if c := l.peek(0); c == 'e' || c == 'E' {
		n := 1
		if s := l.peek(1); s == '+' || s == '-' {
			n = 2
		}
		if isDigit(l.peek(n)) {
			isFloat = true
			l.pos += n
			l.digits()
		}
	}
	// Letters, digits or `_` stuck to a number, as in `12abc` or `1__0`.
	bad := false
	for l.pos < len(l.text) {
		r, size := utf8.DecodeRune(l.text[l.pos:])
		if !isLetter(r) && !unicode.IsDigit(r) {
			break
		}
		bad = true
		l.pos += size
	}
	text := string(l.text[start:l.pos])
	tok := Token{Kind: INT, Text: text, Span: l.span(start, l.pos)}
	if bad {
		l.errorf(start, l.pos, "invalid number `%s`", text)
		tok.Kind = ILLEGAL
		l.emit(tok)
		return
	}
	clean := strings.ReplaceAll(text, "_", "")
	if isFloat {
		tok.Kind = FLOAT
		f, err := strconv.ParseFloat(clean, 64)
		if err != nil {
			l.errorf(start, l.pos, "float literal `%s` is out of range", text)
			tok.Kind = ILLEGAL
		}
		tok.Float = f
	} else {
		n, err := strconv.ParseInt(clean, 10, 64)
		if err != nil {
			l.errorf(start, l.pos, "integer literal `%s` is too large", text).Notes = []string{"the largest Int is 9223372036854775807"}
			tok.Kind = ILLEGAL
		}
		tok.Int = n
	}
	l.emit(tok)
}

func (l *lexer) string() {
	start := l.pos
	l.pos++ // opening quote
	var b strings.Builder
	ok := true
	for {
		if l.pos >= len(l.text) || l.text[l.pos] == '\n' {
			l.errorf(start, start+1, "string is not closed").Help = []string{"add a `\"` at the end of the string"}
			ok = false
			break
		}
		c := l.text[l.pos]
		if c == '"' {
			l.pos++
			break
		}
		if c == '{' {
			d := l.errorf(l.pos, l.pos+1, "string interpolation is not supported yet")
			d.Help = []string{"write `\\{` for a literal `{`, or join strings with `+`"}
			ok = false
			l.pos++
			continue
		}
		if c == '\\' {
			esc := l.peek(1)
			switch esc {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '"', '\\', '{':
				b.WriteByte(esc)
			default:
				r, size := utf8.DecodeRune(l.text[l.pos+1:])
				if esc == '\n' || esc == 0 {
					size = 0
				}
				d := l.errorf(l.pos, l.pos+1+size, "unknown escape `\\%c` in string", r)
				d.Notes = []string{"the escapes are \\n \\t \\\" \\\\ \\{"}
				ok = false
				l.pos += 1 + size
				continue
			}
			l.pos += 2
			continue
		}
		r, size := utf8.DecodeRune(l.text[l.pos:])
		b.WriteRune(r)
		l.pos += size
	}
	tok := Token{Kind: STRING, Text: string(l.text[start:l.pos]), Span: l.span(start, l.pos), Str: b.String()}
	if !ok {
		tok.Kind = ILLEGAL
	}
	l.emit(tok)
}
