package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/diag"
)

func parse(t *testing.T, text string) (*ast.File, diag.List) {
	t.Helper()
	return Parse(diag.NewSource("t.brz", []byte(text)))
}

// body parses statements inside a function and dumps its body.
func body(t *testing.T, stmts string) string {
	t.Helper()
	f, diags := parse(t, "func f()\n{\n"+stmts+"\n}\n")
	for _, d := range diags {
		t.Errorf("unexpected error: %d:%d %s", d.Span.Start.Line, d.Span.Start.Col, d.Message)
	}
	return ast.Dump(f.Decls[0].(*ast.FuncDecl).Body)
}

func TestFunc(t *testing.T) {
	f, diags := parse(t, "func add(a: Int, b: Int) -> Int\n{\n    return a + b\n}\n\nfunc main() -> Int { return 0 }")
	if len(diags) > 0 {
		t.Fatal(diags[0].Message)
	}
	want := "(func add (a Int) (b Int) -> Int (block (return (+ a b))))\n(func main -> Int (block (return 0)))"
	if got := ast.Dump(f); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestStatements(t *testing.T) {
	tests := []struct{ src, want string }{
		{"con x = 1", "(block (con x 1))"},
		{"mut y: Float = 2.5", "(block (mut y: Float 2.5f))"},
		{"x = x + 1", "(block (= x (+ x 1)))"},
		{"print(\"hi\")", `(block (call print "hi"))`},
		{"return", "(block (return))"},
		{"while i <= n\n{\ni = i + 1\n}", "(block (while (<= i n) (block (= i (+ i 1)))))"},
		{"if a { f() }", "(block (if a (block (call f))))"},
		{"if a\n{\nf()\n}\nelse if b\n{\ng()\n}\nelse\n{\nh()\n}",
			"(block (if a (block (call f)) else (if b (block (call g)) else (block (call h)))))"},
		{"if a { f() }\ng()", "(block (if a (block (call f))) (call g))"},
		{"x += 1\nthis.n -= y * 2\nx *= 3\nx /= 4\nx %= 5",
			"(block (+= x 1) (-= (. this n) (* y 2)) (*= x 3) (/= x 4) (%= x 5))"},
		{"while true\n{\nbreak\ncontinue\n}", "(block (while true (block (break) (continue))))"},
		{"for i in 0..n\n{\nf(i)\n}", "(block (for i in (.. 0 n) (block (call f i))))"},
		{"for _, x in xs { f(x) }", "(block (for _ x in xs (block (call f x))))"},
		{"con xs: List<List<Int>> = [[1, 2], [], [3,\n4,],]", "(block (con xs: List<List<Int>> (list (list 1 2) (list) (list 3 4))))"},
		{"mut m: Map<String, List<Int>?>= xs[i + 1][0]\nxs[0] = 1\nxs[i] += 2", "(block (mut m: Map<String, List<Int>?> (index (index xs (+ i 1)) 0)) (= (index xs 0) 1) (+= (index xs i) 2))"},
		{"con r = 1 + 1..=n * 2 == xs", "(block (con r (== (..= (+ 1 1) (* n 2)) xs)))"},
	}
	for _, tt := range tests {
		if got := body(t, tt.src); got != tt.want {
			t.Errorf("%q:\ngot  %s\nwant %s", tt.src, got, tt.want)
		}
	}
}

func TestPrecedence(t *testing.T) {
	tests := []struct{ src, want string }{
		{"a + b * c", "(+ a (* b c))"},
		{"a - b - c", "(- (- a b) c)"},
		{"a * (b + c)", "(* a (paren (+ b c)))"},
		{"-a * b", "(* (- a) b)"},
		{"not a and b or c", "(or (and (not a) b) c)"},
		{"n % 2 == 0", "(== (% n 2) 0)"},
		{"n >= low and n <= high", "(and (>= n low) (<= n high))"},
		{"Float(a + b) / 2.0", "(/ (call Float (+ a b)) 2f)"},
		{"f(a,\n  b)", "(call f a b)"},
		{"a +\n  b", "(+ a b)"},
	}
	for _, tt := range tests {
		got := body(t, "con x = "+tt.src)
		want := "(block (con x " + tt.want + "))"
		if got != want {
			t.Errorf("%q:\ngot  %s\nwant %s", tt.src, got, want)
		}
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name, src string
		want      []string // "line:col message"
	}{
		{
			"chained comparison",
			"func f()\n{\n    con x = 1 < 2 < 3\n}",
			[]string{"3:19 comparisons can't be chained, use `and`"},
		},
		{
			"several errors",
			"func f()\n{\n    con = 1\n    con y 2\n    print(1\n    g()\n}\n",
			[]string{
				"3:9 expected a name, found `=`",
				"4:11 expected `=` and a value for `y`, found `2`",
				"5:10 `(` is not closed",
			},
		},
		{
			"bad declaration then a good one",
			"con x = 1\nfunc f(a Int) { }\nfunc g() { x }\nfunc h() { }\n",
			[]string{
				"1:1 expected `func`, `class` or `enum`, found `con`",
				"2:10 expected `:` and a type after parameter `a`, found `Int`",
				"3:12 this expression is not used",
			},
		},
		{
			"class members",
			"class A\n{\n    mut x\n    con y = 1 2\n    print(1)\n    func f() { }\n}\nfunc g() { }",
			[]string{
				"3:10 field `x` needs an initial value",
				"4:15 expected the end of the member, found `2`",
				"5:5 expected a field (`con` or `mut`) or a method (`func`), found `print`",
			},
		},
		{
			"block on the next line is skipped",
			"class A\n{\n    func f() -> Int Int\n    {\n        g()\n    }\n}\nfunc h()\n{\n    if x == )\n    {\n        g()\n    }\n}",
			[]string{"3:21 expected `{`, found `Int`", "10:13 expected an expression, found `)`"},
		},
		{
			"enum members",
			"enum A\n{\n    Red Green\n    con x = 1\n    Blue\n}",
			[]string{
				"3:9 expected the end of the case, found `Green`",
				"4:5 expected a case or a method (`func`), found `con`",
			},
		},
		{
			"match arms",
			"func f() -> Int\n{\n    return match x\n    {\n        y => 1\n        1.5 => 2\n        1 2\n        _ => 3\n    }\n}",
			[]string{
				"5:9 `y` is not a pattern",
				"6:9 Float values can't be used as patterns",
				"7:11 expected `=>` after the pattern, found `2`",
			},
		},
		{
			"assign to call",
			"func f()\n{\n    g() = 1\n}",
			[]string{"3:5 cannot assign to this expression"},
		},
		{
			"missing brace",
			"func f()\n{\n    g()\n",
			[]string{"4:1 expected `}`, found end of file"},
		},
		{
			"chained range and for",
			"func f()\n{\n    con r = 1..2..3\n    for i 0..3 { }\n    for x, in xs { }\n}",
			[]string{"3:17 ranges can't be chained", "4:11 expected `in` after the loop variables, found `0`", "5:12 expected a name, found `in`"},
		},
		{
			"reserved word",
			"func f()\n{\n    con data = 1\n}",
			[]string{"3:9 `data` is reserved for a later version, and can't be used as a name"},
		},
		{
			"errors and null",
			"class B use Thing {}\nfunc f() -> Int?? { }\nfunc g()\n{\n    con a = f() else 1 else 2\n    h()\n    else return\n    con b = x is A == true\n}",
			[]string{
				"1:13 expected `Error` after `use`, found `Thing`",
				"2:17 `Int??` is not allowed: a `?` type is already nullable",
				"5:24 an expression can only have one `else`",
				"7:5 `else` must be on the same line as the expression it handles",
				"8:20 `is` can't be combined with another comparison; use `and`",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, diags := parse(t, tt.src)
			if f == nil {
				t.Fatal("no tree")
			}
			var got []string
			for _, d := range diags {
				got = append(got, fmt.Sprintf("%d:%d %s", d.Span.Start.Line, d.Span.Start.Col, d.Message))
			}
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

func TestClass(t *testing.T) {
	src := `class Counter
{
    con name = ""
    mut count: Int = 0

    func init(name: String)
    {
        this.name = name
    }

    func describe() -> String
    {
        return this.name + String(this.count)
    }
}
`
	f, diags := parse(t, src)
	if len(diags) > 0 {
		t.Fatal(diags[0].Message)
	}
	want := `(class Counter (field con name "") (field mut count: Int 0) ` +
		`(func init (name String) (block (= (. this name) name))) ` +
		`(func describe -> String (block (return (+ (. this name) (call String (. this count)))))))`
	if got := ast.Dump(f); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if got := body(t, "c.add(1).total = c.items.count"); got != "(block (= (. (call (. c add) 1) total) (. (. c items) count)))" {
		t.Errorf("members: %s", got)
	}
}

func TestEnumAndMatch(t *testing.T) {
	src := `enum Status
{
    Active
    Frozen

    func label() -> String
    {
        return match this
        {
            Status.Active => "active"
            Status.Frozen => "frozen"
        }
    }
}
`
	f, diags := parse(t, src)
	if len(diags) > 0 {
		t.Fatal(diags[0].Message)
	}
	want := `(enum Status Active Frozen (func label -> String (block (return (match this (Status.Active => "active") (Status.Frozen => "frozen"))))))`
	if got := ast.Dump(f); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}

	got := body(t, "print(match n { 0 => \"zero\"\n -1 => \"minus one\"\n \"a\" => \"\"\n true => \"\"\n _ => \"many\" })")
	want = `(block (call print (match n (0 => "zero") (-1 => "minus one") ("a" => "") (true => "") (_ => "many"))))`
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}

	got = body(t, "match s\n{\n    Suit.Hearts =>\n    {\n        print(1)\n    }\n    _ => { }\n}")
	want = `(block (match s (Suit.Hearts => (block (call print 1))) (_ => (block))))`
	if got != want {
		t.Errorf("match statement:\ngot\n%s\nwant\n%s", got, want)
	}

	// the match keeps the arms that parsed
	f, _ = parse(t, "func f() -> Int\n{\n    return match x\n    {\n        1 => 1\n        oops => 2\n        _ => 3\n    }\n}")
	m := f.Decls[0].(*ast.FuncDecl).Body.Stmts[0].(*ast.Return).Value.(*ast.Match)
	if !m.Bad || len(m.Arms) != 2 {
		t.Errorf("bad %v, %d arms", m.Bad, len(m.Arms))
	}
}

func TestErrorsAndNull(t *testing.T) {
	f, diags := parse(t, "class NotFound use Error\n{\n}\nfunc find(id: Int) -> Account? ! NotFound\n{\n    return null\n}\nfunc save() -> ! Error\n{\n}")
	if len(diags) > 0 {
		t.Fatal(diags[0].Message)
	}
	want := "(class NotFound use Error)\n(func find (id Int) -> Account? ! NotFound (block (return null)))\n(func save -> ! Error (block))"
	if got := ast.Dump(f); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}

	tests := []struct{ src, want string }{
		{"con a = find(1) else return", "(con a (else (call find 1) return))"},
		{"con a = find(1) else return Error(\"x\")", `(con a (else (call find 1) return (call Error "x")))`},
		{"con a = find(1) else (e) return 0", "(con a (else (call find 1) (e) return 0))"},
		{"con a = find(1) else (e)\n{\n    print(e.message)\n    return\n}", "(con a (else (call find 1) (e) (block (call print (. e message)) (return))))"},
		{"con a = find(1) else 0", "(con a (else (call find 1) 0))"},
		{"con a = find(1) else (x)", "(con a (else (call find 1) (paren x)))"},
		{"con a = f() else panic(\"no\")", `(con a (else (call f) (call panic "no")))`},
		{"save() else return", "(block (else (call save) return))"},
		{"con b = e is NotFound and x == null", "(con b (and (is e NotFound) (== x null)))"},
		{"con c = a + 1 else b or c", "(con c (else (+ a 1) (or b c)))"},
	}
	for _, tt := range tests {
		got := body(t, tt.src)
		want := "(block " + tt.want + ")"
		if strings.HasPrefix(tt.want, "(block") {
			want = tt.want
		}
		if got != want {
			t.Errorf("%q:\ngot  %s\nwant %s", tt.src, got, want)
		}
	}
}

func TestRecoveredTree(t *testing.T) {
	f, _ := parse(t, "func f()\n{\n    con = 1\n    g()\n}\nfunc h() { }")
	want := "(func f (block (bad-stmt) (call g)))\n(func h (block))"
	if got := ast.Dump(f); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}
