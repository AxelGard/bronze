package check

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/AxelGard/bronze/internal/diag"
	"github.com/AxelGard/bronze/internal/parser"
)

// errors checks a program and returns its errors as "line:col message".
func errors(t *testing.T, text string) []string {
	t.Helper()
	src := diag.NewSource("t.brz", []byte(text))
	f, pdiags := parser.Parse(src)
	if len(pdiags) > 0 {
		t.Fatalf("syntax error: %s", pdiags[0].Message)
	}
	_, diags := Check(f, src)
	diags.Sort()
	var out []string
	for _, d := range diags {
		out = append(out, fmt.Sprintf("%d:%d %s", d.Span.Start.Line, d.Span.Start.Col, d.Message))
	}
	return out
}

const mainFunc = "\nfunc main() -> Int\n{\n    return 0\n}\n"

func TestValidPrograms(t *testing.T) {
	text, err := os.ReadFile("../../examples/core.brz")
	if err != nil {
		t.Fatal(err)
	}
	if errs := errors(t, string(text)); len(errs) > 0 {
		t.Errorf("core.brz: %s", strings.Join(errs, "\n"))
	}

	ok := []string{
		"func f() -> Int\n{\n    if true\n    {\n        return 1\n    }\n    else\n    {\n        panic(\"no\")\n    }\n}",
		"func f(a: Int) -> String\n{\n    return String(a) + String(1.5) + String(true)\n}",
		"func f()\n{\n    mut x: Float = 1.0\n    x = x * 2.0\n    assert(x > 1.0, \"big\")\n}",
		"func f() -> Int\n{\n    con x = panic(\"later\")\n    return x + 1\n}",
	}
	for _, src := range ok {
		if errs := errors(t, src+mainFunc); len(errs) > 0 {
			t.Errorf("%q:\n%s", src, strings.Join(errs, "\n"))
		}
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name, src string
		want      []string
	}{
		{"unknown name", "func f() -> Int\n{\n    return y\n}",
			[]string{"3:12 unknown name `y`"}},
		{"unknown function", "func f()\n{\n    g(1)\n}",
			[]string{"3:5 unknown function `g`"}},
		{"unknown type", "func f(a: Foo)\n{\n}",
			[]string{"1:11 unknown type `Foo`"}},
		{"con assign", "func f()\n{\n    con x = 1\n    x = 2\n}",
			[]string{"4:5 cannot assign to `x`: it is declared with `con`"}},
		{"param assign", "func f(a: Int)\n{\n    a = 2\n}",
			[]string{"3:5 cannot assign to `a`: parameters can't be changed"}},
		{"undeclared assign", "func f()\n{\n    z = 2\n}",
			[]string{"3:5 cannot assign to `z`: it is not declared"}},
		{"assign wrong type", "func f()\n{\n    mut x = 1\n    x = \"a\"\n}",
			[]string{"4:9 cannot assign String to `x`, which is Int"}},
		{"mixed arithmetic", "func f(a: Int) -> Float\n{\n    return a / 2.0\n}",
			[]string{"3:12 cannot divide Int by Float"}},
		{"float remainder", "func f() -> Float\n{\n    return 1.0 % 2.0\n}",
			[]string{"3:12 cannot use `%` on Float"}},
		{"string plus int", "func f() -> String\n{\n    return \"a\" + 1\n}",
			[]string{"3:12 cannot add String and Int"}},
		{"compare types", "func f() -> Bool\n{\n    return 1 == true\n}",
			[]string{"3:12 cannot compare Int and Bool with `==`"}},
		{"and needs bool", "func f() -> Bool\n{\n    return 1 and true\n}",
			[]string{"3:12 `and` needs Bool on both sides, found Int"}},
		{"condition", "func f()\n{\n    while 1\n    {\n    }\n}",
			[]string{"3:11 condition must be Bool, found Int"}},
		{"arg count", "func g(a: Int, b: Int) -> Int\n{\n    return a\n}\nfunc f()\n{\n    g(1)\n}",
			[]string{"7:5 `g` takes 2 arguments, but 1 was given"}},
		{"arg type", "func g(a: Int)\n{\n}\nfunc f()\n{\n    g(1.5)\n}",
			[]string{"6:7 argument `a` of `g` must be Int, found Float"}},
		{"wrong return type", "func f() -> Int\n{\n    return \"a\"\n}",
			[]string{"3:12 `f` returns Int, but this is String"}},
		{"bare return", "func f() -> Int\n{\n    return\n}",
			[]string{"3:5 `f` must return Int, but this `return` has no value"}},
		{"return value from nothing", "func f()\n{\n    return 1\n}",
			[]string{"3:12 `f` returns nothing, so `return` can't have a value"}},
		{"missing return", "func f(a: Int) -> Int\n{\n    if a > 0\n    {\n        return 1\n    }\n}",
			[]string{"7:1 missing return at the end of `f`"}},
		{"missing return while", "func f() -> Int\n{\n    while true\n    {\n        return 1\n    }\n}",
			[]string{"7:1 missing return at the end of `f`"}},
		{"no value used", "func g()\n{\n}\nfunc f()\n{\n    con x = g()\n}",
			[]string{"6:13 `g` returns nothing, so it can't be used as a value"}},
		{"redeclare", "func f()\n{\n    con x = 1\n    if true\n    {\n        con x = 2\n    }\n}",
			[]string{"6:13 `x` is already a variable"}},
		{"shadow param", "func f(a: Int)\n{\n    con a = 1\n}",
			[]string{"3:9 `a` is already a parameter"}},
		{"shadow function", "func f()\n{\n    con f = 1\n}",
			[]string{"3:9 `f` is already the name of a function"}},
		{"duplicate function", "func f()\n{\n}\nfunc f()\n{\n}",
			[]string{"4:6 `f` is declared twice"}},
		{"annotation", "func f()\n{\n    con x: Int = 1.0\n}",
			[]string{"3:18 `x` is declared as Int, but the value is Float"}},
		{"print object", "func g()\n{\n}\nfunc f()\n{\n    print(1, 2)\n}",
			[]string{"6:5 `print` takes 1 argument, but 2 were given"}},
		{"float of float", "func f() -> Float\n{\n    return Float(1.0)\n}",
			[]string{"3:18 this is already a Float"}},
		{"no cascade", "func f() -> Int\n{\n    con x = y + 1\n    return x * 2\n}",
			[]string{"3:13 unknown name `y`"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errors(t, tt.src+mainFunc)
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

func TestMainFunc(t *testing.T) {
	if got := errors(t, "func f()\n{\n}"); len(got) != 1 || got[0] != "1:1 the program has no `main` function" {
		t.Errorf("no main: %v", got)
	}
	if got := errors(t, "func main(a: Int) -> Int\n{\n    return a\n}"); len(got) != 1 || !strings.Contains(got[0], "`main` must be declared") {
		t.Errorf("bad main: %v", got)
	}
}

func TestHelp(t *testing.T) {
	src := diag.NewSource("t.brz", []byte("func f(a: Int) -> Float\n{\n    return a + 2.0\n}"+mainFunc))
	f, _ := parser.Parse(src)
	_, diags := Check(f, src)
	if len(diags) != 1 || len(diags[0].Help) != 1 || diags[0].Help[0] != "convert explicitly: Float(a) + 2.0" {
		t.Errorf("got %+v", diags)
	}
}
