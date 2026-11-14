package parser

import (
	"sync"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/diag"
)

// preludeSource declares what is built into every program.
const preludeSource = `class Error
{
    mut message = ""

    func init(message: String)
    {
        this.message = message
    }
}
`

var prelude = sync.OnceValue(func() []ast.Decl {
	f, diags := Parse(diag.NewSource("<builtin>", []byte(preludeSource)))
	if len(diags) > 0 {
		panic("parser: the prelude has an error: " + diags[0].Message)
	}
	for _, d := range f.Decls {
		if c, ok := d.(*ast.ClassDecl); ok {
			c.Builtin = true
		}
	}
	return f.Decls
})

// Prelude returns the built-in declarations, such as the Error class. The
// checker and the interpreter add them to every program.
func Prelude() []ast.Decl { return prelude() }
