package check

import (
	"fmt"

	"github.com/AxelGard/bronze/internal/ast"
)

// Visibility: fields and methods are public by default. A `private` one can
// be used only in the methods and inits of the class that declares it, on
// any object of that class. A `protected` one also in the class's
// children.

// canUse reports whether the function being checked may use a member with
// visibility vis, declared in class cls or enum en.
func (c *checker) canUse(vis ast.Visibility, cls *Class, en *Enum) bool {
	if vis == ast.Public {
		return true
	}
	if c.fn == nil {
		return false
	}
	if en != nil {
		return c.fn.enum == en
	}
	if c.fn.class == nil {
		return false
	}
	if vis == ast.Private {
		return c.fn.class == cls
	}
	return assignable(c.fn.class, cls)
}

// access reports an error if the member called name, with visibility vis
// in class cls or enum en, can't be used here.
func (c *checker) access(at *ast.Ident, vis ast.Visibility, cls *Class, en *Enum, line int) {
	if c.canUse(vis, cls, en) {
		return
	}
	owner := ""
	if cls != nil {
		owner = cls.Name
	} else {
		owner = en.Name
	}
	if vis == ast.Private {
		e := c.errorf(at.Sp, "`%s` is private to `%s`", at.Name, owner)
		e.Notes = []string{fmt.Sprintf("it is declared at line %d; only `%s`'s own methods can use it", line, owner)}
		return
	}
	e := c.errorf(at.Sp, "`%s` is protected: only `%s` and its children can use it", at.Name, owner)
	e.Notes = []string{fmt.Sprintf("it is declared at line %d", line)}
}

// sigVis is the visibility of a function, method or init.
func sigVis(sig *funcSig) ast.Visibility {
	if sig.decl == nil {
		return ast.Public
	}
	return sig.decl.Vis
}

// accessSig checks that the method or init sig can be called here.
func (c *checker) accessSig(at *ast.Ident, sig *funcSig) {
	c.access(at, sigVis(sig), sig.class, sig.enum, sig.line)
}

// visPrefix is "private " or "protected ", for hover text.
func visPrefix(vis ast.Visibility) string {
	if vis == ast.Public {
		return ""
	}
	return vis.String() + " "
}
