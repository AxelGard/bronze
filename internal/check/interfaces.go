package check

import (
	"fmt"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/diag"
)

// Interfaces: `interface Shape { func area() -> Float }`. A class that says
// `use Shape` must have each of its methods, with exactly the same types,
// and is then a Shape. An interface can use other interfaces.

// Interface is an interface type. Its values are objects of classes that
// use it.
type Interface struct {
	Name    string
	Decl    *ast.InterfaceDecl
	Parents []*Interface
	Methods map[string]*funcSig // its own and its parents'
	all     map[*Interface]bool // itself and every interface it uses
	Sym     *Symbol
	state   int
}

func (i *Interface) String() string { return i.Name }

// interfaceUses resolves the names after `use` in an interface.
func (c *checker) interfaceUses(i *Interface) {
	for _, u := range i.Decl.Uses {
		switch {
		case c.ifaces[u.Name] != nil:
			p := c.ifaces[u.Name]
			c.ref(u, p.Sym)
			i.Parents = append(i.Parents, p)
		case c.classes[u.Name] != nil:
			c.ref(u, c.classes[u.Name].Sym)
			d := c.errorf(u.Sp, "an interface can only use other interfaces, and `%s` is a class", u.Name)
			d.Notes = []string{"an interface has no fields or bodies of its own to inherit"}
		case c.broken[u.Name]:
		default:
			c.errorf(u.Sp, "unknown interface `%s`", u.Name)
		}
	}
}

// interfaceMembers records the methods of an interface, after those of
// the interfaces it uses.
func (c *checker) interfaceMembers(i *Interface) {
	switch i.state {
	case 2:
		return
	case 1:
		c.errorf(i.Decl.Name.Sp, "`%s` uses itself", i.Name).Notes = []string{"the interfaces it uses go around in a circle"}
		return
	}
	i.state = 1
	i.all = map[*Interface]bool{i: true}
	for _, p := range i.Parents {
		c.interfaceMembers(p)
		if p.state != 2 {
			continue // a circle, reported once
		}
		for k := range p.all {
			i.all[k] = true
		}
		for name, sig := range p.Methods {
			if prev, ok := i.Methods[name]; ok && prev != sig {
				if why := sigDiff(sig, prev); why != "" {
					d := c.errorf(i.Decl.Name.Sp, "`%s` gets `%s` from both `%s` and `%s`, with different types", i.Name, name, prev.iface.Name, sig.iface.Name)
					d.Notes = []string{"`" + sigHover(prev) + "` and `" + sigHover(sig) + "`"}
				}
				continue
			}
			i.Methods[name] = sig
		}
	}
	for _, m := range i.Decl.Methods {
		name := m.Name.Name
		sig := c.signature(m, nil)
		sig.iface = i
		if name == "init" {
			d := c.errorf(m.Name.Sp, "an interface can't have an `init`")
			d.Notes = []string{"it is never created; the classes that use it are"}
			continue
		}
		if prev, ok := i.Methods[name]; ok {
			if prev.iface == i {
				c.errorf(m.Name.Sp, "`%s` already has a method `%s`", i.Name, name).Notes = []string{
					fmt.Sprintf("the first one is at line %d", prev.line),
				}
				continue
			}
			if why := sigDiff(sig, prev); why != "" {
				c.errorf(m.Name.Sp, "`%s` must %s, like `%s.%s`", name, why, prev.iface.Name, name)
				continue
			}
		}
		sig.sym = c.newSym(m.Name, SymMethod, false)
		i.Methods[name] = sig
	}
	i.state = 2
}

// implements records the interfaces a class is, and checks that it has
// their methods.
func (c *checker) implements(cls *Class) {
	if cls.impl != nil {
		return
	}
	cls.impl = map[*Interface]bool{}
	if cls.Super != nil {
		c.implements(cls.Super)
		for k := range cls.Super.impl {
			cls.impl[k] = true
		}
	}
	for idx, i := range cls.Ifaces {
		for k := range i.all {
			cls.impl[k] = true
		}
		at := cls.Decl.Uses[cls.ifaceAt[idx]]
		for _, name := range sortedKeys(i.Methods) {
			want := i.Methods[name]
			sig := cls.Methods[name]
			switch {
			case sig == nil || sig.static:
				d := c.errorf(at.Sp, "`%s` uses `%s`, but has no method `%s`", cls.Name, i.Name, name)
				d.Help = []string{fmt.Sprintf("add `%s { ... }`", sigHover(want))}
			case sigVis(sig) != ast.Public:
				c.errorf(c.methodPos(cls, sig, at), "`%s` is %s, but `%s` needs it public", name, sigVis(sig), i.Name)
			default:
				if why := sigDiff(sig, want); why != "" {
					d := c.errorf(c.methodPos(cls, sig, at), "`%s` must %s, like `%s.%s`", name, why, want.iface.Name, name)
					d.Notes = []string{fmt.Sprintf("`%s.%s` is `%s`", want.iface.Name, name, sigHover(want))}
				}
			}
		}
	}
}

// methodPos is where to report a problem with a method of cls: at its
// name if cls declares it, otherwise at the interface after `use`.
func (c *checker) methodPos(cls *Class, sig *funcSig, use *ast.Ident) diag.Span {
	if sig.class == cls && sig.decl != nil {
		return sig.decl.Name.Sp
	}
	return use.Sp
}

// ifaceMethod checks a call to a method of an interface value.
func (c *checker) ifaceMethod(e *ast.Call, m *ast.Member, i *Interface) Type {
	name := m.Name.Name
	sig := i.Methods[name]
	if sig == nil {
		c.args(e)
		if !i.Decl.Bad {
			c.errorf(m.Name.Sp, "`%s` has no method `%s`", i.Name, name)
		}
		return Invalid
	}
	c.ref(m.Name, sig.sym)
	if sig.mut {
		c.changes(m.X, fmt.Sprintf("`%s` is a `mut func`", name))
	}
	c.checkArgs(e, name, sig, c.argsFor(e, sig.params))
	return sig.result
}
