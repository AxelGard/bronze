package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

// client talks to a server running in a goroutine.
type client struct {
	t      *testing.T
	conn   *conn
	nextID int
	done   chan int
	notes  []*message // notifications received while waiting for responses
}

func newClient(t *testing.T) *client {
	t.Helper()
	toServer, fromClient := io.Pipe()
	fromServer, toClient := io.Pipe()
	c := &client{t: t, conn: &conn{in: bufio.NewReader(fromServer), out: fromClient}, done: make(chan int, 1)}
	go func() {
		c.done <- Serve(toServer, toClient, io.Discard, "test")
		toClient.Close()
	}()
	return c
}

func (c *client) send(method string, id *int, params any) {
	c.t.Helper()
	raw, err := json.Marshal(params)
	if err != nil {
		c.t.Fatal(err)
	}
	m := &message{Method: method, Params: raw}
	if id != nil {
		r := json.RawMessage(fmt.Sprint(*id))
		m.ID = &r
	}
	if err := c.conn.write(m); err != nil {
		c.t.Fatal(err)
	}
}

func (c *client) notify(method string, params any) { c.send(method, nil, params) }

// request sends a request and returns its response.
func (c *client) request(method string, params any) *message {
	c.t.Helper()
	c.nextID++
	id := c.nextID
	c.send(method, &id, params)
	for {
		m, err := c.conn.read()
		if err != nil {
			c.t.Fatalf("%s: %v", method, err)
		}
		if m.ID != nil && string(*m.ID) == fmt.Sprint(id) {
			return m
		}
		c.notes = append(c.notes, m)
	}
}

// diagnostics waits for the next publishDiagnostics notification.
func (c *client) diagnostics() publishDiagnosticsParams {
	c.t.Helper()
	for {
		var m *message
		if len(c.notes) > 0 {
			m, c.notes = c.notes[0], c.notes[1:]
		} else {
			var err error
			if m, err = c.conn.read(); err != nil {
				c.t.Fatal(err)
			}
		}
		if m.Method == "textDocument/publishDiagnostics" {
			var p publishDiagnosticsParams
			if err := json.Unmarshal(m.Params, &p); err != nil {
				c.t.Fatal(err)
			}
			return p
		}
	}
}

func result[T any](t *testing.T, m *message) T {
	t.Helper()
	var v T
	if m.Error != nil {
		t.Fatalf("error: %s", m.Error.Message)
	}
	if err := json.Unmarshal(*m.Result, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

const uri = "file:///home/ada/app/main.brz"

const program = `class Counter
{
    mut count = 0

    mut func add(n: Int)
    {
        this.count = this.count + n
    }
}

func main() -> Int
{
    con c = Counter()
    c.add(2)
    print("é: " + String(c.count))
    return 0
}
`

// at returns the position of the n-th (1-based) occurrence of word, plus
// shift characters, in UTF-16 units.
func at(text, word string, n, shift int) position {
	offset := -1
	for i := 0; i < n; i++ {
		offset += 1 + strings.Index(text[offset+1:], word)
	}
	offset += shift
	line := strings.Count(text[:offset], "\n")
	start := strings.LastIndex(text[:offset], "\n") + 1
	return position{Line: line, Character: len([]rune(text[start:offset]))}
}

func open(c *client, text string) publishDiagnosticsParams {
	c.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": uri, "languageId": "bronze", "version": 1, "text": text},
	})
	return c.diagnostics()
}

func TestSession(t *testing.T) {
	c := newClient(t)
	init := result[map[string]any](t, c.request("initialize", map[string]any{"capabilities": map[string]any{}}))
	caps := init["capabilities"].(map[string]any)
	if caps["positionEncoding"] != "utf-16" || caps["definitionProvider"] != true {
		t.Errorf("capabilities: %v", caps)
	}
	c.notify("initialized", map[string]any{})

	if d := open(c, program); len(d.Diagnostics) != 0 {
		t.Errorf("unexpected diagnostics: %+v", d.Diagnostics)
	}

	// go to definition: `count` in String(c.count) goes to the field
	pos := at(program, "count", 4, 1)
	loc := result[location](t, c.request("textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": uri}, "position": pos,
	}))
	if want := at(program, "count", 1, 0); loc.URI != uri || loc.Range.Start != want {
		t.Errorf("definition: %+v, want %+v", loc, want)
	}

	// definition of a built-in has no location
	m := c.request("textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": uri}, "position": at(program, "print", 1, 0),
	})
	if m.Error != nil || (m.Result != nil && string(*m.Result) != "null") { // json decodes null as nil
		t.Errorf("definition of print: %+v", m)
	}

	// rename the method add to increase
	pos = at(program, "add", 2, 0)
	prep := result[map[string]any](t, c.request("textDocument/prepareRename", map[string]any{
		"textDocument": map[string]any{"uri": uri}, "position": pos,
	}))
	if prep["placeholder"] != "add" {
		t.Errorf("prepareRename: %v", prep)
	}
	edit := result[workspaceEdit](t, c.request("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": uri}, "position": pos, "newName": "increase",
	}))
	edits := edit.Changes[uri]
	if len(edits) != 2 || edits[0].Range.Start != at(program, "add", 1, 0) || edits[1].NewText != "increase" {
		t.Errorf("rename: %+v", edits)
	}

	// a bad new name is refused
	m = c.request("textDocument/rename", map[string]any{
		"textDocument": map[string]any{"uri": uri}, "position": pos, "newName": "while",
	})
	if m.Error == nil || !strings.Contains(m.Error.Message, "keyword") {
		t.Errorf("rename to while: %+v", m)
	}

	// a change with errors publishes them; positions count UTF-16 units
	broken := strings.Replace(program, `String(c.count)`, `c.count`, 1)
	c.notify("textDocument/didChange", map[string]any{
		"textDocument":   map[string]any{"uri": uri, "version": 2},
		"contentChanges": []map[string]any{{"text": broken}},
	})
	d := c.diagnostics()
	if len(d.Diagnostics) != 1 || !strings.HasPrefix(d.Diagnostics[0].Message, "cannot add String and Int") {
		t.Fatalf("diagnostics: %+v", d.Diagnostics)
	}
	if got, want := d.Diagnostics[0].Range.Start, at(broken, `"é`, 1, 0); got != want {
		t.Errorf("diagnostic at %+v, want %+v", got, want)
	}

	if m := c.request("textDocument/formatting", map[string]any{}); m.Error == nil || m.Error.Code != codeMethodNotFound {
		t.Errorf("unknown method: %+v", m)
	}

	c.request("shutdown", nil)
	c.notify("exit", nil)
	if code := <-c.done; code != 0 {
		t.Errorf("exit code %d", code)
	}
}

func TestRenameRefusals(t *testing.T) {
	c := newClient(t)
	c.request("initialize", map[string]any{"capabilities": map[string]any{}})

	src := "class Oops use Error\n{\n}\nfunc main() -> Int\n{\n    con e = Oops()\n    print(e.message)\n    return 0\n}\n"
	open(c, src)
	m := c.request("textDocument/prepareRename", map[string]any{
		"textDocument": map[string]any{"uri": uri}, "position": at(src, "message", 1, 0),
	})
	if m.Error == nil || !strings.Contains(m.Error.Message, "built in") {
		t.Errorf("rename message: %+v", m)
	}

	// with a syntax error, rename is refused
	c.notify("textDocument/didChange", map[string]any{
		"textDocument":   map[string]any{"uri": uri, "version": 2},
		"contentChanges": []map[string]any{{"text": strings.Replace(src, "return 0", "return )", 1)}},
	})
	c.diagnostics()
	m = c.request("textDocument/prepareRename", map[string]any{
		"textDocument": map[string]any{"uri": uri}, "position": at(src, "Oops", 1, 0),
	})
	if m.Error == nil || !strings.Contains(m.Error.Message, "syntax errors") {
		t.Errorf("rename with syntax error: %+v", m)
	}

	c.notify("exit", nil)
	if code := <-c.done; code != 1 {
		t.Errorf("exit without shutdown: code %d", code)
	}
}

func TestUTF8Positions(t *testing.T) {
	c := newClient(t)
	init := result[map[string]any](t, c.request("initialize", map[string]any{
		"capabilities": map[string]any{"general": map[string]any{"positionEncodings": []string{"utf-8", "utf-16"}}},
	}))
	if init["capabilities"].(map[string]any)["positionEncoding"] != "utf-8" {
		t.Fatalf("encoding: %v", init)
	}
	src := "func main() -> Int\n{\n    con é = 1\n    return é\n}\n"
	open(c, src)
	loc := result[location](t, c.request("textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": uri}, "position": position{Line: 3, Character: 11},
	}))
	// `é` is at byte 8 on line 2; it ends at byte 10
	if loc.Range.Start != (position{Line: 2, Character: 8}) || loc.Range.End != (position{Line: 2, Character: 10}) {
		t.Errorf("definition: %+v", loc.Range)
	}
	c.notify("exit", nil)
	<-c.done
}

func TestHover(t *testing.T) {
	c := newClient(t)
	c.request("initialize", map[string]any{"capabilities": map[string]any{}})
	src := `class Account
{
    mut balance = 0

    mut func withdraw(amount: Int) -> Int ! Error
    {
        this.balance -= amount
        return this.balance
    }

    static func opened() -> Account
    {
        return Account()
    }
}

enum Status
{
    Active
    Frozen
}

func top_up(a: mut Account, by: Int)
{
    a.balance += by
}

enum Role: String
{
    Admin = "admin"
}

func role_of(s: String) -> Role
{
    con r = Role.from(s) else Role.Admin
    print(r.value)
    return r
}

func main() -> Int
{
    con account = Account.opened()
    mut names: List<String> = []
    for n in 0..3
    {
        names.append("{n}")
    }
    con left = account.withdraw(5) else 0
    con s = Status.Frozen
    print(left)
    con oops = Error("x")
    print(oops.message)
    return 0
}
`
	open(c, src)
	tests := []struct {
		word  string
		n     int
		want  string
		extra string
	}{
		{"account", 2, "con account: Account", ""},
		{"names", 1, "mut names: List<String>", ""},
		{"n in", 1, "n: Int", ""},
		{"amount", 2, "amount: Int", ""},
		{"balance", 2, "mut balance: Int", "in `Account`"},
		{"withdraw", 2, "mut func withdraw(amount: Int) -> Int ! Error", "in `Account`"},
		{"top_up", 1, "func top_up(a: mut Account, by: Int)", ""},
		{"a.balance", 1, "a: mut Account", ""},
		{"opened", 2, "static func opened() -> Account", "in `Account`"},
		{"Role", 1, "enum Role: String", ""},
		{"Admin", 1, "Role.Admin = \"admin\"", ""},
		{"from", 1, "func from(value: String) -> Role?", "in `Role`"},
		{"value", 1, "con value: String", "in `Role`"},
		{"Account", 2, "class Account", ""},
		{"Status", 2, "enum Status", ""},
		{"Frozen", 2, "Status.Frozen", ""},
		{"append", 1, "func append(x: String)", "in `List<String>`"},
		{"print", 1, "func print(x: T)", "prints `x`"},
		{"main", 1, "func main() -> Int", ""},
		{"message", 1, "mut message: String", "in `Error`"},
		{"Error(", 1, "class Error", ""},
	}
	for _, tt := range tests {
		m := c.request("textDocument/hover", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     at(src, tt.word, tt.n, 1),
		})
		h := result[hoverResult](t, m)
		want := "```bronze\n" + tt.want + "\n```"
		if !strings.HasPrefix(h.Contents.Value, want) || !strings.Contains(h.Contents.Value, tt.extra) {
			t.Errorf("hover on %q: got %q, want %q and %q", tt.word, h.Contents.Value, want, tt.extra)
		}
	}
	m := c.request("textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     at(src, "return 0", 1, 0),
	})
	if m.Error != nil || m.Result != nil && string(*m.Result) != "null" {
		t.Errorf("hover on a keyword: %+v", m)
	}
}
