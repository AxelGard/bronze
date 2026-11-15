package check

import (
	"fmt"
	"strings"

	"github.com/AxelGard/bronze/internal/ast"
)

// Data classes: `data Money(amount: Int, currency: String)`, or the long
// form `data Wallet { mut amount: Int ... }`. They are classes with a
// constructor made from their fields (the ones without an initial value,
// in order), `==` by value, and printing. They can use interfaces and
// Error, but not classes, and nothing can use them.

// dataConstructor makes the constructor of a data class, as an init that
// takes the fields without an initial value.
func (c *checker) dataConstructor(cls *Class) {
	d := cls.Decl
	for _, m := range d.Methods {
		if m.Name.Name == "init" {
			e := c.errorf(m.Name.Sp, "a data class gets its constructor from its fields, so it can't have an `init`")
			e.Notes = []string{fmt.Sprintf("`%s(...)` takes %s", cls.Name, dataParamsText(d))}
		}
	}
	fn := &ast.FuncDecl{Name: &ast.Ident{Name: "init", Sp: d.Name.Sp}, Body: &ast.Block{}, Sp: d.Name.Sp}
	for _, f := range d.Fields {
		if f.Value == nil && f.Type != nil {
			fn.Params = append(fn.Params, &ast.Param{Name: f.Name, Type: f.Type, Sp: f.Sp})
		}
	}
	sig := c.signature(fn, cls)
	cls.Inits = map[int]*funcSig{len(fn.Params): sig}
}

// dataParamsText is "(amount: Int, currency: String)", the fields a data
// class's constructor takes.
func dataParamsText(d *ast.ClassDecl) string {
	var ps []string
	for _, f := range d.Fields {
		if f.Value == nil && f.Type != nil {
			ps = append(ps, f.Name.Name+": "+ast.TypeString(f.Type))
		}
	}
	return "(" + strings.Join(ps, ", ") + ")"
}

// ownFields returns the fields a data class declares, in order: the ones
// that `==`, printing and map keys look at.
func ownFields(cls *Class) []*Field {
	var fs []*Field
	for _, f := range cls.Decl.Fields {
		if fd := cls.Fields[f.Name.Name]; fd != nil && fd.Decl == f {
			fs = append(fs, fd)
		}
	}
	return fs
}

// dataInfo works out whether a data class can change, can be printed, and
// can be a map key, from the types of its fields.
func (c *checker) dataInfo(cls *Class) {
	if cls.dataState != 0 {
		return
	}
	cls.dataState = 1 // a data class that holds itself is changeable through a T?
	frozen, printable, keyable := true, true, true
	for _, f := range ownFields(cls) {
		t := c.fieldType(f)
		if k, ok := nonNull(t).(*Class); ok && k.Data {
			c.dataInfo(k)
		}
		if f.Mut || changeable(t) {
			frozen = false
		}
		if !printableType(t) {
			printable = false
		}
		if f.Mut || !validKey(t) {
			keyable = false
		}
	}
	cls.frozen, cls.printable, cls.keyable = frozen, printable, keyable
	cls.dataState = 2
}

// checkDataKeys reports the map key types that are data classes that
// can't be keys. They are checked after the classes are collected, since
// the types of their fields are needed.
func (c *checker) checkDataKeys() {
	for _, k := range c.dataKeys {
		if !k.cls.keyable {
			c.dataKeyError(k.at, k.cls)
		}
	}
}

func (c *checker) dataKeyError(at ast.Node, cls *Class) {
	d := c.errorf(at.Span(), "`%s` can't be a map key, since its fields aren't all `con` and keys themselves", cls.Name)
	d.Notes = []string{"a key can be an Int, String, Bool, enum, or a data class with only con fields of those; changing a key would lose its entry"}
}

type dataKey struct {
	at  ast.Node
	cls *Class
}
