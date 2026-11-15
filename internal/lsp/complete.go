package lsp

import (
	"unicode"
	"unicode/utf8"

	"github.com/AxelGard/bronze/internal/check"
	"github.com/AxelGard/bronze/internal/diag"
	"github.com/AxelGard/bronze/internal/parser"
)

// LSP completion item kinds.
var completionKinds = map[check.CompletionKind]int{
	check.CompleteVariable: 6,
	check.CompleteFunction: 3,
	check.CompleteMethod:   2,
	check.CompleteField:    5,
	check.CompleteClass:    7,
	check.CompleteEnum:     13,
	check.CompleteCase:     20,
	check.CompleteKeyword:  14,
	check.CompleteType:     25,
}

// completion lists what can be written at the cursor. The word under the
// cursor gets check.Marker and `()` after it, and the file is checked
// again: the checker then reports what fits where the marker is. The
// editor filters the items by the part of the word already written.
func (s *server) completion(p textDocumentPositionParams) any {
	doc := s.docs[p.TextDocument.URI]
	if doc == nil {
		return nil
	}
	text := doc.src.Text
	at := s.toOffset(doc.src, p.Position)
	end := at
	for end < len(text) {
		r, size := utf8.DecodeRune(text[end:])
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}
		end += size
	}
	marked := append(append(append([]byte{}, text[:end]...), check.Marker+"()"...), text[end:]...)
	src := diag.NewSource(doc.src.Name, marked)
	file, _ := parser.Parse(src)
	comps := check.Complete(file, src)

	items := comps.Items
	if !comps.Found {
		if !inType(text, at) {
			return []completionItem{}
		}
		items = comps.Types
	}
	out := make([]completionItem, 0, len(items))
	for _, it := range items {
		out = append(out, completionItem{Label: it.Label, Kind: completionKinds[it.Kind], Detail: it.Detail})
	}
	return out
}

// inType reports whether the word at offset at is in a written type: right
// after `:`, `->`, `<`, `!` or a `,` inside `< >`.
func inType(text []byte, at int) bool {
	i := at
	for i > 0 {
		r, size := utf8.DecodeLastRune(text[:i])
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}
		i -= size
	}
	for i > 0 && (text[i-1] == ' ' || text[i-1] == '\t') {
		i--
	}
	if i == 0 {
		return false
	}
	switch text[i-1] {
	case ':', '<', '!':
		return true
	case '>':
		return i >= 2 && text[i-2] == '-'
	case ',':
		// a `,` between type arguments: an unclosed `<` on this line
		depth := 0
		for j := i - 1; j > 0 && text[j-1] != '\n'; j-- {
			switch text[j-1] {
			case '>':
				depth--
			case '<':
				depth++
				if depth > 0 {
					return true
				}
			case '(', '[', '{':
				return false
			}
		}
	}
	return false
}
