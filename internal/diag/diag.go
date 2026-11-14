package diag

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Pos is a position in a source file. Line and Col are 1-based, and Col
// counts characters (runes), not bytes. Offset is the 0-based byte offset.
type Pos struct {
	Offset int
	Line   int
	Col    int
}

// Before reports whether p comes before q.
func (p Pos) Before(q Pos) bool { return p.Offset < q.Offset }

// Span is a range of source text, from Start (inclusive) to End (exclusive).
type Span struct {
	Start Pos
	End   Pos
}

// To returns the span from the start of s to the end of t.
func (s Span) To(t Span) Span { return Span{s.Start, t.End} }

// Source is one source file.
type Source struct {
	Name  string
	Text  []byte
	lines []int // byte offset of the start of each line
}

// NewSource makes a Source and indexes its lines.
func NewSource(name string, text []byte) *Source {
	s := &Source{Name: name, Text: text, lines: []int{0}}
	for i, b := range text {
		if b == '\n' {
			s.lines = append(s.lines, i+1)
		}
	}
	return s
}

// Line returns the text of line n (1-based), without its line ending.
func (s *Source) Line(n int) string {
	if n < 1 || n > len(s.lines) {
		return ""
	}
	start := s.lines[n-1]
	end := len(s.Text)
	if n < len(s.lines) {
		end = s.lines[n] - 1
	}
	return strings.TrimRight(string(s.Text[start:end]), "\r")
}

// LineStart returns the byte offset where line n (1-based) starts, or the
// end of the text if there is no such line.
func (s *Source) LineStart(n int) int {
	if n < 1 {
		return 0
	}
	if n > len(s.lines) {
		return len(s.Text)
	}
	return s.lines[n-1]
}

// Lines returns the number of lines.
func (s *Source) Lines() int { return len(s.lines) }

// PosAt returns the position of a byte offset.
func (s *Source) PosAt(offset int) Pos {
	line := sort.Search(len(s.lines), func(i int) bool { return s.lines[i] > offset }) - 1
	col := utf8.RuneCount(s.Text[s.lines[line]:offset]) + 1
	return Pos{Offset: offset, Line: line + 1, Col: col}
}

// Diagnostic is one error found in a source file.
type Diagnostic struct {
	Span    Span
	Message string   // starts lowercase, no trailing period
	Label   string   // optional, printed after the carets
	Help    []string // optional "= help:" lines
	Notes   []string // optional "= note:" lines
}

// List collects diagnostics.
type List []*Diagnostic

// Errorf adds a diagnostic and returns it, so help and notes can be added.
func (l *List) Errorf(span Span, format string, args ...any) *Diagnostic {
	d := &Diagnostic{Span: span, Message: fmt.Sprintf(format, args...)}
	*l = append(*l, d)
	return d
}

// Sort puts the diagnostics in source order. Diagnostics at the same
// position keep their order.
func (l List) Sort() {
	sort.SliceStable(l, func(i, j int) bool { return l[i].Span.Start.Before(l[j].Span.Start) })
}

// Format writes the diagnostics in source order to w:
//
//	error: cannot add Int and Float
//	  --> core.brz:14:12
//	   |
//	14 |     return a + 2.0
//	   |            ^^^^^^^ Int + Float
//	   |
//	   = help: convert explicitly: Float(a) + 2.0
//
// Several diagnostics are separated by a blank line.
func Format(w io.Writer, src *Source, diags List) {
	diags.Sort()
	for i, d := range diags {
		if i > 0 {
			fmt.Fprintln(w)
		}
		formatOne(w, src, d)
	}
}

func formatOne(w io.Writer, src *Source, d *Diagnostic) {
	start := d.Span.Start
	fmt.Fprintf(w, "error: %s\n", d.Message)
	fmt.Fprintf(w, "  --> %s:%d:%d\n", src.Name, start.Line, start.Col)

	num := strconv.Itoa(start.Line)
	pad := strings.Repeat(" ", len(num))
	line := src.Line(start.Line)

	// Underline to the end of the span, or to the end of the line if the
	// span covers several lines. At least one caret.
	width := 1
	if d.Span.End.Line == start.Line {
		width = d.Span.End.Col - start.Col
	} else {
		width = utf8.RuneCountInString(line) - start.Col + 1
	}
	width = max(width, 1)

	// Keep tabs in the indentation so the carets line up.
	var indent strings.Builder
	for i, r := range []rune(line) {
		if i >= start.Col-1 {
			break
		}
		if r == '\t' {
			indent.WriteRune('\t')
		} else {
			indent.WriteRune(' ')
		}
	}

	fmt.Fprintf(w, "%s |\n", pad)
	fmt.Fprintf(w, "%s | %s\n", num, line)
	carets := strings.Repeat("^", width)
	if d.Label != "" {
		carets += " " + d.Label
	}
	fmt.Fprintf(w, "%s | %s%s\n", pad, indent.String(), carets)
	if len(d.Help) > 0 || len(d.Notes) > 0 {
		fmt.Fprintf(w, "%s |\n", pad)
	}
	for _, h := range d.Help {
		fmt.Fprintf(w, "%s = help: %s\n", pad, h)
	}
	for _, n := range d.Notes {
		fmt.Fprintf(w, "%s = note: %s\n", pad, n)
	}
}
