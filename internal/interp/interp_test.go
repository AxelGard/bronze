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
