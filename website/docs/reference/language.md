# Language reference

Bronze 0.2.0 in one place. The [guide](../guide/first-program.md) explains the same things with examples.

## Source files

A program is one UTF-8 file ending in `.brz`. Its top level holds `func`, `class` and `enum` declarations, in any order. It must have `func main() -> Int` with no parameters; the value `main` returns is the exit code.

## Tokens

| Kind | Examples |
|---|---|
| Names | `add`, `total`, `is_even`, `_tmp`: a letter or `_`, then letters, digits and `_`. A lone `_` is only a `match` pattern |
| `Int` literals | `0`, `42`, `1_000_000` (decimal only) |
| `Float` literals | `2.0`, `3.14159`, `1.5e-3`: digits on both sides of the `.` |
| `String` literals | `"hello"`, with the escapes `\n` `\t` `\"` `\\` `\{` |
| Operators | `+ - * / %` `== != < <= > >=` `..` `..=` `=` `+= -= *= /= %=` `->` `=>` `!` `?` `.` |
| Punctuation | `( ) { } [ ] , :` |
| Comments | `// to the end of the line`, `/* a block */` (block comments don't nest) |

A `{` in a string without a `\` starts an interpolation: `"total: {a + b}"`. It ends at the matching `}` on the same line, and the value inside is converted with `String(...)`. A `}` outside an interpolation is a literal `}`.

### Keywords

```text
func  return  con  mut  if  else  while  for  in  break  continue
and  or  not  true  false  class  enum  use  this  null  match  is
static  override  super
```

These are reserved for later versions, and can't be used as names:

```text
data  interface  namespace  import  private  protected  test  type  copy
```

## Newlines

A newline ends a statement. It doesn't when:

- it is inside `( )` or `[ ]`
- the line ends in a binary operator (`+`, `and`, `==`, ...), `,`, `->`, `=>`, `=`, `+=` (and the other `op=`), `.` or `!`

A `{` may be on the next line, and so may the `else` of an `if`. The `else` that handles an error or a null must be on the same line as the expression it handles.

## Types

| Type | Values |
|---|---|
| `Int` | 64-bit signed integer |
| `Float` | 64-bit IEEE 754 floating point |
| `Bool` | `true`, `false` |
| `String` | UTF-8 text |
| a class `C` | a reference to an object of `C` |
| an enum `E` | one of its cases |
| `List<T>` | a shared reference to a list of `T`s; `T` can't be a `T?` |
| `Map<K, V>` | a shared reference to a map from `K` to `V`, in insertion order; `K` is `Int`, `String`, `Bool` or an enum |
| `Error` | the built-in error class |
| `T?` | a `T`, or `null`, for any type `T` (but not `T??`) |
| `T ! E` | return types only: a `T`, or a failure with `E` |
| `! E` | return types only: nothing, or a failure with `E` |

## Operators

From the lowest precedence to the highest:

| Operators | Operands | Result |
|---|---|---|
| `x else handler` | `x` is a `T ! E` or a `T?` | `T` (see [Errors](../guide/errors.md), [Null](../guide/null.md)) |
| `or` | `Bool` | `Bool`, short-circuit |
| `and` | `Bool` | `Bool`, short-circuit |
| `== !=` | two values of the same type, a `T?` and a `T`, or a `T?` and `null`; lists and maps by value | `Bool` |
| `< <= > >=` | two `Int`s, two `Float`s or two `String`s | `Bool` |
| `x is C` | `x` is an object (or a `T?` of one) and `C` a class it could be | `Bool`; inside `if x is C`, a `con` or parameter `x` is a `C` |
| `a..b`, `a..=b` | two `Int`s | the `Int`s from `a` up to `b`, without or with `b`; a `List<Int>` outside a `for` |
| `+ -` | two `Int`s or two `Float`s; `+` also joins two `String`s | the same type |
| `* / %` | two `Int`s or two `Float`s (`%` only `Int`) | the same type |
| `not x`, `-x` | `Bool`; `Int` or `Float` | the same type |
| `f(...)`, `x.name`, `xs[i]`, `xs[a..b]` | `xs[i]`: a `List<T>` or `String` and an `Int`, or a `Map<K, V>` and a `K`. `xs[a..b]`: a `List<T>` or `String` and `Int`s | `xs[i]` is a `T?`, `String?` or `V?`; a slice is a new `List<T>` or a `String` |

Comparisons and ranges can't be chained (`a < b < c` and `a..b..c` are errors). `Int` and `Float` never mix: convert with `Float(...)`. `==` on objects compares identity; on enums, the case.

## Built-ins

| Name | Takes | Does |
|---|---|---|
| `print(x)` | one `Int`, `Float`, `Bool`, `String` or enum value, or a list or map of them | prints it and a newline |
| `String(x)` | the same types | the text `print` would show |
| `Float(n)` | an `Int` | the same number as a `Float` |
| `Error(message)` | a `String` | a new `Error` |
| `panic(message)` | a `String` | stops the program; usable wherever a value is expected |
| `assert(condition, message)` | a `Bool` and a `String` | panics with `assertion failed: message` if the condition is false |

Lists have the methods `length`, `append`, `insert`, `remove_at`, `pop`, `contains`, `index_of`, `reverse`, `sort` and `join`; see [Lists](../guide/lists.md#methods). Maps have `length`, `contains`, `remove`, `keys` and `values`; see [Maps](../guide/maps.md#methods). Strings have `length`, `contains`, `starts_with`, `ends_with`, `index_of`, `split`, `trim`, `to_upper`, `to_lower` and `replace`; see [Strings](../guide/strings.md#methods).

## Statements

| Statement | |
|---|---|
| `con x = e`, `con x: T = e` | a variable that can't be assigned again |
| `mut x = e`, `mut x: T = e` | a variable that can |
| `x = e`, `obj.field = e`, `xs[i] = e`, `m[k] = e` | assignment; `xs[i]` out of range panics |
| `x += e`, `obj.field -= e`, `xs[i] *= e`, also `/=`, `%=` | `x = x + e`, with the same type rules as the operator; the target is evaluated once |
| `if c { } else if c { } else { }` | |
| `while c { }` | |
| `for x in e { }`, `for i, x in e { }` | `e` is a range, a list, a map or a `String`; over a list, `i` is the index; over a map, one variable gives the keys and two give keys and values. `x` can be `_`. Loop variables can't be assigned to |
| `break`, `continue` | leave the innermost loop, or go on to its next pass; only inside a loop |
| `return`, `return e` | |
| `match e { ... }` | a `match` whose value is not used; its arms are calls or blocks |
| `f(...)` | a call; a call that can fail needs an `else` |

Names are visible from their declaration to the end of the block. A name can't be declared again while it is visible, and a variable can't share a name with a parameter, function, class or enum. Parameters can't be assigned to.

## Declarations

```text
func name(a: T, b: T) -> R { ... }      returns R
func name(a: T)                         returns nothing
func name() -> R ! E                    returns R, or fails with E
func name() -> ! E                      returns nothing, or fails with E

class Name { fields and methods }
class Name use Parent { ... }           a child class; `use Error` makes an error class
enum Name { cases, one per line, then methods }
enum Name: String { Case = "value" ... }  an enum with values (Int or String)
```

- **Fields**: `con name = value` or `mut name: T = value`. An initial value is required. A `con` field can be assigned only in `init`.
- **Methods**: `mut func name(...)` for a method that changes `this`. In other methods `this` is read only.
- **Inheritance**: one class parent. `override func` replaces a parent's method with exactly its signature; calls are dynamic. `super(Parent).method(...)` calls the parent's version, and `super(Parent).init(...)` may only be the first statement of an `init`; without it, the parent's `init()` runs first. `init`s and static methods aren't inherited.
- **Static methods**: `static func name(...)` belongs to the class, has no `this`, and is called as `Class.name(...)`.
- **Parameters**: `name: T` is read only (for objects, lists and maps, all the way down), `name: mut T` may change the caller's value. A read-only value can't be changed, stored, passed as `mut`, or returned (a method that isn't `mut` may return what it read from `this`; the result is as read only as the object). `mut` on `Int`, `Float`, `Bool`, `String` or an enum is an error.
- **Enums with values**: every case has a literal value, all different. `r.value` gives it, and `Role.from(x)` gives a `Role?`.
- **Constructors**: methods named `init`. A class can have several, with different numbers of parameters. Without any, it gets an empty `init()`.
- **Returns**: a function that returns a value must end in a `return`, a `panic(...)`, or an `if`/`else` or `match` whose branches all end that way.

## match patterns

`Int`, `String` and `Bool` literals (`-1` too), enum cases (`Status.Active`), and `_`. A `match` must cover every value: all cases of an enum, `true` and `false` for a `Bool`, and a `_` for an `Int` or a `String`. A repeated pattern, or an arm after `_`, is an error.

## Running

- Expressions are evaluated left to right, and arguments before the call.
- `Int` arithmetic wraps around on overflow.
- `Int` `/` rounds toward zero, and `%` has the sign of the left operand. Dividing an `Int` by zero panics.
- `Float` division by zero gives `inf`, `-inf` or `NaN`.
- Objects, lists and maps are shared: assigning or passing one never copies it.
- Changing a list's length or order, or adding or removing a map's key, while a `for` loops over it panics.
- Calls nested more than 10 000 deep panic with `stack overflow`.
- A panic prints a stack trace to stderr and exits with 101.

## print format

| Type | Printed as |
|---|---|
| `Int` | `-12` |
| `Float` | the shortest form that reads back the same, always with a `.`: `6.0`, `3.5`, `0.1`. With an exponent when the size is below `1e-7` or at least `1e21`: `1.5e-08`, `1e+21`. Also `inf`, `-inf`, `NaN` |
| `Bool` | `true`, `false` |
| `String` | the text as it is |
| enum | the case name: `Frozen` |
| `List<T>` | `[1, 2, 3]`, `[]`; a `String` inside a list or map is quoted, with `\n`, `\t`, `\"` and `\\` escaped: `["a, b"]` |
| `Map<K, V>` | `{"Ada": 36, "Grace": 85}`, `{}` |

## Grammar

```text
program     = { NEWLINE } { decl { NEWLINE } } EOF
decl        = func_decl | class_decl | enum_decl

func_decl   = "func" IDENT "(" [ params ] ")" [ "->" ret_type ] block
params      = param { "," param }
param       = IDENT ":" [ "mut" ] type
ret_type    = type [ "!" type ]                     // T, or T ! E
            | "!" type                              // nothing, or E
type        = type_name [ "?" ]
type_name   = "Int" | "Float" | "Bool" | "String"
            | "List" "<" type ">"
            | "Map" "<" type "," type ">"
            | IDENT

class_decl  = "class" IDENT [ "use" IDENT { "," IDENT } ] "{" { NEWLINE } { member { NEWLINE } } "}"
member      = field | [ "static" | "override" ] [ "mut" ] func_decl
field       = ( "con" | "mut" ) IDENT [ ":" type ] "=" expr

enum_decl   = "enum" IDENT [ ":" ( "Int" | "String" ) ] "{" { NEWLINE } { ( case | func_decl ) { NEWLINE } } "}"
case        = IDENT [ "=" ( INT | "-" INT | STRING ) ]

block       = "{" { NEWLINE } { stmt NEWLINE { NEWLINE } } "}"

stmt        = var_decl | assign | compound | if_stmt | while_stmt | for_stmt
            | "break" | "continue" | return_stmt
            | match_expr                            // a match statement
            | expr                                  // a call, maybe with an error else

var_decl    = ( "con" | "mut" ) IDENT [ ":" type ] "=" expr
assign      = target "=" expr
compound    = target ( "+=" | "-=" | "*=" | "/=" | "%=" ) expr
target      = IDENT | postfix "." IDENT | postfix "[" expr "]"
if_stmt     = "if" expr block { "else" "if" expr block } [ "else" block ]
while_stmt  = "while" expr block
for_stmt    = "for" loop_var [ "," loop_var ] "in" expr block
loop_var    = IDENT | "_"
return_stmt = "return" [ expr ]

expr        = or_expr [ "else" handler ]
handler     = "return" [ or_expr ]
            | "(" IDENT ")" "return" [ or_expr ]
            | "(" IDENT ")" block
            | or_expr                               // a fallback value, or panic(...)

or_expr     = and_expr { "or" and_expr }
and_expr    = cmp_expr { "and" cmp_expr }
cmp_expr    = range_expr [ ( "==" | "!=" | "<" | "<=" | ">" | ">=" ) range_expr ]
            | range_expr "is" IDENT
range_expr  = add_expr [ ( ".." | "..=" ) add_expr ]
add_expr    = mul_expr { ( "+" | "-" ) mul_expr }
mul_expr    = unary { ( "*" | "/" | "%" ) unary }
unary       = ( "not" | "-" ) unary | postfix
postfix     = primary { "(" [ args ] ")" | "." IDENT | "[" index "]" }
index       = expr                                  // an index
            | [ add_expr ] ( ".." | "..=" ) [ add_expr ]   // a slice; `..=` needs an end
args        = expr { "," expr }
primary     = INT | FLOAT | STRING | "true" | "false" | "null" | "this" | "super" "(" IDENT ")"
            | IDENT | "(" expr ")" | match_expr | list_lit | map_lit
list_lit    = "[" [ expr { "," expr } [ "," ] ] "]"
map_lit     = "{" [ entry { "," entry } [ "," ] ] "}"      // newlines allowed after `{`, around `,`, before `}`
entry       = expr ":" expr

match_expr  = "match" expr "{" { NEWLINE } { arm NEWLINE { NEWLINE } } "}"
arm         = pattern "=>" ( expr | block )         // a block only in a match statement
pattern     = INT | "-" INT | STRING | "true" | "false" | IDENT "." IDENT | "_"
```
