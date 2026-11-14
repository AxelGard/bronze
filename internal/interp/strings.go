package interp

import (
	"strings"
	"unicode/utf8"

	"github.com/AxelGard/bronze/internal/ast"
)

// clamp resolves the bounds of a slice of something of length n: negative
// bounds count from the end, and then both are clamped to 0..n. hi is
// exclusive; inclusive means it was written with `..=`. A slice never
// fails: if lo is past hi it is empty.
func clamp(lo, hi *int64, inclusive bool, n int) (int, int) {
	l, h := int64(0), int64(n)
	if lo != nil {
		l = *lo
		if l < 0 {
			l += int64(n)
		}
	}
	if hi != nil {
		h = *hi
		if h < 0 {
			h += int64(n)
		}
		if inclusive && h < int64(n) {
			h++
		}
	}
	l = min(max(l, 0), int64(n))
	h = min(max(h, 0), int64(n))
	if l > h {
		l = h
	}
	return int(l), int(h)
}

// slice runs `x[a..b]` and the other slices. The result is a new list or
// String.
func (in *interp) slice(x *ast.Slice, e *env) Value {
	v := in.expr(x.X, e)
	var lo, hi *int64
	if x.Lo != nil {
		n := in.expr(x.Lo, e).(int64)
		lo = &n
	}
	if x.Hi != nil {
		n := in.expr(x.Hi, e).(int64)
		hi = &n
	}
	switch v := v.(type) {
	case *List:
		l, h := clamp(lo, hi, x.Inclusive, len(v.Elems))
		return &List{Elems: append([]Value(nil), v.Elems[l:h]...)}
	case string:
		rs := []rune(v)
		l, h := clamp(lo, hi, x.Inclusive, len(rs))
		return string(rs[l:h])
	}
	panic("interp: cannot slice")
}

// charAt returns the character at index i of s as a String, or null.
func charAt(s string, i int64) Value {
	rs := []rune(s)
	if j, ok := position(i, len(rs)); ok {
		return string(rs[j])
	}
	return nil
}

// stringMethod calls a method of a String. Lengths and indexes count
// characters, not bytes.
func stringMethod(s, name string, args []Value) Value {
	arg := func(i int) string { return args[i].(string) }
	switch name {
	case "length":
		return int64(utf8.RuneCountInString(s))
	case "contains":
		return strings.Contains(s, arg(0))
	case "starts_with":
		return strings.HasPrefix(s, arg(0))
	case "ends_with":
		return strings.HasSuffix(s, arg(0))
	case "index_of":
		if i := strings.Index(s, arg(0)); i >= 0 {
			return int64(utf8.RuneCountInString(s[:i]))
		}
		return nil
	case "split":
		parts := strings.Split(s, arg(0))
		l := &List{Elems: make([]Value, len(parts))}
		for i, p := range parts {
			l.Elems[i] = p
		}
		return l
	case "trim":
		return strings.TrimSpace(s)
	case "to_upper":
		return strings.ToUpper(s)
	case "to_lower":
		return strings.ToLower(s)
	case "replace":
		if arg(0) == "" {
			return s
		}
		return strings.ReplaceAll(s, arg(0), arg(1))
	}
	panic("interp: unknown String method " + name)
}
