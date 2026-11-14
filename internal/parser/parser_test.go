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
			"class A\n{\n    func f() -> ! E\n    {\n        g()\n    }\n}\nfunc h()\n{\n    if x is B\n    {\n        g()\n    }\n}",
			[]string{"3:17 functions that can fail are not supported yet", "10:10 `is` is not supported yet"},
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
			"reserved word",
			"func f()\n{\n    con for = 1\n}",
			[]string{"3:9 `for` is reserved for a later version, and can't be used as a name"},
		},
		{
			"not yet",
			"enum A {}\nclass B use Error {}\nfunc f()\n{\n    con a = null\n}",
			[]string{"1:1 enums are not supported yet", "2:9 `use Error` is not supported yet", "5:13 `null` is not supported yet"},
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

func TestRecoveredTree(t *testing.T) {
	f, _ := parse(t, "func f()\n{\n    con = 1\n    g()\n}\nfunc h() { }")
	want := "(func f (block (bad-stmt) (call g)))\n(func h (block))"
	if got := ast.Dump(f); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}
