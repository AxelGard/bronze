package check

import (
	"fmt"
	"strings"

	"github.com/AxelGard/bronze/internal/ast"
)

// Inheritance: `class Dog use Animal`. A class has at most one class
// parent (Error counts as one). It has every field and method of its
// parent, except static methods and inits, and can be used wherever its
// parent can. Replacing a parent's method needs `override`, with exactly
// the parent's signature.

// classUses resolves the names after `use`.
func (c *checker) classUses(cls *Class) {
	for idx, u := range cls.Decl.Uses {
		parent, ok := c.classes[u.Name]
		switch {
		case c.ifaces[u.Name] != nil:
			c.ref(u, c.ifaces[u.Name].Sym)
			cls.Ifaces = append(cls.Ifaces, c.ifaces[u.Name])
			cls.ifaceAt = append(cls.ifaceAt, idx)
		case ok && parent.Data:
			c.ref(u, parent.Sym)
			d := c.errorf(u.Sp, "`%s` is a data class, and can't be used", u.Name)
			d.Notes = []string{"a data class is complete as it is; use an interface to share methods"}
		case ok && cls.Data && !parent.Decl.Builtin:
			c.ref(u, parent.Sym)
			d := c.errorf(u.Sp, "a data class can use interfaces and `Error`, but not the class `%s`", u.Name)
			d.Help = []string{fmt.Sprintf("make `%s` a class, or `%s` an interface", cls.Name, u.Name)}
		case ok:
			c.ref(u, parent.Sym)
			if cls.Super != nil {
				d := c.errorf(u.Sp, "a class can use only one class in 0.3, and `%s` already uses `%s`", cls.Name, cls.Super.Name)
				d.Notes = []string{"using several classes is planned for after 1.0"}
				continue
			}
			cls.Super = parent
		case c.enums[u.Name] != nil:
			c.ref(u, c.enums[u.Name].Sym)
			c.errorf(u.Sp, "`%s` is an enum, and can't be used", u.Name)
		case builtinTypes[u.Name] != nil || genericTypes[u.Name]:
			c.errorf(u.Sp, "`%s` is a built-in type, and can't be used", u.Name)
		case c.broken[u.Name]:
		default:
			c.errorf(u.Sp, "unknown class or interface `%s`", u.Name)
		}
	}
}

// breakCycle reports a class that uses itself, through its parents, and
// removes the parent that closes the circle.
func (c *checker) breakCycle(cls *Class) {
	seen := map[*Class]bool{cls: true}
	for k := cls; k.Super != nil; k = k.Super {
		if k.Super == cls {
			var names []string
			for p := cls; ; p = p.Super {
				names = append(names, p.Name)
				if p == k {
					break
				}
			}
			d := c.errorf(cls.Decl.Name.Sp, "`%s` uses itself", cls.Name)
			d.Notes = []string{"the parents go around in a circle: " + strings.Join(append(names, cls.Name), " uses ")}
			cls.Super = nil
			return
		}
		if seen[k.Super] {
			return // a circle further up, reported at its own classes
		}
		seen[k.Super] = true
	}
}

// classMembers records the fields, methods and inits of a class, after
// those of its parent. The types of fields are found later, from their
// initial values.
func (c *checker) classMembers(cls *Class) {
	if cls.state != 0 {
		return
	}
	cls.state = 1
	d := cls.Decl
	parent := cls.Super
	if parent != nil {
		c.classMembers(parent)
		for name, f := range parent.Fields {
			cls.Fields[name] = f
		}
		for name, sig := range parent.Methods {
			if !sig.static {
				cls.Methods[name] = sig
			}
		}
	}
	for _, f := range d.Fields {
		if pf := c.parentField(cls, f.Name.Name); pf != nil {
			d := c.errorf(f.Name.Sp, "`%s` is already a field of `%s`", f.Name.Name, pf.Owner.Name)
			d.Help = []string{fmt.Sprintf("set it in `init` instead: `this.%s = ...`", f.Name.Name)}
			continue
		}
		if prev, ok := cls.Fields[f.Name.Name]; ok {
			c.errorf(f.Name.Sp, "`%s` already has a field `%s`", cls.Name, f.Name.Name).Notes = []string{
				fmt.Sprintf("the first one is at line %d", prev.Decl.Name.Sp.Start.Line),
			}
			continue
		}
		cls.Fields[f.Name.Name] = &Field{Name: f.Name.Name, Owner: cls, Mut: f.Mut, Decl: f, sym: c.newSym(f.Name, SymField, d.Builtin)}
	}
	own := map[string]*funcSig{}
	for _, m := range d.Methods {
		name := m.Name.Name
		sig := c.signature(m, cls)
		if f, ok := cls.Fields[name]; ok {
			where := fmt.Sprintf("the field is at line %d", f.Decl.Name.Sp.Start.Line)
			if f.Owner != cls {
				where = fmt.Sprintf("the field is declared in `%s`", f.Owner.Name)
			}
			c.errorf(m.Name.Sp, "`%s` already has a field `%s`", cls.Name, name).Notes = []string{
				where + "; a method can't have the same name",
			}
			continue
		}
		if sig.isInit {
			if m.Override {
				d := c.errorf(m.OverrideSp, "`init` can't be overridden")
				d.Notes = []string{"each class has its own `init`s; call the parent's with `super(...).init(...)`"}
			}
			if prev, ok := cls.Inits[len(sig.params)]; ok {
				d := c.errorf(m.Name.Sp, "`%s` already has an `init` with %s", cls.Name, plural(len(sig.params), "parameter"))
				d.Notes = []string{
					fmt.Sprintf("the other one is at line %d", prev.line),
					"several `init`s must have different numbers of parameters",
				}
				continue
			}
			cls.Inits[len(sig.params)] = sig
			continue
		}
		if prev, ok := own[name]; ok {
			c.errorf(m.Name.Sp, "`%s` already has a method `%s`", cls.Name, name).Notes = []string{
				fmt.Sprintf("the first one is at line %d", prev.line),
				"only `init` can be declared several times",
			}
			continue
		}
		own[name] = sig
		sig.sym = c.newSym(m.Name, SymMethod, d.Builtin)
		c.override(cls, sig, parentMethod(parent, name))
		cls.Methods[name] = sig
	}
	if cls.Data {
		c.dataConstructor(cls)
	}
	if len(cls.Inits) == 0 {
		// A class without an init gets an empty one.
		cls.Inits[0] = &funcSig{name: "init", result: NoValue, class: cls, isInit: true, line: d.Name.Sp.Start.Line}
		if parent != nil && needsParentInit(parent) {
			d := c.errorf(d.Name.Sp, "`%s` must call `super(%s).init(...)` in an `init`, since `%s` has no `init()`", cls.Name, parent.Name, parent.Name)
			d.Notes = []string{fmt.Sprintf("`%s` gets an empty `init()`, which would run `%s`'s `init()`", cls.Name, parent.Name)}
			d.Help = []string{fmt.Sprintf("add `func init(...) { super(%s).init(...) }`", parent.Name)}
		}
	}
	cls.state = 2
}

// parentField returns the field of an ancestor of cls with that name.
func (c *checker) parentField(cls *Class, name string) *Field {
	if cls.Super == nil {
		return nil
	}
	return cls.Super.Fields[name]
}

// parentMethod returns the method of parent (or its ancestors) that a
// method with that name would replace, or nil.
func parentMethod(parent *Class, name string) *funcSig {
	if parent == nil {
		return nil
	}
	if sig := parent.Methods[name]; sig != nil {
		return sig
	}
	return nil
}

// needsParentInit reports whether a child of parent must call
// `super(parent).init(...)`: parent has no `init()` without parameters.
// The built-in Error is set up by its fields alone, so it never needs it.
func needsParentInit(parent *Class) bool {
	return parent.Inits[0] == nil && !parent.Decl.Builtin
}

// override checks a method against the parent method it replaces, if any.
func (c *checker) override(cls *Class, sig, prev *funcSig) {
	m := sig.decl
	switch {
	case prev == nil && m.Override:
		d := c.errorf(m.Name.Sp, "`%s` doesn't override anything", m.Name.Name)
		for _, i := range cls.Ifaces {
			if i.Methods[m.Name.Name] != nil {
				d.Notes = []string{fmt.Sprintf("`%s` is for the interface `%s`, and a method for an interface is written without `override`", m.Name.Name, i.Name)}
				d.Help = []string{"remove `override`"}
				return
			}
		}
		if cls.Super == nil {
			d.Notes = []string{fmt.Sprintf("`%s` has no class parent", cls.Name)}
		} else {
			d.Notes = []string{fmt.Sprintf("`%s` has no method `%s`", cls.Super.Name, m.Name.Name)}
		}
		d.Help = []string{"remove `override`"}
	case prev == nil:
	case sigVis(prev) == ast.Private:
		d := c.errorf(m.Name.Sp, "`%s` already has a private method `%s`", prev.class.Name, m.Name.Name)
		d.Notes = []string{"a private method can't be overridden, and a child can't use its name"}
	case prev.static || m.Static:
		d := c.errorf(m.Name.Sp, "`%s` is already a method of `%s`", m.Name.Name, prev.class.Name)
		d.Notes = []string{"static methods can't be overridden, or override a method"}
	case !m.Override:
		d := c.errorf(m.Name.Sp, "`%s` replaces `%s.%s`, and needs `override`", m.Name.Name, prev.class.Name, m.Name.Name)
		d.Help = []string{fmt.Sprintf("write `override func %s(...)`", m.Name.Name)}
	default:
		if why := sigDiff(sig, prev); why != "" {
			d := c.errorf(m.Name.Sp, "`%s` must %s, like `%s.%s`", m.Name.Name, why, prev.class.Name, m.Name.Name)
			d.Notes = []string{fmt.Sprintf("`%s.%s` is `%s`", prev.class.Name, m.Name.Name, sigHover(prev))}
		}
		c.link(sig.sym, prev.sym)
		if prev.sym != nil && !prev.sym.Builtin && !c.inPrelude {
			c.info.Defs = append(c.info.Defs, Ref{Span: m.OverrideSp, Sym: prev.sym})
		}
	}
}

// sigDiff says how sig differs from want, as in "`speak` must return
// String", or "" if they are the same.
func sigDiff(sig, want *funcSig) string {
	if len(sig.params) != len(want.params) {
		return "take " + plural(len(want.params), "parameter")
	}
	for i, p := range sig.params {
		wp := want.params[i]
		if !identical(p, wp) && !silent(p) && !silent(wp) {
			return fmt.Sprintf("take %s as parameter %d", wp, i+1)
		}
		mode, wmode := paramMode(sig, i), paramMode(want, i)
		if mode != wmode {
			if wmode == "" {
				return fmt.Sprintf("take parameter %d without `%s`", i+1, mode)
			}
			return fmt.Sprintf("take parameter %d as `%s`", i+1, wmode)
		}
	}
	if !sameResult(sig.result, want.result) {
		if want.result == NoValue {
			return "return nothing"
		}
		return "return " + resultText(want.result)
	}
	if sigVis(sig) != sigVis(want) {
		return "be " + sigVis(want).String()
	}
	if sig.mut != want.mut {
		if want.mut {
			return "be a `mut func`"
		}
		return "not be a `mut func`"
	}
	return ""
}

func paramMode(sig *funcSig, i int) string {
	if sig.decl != nil && sig.decl.Params[i].Mut {
		return "mut"
	}
	if sig.decl != nil && sig.decl.Params[i].Copy {
		return "copy"
	}
	return ""
}

func sameResult(a, b Type) bool {
	fa, okA := a.(*Failable)
	fb, okB := b.(*Failable)
	if okA || okB {
		return okA && okB && fa.Err == fb.Err && (fa.Value == fb.Value || identical(fa.Value, fb.Value))
	}
	return a == b || identical(a, b) || silent(a) || silent(b)
}

func resultText(t Type) string {
	if f, ok := t.(*Failable); ok && f.Value == NoValue {
		return "nothing, or fail with " + f.Err.Name
	}
	return t.String()
}

// ---------------------------------------------------------------------
// super

// superInit reports whether call is `super(X).init(...)`.
func superInit(e ast.Expr) bool {
	call, ok := e.(*ast.Call)
	if !ok {
		return false
	}
	m, ok := call.Fn.(*ast.Member)
	if !ok {
		return false
	}
	_, ok = m.X.(*ast.Super)
	return ok && m.Name.Name == "init"
}

// firstSuperInit returns the `super(X).init(...)` call that is the first
// statement of an init's body, or nil.
func firstSuperInit(fn *ast.FuncDecl) *ast.Call {
	if fn == nil || fn.Body == nil || len(fn.Body.Stmts) == 0 {
		return nil
	}
	if s, ok := fn.Body.Stmts[0].(*ast.ExprStmt); ok && superInit(s.X) {
		return s.X.(*ast.Call)
	}
	return nil
}

// checkParentInit checks that an init of a child either calls
// `super(Parent).init(...)` first, or that the parent has an `init()` to
// run by itself.
func (c *checker) checkParentInit(sig *funcSig) {
	if !sig.isInit || sig.decl == nil || sig.class.Super == nil {
		return
	}
	parent := sig.class.Super
	for _, st := range sig.decl.Body.Stmts {
		if es, ok := st.(*ast.ExprStmt); ok && superInit(es.X) {
			return // reported where it is, if it isn't first
		}
	}
	if needsParentInit(parent) {
		d := c.errorf(sig.decl.Name.Sp, "`%s` must call `super(%s).init(...)`, since `%s` has no `init()`", sig.class.Name, parent.Name, parent.Name)
		d.Notes = []string{fmt.Sprintf("without it, `%s`'s `init()` would run before this `init`", parent.Name)}
		d.Help = []string{fmt.Sprintf("make `super(%s).init(...)` the first statement", parent.Name)}
	}
}

// superCall checks `super(X).method(...)` and `super(X).init(...)`.
func (c *checker) superCall(e *ast.Call, m *ast.Member, sup *ast.Super) Type {
	parent := c.superClass(sup)
	if parent == nil {
		c.args(e)
		return Invalid
	}
	name := m.Name.Name
	if name == "init" {
		if c.fn.decl == nil || !c.fn.isInit || firstSuperInit(c.fn.decl) != e {
			d := c.errorf(e.Sp, "`super(%s).init(...)` must be the first statement of `init`", parent.Name)
			d.Notes = []string{"the parent is set up before the rest of the child's `init` runs"}
		}
		sig, ok := parent.Inits[len(e.Args)]
		if !ok {
			types := c.args(e)
			arities := parent.initArities()
			if len(arities) == 1 {
				c.arityError(e, "super("+parent.Name+").init", arities[0], parent.Inits[arities[0]])
				return NoValue
			}
			d := c.errorf(e.Sp, "`%s` has no `init` that takes %s", parent.Name, plural(len(types), "argument"))
			var ns []string
			for _, n := range arities {
				ns = append(ns, fmt.Sprint(n))
			}
			d.Notes = []string{fmt.Sprintf("its `init`s take %s arguments", strings.Join(ns, " or "))}
			return NoValue
		}
		if sig.sym != nil {
			c.ref(m.Name, sig.sym)
		}
		c.accessSig(m.Name, sig)
		c.checkArgs(e, parent.Name, sig, c.argsFor(e, sig.params))
		return NoValue
	}
	sig := parent.Methods[name]
	if sig == nil || sig.static {
		c.args(e)
		switch {
		case parent.Fields[name] != nil:
			c.errorf(m.Name.Sp, "`%s` is a field, not a method", name)
		case !parent.Decl.Bad:
			c.errorf(m.Name.Sp, "`%s` has no method `%s`", parent.Name, name)
		}
		return Invalid
	}
	c.ref(m.Name, sig.sym)
	c.accessSig(m.Name, sig)
	if sig.mut && c.thisReadOnly() {
		d := c.errorf(sup.Sp, "`this` is read only")
		d.Label = fmt.Sprintf("`%s` is a `mut func`", name)
		d.Help = c.readOnlyHelp(&ast.This{Sp: sup.Sp})
	}
	c.checkArgs(e, name, sig, c.argsFor(e, sig.params))
	return sig.result
}

// superClass checks `super(X)` and returns X, the class parent of the
// class being checked, or nil after reporting an error.
func (c *checker) superClass(sup *ast.Super) *Class {
	switch {
	case c.fn == nil:
		c.errorf(sup.Sp, "`super` can't be used in the initial value of a field")
		return nil
	case c.fn.class == nil:
		c.errorf(sup.Sp, "`super` can only be used inside methods")
		return nil
	case c.fn.static:
		c.errorf(sup.Sp, "`super` can't be used in a static method")
		return nil
	}
	cls := c.fn.class
	name := sup.Class.Name
	if k, ok := c.classes[name]; ok {
		c.ref(sup.Class, k.Sym)
	}
	switch {
	case cls.Super == nil:
		d := c.errorf(sup.Class.Sp, "`%s` has no class parent", cls.Name)
		d.Notes = []string{fmt.Sprintf("`super(...)` calls the methods of the class after `use`, as in `class %s use Parent`", cls.Name)}
		return nil
	case cls.Super.Name != name:
		d := c.errorf(sup.Class.Sp, "`%s` is not the class parent of `%s`", name, cls.Name)
		d.Help = []string{fmt.Sprintf("write `super(%s)`", cls.Super.Name)}
		return nil
	}
	return cls.Super
}
