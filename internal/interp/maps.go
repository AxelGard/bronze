package interp

import (
	"slices"

	"github.com/AxelGard/bronze/internal/diag"
)

// Map is a map. Maps are shared, like objects, and keep their keys in the
// order they were first added. Keys are int64, string, bool or *Case
// values, so they can be Go map keys.
type Map struct {
	keys    []Value
	vals    []Value
	pos     map[Value]int // the position of each key in keys
	looping int           // how many `for` loops are looping over it
}

// NewMap returns an empty map.
func NewMap() *Map { return &Map{pos: map[Value]int{}} }

// Len returns the number of keys.
func (m *Map) Len() int { return len(m.keys) }

// Get returns the value for a key, and whether the key is there.
func (m *Map) Get(k Value) (Value, bool) {
	if i, ok := m.pos[k]; ok {
		return m.vals[i], true
	}
	return nil, false
}

// Set adds a key, or replaces its value in place.
func (m *Map) Set(k, v Value) {
	if i, ok := m.pos[k]; ok {
		m.vals[i] = v
		return
	}
	m.pos[k] = len(m.keys)
	m.keys = append(m.keys, k)
	m.vals = append(m.vals, v)
}

// Remove removes a key, and returns its value and whether it was there.
func (m *Map) Remove(k Value) (Value, bool) {
	i, ok := m.pos[k]
	if !ok {
		return nil, false
	}
	v := m.vals[i]
	m.keys = slices.Delete(m.keys, i, i+1)
	m.vals = slices.Delete(m.vals, i, i+1)
	delete(m.pos, k)
	for j := i; j < len(m.keys); j++ {
		m.pos[m.keys[j]] = j
	}
	return v, true
}

// mapChanging panics if a map is being looped over, before a key is added
// or removed.
func (in *interp) mapChanging(m *Map, at diag.Span) {
	if m.looping > 0 {
		in.fail(at, "map changed while looping over it")
	}
}

// mapMethod calls a method of a map.
func (in *interp) mapMethod(m *Map, name string, args []Value, at diag.Span) Value {
	switch name {
	case "length":
		return int64(m.Len())
	case "contains":
		_, ok := m.pos[args[0]]
		return ok
	case "remove":
		in.mapChanging(m, at)
		v, _ := m.Remove(args[0])
		return v
	case "keys":
		return &List{Elems: slices.Clone(m.keys)}
	case "values":
		return &List{Elems: slices.Clone(m.vals)}
	}
	panic("interp: unknown map method " + name)
}

// mapEqual compares two maps by value: the same keys, each with an equal
// value, in any order.
func mapEqual(a, b *Map) bool {
	if a.Len() != b.Len() {
		return false
	}
	for i, k := range a.keys {
		v, ok := b.Get(k)
		if !ok || !equal(a.vals[i], v) {
			return false
		}
	}
	return true
}
