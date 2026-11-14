package lsp

import (
	"unicode/utf8"

	"github.com/AxelGard/bronze/internal/diag"
)

// LSP positions are a 0-based line and a column counted in UTF-16 code
// units (the default) or bytes (if the client offers "utf-8").

// units is the width of r in the position encoding.
func (s *server) units(r rune) int {
	if s.encoding == "utf-8" {
		return utf8.RuneLen(r)
	}
	if r >= 0x10000 {
		return 2
	}
	return 1
}

// toPosition converts a byte offset to an LSP position.
func (s *server) toPosition(src *diag.Source, offset int) position {
	p := src.PosAt(offset)
	start := src.LineStart(p.Line)
	col := 0
	for _, r := range string(src.Text[start:offset]) {
		col += s.units(r)
	}
	return position{Line: p.Line - 1, Character: col}
}

func (s *server) toRange(src *diag.Source, span diag.Span) lspRange {
	if src == nil {
		return lspRange{}
	}
	return lspRange{Start: s.toPosition(src, span.Start.Offset), End: s.toPosition(src, span.End.Offset)}
}

// toOffset converts an LSP position to a byte offset, clamped to the line.
func (s *server) toOffset(src *diag.Source, pos position) int {
	if pos.Line >= src.Lines() {
		return len(src.Text)
	}
	offset := src.LineStart(pos.Line + 1)
	col := 0
	for col < pos.Character && offset < len(src.Text) {
		r, size := utf8.DecodeRune(src.Text[offset:])
		if r == '\n' {
			break
		}
		col += s.units(r)
		offset += size
	}
	return offset
}
