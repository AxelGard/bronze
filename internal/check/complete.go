package check

import (
	"slices"
	"sort"
	"strings"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/diag"
	"github.com/AxelGard/bronze/internal/parser"
)

// Marker is put at the cursor for completion, as a call: `acc` becomes
// `acc__complete__()`. A call is a valid statement and a valid value, so
// the checker reaches it wherever the cursor is in an expression.
const Marker = "__complete__"

// CompletionKind is what a completion item is.
type CompletionKind int

const (
	CompleteVariable CompletionKind = iota
	CompleteFunction
	CompleteMethod
	CompleteField
	CompleteClass
	CompleteEnum
	CompleteCase
	CompleteKeyword
	CompleteType
)

// Completion is one thing that can be written at the cursor.
type Completion struct {
	Label  string
	Kind   CompletionKind
	Detail string // its hover text, such as `con total: Int`
}

// Completions are what can be written at the marker.
type Completions struct {
	// Found is true if the checker reached the marker. Items are then what
	// fits there: the members of a value after a `.`, or the names in
	// scope.
	Found bool
	Items []Completion
	// Types are the type names, for a cursor in a written type, where the
	// checker doesn't see the marker.
	Types []Completion
}

// Complete checks a program with the Marker in it and returns what can be
// written there. Bodies with syntax errors are checked too, so that one
// unfinished line doesn't stop completion in the rest of the function.
func Complete(file *ast.File, src *diag.Source) *Completions {
	c := &checker{
		src:      src,
		info:     &Info{Types: map[ast.Expr]Type{}},
		funcs:    map[string]*funcSig{},
		classes:  map[string]*Class{},
		enums:    map[string]*Enum{},
		broken:   map[string]bool{},
		complete: &Completions{},
	}
	c.run(file, append(append([]ast.Decl{}, parser.Prelude()...), file.Decls...))
	c.complete.Types = c.typeNames()
	return c.complete
}

func isMarker(name string) bool { return strings.HasSuffix(name, Marker) }

// atMarker reports whether id is the marker, and the checker should record
// completions instead of checking it.
func (c *checker) atMarker(id *ast.Ident) bool {
	return c.complete != nil && !c.complete.Found && isMarker(id.Name)
}

// completeScope records every name that can be used at this point.
func (c *checker) completeScope() {
	c.complete.Found = true
	var items []Completion
	seen := map[string]bool{}
	add := func(it Completion) {
		if !seen[it.Label] && !isMarker(it.Label) {
			seen[it.Label] = true
			items = append(items, it)
		}
	}
	for s := c.scope; s != nil; s = s.parent {
		names := make([]string, 0, len(s.vars))
		for n := range s.vars {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			add(Completion{Label: n, Kind: CompleteVariable, Detail: varHover(n, s.vars[n])})
		}
	}
	if c.fn != nil && (c.fn.class != nil || c.fn.enum != nil) {
		add(Completion{Label: "this", Kind: CompleteKeyword})
	}
	for _, n := range sortedKeys(c.funcs) {
		add(Completion{Label: n, Kind: CompleteFunction, Detail: sigHover(c.funcs[n])})
	}
	for _, n := range sortedKeys(c.classes) {
		add(Completion{Label: n, Kind: CompleteClass, Detail: "class " + n})
	}
	for _, n := range sortedKeys(c.enums) {
		add(Completion{Label: n, Kind: CompleteEnum, Detail: "enum " + n})
	}
	for _, n := range sortedKeys(builtinFuncHovers) {
		add(Completion{Label: n, Kind: CompleteFunction, Detail: builtinFuncHovers[n][0]})
	}
	for _, kw := range []string{"con", "mut", "if", "else", "while", "for", "in", "break", "continue",
		"return", "match", "true", "false", "null", "not", "and", "or", "is"} {
		add(Completion{Label: kw, Kind: CompleteKeyword})
	}
	c.complete.Items = items
}

// completeMember records the members of x, for `x.` with the marker after
// the dot.
func (c *checker) completeMember(x ast.Expr) {
	c.complete.Found = true
	if cls := c.className(x); cls != nil {
		for _, n := range sortedKeys(cls.Methods) {
			if sig := cls.Methods[n]; sig.static {
				c.complete.Items = append(c.complete.Items, Completion{Label: n, Kind: CompleteMethod, Detail: sigHover(sig)})
			}
		}
		return
	}
	if en := c.enumName(x); en != nil {
		for _, cs := range en.Decl.Cases {
			c.complete.Items = append(c.complete.Items, Completion{Label: cs.Name, Kind: CompleteCase, Detail: caseHover(en, cs.Name)})
		}
		if en.Value != nil {
			c.complete.Items = append(c.complete.Items, Completion{Label: "from", Kind: CompleteMethod, Detail: fromMethod(en).Signature()})
		}
		return
	}
	t := nonNull(c.value(x))
	var items []Completion
	builtins := func(ms map[string]*method) {
		for _, n := range sortedKeys(ms) {
			items = append(items, Completion{Label: n, Kind: CompleteMethod, Detail: ms[n].Signature()})
		}
	}
	switch t := t.(type) {
	case *Class:
		for _, n := range sortedKeys(t.Fields) {
			f := t.Fields[n]
			kw := "con "
			if f.Mut {
				kw = "mut "
			}
			detail := kw + n
			if ft := typeText(c.fieldType(f)); ft != "" {
				detail += ": " + ft
			}
			items = append(items, Completion{Label: n, Kind: CompleteField, Detail: detail})
		}
		for _, n := range sortedKeys(t.Methods) {
			if !t.Methods[n].static {
				items = append(items, Completion{Label: n, Kind: CompleteMethod, Detail: sigHover(t.Methods[n])})
			}
		}
	case *Enum:
		if t.Value != nil {
			items = append(items, Completion{Label: "value", Kind: CompleteField, Detail: valueSym(t).Hover})
		}
		for _, n := range sortedKeys(t.Methods) {
			items = append(items, Completion{Label: n, Kind: CompleteMethod, Detail: sigHover(t.Methods[n])})
		}
	case *List:
		builtins(listMethods(t))
	case *Map:
		builtins(mapMethods(t))
	default:
		if t == String {
			builtins(stringMethods())
		}
	}
	c.complete.Items = items
}

// typeNames returns every type name that can be written.
func (c *checker) typeNames() []Completion {
	var items []Completion
	for _, n := range []string{"Int", "Float", "Bool", "String", "List", "Map"} {
		items = append(items, Completion{Label: n, Kind: CompleteType})
	}
	for _, n := range sortedKeys(c.classes) {
		items = append(items, Completion{Label: n, Kind: CompleteClass, Detail: "class " + n})
	}
	for _, n := range sortedKeys(c.enums) {
		items = append(items, Completion{Label: n, Kind: CompleteEnum, Detail: "enum " + n})
	}
	return items
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
