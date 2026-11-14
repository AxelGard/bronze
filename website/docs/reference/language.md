# Language reference

Bronze 0.1.0 in one place. The [guide](../guide/first-program.md) explains the same things with examples.

## Source files

A program is one UTF-8 file ending in `.brz`. Its top level holds `func`, `class` and `enum` declarations, in any order. It must have `func main() -> Int` with no parameters; the value `main` returns is the exit code.

## Tokens

| Kind | Examples |
|---|---|
| Names | `add`, `total`, `is_even`, `_tmp`: a letter or `_`, then letters, digits and `_`. A lone `_` is only a `match` pattern |
| `Int` literals | `0`, `42`, `1_000_000` (decimal only) |
| `Float` literals | `2.0`, `3.14159`, `1.5e-3`: digits on both sides of the `.` |
| `String` literals | `"hello"`, with the escapes `\n` `\t` `\"` `\\` `\{` |
| Operators | `+ - * / %` `== != < <= > >=` `=` `->` `=>` `!` `?` `.` |
| Punctuation | `( ) { } , :` |
| Comments | `// to the end of the line`, `/* a block */` (block comments don't nest) |

A `{` in a string without a `\` is an error, so that string interpolation can be added later without changing what existing programs mean.

### Keywords

```text
func  return  con  mut  if  else  while  and  or  not  true  false
class  enum  use  this  null  match  is
```

These are reserved for later versions, and can't be used as names:

```text
data  interface  namespace  import  override  static  private  protected
for  in  break  continue  test  type  super  copy
```

## Newlines

A newline ends a statement. It doesn't when:

- it is inside `( )`
- the line ends in a binary operator (`+`, `and`, `==`, ...), `,`, `->`, `=>`, `=`, `.` or `!`

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
| `Error` | the built-in error class |
| `T?` | a `T`, or `null` (`T` is a class, an enum or `Error`) |
| `T ! E` | return types only: a `T`, or a failure with `E` |
| `! E` | return types only: nothing, or a failure with `E` |

## Operators

From the lowest precedence to the highest:

| Operators | Operands | Result |
|---|---|---|
| `x else handler` | `x` is a `T ! E` or a `T?` | `T` (see [Errors](../guide/errors.md), [Null](../guide/null.md)) |
| `or` | `Bool` | `Bool`, short-circuit |
| `and` | `Bool` | `Bool`, short-circuit |
| `== !=` | two values of the same type, or a `T?` and `null` | `Bool` |
| `< <= > >=` | two `Int`s, two `Float`s or two `String`s | `Bool` |
| `x is C` | `x` is an error, `C` is `Error` or a class that uses it | `Bool` |
| `+ -` | two `Int`s or two `Float`s; `+` also joins two `String`s | the same type |
| `* / %` | two `Int`s or two `Float`s (`%` only `Int`) | the same type |
| `not x`, `-x` | `Bool`; `Int` or `Float` | the same type |
| `f(...)`, `x.name` | | |

Comparisons can't be chained (`a < b < c` is an error). `Int` and `Float` never mix: convert with `Float(...)`. `==` on objects compares identity; on enums, the case.

## Built-ins

| Name | Takes | Does |
|---|---|---|
| `print(x)` | one `Int`, `Float`, `Bool`, `String` or enum value | prints it and a newline |
| `String(x)` | the same types | the text `print` would show |
| `Float(n)` | an `Int` | the same number as a `Float` |
| `Error(message)` | a `String` | a new `Error` |
| `panic(message)` | a `String` | stops the program; usable wherever a value is expected |
| `assert(condition, message)` | a `Bool` and a `String` | panics with `assertion failed: message` if the condition is false |

## Statements

| Statement | |
|---|---|
| `con x = e`, `con x: T = e` | a variable that can't be assigned again |
| `mut x = e`, `mut x: T = e` | a variable that can |
| `x = e`, `obj.field = e` | assignment |
| `if c { } else if c { } else { }` | |
| `while c { }` | |
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
class Name use Error { ... }            an error class, with a `message` field
enum Name { cases, one per line, then methods }
```

- **Fields**: `con name = value` or `mut name: T = value`. An initial value is required. A `con` field can be assigned only in `init`.
- **Constructors**: methods named `init`. A class can have several, with different numbers of parameters. Without any, it gets an empty `init()`.
- **Returns**: a function that returns a value must end in a `return`, a `panic(...)`, or an `if`/`else` or `match` whose branches all end that way.

## match patterns

`Int`, `String` and `Bool` literals (`-1` too), enum cases (`Status.Active`), and `_`. A `match` must cover every value: all cases of an enum, `true` and `false` for a `Bool`, and a `_` for an `Int` or a `String`. A repeated pattern, or an arm after `_`, is an error.

## Running

- Expressions are evaluated left to right, and arguments before the call.
- `Int` arithmetic wraps around on overflow.
- `Int` `/` rounds toward zero, and `%` has the sign of the left operand. Dividing an `Int` by zero panics.
- `Float` division by zero gives `inf`, `-inf` or `NaN`.
- Objects are shared: assigning or passing one never copies it.
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

## Grammar

```text
program     = { NEWLINE } { decl { NEWLINE } } EOF
decl        = func_decl | class_decl | enum_decl

func_decl   = "func" IDENT "(" [ params ] ")" [ "->" ret_type ] block
params      = param { "," param }
param       = IDENT ":" type
ret_type    = type [ "!" type ]                     // T, or T ! E
            | "!" type                              // nothing, or E
type        = type_name [ "?" ]
type_name   = "Int" | "Float" | "Bool" | "String" | IDENT

class_decl  = "class" IDENT [ "use" "Error" ] "{" { NEWLINE } { member { NEWLINE } } "}"
member      = field | func_decl
field       = ( "con" | "mut" ) IDENT [ ":" type ] "=" expr

enum_decl   = "enum" IDENT "{" { NEWLINE } { ( IDENT | func_decl ) { NEWLINE } } "}"

block       = "{" { NEWLINE } { stmt NEWLINE { NEWLINE } } "}"

stmt        = var_decl | assign | if_stmt | while_stmt | return_stmt
            | match_expr                            // a match statement
            | expr                                  // a call, maybe with an error else

var_decl    = ( "con" | "mut" ) IDENT [ ":" type ] "=" expr
assign      = target "=" expr
target      = IDENT | postfix "." IDENT
if_stmt     = "if" expr block { "else" "if" expr block } [ "else" block ]
while_stmt  = "while" expr block
return_stmt = "return" [ expr ]

expr        = or_expr [ "else" handler ]
handler     = "return" [ or_expr ]
            | "(" IDENT ")" "return" [ or_expr ]
            | "(" IDENT ")" block
            | or_expr                               // a fallback value, or panic(...)

or_expr     = and_expr { "or" and_expr }
and_expr    = cmp_expr { "and" cmp_expr }
cmp_expr    = add_expr [ ( "==" | "!=" | "<" | "<=" | ">" | ">=" ) add_expr ]
            | add_expr "is" IDENT
add_expr    = mul_expr { ( "+" | "-" ) mul_expr }
mul_expr    = unary { ( "*" | "/" | "%" ) unary }
unary       = ( "not" | "-" ) unary | postfix
postfix     = primary { "(" [ args ] ")" | "." IDENT }
args        = expr { "," expr }
primary     = INT | FLOAT | STRING | "true" | "false" | "null" | "this"
            | IDENT | "(" expr ")" | match_expr

match_expr  = "match" expr "{" { NEWLINE } { arm NEWLINE { NEWLINE } } "}"
arm         = pattern "=>" ( expr | block )         // a block only in a match statement
pattern     = INT | "-" INT | STRING | "true" | "false" | IDENT "." IDENT | "_"
```
