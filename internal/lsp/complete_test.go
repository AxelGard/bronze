package lsp

import (
	"slices"
	"strings"
	"testing"
)

// complete opens src, where `|` marks the cursor, and returns the labels
// of the completion items there.
func complete(t *testing.T, src string) []string {
	t.Helper()
	c := newClient(t)
	c.request("initialize", map[string]any{"capabilities": map[string]any{}})
	i := strings.Index(src, "|")
	text := src[:i] + src[i+1:]
	open(c, text)
	m := c.request("textDocument/completion", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     at(text+" ", text[i:]+" ", 1, 0),
	})
	var labels []string
	for _, it := range result[[]completionItem](t, m) {
		labels = append(labels, it.Label)
	}
	return labels
}

const completeProgram = `class Account
{
    mut balance = 0
    con owner = "Ada"

    func deposit(amount: Int)
    {
        this.|
    }
}

enum Status
{
    Active
    Frozen
}

func helper() -> Int
{
    return 1
}

func main() -> Int
{
    con account = Account()
    mut names: List<String> = []
    for n in 0..3
    {
        CURSOR
    }
    return 0
}
`

func TestCompletion(t *testing.T) {
	inMain := func(line string) string {
		return strings.Replace(strings.Replace(completeProgram, "this.|", "this.balance += amount", 1), "CURSOR", line, 1)
	}
	tests := []struct {
		name     string
		src      string
		want     []string
		dontWant []string
	}{
		{"scope", inMain("print(na|)"), []string{"names", "account", "n", "helper", "Account", "Status", "print", "for"}, []string{"amount", "this"}},
		{"members", inMain("account.|"), []string{"balance", "owner", "deposit"}, []string{"names"}},
		{"members, partly written", inMain("account.dep|"), []string{"deposit"}, nil},
		{"members before a call", inMain("account.dep|(5)"), []string{"deposit"}, nil},
		{"enum cases", inMain("con s = Status.|"), []string{"Active", "Frozen"}, nil},
		{"list methods", inMain("names.|"), []string{"append", "length", "join"}, []string{"keys"}},
		{"string methods", inMain("con up = \"x\".to|"), []string{"to_upper", "split"}, nil},
		{"this", strings.Replace(completeProgram, "CURSOR", "", 1), []string{"balance", "deposit"}, nil},
		{"scope in a method", strings.Replace(strings.Replace(completeProgram, "this.|", "print(am|)", 1), "CURSOR", "", 1), []string{"amount", "this"}, []string{"names"}},
		{"type", strings.Replace(inMain(""), "mut names: List<String>", "mut names: Acc|", 1), []string{"Account", "Int", "List", "Status"}, []string{"names"}},
	}
	for _, tt := range tests {
		got := complete(t, tt.src)
		for _, w := range tt.want {
			if !slices.Contains(got, w) {
				t.Errorf("%s: %q missing from %v", tt.name, w, got)
			}
		}
		for _, w := range tt.dontWant {
			if slices.Contains(got, w) {
				t.Errorf("%s: %q should not be in %v", tt.name, w, got)
			}
		}
	}
}

const classesProgram = `class Account
{
    mut balance = 0
    private mut pin = ""

    static func open() -> Account
    {
        return Account()
    }

    func show() -> String
    {
        return ""
    }

    func peek(other: Account) -> String
    {
        return INSIDE
    }
}

class Savings use Account
{
    func rate() -> Int
    {
        return 1
    }
}

enum Role: String
{
    Admin = "admin"
}

func main() -> Int
{
    con s = Savings()
    CURSOR
    return 0
}
`

func TestCompletionClasses(t *testing.T) {
	inMain := func(line string) string {
		return strings.Replace(strings.Replace(classesProgram, "INSIDE", `""`, 1), "CURSOR", line, 1)
	}
	tests := []struct {
		name     string
		src      string
		want     []string
		dontWant []string
	}{
		{"inherited, without private and static", inMain("s.|"), []string{"balance", "show", "rate"}, []string{"pin", "open"}},
		{"private inside the class", strings.Replace(strings.Replace(classesProgram, "INSIDE", "other.|", 1), "CURSOR", "", 1), []string{"pin", "balance"}, []string{"open"}},
		{"static methods", inMain("con a = Account.|"), []string{"open"}, []string{"show", "balance"}},
		{"enum from", inMain("con r = Role.|"), []string{"Admin", "from"}, nil},
		{"enum value", inMain("con v = Role.Admin.|"), []string{"value"}, nil},
	}
	for _, tt := range tests {
		got := complete(t, tt.src)
		for _, w := range tt.want {
			if !slices.Contains(got, w) {
				t.Errorf("%s: %q missing from %v", tt.name, w, got)
			}
		}
		for _, w := range tt.dontWant {
			if slices.Contains(got, w) {
				t.Errorf("%s: %q should not be in %v", tt.name, w, got)
			}
		}
	}
}
