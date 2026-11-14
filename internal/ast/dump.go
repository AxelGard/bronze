package ast

import (
	"fmt"
	"strconv"
	"strings"
)

// Dump writes a node as a compact S-expression, for tests and debugging:
//
//	(func add (a Int) (b Int) -> Int (block (return (+ a b))))
func Dump(n Node) string {
	var b strings.Builder
	dump(&b, n)
	return b.String()
}

func dump(b *strings.Builder, n Node) {
	switch n := n.(type) {
	case *File:
		for i, d := range n.Decls {
			if i > 0 {
				b.WriteByte('\n')
			}
			dump(b, d)
		}
	case *FuncDecl:
		fmt.Fprintf(b, "(func %s", n.Name.Name)
		for _, p := range n.Params {
			fmt.Fprintf(b, " (%s %s)", p.Name.Name, typeString(p.Type))
		}
		if n.Result != nil {
			fmt.Fprintf(b, " -> %s", typeString(n.Result))
		}
		b.WriteByte(' ')
		dump(b, n.Body)
		b.WriteByte(')')
	case *BadDecl:
		b.WriteString("(bad-decl)")
	case *Block:
		b.WriteString("(block")
		for _, s := range n.Stmts {
			b.WriteByte(' ')
			dump(b, s)
		}
		b.WriteByte(')')
	case *VarDecl:
		kw := "con"
		if n.Mut {
			kw = "mut"
		}
		fmt.Fprintf(b, "(%s %s", kw, n.Name.Name)
		if n.Type != nil {
			fmt.Fprintf(b, ": %s", typeString(n.Type))
		}
		b.WriteByte(' ')
		dump(b, n.Value)
		b.WriteByte(')')
	case *Assign:
		b.WriteString("(= ")
		dump(b, n.Target)
		b.WriteByte(' ')
		dump(b, n.Value)
		b.WriteByte(')')
	case *If:
		b.WriteString("(if ")
		dump(b, n.Cond)
		b.WriteByte(' ')
		dump(b, n.Then)
		if n.Else != nil {
			b.WriteString(" else ")
			dump(b, n.Else)
		}
		b.WriteByte(')')
	case *While:
		b.WriteString("(while ")
		dump(b, n.Cond)
		b.WriteByte(' ')
		dump(b, n.Body)
		b.WriteByte(')')
	case *Return:
		b.WriteString("(return")
		if n.Value != nil {
			b.WriteByte(' ')
			dump(b, n.Value)
		}
		b.WriteByte(')')
	case *ExprStmt:
		dump(b, n.X)
	case *BadStmt:
		b.WriteString("(bad-stmt)")
	case *Ident:
		b.WriteString(n.Name)
	case *IntLit:
		b.WriteString(strconv.FormatInt(n.Value, 10))
	case *FloatLit:
		b.WriteString(strconv.FormatFloat(n.Value, 'g', -1, 64) + "f")
	case *StringLit:
		b.WriteString(strconv.Quote(n.Value))
	case *BoolLit:
		b.WriteString(strconv.FormatBool(n.Value))
	case *Unary:
		fmt.Fprintf(b, "(%s ", strings.Trim(n.Op.String(), "`"))
		dump(b, n.X)
		b.WriteByte(')')
	case *Binary:
		fmt.Fprintf(b, "(%s ", strings.Trim(n.Op.String(), "`"))
		dump(b, n.X)
		b.WriteByte(' ')
		dump(b, n.Y)
		b.WriteByte(')')
	case *Call:
		b.WriteString("(call ")
		dump(b, n.Fn)
		for _, a := range n.Args {
			b.WriteByte(' ')
			dump(b, a)
		}
		b.WriteByte(')')
	case *Paren:
		b.WriteString("(paren ")
		dump(b, n.X)
		b.WriteByte(')')
	case *BadExpr:
		b.WriteString("(bad-expr)")
	default:
		fmt.Fprintf(b, "(unknown %T)", n)
	}
}

func typeString(t *TypeExpr) string {
	if t.Optional {
		return t.Name.Name + "?"
	}
	return t.Name.Name
}
