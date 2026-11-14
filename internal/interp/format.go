package interp

import (
	"math"
	"strconv"
	"strings"
)

// Format returns the text `print` and `String()` show for a value.
func Format(v Value) string {
	switch v := v.(type) {
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return FormatFloat(v)
	case bool:
		return strconv.FormatBool(v)
	case string:
		return v
	case *Case:
		return v.Name
	case *List:
		var b strings.Builder
		b.WriteByte('[')
		for i, x := range v.Elems {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(formatInner(x))
		}
		b.WriteByte(']')
		return b.String()
	case *Map:
		var b strings.Builder
		b.WriteByte('{')
		for i, k := range v.keys {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(formatInner(k))
			b.WriteString(": ")
			b.WriteString(formatInner(v.vals[i]))
		}
		b.WriteByte('}')
		return b.String()
	}
	panic("interp: cannot format value")
}

// formatInner formats a value inside a list or a map, where Strings are
// quoted.
func formatInner(v Value) string {
	if s, ok := v.(string); ok {
		return quote(s)
	}
	return Format(v)
}

// quote writes a String in quotes, with `\n`, `\t`, `"` and `\` escaped.
func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// FormatFloat writes the shortest text that reads back as the same
// Float, always with a `.` or an exponent: 6.0, 0.1, 1e+21, 1e-07.
// Plain decimals are used for 1e-7 <= |f| < 1e21.
func FormatFloat(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	case math.IsNaN(f):
		return "NaN"
	}
	if a := math.Abs(f); a != 0 && (a < 1e-7 || a >= 1e21) {
		return strconv.FormatFloat(f, 'e', -1, 64)
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}
