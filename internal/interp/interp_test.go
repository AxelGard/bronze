package interp

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/AxelGard/bronze/internal/check"
	"github.com/AxelGard/bronze/internal/diag"
	"github.com/AxelGard/bronze/internal/parser"
)

func run(t *testing.T, text string) (string, int, *Panic) {
	t.Helper()
	src := diag.NewSource("t.brz", []byte(text))
	f, diags := parser.Parse(src)
	if len(diags) == 0 {
		_, diags = check.Check(f, src)
	}
	if len(diags) > 0 {
		t.Fatalf("%d:%d %s", diags[0].Span.Start.Line, diags[0].Span.Start.Col, diags[0].Message)
	}
	var out strings.Builder
	code, p := Run(f, src, &out)
	return out.String(), code, p
}

// prints runs a main body and returns what it printed.
func prints(t *testing.T, body string) string {
	t.Helper()
	out, _, p := run(t, "func main() -> Int\n{\n"+body+"\n    return 0\n}\n")
	if p != nil {
		t.Fatalf("panic: %s", p.Message)
	}
	return out
}

func TestCore(t *testing.T) {
	text, err := os.ReadFile("../../examples/core.brz")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("../../tests/run/core.out")
	if err != nil {
		t.Fatal(err)
	}
	out, code, p := run(t, string(text))
	if p != nil || code != 0 {
		t.Fatalf("code %d, panic %v", code, p)
	}
	if out != string(want) {
		t.Errorf("got\n%s", out)
	}
}

func TestArithmetic(t *testing.T) {
	tests := []struct{ expr, want string }{
		{"-7 / 2", "-3"},
		{"-7 % 2", "-1"},
		{"7 % -2", "1"},
		{"9223372036854775807 + 1", "-9223372036854775808"},
		{"1.0 / 0.0", "inf"},
		{"-1.0 / 0.0", "-inf"},
		{"0.1 + 0.2", "0.30000000000000004"},
		{"\"a\" + \"b\"", "ab"},
		{"\"a\" < \"b\"", "true"},
		{"1.5 >= 1.5", "true"},
		{"2 * 3 + 4", "10"},
		{"not (1 == 1) or 2 != 3", "true"},
		{"String(12) + String(2.5) + String(false)", "122.5false"},
	}
	for _, tt := range tests {
		if got := prints(t, "    print("+tt.expr+")"); got != tt.want+"\n" {
			t.Errorf("%s = %q, want %q", tt.expr, got, tt.want)
		}
	}
}

func TestShortCircuit(t *testing.T) {
	got := prints(t, "    print(false and 1 / 0 == 0)\n    print(true or panic(\"no\") == 0)")
	if got != "false\ntrue\n" {
		t.Errorf("got %q", got)
	}
}

func TestExitCode(t *testing.T) {
	_, code, _ := run(t, "func main() -> Int\n{\n    return 3\n}")
	if code != 3 {
		t.Errorf("code %d", code)
	}
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		f    float64
		want string
	}{
		{6, "6.0"}, {3.5, "3.5"}, {0.1, "0.1"}, {-2, "-2.0"}, {0, "0.0"},
		{1e20, "100000000000000000000.0"}, {1e21, "1e+21"}, {1.5e-8, "1.5e-08"}, {1.5e-7, "0.00000015"}, {1e-6, "0.000001"},
		{math.Inf(1), "inf"}, {math.Inf(-1), "-inf"}, {math.NaN(), "NaN"},
	}
	for _, tt := range tests {
		if got := FormatFloat(tt.f); got != tt.want {
			t.Errorf("FormatFloat(%v) = %q, want %q", tt.f, got, tt.want)
		}
	}
}

func trace(p *Panic) string {
	var b strings.Builder
	p.Format(&b)
	return b.String()
}

func TestPanics(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		{"division by zero", `func divide(a: Int, b: Int) -> Int
{
    return a / b
}

func main() -> Int
{
    print(divide(1, 0))
    return 0
}`, "panic: division by zero\n  at divide (t.brz:3)\n  at main   (t.brz:8)\n"},
		{"panic", "func main() -> Int\n{\n    panic(\"boom\")\n}",
			"panic: boom\n  at main (t.brz:3)\n"},
		{"assert", "func main() -> Int\n{\n    assert(1 > 2, \"one is big\")\n    return 0\n}",
			"panic: assertion failed: one is big\n  at main (t.brz:3)\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, p := run(t, tt.src)
			if p == nil {
				t.Fatal("no panic")
			}
			if got := trace(p); got != tt.want {
				t.Errorf("got\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestStackOverflow(t *testing.T) {
	_, _, p := run(t, "func f(n: Int) -> Int\n{\n    return f(n + 1)\n}\nfunc main() -> Int\n{\n    return f(0)\n}")
	if p == nil || p.Message != "stack overflow" {
		t.Fatalf("got %v", p)
	}
	if len(p.Trace) != MaxDepth {
		t.Errorf("trace has %d frames", len(p.Trace))
	}
	got := trace(p)
	if !strings.Contains(got, "  ... 9980 more\n") || !strings.HasSuffix(got, "  at main (t.brz:7)\n") {
		t.Errorf("got\n%s", got)
	}
}

func TestOutputBeforePanic(t *testing.T) {
	out, _, p := run(t, "func main() -> Int\n{\n    print(1)\n    print(1 / 0)\n    return 0\n}")
	if p == nil || out != "1\n" {
		t.Errorf("out %q, panic %v", out, p)
	}
}

func TestCounter(t *testing.T) {
	text, err := os.ReadFile("../../examples/projects/counter.brz")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("../../tests/run/counter.out")
	if err != nil {
		t.Fatal(err)
	}
	out, code, p := run(t, string(text))
	if p != nil || code != 0 {
		t.Fatalf("code %d, panic %v", code, p)
	}
	if out != string(want) {
		t.Errorf("got\n%s", out)
	}
}

const node = `class Node
{
    mut value = next_id()
    mut label = "?"

    func init()
    {
    }

    func init(label: String)
    {
        print("init " + label + " " + String(this.value))
        this.label = label
    }

    func bump() -> Int
    {
        this.value = this.value + 1
        return this.value
    }

    func twice() -> Int
    {
        this.bump()
        return this.bump()
    }
}

class Empty
{
}

func next_id() -> Int
{
    print("next_id")
    return 7
}
`

func TestObjects(t *testing.T) {
	tests := []struct{ body, want string }{
		// field values are evaluated before init, for each new object
		{`con a = Node("a")`, "next_id\ninit a 7\n"},
		{`con a = Node()
    con b = Node()
    print(a == b)
    print(a == a)`, "next_id\nnext_id\nfalse\ntrue\n"},
		// methods calling methods, and shared objects
		{`con a = Node()
    con b = a
    print(b.twice())
    print(a.value)`, "next_id\n9\n9\n"},
		{`con e = Empty()
    print(e == e)`, "true\n"},
		{`mut a = Node()
    con first = a
    a = Node()
    first.label = "first"
    print(a.label + " " + first.label)`, "next_id\nnext_id\n? first\n"},
	}
	for _, tt := range tests {
		out, _, p := run(t, node+"func main() -> Int\n{\n    "+tt.body+"\n    return 0\n}\n")
		if p != nil {
			t.Fatalf("panic: %s", p.Message)
		}
		if out != tt.want {
			t.Errorf("%s:\ngot\n%s\nwant\n%s", tt.body, out, tt.want)
		}
	}
}

func TestMethodPanic(t *testing.T) {
	src := `class Account
{
    mut balance = 0

    func init(balance: Int)
    {
        assert(balance >= 0, "negative balance")
        this.balance = balance
    }

    func share(people: Int) -> Int
    {
        return this.balance / people
    }
}

func main() -> Int
{
    con a = Account(10)
    print(a.share(2))
    print(a.share(0))
    con b = Account(-1)
    return 0
}
`
	out, _, p := run(t, src)
	if p == nil || out != "5\n" {
		t.Fatalf("out %q, panic %v", out, p)
	}
	want := "panic: division by zero\n  at Account.share (t.brz:13)\n  at main          (t.brz:21)\n"
	if got := trace(p); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}

	_, _, p = run(t, strings.Replace(src, "print(a.share(0))", "", 1))
	want = "panic: assertion failed: negative balance\n  at Account.init (t.brz:7)\n  at main         (t.brz:22)\n"
	if p == nil {
		t.Fatal("no panic")
	}
	if got := trace(p); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

const light = `enum Light
{
    Red
    Yellow
    Green

    func next() -> Light
    {
        return match this
        {
            Light.Red => Light.Green
            Light.Green => Light.Yellow
            Light.Yellow => Light.Red
        }
    }

    func stop() -> Bool
    {
        return this == Light.Red or this == Light.Yellow
    }
}

class Crossing
{
    mut light = Light.Red
}
`

func TestEnumsAndMatch(t *testing.T) {
	tests := []struct{ body, want string }{
		{"print(Light.Yellow)", "Yellow\n"},
		{"print(Light.Red.next().next())", "Yellow\n"},
		{"con l = Light.Green\n    print(l == Light.Green)\n    print(l != Light.Green)\n    print(l.stop())", "true\nfalse\nfalse\n"},
		{"con c = Crossing()\n    c.light = c.light.next()\n    print(c.light)", "Green\n"},
		// patterns of each kind, first match wins
		{"print(match -1 { -1 => \"minus\"\n _ => \"other\" })", "minus\n"},
		{"print(match \"b\" { \"a\" => 1\n \"b\" => 2\n _ => 3 })", "2\n"},
		{"print(match 1 < 2 { true => \"yes\"\n false => \"no\" })", "yes\n"},
		{"print(match 7 { 1 => \"one\"\n _ => \"any\" })", "any\n"},
		// only the chosen arm is evaluated
		{"print(match Light.Red { Light.Red => 1\n _ => panic(\"not this\") })", "1\n"},
	}
	for _, tt := range tests {
		out, _, p := run(t, light+"func main() -> Int\n{\n    "+tt.body+"\n    return 0\n}\n")
		if p != nil {
			t.Fatalf("%s: panic: %s", tt.body, p.Message)
		}
		if out != tt.want {
			t.Errorf("%s:\ngot\n%s\nwant\n%s", tt.body, out, tt.want)
		}
	}
}

func TestEnumMethodTrace(t *testing.T) {
	src := `enum Coin
{
    Heads
    Tails

    func flip() -> Coin
    {
        return match this
        {
            Coin.Heads => Coin.Tails
            Coin.Tails => panic("no more flips")
        }
    }
}

func main() -> Int
{
    print(Coin.Heads.flip().flip())
    return 0
}
`
	_, _, p := run(t, src)
	if p == nil {
		t.Fatal("no panic")
	}
	want := "panic: no more flips\n  at Coin.flip (t.brz:11)\n  at main      (t.brz:18)\n"
	if got := trace(p); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestMatchStatement(t *testing.T) {
	src := light + `func check(l: Light) -> String
{
    match l
    {
        Light.Red =>
        {
            print("stop")
            return "red"
        }
        Light.Yellow => print("slow")
        _ => { }
    }
    return String(l)
}

func main() -> Int
{
    print(check(Light.Red))
    print(check(Light.Yellow))
    print(check(Light.Green))
    return 0
}
`
	out, _, p := run(t, src)
	if p != nil {
		t.Fatal(p.Message)
	}
	if want := "stop\nred\nslow\nYellow\nGreen\n"; out != want {
		t.Errorf("got\n%s\nwant\n%s", out, want)
	}
}

const errorProgram = `class NotFound use Error
{
    con id = 0

    func init(id: Int)
    {
        this.id = id
        this.message = "no item " + String(id)
    }
}

class Item
{
    con id = 0

    func init(id: Int)
    {
        this.id = id
    }
}

func find(id: Int) -> Item ! NotFound
{
    if id < 0
    {
        return NotFound(id)
    }
    return Item(id)
}

func check(id: Int) -> ! Error
{
    if id == 0
    {
        return Error("zero")
    }
}

// passes errors on, and turns NotFound into a plain Error
func twice(id: Int) -> Int ! Error
{
    check(id) else return
    con item = find(id) else (e) return Error("wrapped: " + e.message)
    return item.id * 2
}

func lookup(id: Int) -> Item?
{
    if id > 10
    {
        return null
    }
    return Item(id)
}

func first(id: Int) -> Item? ! Error
{
    check(id) else return
    return lookup(id)
}

func describe(id: Int) -> String
{
    con n = twice(id) else (e)
    {
        if e is NotFound
        {
            return "never: twice wraps NotFound"
        }
        return "failed: " + e.message
    }
    return "ok: " + String(n)
}
`

func TestErrorsAndNull(t *testing.T) {
	tests := []struct{ body, want string }{
		{"print((find(3) else Item(0)).id)\n    print((find(-1) else Item(0)).id)", "3\n0\n"},
		{"print(describe(4))", "ok: 8\n"},
		{"print(describe(0))", "failed: zero\n"},
		{"print(describe(-2))", "failed: wrapped: no item -2\n"},
		{"con e = Error(\"plain\")\n    print(e.message)\n    print(e is Error)\n    print(e is NotFound)", "plain\ntrue\nfalse\n"},
		{"con e = NotFound(5)\n    print(e.message)\n    print(e.id)\n    print(e is NotFound and e is Error)", "no item 5\n5\ntrue\n"},
		{"print((lookup(11) else Item(99)).id)", "99\n"},
		{"print((lookup(3) else Item(99)).id)", "3\n"},
		{"con a = lookup(12)\n    print(a == null)\n    con b = lookup(1)\n    print(b != null)", "true\ntrue\n"},
		{"con a = first(20) else panic(\"no\")\n    print(a == null)", "true\n"},
		{"con a = first(0) else Item(7)\n    print((a else Item(8)).id)", "7\n"},
		{"check(1) else panic(\"no\")\n    print(\"fine\")", "fine\n"},
	}
	for _, tt := range tests {
		out, _, p := run(t, errorProgram+"func main() -> Int\n{\n    "+tt.body+"\n    return 0\n}\n")
		if p != nil {
			t.Fatalf("%s: panic: %s", tt.body, p.Message)
		}
		if out != tt.want {
			t.Errorf("%s:\ngot\n%s\nwant\n%s", tt.body, out, tt.want)
		}
	}
}

func TestElseReturn(t *testing.T) {
	src := errorProgram + `func size(id: Int) -> Int
{
    // a return deep inside an expression leaves the function
    con n = 1 + (lookup(id) else return -1).id
    return n
}

func report(id: Int)
{
    con item = lookup(id) else return
    print("found " + String(item.id))
}

func main() -> Int
{
    print(size(4))
    print(size(40))
    report(2)
    report(20)
    con x = find(-5) else panic("find failed")
    return 0
}
`
	out, _, p := run(t, src)
	if want := "5\n-1\nfound 2\n"; out != want {
		t.Errorf("got\n%s\nwant\n%s", out, want)
	}
	if p == nil || p.Message != "find failed" || len(p.Trace) != 1 {
		t.Errorf("panic %+v", p)
	}
}
