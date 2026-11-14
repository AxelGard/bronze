package check

import (
	"fmt"
	"strings"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/diag"
	"github.com/AxelGard/bronze/internal/lexer"
)

// value checks an expression whose value is used.
func (c *checker) value(e ast.Expr) Type {
	t := c.expr(e)
	if f, ok := t.(*Failable); ok {
		c.unhandled(e)
		if f.Value == NoValue {
			return Invalid
		}
		return f.Value
	}
	if t == NoValue {
		name := "this call"
		if call, ok := e.(*ast.Call); ok {
			switch fn := call.Fn.(type) {
			case *ast.Ident:
				name = "`" + fn.Name + "`"
			case *ast.Member:
				name = "`" + fn.Name.Name + "`"
			}
		}
		c.errorf(e.Span(), "%s returns nothing, so it can't be used as a value", name)
		return Invalid
	}
	return t
}

func (c *checker) expr(e ast.Expr) Type {
	t := c.expr1(e)
	c.info.Types[e] = t
	return t
}

func (c *checker) expr1(e ast.Expr) Type {
	switch e := e.(type) {
	case *ast.IntLit:
		return Int
	case *ast.FloatLit:
		return Float
	case *ast.StringLit:
		return String
	case *ast.BoolLit:
		return Bool
	case *ast.NullLit:
		return Null
	case *ast.Is:
		return c.isExpr(e)
	case *ast.Else:
		return c.elseExpr(e)
	case *ast.Paren:
		return c.value(e.X)
	case *ast.Ident:
		return c.ident(e)
	case *ast.This:
		return c.this(e)
	case *ast.Member:
		return c.member(e)
	case *ast.Unary:
		return c.unary(e)
	case *ast.Binary:
		return c.binary(e)
	case *ast.Range:
		return c.rangeExpr(e)
	case *ast.ListLit:
		return c.listLit(e, nil)
	case *ast.MapLit:
		return c.mapLit(e, nil)
	case *ast.Index:
		return c.index(e)
	case *ast.Call:
		return c.call(e)
	case *ast.Match:
		return c.match(e)
	case *ast.BadExpr:
		return Invalid
	}
	panic(fmt.Sprintf("check: unexpected expression %T", e))
}

func (c *checker) ident(e *ast.Ident) Type {
	if v := c.scope.lookup(e.Name); v != nil {
		c.ref(e, v.sym)
		return v.typ
	}
	if sig, ok := c.funcs[e.Name]; ok {
		c.ref(e, sig.sym)
		d := c.errorf(e.Sp, "`%s` is a function, and functions can't be used as values yet", e.Name)
		d.Help = []string{fmt.Sprintf("to call it, write `%s(...)`", e.Name)}
		return Invalid
	}
	if cls, ok := c.classes[e.Name]; ok {
		c.ref(e, cls.Sym)
		d := c.errorf(e.Sp, "`%s` is a class, not a value", e.Name)
		d.Help = []string{fmt.Sprintf("to create an object, write `%s(...)`", e.Name)}
		return Invalid
	}
	if en, ok := c.enums[e.Name]; ok {
		c.ref(e, en.Sym)
		d := c.errorf(e.Sp, "`%s` is an enum, not a value", e.Name)
		d.Help = []string{fmt.Sprintf("use one of its cases, like `%s.%s`", en.Name, en.someCase())}
		return Invalid
	}
	if builtinTypes[e.Name] != nil || genericTypes[e.Name] {
		c.errorf(e.Sp, "`%s` is a type, not a value", e.Name)
		return Invalid
	}
	if builtinFuncs[e.Name] {
		c.errorf(e.Sp, "`%s` is a built-in function, and functions can't be used as values yet", e.Name)
		return Invalid
	}
	if c.broken[e.Name] {
		return Invalid
	}
	d := c.errorf(e.Sp, "unknown name `%s`", e.Name)
	if c.thisField(e.Name) != nil {
		d.Help = []string{fmt.Sprintf("to use the field, write `this.%s`", e.Name)}
	}
	return Invalid
}

func (c *checker) this(e *ast.This) Type {
	switch {
	case c.fn == nil:
		c.errorf(e.Sp, "`this` can't be used in the initial value of a field")
		return Invalid
	case c.fn.enum != nil:
		return c.fn.enum
	case c.fn.class == nil:
		c.errorf(e.Sp, "`this` can only be used inside methods")
		return Invalid
	}
	return c.fn.class
}

// fieldOf checks x.name and returns the field it refers to and its class,
// or nil after reporting an error.
func (c *checker) fieldOf(m *ast.Member) (*Field, *Class) {
	t := c.value(m.X)
	if c.mayBeNull(m.X, t) {
		t = t.(*Optional).Inner
	}
	cls, ok := t.(*Class)
	if !ok {
		if !silent(t) {
			c.errorf(m.Name.Sp, "%s has no fields", t)
		}
		return nil, nil
	}
	if f := cls.Fields[m.Name.Name]; f != nil {
		c.ref(m.Name, f.sym)
		return f, cls
	}
	switch {
	case cls.Decl.Bad:
		// The class had a syntax error; the field may be the broken one.
	case cls.Methods[m.Name.Name] != nil:
		d := c.errorf(m.Name.Sp, "`%s` is a method, not a field", m.Name.Name)
		d.Help = []string{fmt.Sprintf("to call it, write `%s(...)`", c.text(m))}
	default:
		c.errorf(m.Name.Sp, "`%s` has no field `%s`", cls.Name, m.Name.Name)
	}
	return nil, nil
}

func (c *checker) member(m *ast.Member) Type {
	if en := c.enumName(m.X); en != nil {
		c.ref(m.X.(*ast.Ident), en.Sym)
		return c.enumCase(m, en)
	}
	f, _ := c.fieldOf(m)
	if f == nil {
		return Invalid
	}
	return c.fieldType(f)
}

// enumName returns the enum that x names, if x is the name of an enum.
func (c *checker) enumName(x ast.Expr) *Enum {
	id, ok := x.(*ast.Ident)
	if !ok || c.scope.lookup(id.Name) != nil {
		return nil
	}
	return c.enums[id.Name]
}

// enumCase checks `Enum.Case`.
func (c *checker) enumCase(m *ast.Member, en *Enum) Type {
	name := m.Name.Name
	if _, ok := en.Cases[name]; ok {
		c.ref(m.Name, en.CaseSym[name])
		return en
	}
	switch {
	case en.Decl.Bad:
	case en.Methods[name] != nil:
		d := c.errorf(m.Name.Sp, "`%s` is a method of each `%s` case, not a case", name, en.Name)
		d.Help = []string{fmt.Sprintf("call it on a case, like `%s.%s.%s()`", en.Name, en.someCase(), name)}
	default:
		c.errorf(m.Name.Sp, "`%s` has no case `%s`", en.Name, name)
	}
	return Invalid
}

func (c *checker) unary(e *ast.Unary) Type {
	t := c.value(e.X)
	if silent(t) || c.mayBeNull(e.X, t) {
		return Invalid
	}
	switch e.Op {
	case lexer.MINUS:
		if isNumeric(t) {
			return t
		}
		c.errorf(e.Sp, "cannot negate %s", t)
	case lexer.NOT:
		if t == Bool {
			return Bool
		}
		c.errorf(e.Sp, "`not` needs a Bool, found %s", t)
	}
	return Invalid
}

var arithVerbs = map[lexer.Kind]string{
	lexer.PLUS:    "add %s and %s",
	lexer.MINUS:   "subtract %[2]s from %[1]s",
	lexer.STAR:    "multiply %s by %s",
	lexer.SLASH:   "divide %s by %s",
	lexer.PERCENT: "take the remainder of %s by %s",
}

func opText(k lexer.Kind) string {
	s := k.String()
	return s[1 : len(s)-1]
}

func (c *checker) binary(e *ast.Binary) Type {
	if e.Op == lexer.AND || e.Op == lexer.OR {
		x, y := c.value(e.X), c.value(e.Y)
		for _, side := range []struct {
			t Type
			e ast.Expr
		}{{x, e.X}, {y, e.Y}} {
			if !assignable(side.t, Bool) && !c.nullMismatch(side.e, side.t, Bool) {
				c.errorf(side.e.Span(), "`%s` needs Bool on both sides, found %s", opText(e.Op), side.t)
			}
		}
		return Bool
	}

	var x, y Type
	switch {
	case isLiteral(e.Y):
		x = c.value(e.X)
		y = c.valueFor(e.Y, x)
	case isLiteral(e.X):
		y = c.value(e.Y)
		x = c.valueFor(e.X, y)
	default:
		x, y = c.value(e.X), c.value(e.Y)
	}
	if silent(x) || silent(y) {
		if isCmp(e.Op) {
			return Bool
		}
		return Invalid
	}
	label := fmt.Sprintf("%s %s %s", x, opText(e.Op), y)

	switch e.Op {
	case lexer.EQ, lexer.NE:
		// A T? can be compared with a T, and with null.
		if identical(nonNull(x), nonNull(y)) {
			return Bool
		}
		_, xOpt := x.(*Optional)
		_, yOpt := y.(*Optional)
		switch {
		case x == Null && yOpt, y == Null && xOpt:
			return Bool
		case x == Null || y == Null:
			other, t := e.X, x
			if x == Null {
				other, t = e.Y, y
			}
			d := c.errorf(e.Span(), "`%s` can never be null", c.text(other))
			d.Notes = []string{fmt.Sprintf("it is %s; only `T?` values can be null", t)}
			return Bool
		}
		d := c.errorf(e.Span(), "cannot compare %s and %s with `%s`", x, y, opText(e.Op))
		d.Label = label
		c.mixedHelp(d, e, e.X, e.Y, x, y, false)
		return Bool

	case lexer.LT, lexer.LE, lexer.GT, lexer.GE:
		if c.eitherMayBeNull(e, x, y) {
			return Bool
		}
		if x == y && (isNumeric(x) || x == String) {
			return Bool
		}
		var d *diag.Diagnostic
		if identical(x, y) {
			d = c.errorf(e.Span(), "cannot compare %s values with `%s`", x, opText(e.Op))
		} else {
			d = c.errorf(e.Span(), "cannot compare %s and %s with `%s`", x, y, opText(e.Op))
		}
		d.Label = label
		c.mixedHelp(d, e, e.X, e.Y, x, y, false)
		return Bool
	}

	if c.eitherMayBeNull(e, x, y) {
		return Invalid
	}
	return c.arith(e.Op, e, e.X, e.Y, x, y, label, false)
}

// rangeExpr checks `lo..hi` or `lo..=hi`, which is a List<Int> outside a
// `for` and `[ ]`.
func (c *checker) rangeExpr(e *ast.Range) Type {
	for _, side := range []ast.Expr{e.Lo, e.Hi} {
		t := c.value(side)
		if !assignable(t, Int) && !c.nullMismatch(side, t, Int) {
			d := c.errorf(side.Span(), "a range needs Int bounds, found %s", t)
			d.Notes = []string{"ranges count whole numbers, from the first bound up"}
		}
	}
	return &List{Elem: Int}
}

// eitherMayBeNull reports "may be null" for each side of e that is a T?,
// and says whether it did.
func (c *checker) eitherMayBeNull(e *ast.Binary, x, y Type) bool {
	nx := c.mayBeNull(e.X, x)
	ny := c.mayBeNull(e.Y, y)
	return nx || ny
}

// nonNull returns T for a T?, and t itself otherwise.
func nonNull(t Type) Type {
	if o, ok := t.(*Optional); ok {
		return o.Inner
	}
	return t
}

// arith checks `x op y` for an arithmetic operator, where e is the whole
// expression and ex and ey its sides. In a compound assignment (`ex op=
// ey`), ex is the target.
func (c *checker) arith(op lexer.Kind, e, ex, ey ast.Node, x, y Type, label string, compound bool) Type {
	switch {
	case x == y && x == Int:
		return Int
	case x == y && x == Float && op != lexer.PERCENT:
		return Float
	case x == y && x == String && op == lexer.PLUS:
		return String
	}
	var d *diag.Diagnostic
	if x == Float && y == Float {
		d = c.errorf(e.Span(), "cannot use `%%` on Float")
	} else {
		d = c.errorf(e.Span(), "cannot "+arithVerbs[op], x, y)
	}
	d.Label = label
	c.mixedHelp(d, e, ex, ey, x, y, compound)
	if op == lexer.PLUS && (x == String) != (y == String) {
		other := ex
		if x == String {
			other = ey
		}
		if t := c.info.Types[other.(ast.Expr)]; (t == Int || t == Float || t == Bool) && !(compound && other == ex) {
			d.Help = append(d.Help, "convert explicitly: "+c.replaced(e, other, "String"))
		}
	}
	return Invalid
}

// mixedHelp adds "convert explicitly: Float(a) / 2.0" when one side of e is
// an Int and the other a Float. The target of a compound assignment can't
// be converted.
func (c *checker) mixedHelp(d *diag.Diagnostic, e, ex, ey ast.Node, x, y Type, compound bool) {
	if !(isNumeric(x) && isNumeric(y) && x != y) {
		return
	}
	intSide := ex
	if y == Int {
		intSide = ey
	}
	if compound && intSide == ex {
		return
	}
	d.Help = append(d.Help, "convert explicitly: "+c.replaced(e, intSide, "Float"))
}

// replaced returns the source text of e with part wrapped in a call to fn.
func (c *checker) replaced(e, part ast.Node, fn string) string {
	text := c.src.Text
	s, p := e.Span(), part.Span()
	return string(text[s.Start.Offset:p.Start.Offset]) + fn + "(" + c.text(part) + ")" +
		string(text[p.End.Offset:s.End.Offset])
}

func isCmp(k lexer.Kind) bool {
	switch k {
	case lexer.EQ, lexer.NE, lexer.LT, lexer.LE, lexer.GT, lexer.GE:
		return true
	}
	return false
}

// ---------------------------------------------------------------------
// Calls

func (c *checker) call(e *ast.Call) Type {
	switch fn := e.Fn.(type) {
	case *ast.Ident:
		return c.callName(e, fn)
	case *ast.Member:
		return c.callMethod(e, fn)
	}
	c.value(e.Fn)
	c.args(e)
	c.errorf(e.Fn.Span(), "this expression can't be called")
	return Invalid
}

func (c *checker) callName(e *ast.Call, id *ast.Ident) Type {
	if v := c.scope.lookup(id.Name); v != nil {
		c.ref(id, v.sym)
		c.args(e)
		c.errorf(id.Sp, "`%s` is a variable, not a function", id.Name)
		return Invalid
	}
	if sig, ok := c.funcs[id.Name]; ok {
		c.ref(id, sig.sym)
		c.checkArgs(e, id.Name, sig, c.argsFor(e, sig.params))
		return sig.result
	}
	if cls, ok := c.classes[id.Name]; ok {
		c.ref(id, cls.Sym)
		return c.construct(e, cls)
	}
	if en, ok := c.enums[id.Name]; ok {
		c.ref(id, en.Sym)
		c.args(e)
		d := c.errorf(id.Sp, "`%s` is an enum, and can't be called", id.Name)
		d.Help = []string{fmt.Sprintf("use one of its cases, like `%s.%s`", en.Name, en.someCase())}
		return Invalid
	}
	if builtinFuncs[id.Name] {
		return c.builtin(e, id.Name)
	}
	c.args(e)
	if c.broken[id.Name] {
		return Invalid
	}
	d := c.errorf(id.Sp, "unknown function `%s`", id.Name)
	if c.fn != nil && (c.fn.class != nil && c.fn.class.Methods[id.Name] != nil ||
		c.fn.enum != nil && c.fn.enum.Methods[id.Name] != nil) {
		d.Help = []string{fmt.Sprintf("to call the method, write `this.%s(...)`", id.Name)}
	}
	return Invalid
}

// construct checks `Counter(args)`: the init is chosen by the number of
// arguments.
func (c *checker) construct(e *ast.Call, cls *Class) Type {
	sig, ok := cls.Inits[len(e.Args)]
	if !ok {
		types := c.args(e)
		arities := cls.initArities()
		if len(arities) == 1 {
			c.arityError(e, cls.Name, arities[0], cls.Inits[arities[0]])
			return cls
		}
		var ns []string
		for _, n := range arities {
			ns = append(ns, fmt.Sprint(n))
		}
		d := c.errorf(e.Sp, "`%s` has no `init` that takes %s", cls.Name, plural(len(types), "argument"))
		d.Notes = []string{fmt.Sprintf("its `init`s take %s arguments", strings.Join(ns, " or "))}
		return cls
	}
	c.checkArgs(e, cls.Name, sig, c.argsFor(e, sig.params))
	return cls
}

func (c *checker) callMethod(e *ast.Call, m *ast.Member) Type {
	t := c.value(m.X)
	name := m.Name.Name
	if c.mayBeNull(m.X, t) {
		t = t.(*Optional).Inner
	}
	switch t := t.(type) {
	case *List:
		return c.listMethod(e, m, t)
	case *Map:
		return c.mapMethod(e, m, t)
	}
	if en, ok := t.(*Enum); ok {
		sig := en.Methods[name]
		if sig == nil {
			c.args(e)
			if !en.Decl.Bad {
				c.errorf(m.Name.Sp, "`%s` has no method `%s`", en.Name, name)
			}
			return Invalid
		}
		c.ref(m.Name, sig.sym)
		c.checkArgs(e, name, sig, c.argsFor(e, sig.params))
		return sig.result
	}
	cls, ok := t.(*Class)
	if !ok {
		c.args(e)
		if !silent(t) {
			c.errorf(m.Name.Sp, "%s has no methods", t)
		}
		return Invalid
	}
	sig := cls.Methods[name]
	if sig == nil {
		c.args(e)
		switch {
		case name == "init":
			d := c.errorf(m.Name.Sp, "`init` can't be called directly")
			d.Help = []string{fmt.Sprintf("to create an object, write `%s(...)`", cls.Name)}
		case cls.Fields[name] != nil:
			c.errorf(m.Name.Sp, "`%s` is a field, not a method", name)
		case cls.Decl.Bad:
		default:
			c.errorf(m.Name.Sp, "`%s` has no method `%s`", cls.Name, name)
		}
		return Invalid
	}
	c.ref(m.Name, sig.sym)
	c.checkArgs(e, name, sig, c.argsFor(e, sig.params))
	return sig.result
}

func (c *checker) args(e *ast.Call) []Type {
	var types []Type
	for _, a := range e.Args {
		types = append(types, c.value(a))
	}
	return types
}

// checkArgs checks the arguments of a call to a function, method or init.
func (c *checker) checkArgs(e *ast.Call, name string, sig *funcSig, types []Type) {
	if len(types) != len(sig.params) {
		c.arityError(e, name, len(sig.params), sig)
		return
	}
	for i, t := range types {
		if !assignable(t, sig.params[i]) && !c.nullMismatch(e.Args[i], t, sig.params[i]) {
			p := sig.decl.Params[i]
			c.errorf(e.Args[i].Span(), "argument `%s` of `%s` must be %s, found %s", p.Name.Name, name, sig.params[i], t).Notes = []string{
				sig.declNote(),
			}
		}
	}
}

// arityError reports a call with the wrong number of arguments. sig is nil
// for built-ins.
func (c *checker) arityError(e *ast.Call, name string, want int, sig *funcSig) {
	verb := "were"
	if len(e.Args) == 1 {
		verb = "was"
	}
	d := c.errorf(e.Sp, "`%s` takes %s, but %d %s given", name, plural(want, "argument"), len(e.Args), verb)
	if sig != nil {
		d.Notes = []string{sig.declNote()}
	}
}

// declNote says where a function, method or init is declared.
func (sig *funcSig) declNote() string {
	switch {
	case sig.class != nil && sig.class.Decl.Builtin:
		return fmt.Sprintf("`%s` is built in", sig.class.Name)
	case sig.isInit && sig.decl == nil:
		return fmt.Sprintf("`%s` has no `init`, so it is created with `%s()`", sig.class.Name, sig.class.Name)
	case sig.isInit:
		return fmt.Sprintf("its `init` is declared at line %d", sig.line)
	case sig.class != nil:
		return fmt.Sprintf("`%s.%s` is declared at line %d", sig.class.Name, sig.name, sig.line)
	case sig.enum != nil:
		return fmt.Sprintf("`%s.%s` is declared at line %d", sig.enum.Name, sig.name, sig.line)
	}
	return fmt.Sprintf("`%s` is declared at line %d", sig.name, sig.line)
}

// builtin checks a call to a built-in function.
func (c *checker) builtin(e *ast.Call, name string) Type {
	types := c.args(e)

	// expect checks the argument count and types.
	expect := func(want ...Type) {
		if len(types) != len(want) {
			c.arityError(e, name, len(want), nil)
			return
		}
		for i, t := range types {
			if !assignable(t, want[i]) && !c.mayBeNull(e.Args[i], t) {
				c.errorf(e.Args[i].Span(), "`%s` needs %s here, found %s", name, want[i], t)
			}
		}
	}

	switch name {
	case "print":
		if len(types) != 1 {
			c.arityError(e, name, 1, nil)
			return NoValue
		}
		if c.mayBeNull(e.Args[0], types[0]) {
			return NoValue
		}
		switch t := types[0]; t.(type) {
		case *Class:
			d := c.errorf(e.Args[0].Span(), "cannot print an object of class %s", t)
			d.Help = []string{"print its fields instead"}
		default:
			if !silent(t) && !printable(t) {
				c.errorf(e.Args[0].Span(), "cannot print a value of type %s", t)
			}
		}
		return NoValue
	case "Float":
		if len(types) == 1 && types[0] == Float {
			c.errorf(e.Args[0].Span(), "this is already a Float")
			return Float
		}
		expect(Int)
		return Float
	case "String":
		if len(types) != 1 {
			c.arityError(e, name, 1, nil)
			return String
		}
		switch t := types[0]; {
		case c.mayBeNull(e.Args[0], t):
		case t == String:
			c.errorf(e.Args[0].Span(), "this is already a String")
		case !printable(t) && !silent(t):
			c.errorf(e.Args[0].Span(), "cannot convert %s to String", t)
		}
		return String
	case "Int":
		d := c.errorf(e.Fn.Span(), "there is no `Int(...)` conversion yet")
		d.Notes = []string{"the rounding rules for Float to Int are not decided"}
		return Invalid
	case "Bool":
		c.errorf(e.Fn.Span(), "there is no `Bool(...)` conversion")
		return Invalid
	case "panic":
		expect(String)
		return Never
	case "assert":
		expect(Bool, String)
		return NoValue
	}
	panic("check: unknown built-in " + name)
}
