package check

import (
	"fmt"
	"strings"
	"testing"

	"github.com/AxelGard/bronze/internal/diag"
	"github.com/AxelGard/bronze/internal/parser"
)

const symbolProgram = `enum Status
{
    Active
    Closed
}

class Account use Error
{
    mut status = Status.Active
    con owner = ""

    func init(owner: String)
    {
        this.owner = owner
    }

    mut func close() -> Status
    {
        this.status = Status.Closed
        return this.status
    }
}

func find(owner: String) -> Account?
{
    return Account(owner)
}

func main() -> Int
{
    con a = find("ada") else Account("x")
    con s = a.close()
    match s
    {
        Status.Active => print(a.owner + a.message)
        _ => { }
    }
    return 0
}
`

// refsOf finds the symbol at the n-th occurrence of word and returns where
// all its references are, as "line:col".
func refsOf(t *testing.T, word string, n int) (*Symbol, []string) {
	t.Helper()
	src := diag.NewSource("t.brz", []byte(symbolProgram))
	f, diags := parser.Parse(src)
	info, cdiags := Check(f, src)
	if len(diags)+len(cdiags) > 0 {
		t.Fatalf("errors: %v %v", diags, cdiags)
	}
	offset := -1
	for i := 0; i < n; i++ {
		next := strings.Index(symbolProgram[offset+1:], word)
		if next < 0 {
			t.Fatalf("%q occurs fewer than %d times", word, n)
		}
		offset += 1 + next
	}
	r := info.RefAt(offset)
	if r == nil {
		t.Fatalf("no symbol at %q #%d", word, n)
	}
	var at []string
	for _, ref := range info.RefsTo(r.Sym) {
		at = append(at, fmt.Sprintf("%d:%d", ref.Span.Start.Line, ref.Span.Start.Col))
	}
	return r.Sym, at
}

func TestSymbols(t *testing.T) {
	tests := []struct {
		word string
		n    int
		kind SymbolKind
		want string
	}{
		{"Status", 1, SymEnum, "1:6 9:18 17:25 19:23 35:9"},
		{"Closed", 2, SymCase, "4:5 19:30"},
		{"Active", 2, SymCase, "3:5 9:25 35:16"},
		{"Account", 3, SymClass, "7:7 24:29 26:12 31:30"},
		{"owner", 1, SymField, "10:9 14:14 35:34"},
		{"owner", 2, SymParameter, "12:15 14:22"},
		{"close", 2, SymMethod, "17:14 32:15"},
		{"status", 2, SymField, "9:9 19:14 20:21"},
		{"find", 2, SymFunction, "24:6 31:13"},
		{"a = find", 1, SymVariable, "31:9 32:13 35:32 35:42"},
		{"s = a", 1, SymVariable, "32:9 33:11"},
	}
	for _, tt := range tests {
		sym, at := refsOf(t, tt.word, tt.n)
		if sym.Kind != tt.kind || strings.Join(at, " ") != tt.want {
			t.Errorf("%s #%d: %s at %s, want %s at %s", tt.word, tt.n, sym.Kind, strings.Join(at, " "), tt.kind, tt.want)
		}
	}

	// the inherited message field is built in
	if sym, _ := refsOf(t, "message", 1); !sym.Builtin || sym.Kind != SymField {
		t.Errorf("message: %+v", sym)
	}
}
