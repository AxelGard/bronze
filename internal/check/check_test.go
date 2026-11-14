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

const counterClass = `class Counter
{
    con name = ""
    mut count = 0

    func init(name: String)
    {
        this.name = name
    }

    func init(name: String, count: Int)
    {
        this.name = name
        this.count = count
    }

    func increment()
    {
        this.count = this.count + 1
    }

    func get() -> Int
    {
        return this.count
    }
}
`

func TestClasses(t *testing.T) {
	text, err := os.ReadFile("../../examples/projects/counter.brz")
	if err != nil {
		t.Fatal(err)
	}
	if errs := errors(t, string(text)); len(errs) > 0 {
		t.Errorf("counter.brz: %s", strings.Join(errs, "\n"))
	}

	ok := []string{
		"func f()\n{\n    con c = Counter(\"a\", 2)\n    c.increment()\n    c.count = c.get() + 1\n    print(c == Counter(\"b\"))\n}",
		"class Empty\n{\n}\nfunc f()\n{\n    con e = Empty()\n    con same: Empty = e\n}",
		"class Box\n{\n    mut inner = Counter(\"x\")\n}\nfunc f() -> Int\n{\n    return Box().inner.get()\n}",
		"func make() -> Counter\n{\n    return Counter(\"m\")\n}\nfunc use_it(c: Counter) -> String\n{\n    return c.name\n}",
	}
	for _, src := range ok {
		if errs := errors(t, counterClass+src+mainFunc); len(errs) > 0 {
			t.Errorf("%q:\n%s", src, strings.Join(errs, "\n"))
		}
	}
}

func TestClassErrors(t *testing.T) {
	// Line numbers are counted within src.
	tests := []struct {
		name, src string
		want      []string
	}{
		{"con field outside init", "func f(c: Counter)\n{\n    c.name = \"x\"\n}",
			[]string{"3:7 cannot assign to `name`: it is a `con` field"}},
		{"unknown field", "func f(c: Counter) -> Int\n{\n    return c.size\n}",
			[]string{"3:14 `Counter` has no field `size`"}},
		{"unknown method", "func f(c: Counter)\n{\n    c.reset()\n}",
			[]string{"3:7 `Counter` has no method `reset`"}},
		{"method as field", "func f(c: Counter) -> Int\n{\n    return c.get\n}",
			[]string{"3:14 `get` is a method, not a field"}},
		{"field as method", "func f(c: Counter)\n{\n    c.count()\n}",
			[]string{"3:7 `count` is a field, not a method"}},
		{"no init with that arity", "func f()\n{\n    con c = Counter()\n}",
			[]string{"3:13 `Counter` has no `init` that takes 0 arguments"}},
		{"init arg type", "func f()\n{\n    con c = Counter(1)\n}",
			[]string{"3:21 argument `name` of `Counter` must be String, found Int"}},
		{"method arg count", "func f(c: Counter)\n{\n    c.increment(1)\n}",
			[]string{"3:5 `increment` takes 0 arguments, but 1 was given"}},
		{"field type", "func f(c: Counter)\n{\n    c.count = 1.5\n}",
			[]string{"3:15 cannot assign Float to `count`, which is Int"}},
		{"this outside method", "func f() -> Int\n{\n    return this.count\n}",
			[]string{"3:12 `this` can only be used inside methods"}},
		{"print object", "func f()\n{\n    print(Counter(\"a\"))\n}",
			[]string{"3:11 cannot print an object of class Counter"}},
		{"class as value", "func f()\n{\n    con c = Counter\n}",
			[]string{"3:13 `Counter` is a class, not a value"}},
		{"call init", "func f(c: Counter)\n{\n    c.init(\"x\")\n}",
			[]string{"3:7 `init` can't be called directly"}},
		{"Int has no fields", "func f() -> Int\n{\n    con x = 1\n    return x.count\n}",
			[]string{"4:14 Int has no fields"}},
		{"compare classes", "class Other\n{\n}\nfunc f(c: Counter) -> Bool\n{\n    return c == Other()\n}",
			[]string{"6:12 cannot compare Counter and Other with `==`"}},
		{"shadow class", "func f()\n{\n    con Counter = 1\n}",
			[]string{"3:9 `Counter` is already the name of a class"}},
		{"duplicate init", "class Two\n{\n    func init(a: Int)\n    {\n    }\n    func init(b: String)\n    {\n    }\n}",
			[]string{"6:10 `Two` already has an `init` with 1 parameter"}},
		{"duplicate member", "class Two\n{\n    mut a = 1\n    func a()\n    {\n    }\n}",
			[]string{"4:10 `Two` already has a field `a`"}},
		{"init return value", "class Two\n{\n    func init()\n    {\n        return 1\n    }\n}",
			[]string{"5:16 `return` in `init` can't have a value"}},
		{"init return type", "class Two\n{\n    func init() -> Int\n    {\n    }\n}",
			[]string{"3:20 `init` returns the new `Two`, so its return type can only be `Two`"}},
		{"this in field", "class Two\n{\n    mut a = 1\n    mut b = this.a\n}",
			[]string{"4:13 `this` can't be used in the initial value of a field"}},
		{"field without this", "class Two\n{\n    mut a = 1\n    func get() -> Int\n    {\n        return a\n    }\n}",
			[]string{"6:16 unknown name `a`"}},
		{"method missing return", "class Two\n{\n    func get() -> Int\n    {\n    }\n}",
			[]string{"5:5 missing return at the end of `get`"}},
		{"duplicate class", "class Counter\n{\n}",
			[]string{"1:7 `Counter` is declared twice"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset := strings.Count(counterClass, "\n")
			var got []string
			for _, e := range errors(t, counterClass+tt.src+mainFunc) {
				var line int
				var rest string
				fmt.Sscanf(e, "%d:", &line)
				rest = e[strings.Index(e, ":"):]
				got = append(got, fmt.Sprint(line-offset)+rest)
			}
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

const statusEnum = `enum Status
{
    Active
    Frozen
    Closed

    func label() -> String
    {
        return match this
        {
            Status.Active => "active"
            Status.Frozen => "frozen"
            Status.Closed => "closed"
        }
    }

    func is_open() -> Bool
    {
        return this != Status.Closed
    }
}
`

// relErrors checks prefix+src and returns errors with line numbers
// counted within src.
func relErrors(t *testing.T, prefix, src string) []string {
	t.Helper()
	offset := strings.Count(prefix, "\n")
	var got []string
	for _, e := range errors(t, prefix+src+mainFunc) {
		var line int
		fmt.Sscanf(e, "%d:", &line)
		got = append(got, fmt.Sprint(line-offset)+e[strings.Index(e, ":"):])
	}
	return got
}

func TestEnums(t *testing.T) {
	ok := []string{
		"func f(s: Status) -> String\n{\n    print(s)\n    print(Status.Active.label())\n    return s.label()\n}",
		"func f(s: Status) -> Int\n{\n    if s == Status.Frozen and s.is_open()\n    {\n        return 1\n    }\n    return match s\n    {\n        Status.Active => 0\n        _ => panic(\"no\")\n    }\n}",
		"class Account\n{\n    mut status = Status.Active\n}\nfunc f(a: Account)\n{\n    a.status = Status.Closed\n}",
		"func f(n: Int) -> String\n{\n    return match n\n    {\n        0 => \"zero\"\n        -1 => \"minus one\"\n        _ => \"other\"\n    }\n}",
		"func f(b: Bool, s: String) -> Int\n{\n    con x = match b\n    {\n        true => 1\n        false => 2\n    }\n    return x + match s\n    {\n        \"a\" => 1\n        _ => 0\n    }\n}",
		"func f(s: Status) -> Int\n{\n    return match s\n    {\n        Status.Active => panic(\"a\")\n        Status.Frozen => 1\n        Status.Closed => panic(\"c\")\n    }\n}",
		"func f(s: Status) -> String\n{\n    return \"is \" + String(s)\n}",
		// match statements
		"func f(s: Status)\n{\n    match s\n    {\n        Status.Active => print(1)\n        Status.Frozen =>\n        {\n            con x = 2\n            print(x)\n        }\n        _ => { }\n    }\n}",
		"func f(s: Status) -> Int\n{\n    match s\n    {\n        Status.Active => { return 1 }\n        _ => panic(\"no\")\n    }\n}",
		"func f(s: Status, n: Int)\n{\n    match s\n    {\n        Status.Active => match n\n        {\n            1 => print(1)\n            _ => { }\n        }\n        _ => { }\n    }\n}",
	}
	for _, src := range ok {
		if errs := relErrors(t, statusEnum, src); len(errs) > 0 {
			t.Errorf("%q:\n%s", src, strings.Join(errs, "\n"))
		}
	}
}

func TestEnumErrors(t *testing.T) {
	tests := []struct {
		name, src string
		want      []string
	}{
		{"unknown case", "func f() -> Status\n{\n    return Status.Open\n}",
			[]string{"3:19 `Status` has no case `Open`"}},
		{"method as case", "func f() -> String\n{\n    return Status.label\n}",
			[]string{"3:19 `label` is a method of each `Status` case, not a case"}},
		{"enum as value", "func f()\n{\n    con s = Status\n}",
			[]string{"3:13 `Status` is an enum, not a value"}},
		{"enum called", "func f()\n{\n    con s = Status()\n}",
			[]string{"3:13 `Status` is an enum, and can't be called"}},
		{"unknown method", "func f(s: Status)\n{\n    s.close()\n}",
			[]string{"3:7 `Status` has no method `close`"}},
		{"no fields", "func f(s: Status) -> Int\n{\n    return s.count\n}",
			[]string{"3:14 Status has no fields"}},
		{"ordering", "func f(s: Status) -> Bool\n{\n    return s < Status.Active\n}",
			[]string{"3:12 cannot compare Status values with `<`"}},
		{"duplicate case", "enum Two\n{\n    A\n    A\n}",
			[]string{"4:5 `Two` already has a case `A`"}},
		{"enum init", "enum Two\n{\n    A\n    func init()\n    {\n    }\n}",
			[]string{"4:10 an enum can't have an `init`"}},
		{"shadow enum", "func f()\n{\n    con Status = 1\n}",
			[]string{"3:9 `Status` is already the name of an enum"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relErrors(t, statusEnum, tt.src)
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

func TestMatchErrors(t *testing.T) {
	m := func(x, arms string) string {
		return "func f(s: Status, n: Int, b: Bool, x: Float) -> Int\n{\n    return match " + x + "\n    {\n" + arms + "    }\n}"
	}
	tests := []struct {
		name, src string
		want      []string
	}{
		{"missing cases", m("s", "        Status.Active => 1\n"),
			[]string{"3:12 match does not cover `Status.Frozen` and `Status.Closed`"}},
		{"missing false", m("b", "        true => 1\n"),
			[]string{"3:12 match does not cover `false`"}},
		{"int needs wildcard", m("n", "        1 => 1\n        2 => 2\n"),
			[]string{"3:12 match on Int needs a `_` arm"}},
		{"duplicate", m("n", "        1 => 1\n        1 => 2\n        _ => 3\n"),
			[]string{"6:9 `1` is already matched"}},
		{"after wildcard", m("n", "        _ => 1\n        2 => 2\n"),
			[]string{"6:9 this arm is never used"}},
		{"wrong pattern type", m("n", "        \"a\" => 1\n        _ => 2\n"),
			[]string{"5:9 the pattern `\"a\"` is String, but the matched value is Int"}},
		{"wrong enum", "enum Other\n{\n    X\n}\n" + m("s", "        Other.X => 1\n        _ => 2\n"),
			[]string{"9:9 the pattern `Other.X` is Other, but the matched value is Status"}},
		{"unknown enum", m("s", "        Nope.X => 1\n        _ => 2\n"),
			[]string{"5:9 unknown enum `Nope`"}},
		{"unknown case pattern", m("s", "        Status.Open => 1\n        _ => 2\n"),
			[]string{"5:16 `Status` has no case `Open`"}},
		{"arm types", m("n", "        1 => 1\n        _ => \"many\"\n"),
			[]string{"6:14 this arm is String, but the arms before it are Int"}},
		{"float", m("x", "        _ => 1\n"),
			[]string{"3:18 cannot match on Float"}},
		{"block arm with value used", m("n", "        1 => { print(1) }\n        _ => 2\n"),
			[]string{"5:14 a `{ }` arm has no value"}},
		{"statement arm value", "func f(n: Int)\n{\n    match n\n    {\n        1 => 5\n        _ => { }\n    }\n}",
			[]string{"5:14 the value of this arm is not used"}},
		{"statement must cover", "func f(s: Status)\n{\n    match s\n    {\n        Status.Active => { }\n    }\n}",
			[]string{"3:5 match does not cover `Status.Frozen` and `Status.Closed`"}},
		{"statement missing return", "func f(s: Status) -> Int\n{\n    match s\n    {\n        Status.Active => { return 1 }\n        _ => { print(2) }\n    }\n}",
			[]string{"8:1 missing return at the end of `f`"}},
		{"match type", "func f(n: Int) -> String\n{\n    return match n\n    {\n        _ => 1\n    }\n}",
			[]string{"3:12 `f` returns String, but this is Int"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relErrors(t, statusEnum, tt.src)
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
