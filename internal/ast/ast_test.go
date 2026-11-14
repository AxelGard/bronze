package ast

import (
	"testing"

	"github.com/AxelGard/bronze/internal/diag"
	"github.com/AxelGard/bronze/internal/lexer"
)

func span(start, end int) diag.Span {
	return diag.Span{Start: diag.Pos{Offset: start, Line: 1, Col: start + 1}, End: diag.Pos{Offset: end, Line: 1, Col: end + 1}}
}

func TestBinarySpan(t *testing.T) {
	// a + b
	x := &Ident{Name: "a", Sp: span(0, 1)}
	y := &Ident{Name: "b", Sp: span(4, 5)}
	b := &Binary{Op: lexer.PLUS, X: x, Y: y, OpSp: span(2, 3)}
	if s := b.Span(); s.Start.Offset != 0 || s.End.Offset != 5 {
		t.Errorf("span %+v", s)
	}
	if got := Dump(b); got != "(+ a b)" {
		t.Errorf("dump %q", got)
	}
}
