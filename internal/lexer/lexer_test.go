package lexer

import (
	"strings"
	"testing"

	"github.com/AxelGard/bronze/internal/diag"
)

func lex(t *testing.T, text string) ([]Token, diag.List) {
	t.Helper()
	return Lex(diag.NewSource("t.brz", []byte(text)))
}

// kinds writes the tokens compactly: words and operators as their text,
// newlines as `;`, literals as their kind. EOF is left out.
func kinds(toks []Token) string {
	var parts []string
	for _, t := range toks {
		switch t.Kind {
		case EOF:
		case NEWLINE:
			parts = append(parts, ";")
		case INT, FLOAT, STRING:
			parts = append(parts, strings.ToUpper(t.Kind.String()))
		default:
			parts = append(parts, t.Text)
		}
	}
	return strings.Join(parts, " ")
}

func TestNewlines(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		{"simple", "con x = 1\nx", "con x = INTEGER ; x ;"},
		{"blank lines collapse", "\n\na\n\n\nb\n\n", "a ; b ;"},
		{"allman braces", "func f()\n{\n    g()\n}\n", "func f ( ) ; { ; g ( ) ; } ;"},
		{"after binary operator", "a +\nb", "a + b ;"},
		{"after and", "a and\nb", "a and b ;"},
		{"after comma", "f(a,\nb)", "f ( a , b ) ;"},
		{"inside parens", "f(\na\n,\nb\n)", "f ( a , b ) ;"},
		{"braces inside parens", "f(match x\n{\n_ => 1\n})", "f ( match x { ; _ => INTEGER ; } ) ;"},
		{"after arrow", "func f() ->\nInt", "func f ( ) -> Int ;"},
		{"after equals", "con x =\n1", "con x = INTEGER ;"},
		{"after dot", "a.\nb", "a . b ;"},
		{"comments", "a // one\n// two\n/* three\n */ b", "a ; b ;"},
		{"block comment same line", "a /* x */ b", "a b ;"},
		{"no newline before operator", "a\n+ b", "a ; + b ;"},
		{"else on next line", "}\nelse", "} ; else ;"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toks, diags := lex(t, tt.src)
			if len(diags) > 0 {
				t.Fatalf("unexpected error: %s", diags[0].Message)
			}
			got := kinds(toks)
			if got != tt.want {
				t.Errorf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestLiterals(t *testing.T) {
	toks, diags := lex(t, `42 1_000_000 2.0 3.14159 1.5e-3 2e10 "a\n\t\"\\\{b"`)
	if len(diags) > 0 {
		t.Fatalf("unexpected error: %s", diags[0].Message)
	}
	if toks[0].Int != 42 || toks[1].Int != 1000000 {
		t.Errorf("ints: %d %d", toks[0].Int, toks[1].Int)
	}
	floats := []float64{2.0, 3.14159, 1.5e-3, 2e10}
	for i, f := range floats {
		if toks[2+i].Kind != FLOAT || toks[2+i].Float != f {
			t.Errorf("float %d: %v %v", i, toks[2+i].Kind, toks[2+i].Float)
		}
	}
	if s := toks[6].Str; s != "a\n\t\"\\{b" {
		t.Errorf("string: %q", s)
	}
}

func TestKeywords(t *testing.T) {
	toks, _ := lex(t, "func con mut for data x")
	want := []Kind{FUNC, CON, MUT, FOR, RESERVED, IDENT}
	for i, k := range want {
		if toks[i].Kind != k {
			t.Errorf("token %d: got %v, want %v", i, toks[i].Kind, k)
		}
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		src, msg string
		col      int
	}{
		{`"abc`, "string is not closed", 1},
		{`x = "a{b}"`, "string interpolation is not supported yet", 7},
		{`"\q"`, "unknown escape `\\q` in string", 2},
		{"a /* b", "block comment is not closed", 3},
		{"99999999999999999999", "integer literal `99999999999999999999` is too large", 1},
		{"12abc", "invalid number `12abc`", 1},
		{"a $ b", "unexpected character '$'", 3},
	}
	for _, tt := range tests {
		_, diags := lex(t, tt.src)
		if len(diags) == 0 {
			t.Errorf("%q: no error", tt.src)
			continue
		}
		if diags[0].Message != tt.msg || diags[0].Span.Start.Col != tt.col {
			t.Errorf("%q: got %q at col %d, want %q at col %d", tt.src, diags[0].Message, diags[0].Span.Start.Col, tt.msg, tt.col)
		}
	}
}

func TestPositions(t *testing.T) {
	toks, _ := lex(t, "a\n  bc")
	b := toks[2]
	if b.Span.Start.Line != 2 || b.Span.Start.Col != 3 || b.Span.End.Col != 5 {
		t.Errorf("bc at %+v", b.Span)
	}
}

func TestOperators02(t *testing.T) {
	toks, _ := lex(t, "a..b ..= += -= *= /= %= [ ] .")
	want := []Kind{IDENT, DOTDOT, IDENT, DOTDOTE, PLUSEQ, MINUSEQ, STAREQ, SLASHEQ, PERCEQ, LBRACK, RBRACK, DOT}
	for i, k := range want {
		if toks[i].Kind != k {
			t.Errorf("token %d: got %v, want %v", i, toks[i].Kind, k)
		}
	}
	toks, _ = lex(t, "1..5")
	if toks[0].Kind != INT || toks[1].Kind != DOTDOT || toks[2].Kind != INT {
		t.Errorf("1..5: %v %v %v", toks[0].Kind, toks[1].Kind, toks[2].Kind)
	}
}

func TestNoNewlineInBrackets(t *testing.T) {
	toks, _ := lex(t, "[1,\n2\n]")
	for _, tok := range toks[:len(toks)-2] {
		if tok.Kind == NEWLINE {
			t.Fatalf("NEWLINE inside [ ]")
		}
	}
}
