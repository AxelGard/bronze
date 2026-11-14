package interp

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/diag"
)

// position resolves an index into a list of length n: negative indexes count
// from the end. It reports false if the index is out of range.
func position(i int64, n int) (int, bool) {
	if i < 0 {
		i += int64(n)
	}
	if i < 0 || i >= int64(n) {
		return 0, false
	}
	return int(i), true
}

// index runs `x[i]`, which is null when there is nothing at i.
func (in *interp) index(x *ast.Index, e *env) Value {
	switch v := in.expr(x.X, e).(type) {
	case *List:
		if j, ok := position(in.expr(x.Index, e).(int64), len(v.Elems)); ok {
			return v.Elems[j]
		}
		return nil
	case *Map:
		val, _ := v.Get(in.expr(x.Index, e))
		return val
	case string:
		return charAt(v, in.expr(x.Index, e).(int64))
	}
	panic("interp: cannot index")
}

// slot is an element of a list or a value in a map, as the target of an
// assignment.
type slot struct {
	list *List
	i    int
	m    *Map
	key  Value
}

// indexTarget evaluates the parts of `x[i]` as an assignment target. An
// index out of range panics.
func (in *interp) indexTarget(x *ast.Index, e *env) slot {
	switch v := in.expr(x.X, e).(type) {
	case *List:
		i := in.expr(x.Index, e).(int64)
		j, ok := position(i, len(v.Elems))
		if !ok {
			in.fail(x.Sp, "index %d out of range for a list of length %d", i, len(v.Elems))
		}
		return slot{list: v, i: j}
	case *Map:
		return slot{m: v, key: in.expr(x.Index, e)}
	}
	panic("interp: cannot assign to an index")
}

// get returns the current value. A key that is not in the map panics, for
// `m[k] += v`.
func (in *interp) get(s slot, at diag.Span) Value {
	if s.m != nil {
		v, ok := s.m.Get(s.key)
		if !ok {
			in.fail(at, "key %s is not in the map", formatInner(s.key))
		}
		return v
	}
	return s.list.Elems[s.i]
}

// set replaces the value. Adding a key to a map that is being looped over
// panics.
func (in *interp) set(s slot, v Value, at diag.Span) {
	if s.m != nil {
		if _, ok := s.m.Get(s.key); !ok {
			in.mapChanging(s.m, at)
		}
		s.m.Set(s.key, v)
		return
	}
	s.list.Elems[s.i] = v
}

// changing panics if a list is being looped over, before a method changes
// its length or order.
func (in *interp) changing(l *List, at diag.Span) {
	if l.looping > 0 {
		in.fail(at, "list changed while looping over it")
	}
}

// listMethod calls a method of a list.
func (in *interp) listMethod(l *List, name string, args []Value, at diag.Span) Value {
	switch name {
	case "length":
		return int64(len(l.Elems))
	case "append":
		in.changing(l, at)
		l.Elems = append(l.Elems, args[0])
		return nil
	case "insert":
		in.changing(l, at)
		i := args[0].(int64)
		if i < 0 || i > int64(len(l.Elems)) {
			in.fail(at, "index %d out of range for insert into a list of length %d", i, len(l.Elems))
		}
		l.Elems = slices.Insert(l.Elems, int(i), args[1])
		return nil
	case "remove_at":
		in.changing(l, at)
		j, ok := position(args[0].(int64), len(l.Elems))
		if !ok {
			return nil
		}
		v := l.Elems[j]
		l.Elems = slices.Delete(l.Elems, j, j+1)
		return v
	case "pop":
		in.changing(l, at)
		if len(l.Elems) == 0 {
			return nil
		}
		v := l.Elems[len(l.Elems)-1]
		l.Elems = l.Elems[:len(l.Elems)-1]
		return v
	case "contains":
		return slices.IndexFunc(l.Elems, func(v Value) bool { return equal(v, args[0]) }) >= 0
	case "index_of":
		if i := slices.IndexFunc(l.Elems, func(v Value) bool { return equal(v, args[0]) }); i >= 0 {
			return int64(i)
		}
		return nil
	case "reverse":
		in.changing(l, at)
		slices.Reverse(l.Elems)
		return nil
	case "sort":
		in.changing(l, at)
		slices.SortStableFunc(l.Elems, compare)
		return nil
	case "join":
		parts := make([]string, len(l.Elems))
		for i, v := range l.Elems {
			parts[i] = v.(string)
		}
		return strings.Join(parts, args[0].(string))
	}
	panic("interp: unknown list method " + name)
}

// compare orders two Ints, Floats or Strings, for sort. NaN goes first.
func compare(a, b Value) int {
	switch a := a.(type) {
	case int64:
		return cmp.Compare(a, b.(int64))
	case float64:
		return cmp.Compare(a, b.(float64))
	case string:
		return cmp.Compare(a, b.(string))
	}
	panic(fmt.Sprintf("interp: cannot sort %T", a))
}

// equal is `==`: lists by value, objects by identity, and everything else
// by value.
func equal(a, b Value) bool {
	switch a := a.(type) {
	case *List:
		b, ok := b.(*List)
		return ok && slices.EqualFunc(a.Elems, b.Elems, equal)
	case *Map:
		b, ok := b.(*Map)
		return ok && mapEqual(a, b)
	}
	return a == b
}
