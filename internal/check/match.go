package check

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/AxelGard/bronze/internal/ast"
	"github.com/AxelGard/bronze/internal/diag"
)

// match checks a match expression: the patterns fit the matched value,
// no pattern is repeated or unreachable, every case is covered, and every
// arm has the same type, which is the type of the match.
func (c *checker) match(m *ast.Match) Type { return c.checkMatch(m, false) }

// matchStmt checks a match used as a statement. Its arms are calls or
// blocks, and their values are not used.
func (c *checker) matchStmt(m *ast.Match) { c.checkMatch(m, true) }

func (c *checker) checkMatch(m *ast.Match, stmt bool) Type {
	t := c.value(m.X)
	if c.mayBeNull(m.X, t) {
		t = Invalid
	}
	en, _ := t.(*Enum)
	known := en != nil || t == Int || t == String || t == Bool
	if !known && !silent(t) {
		d := c.errorf(m.X.Span(), "cannot match on %s", t)
		d.Notes = []string{"match works on enums, Int, String and Bool"}
	}

	seen := map[string]*ast.Pattern{}
	var wildcard *ast.Pattern
	var result Type
	allNever := true
	for _, arm := range m.Arms {
		p := arm.Pattern
		switch {
		case wildcard != nil:
			d := c.errorf(p.Sp, "this arm is never used")
			d.Notes = []string{fmt.Sprintf("the `_` arm at line %d already matches everything", wildcard.Sp.Start.Line)}
		case p.Kind == ast.PatAny:
			wildcard = p
		case known:
			if key, ok := c.pattern(p, t, en); ok {
				if prev := seen[key]; prev != nil {
					d := c.errorf(p.Sp, "`%s` is already matched", c.text(p))
					d.Notes = []string{fmt.Sprintf("by the arm at line %d", prev.Sp.Start.Line)}
				} else {
					seen[key] = p
				}
			}
		}

		if stmt {
			c.stmtArm(arm)
			continue
		}
		if arm.Body != nil {
			d := c.errorf(arm.Body.Sp, "a `{ }` arm has no value")
			d.Notes = []string{"block arms can only be used when the match is a statement, not when its value is used"}
			c.block(arm.Body)
			allNever = false
			continue
		}
		vt := c.value(arm.Value)
		if vt != Never {
			allNever = false
		}
		switch {
		case silent(vt):
		case result == nil:
			result = vt
		case !assignable(vt, result):
			d := c.errorf(arm.Value.Span(), "this arm is %s, but the arms before it are %s", vt, result)
			d.Notes = []string{"every arm of a match must have the same type"}
		}
	}

	if len(m.Arms) == 0 {
		if !m.Bad {
			c.errorf(m.Sp, "match has no arms")
		}
		return Invalid
	}
	if known && wildcard == nil && !m.Bad {
		c.exhaustive(m, t, en, seen)
	}
	if stmt {
		return NoValue
	}
	if result == nil {
		if allNever {
			return Never
		}
		return Invalid
	}
	return result
}

// stmtArm checks an arm of a match statement: a block, a call or another
// match statement.
func (c *checker) stmtArm(arm *ast.Arm) {
	switch v := arm.Value.(type) {
	case nil:
		c.block(arm.Body)
	case *ast.Call:
		c.expr(v)
	case *ast.Match:
		c.matchStmt(v)
	default:
		c.expr(v)
		d := c.errorf(v.Span(), "the value of this arm is not used")
		d.Notes = []string{"in a match used as a statement, each arm must be a call or a `{ }` block"}
	}
}

// pattern checks that a pattern fits the matched type t, and returns a key
// that is the same for equal patterns.
func (c *checker) pattern(p *ast.Pattern, t Type, en *Enum) (string, bool) {
	var pt Type
	var key string
	switch p.Kind {
	case ast.PatInt:
		pt, key = Int, "i:"+strconv.FormatInt(p.Int, 10)
	case ast.PatString:
		pt, key = String, "s:"+p.Str
	case ast.PatBool:
		pt, key = Bool, "b:"+strconv.FormatBool(p.Bool)
	case ast.PatCase:
		name := p.Enum.Name
		pe := c.enums[name]
		switch {
		case pe != nil:
		case c.broken[name]:
			return "", false
		case c.classes[name] != nil || builtinTypes[name] != nil:
			c.errorf(p.Enum.Sp, "`%s` is not an enum", name)
			return "", false
		default:
			c.errorf(p.Enum.Sp, "unknown enum `%s`", name)
			return "", false
		}
		pt, key = pe, "c:"+p.Case.Name
		if pe == en {
			if _, ok := en.Cases[p.Case.Name]; !ok {
				if !en.Decl.Bad {
					c.errorf(p.Case.Sp, "`%s` has no case `%s`", en.Name, p.Case.Name)
				}
				return "", false
			}
		}
	}
	if pt != t {
		d := c.errorf(p.Sp, "the pattern `%s` is %s, but the matched value is %s", c.text(p), pt, t)
		d.Notes = []string{"every pattern must have the type of the value being matched"}
		return "", false
	}
	return key, true
}

// exhaustive reports the values a match without `_` doesn't cover.
func (c *checker) exhaustive(m *ast.Match, t Type, en *Enum, seen map[string]*ast.Pattern) {
	// Point at `match x`, not the whole match.
	at := diag.Span{Start: m.Sp.Start, End: m.X.Span().End}
	var missing []string
	switch {
	case en != nil:
		if en.Decl.Bad {
			return
		}
		for _, cs := range en.Decl.Cases {
			if seen["c:"+cs.Name] == nil {
				missing = append(missing, "`"+en.Name+"."+cs.Name+"`")
			}
		}
	case t == Bool:
		for _, b := range []string{"true", "false"} {
			if seen["b:"+b] == nil {
				missing = append(missing, "`"+b+"`")
			}
		}
	default:
		d := c.errorf(at, "match on %s needs a `_` arm", t)
		what := "an Int can be any number"
		if t == String {
			what = "a String can be any text"
		}
		d.Help = []string{fmt.Sprintf("%s, so add `_ => ...` for the other values", what)}
		return
	}
	if len(missing) == 0 {
		return
	}
	list := missing[0]
	if n := len(missing); n > 1 {
		list = strings.Join(missing[:n-1], ", ") + " and " + missing[n-1]
	}
	d := c.errorf(at, "match does not cover %s", list)
	d.Help = []string{"add an arm for each, or `_ => ...` for all other values"}
}
