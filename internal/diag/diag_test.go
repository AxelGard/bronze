package diag

import (
	"strings"
	"testing"
)

func TestPosAt(t *testing.T) {
	src := NewSource("a.brz", []byte("ab\nçd\n"))
	tests := []struct {
		offset, line, col int
	}{
		{0, 1, 1},
		{1, 1, 2},
		{3, 2, 1},
		{5, 2, 2}, // ç is two bytes
		{7, 3, 1},
	}
	for _, tt := range tests {
		p := src.PosAt(tt.offset)
		if p.Line != tt.line || p.Col != tt.col {
			t.Errorf("PosAt(%d) = %d:%d, want %d:%d", tt.offset, p.Line, p.Col, tt.line, tt.col)
		}
	}
	if got := src.Line(2); got != "çd" {
		t.Errorf("Line(2) = %q", got)
	}
}

func TestFormat(t *testing.T) {
	text := "func f(a: Int) -> Float\n{\n    return a + 2.0\n}\n"
	src := NewSource("core.brz", []byte(text))
	start := strings.Index(text, "a + 2.0")
	var l List
	d := l.Errorf(Span{src.PosAt(start), src.PosAt(start + 7)}, "cannot add Int and Float")
	d.Label = "Int + Float"
	d.Help = append(d.Help, "convert explicitly: Float(a) + 2.0")
	l.Errorf(Span{src.PosAt(0), src.PosAt(4)}, "first")

	var b strings.Builder
	Format(&b, src, l)
	want := `error: first
  --> core.brz:1:1
  |
1 | func f(a: Int) -> Float
  | ^^^^

error: cannot add Int and Float
  --> core.brz:3:12
  |
3 |     return a + 2.0
  |            ^^^^^^^ Int + Float
  |
  = help: convert explicitly: Float(a) + 2.0
`
	if b.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", b.String(), want)
	}
}
