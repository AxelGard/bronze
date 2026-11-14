# Spec for 0.1.0

The exact language that version 0.1.0 implements. It is a subset of Bronze, chosen so that these programs run:

- [`examples/core.brz`](../examples/core.brz): functions, variables, control flow
- [`examples/projects/counter.brz`](../examples/projects/counter.brz): a simple class
- [`examples/projects/bank.brz`](../examples/projects/bank.brz): classes, enums, `match`, errors, null

Where this page makes a choice the design docs haven't made yet, or simplifies a design decision for now, it says **(0.1 choice)**. Those can change later.

## Source files

- UTF-8 text, extension `.brz`.
- A program is **one file** in 0.1.0.
- The top level contains **function**, **class** and **enum** declarations (top-level `con` comes later).

## Tokens

| Kind | Examples |
|---|---|
| Identifier | `add`, `total`, `is_even`, `_tmp`: a letter or `_`, then letters, digits, `_`. A lone `_` is only a match pattern. |
| Keywords | `func` `return` `con` `mut` `if` `else` `while` `and` `or` `not` `true` `false` `class` `enum` `use` `this` `null` `match` `is` |
| Int literal | `0`, `42`, `1_000_000` (decimal only in 0.1. `0x` etc. come later) |
| Float literal | `2.0`, `3.14159`, `1.5e-3`. Digits are required on both sides of `.` |
| String literal | `"hello"`, with escapes `\n` `\t` `\"` `\\` `\{` |
| Operators | `+ - * / %` `== != < <= > >=` `=` `->` `=>` `!` `?` `.` |
| Punctuation | `( ) { } , :` |
| Comments | `// to end of line`, `/* block */` (**0.1 choice:** block comments do not nest) |

Reserved for later (it's an error to use them as identifiers): `data` `interface` `namespace` `import` `override` `static` `private` `protected` `for` `in` `break` `continue` `test` `type` `super` `copy`.

**0.1 choice:** a `{` inside a string literal (without `\`) is an error, "string interpolation is not supported yet", so that adding interpolation later doesn't change the meaning of existing programs.

## Newlines

Newlines end statements and members. The lexer emits a `NEWLINE` token, except:

- **inside `( )`**: an argument list or a parenthesized expression may span lines
- **after a binary operator** (`+`, `and`, `==`, ...), a `,`, `->`, `=>`, `=`, `.` or `!`: the expression continues on the next line
- several newlines in a row count as one

The parser **skips newlines before `{`**, so an opening brace on its own line works. It also skips newlines before the `else` of an `if`, so that `else` may follow `}` on the next line. The **error `else` must be on the same line** as the expression it handles.

## Grammar

```
program     = { NEWLINE } { decl { NEWLINE } } EOF
decl        = func_decl | class_decl | enum_decl

func_decl   = "func" IDENT "(" [ params ] ")" [ "->" ret_type ] block
params      = param { "," param }
param       = IDENT ":" type
ret_type    = type [ "!" type ]                     // T, or T ! E
            | "!" type                              // nothing, or E
type        = type_name [ "?" ]
type_name   = "Int" | "Float" | "Bool" | "String" | IDENT     // IDENT: a class, an enum, or Error

class_decl  = "class" IDENT [ "use" "Error" ] "{" { NEWLINE } { member { NEWLINE } } "}"
member      = field | func_decl
field       = ( "con" | "mut" ) IDENT [ ":" type ] "=" expr

enum_decl   = "enum" IDENT "{" { NEWLINE } { ( IDENT | func_decl ) { NEWLINE } } "}"

block       = "{" { NEWLINE } { stmt NEWLINE { NEWLINE } } "}"

stmt        = var_decl
            | assign
            | if_stmt
            | while_stmt
            | return_stmt
            | expr                                  // must be a call, or a call with an error else

var_decl    = ( "con" | "mut" ) IDENT [ ":" type ] "=" expr
assign      = target "=" expr
target      = IDENT | postfix "." IDENT             // x, or obj.field
if_stmt     = "if" expr block { "else" "if" expr block } [ "else" block ]
while_stmt  = "while" expr block
return_stmt = "return" [ expr ]

expr        = or_expr [ "else" handler ]            // error / null handling, lowest precedence
handler     = "return" [ or_expr ]
            | "(" IDENT ")" "return" [ or_expr ]
            | "(" IDENT ")" block
            | or_expr                               // a fallback value, or panic(...)

or_expr     = and_expr { "or" and_expr }
and_expr    = cmp_expr { "and" cmp_expr }
cmp_expr    = add_expr [ ( "==" | "!=" | "<" | "<=" | ">" | ">=" ) add_expr ]   // no chaining
            | add_expr "is" IDENT
add_expr    = mul_expr { ( "+" | "-" ) mul_expr }
mul_expr    = unary { ( "*" | "/" | "%" ) unary }
unary       = ( "not" | "-" ) unary | postfix
postfix     = primary { "(" [ args ] ")" | "." IDENT }
args        = expr { "," expr }
primary     = INT | FLOAT | STRING | "true" | "false" | "null" | "this"
            | IDENT | "(" expr ")" | match_expr

match_expr  = "match" expr "{" { NEWLINE } { pattern "=>" expr NEWLINE { NEWLINE } } "}"
pattern     = INT | "-" INT | STRING | "true" | "false" | IDENT "." IDENT | "_"
```

Notes:
- `a < b < c` is a syntax error: "comparisons can't be chained, use `and`".
- `Int`, `Float`, `String`, `Error` and class names are also callable: `Float(x)`, `String(x)`, `Error("msg")`, `Counter("clicks")`. A type name followed by `(` is parsed as an identifier.
- In `handler`, `(` IDENT `)` followed by `return` or `{` binds the error. Any other `(` starts a fallback expression.
- `if` and `while` conditions have no parentheses. A parenthesized expression is still allowed, since it's just an expression.

## Types and checking

The checker runs before anything is executed, and reports **all** errors it finds.

### Types

| Type | Values |
|---|---|
| `Int` | 64-bit signed integer |
| `Float` | 64-bit IEEE float |
| `Bool` | `true`, `false` |
| `String` | UTF-8 text |
| a class `C` | a reference to an object of class `C` |
| an enum `E` | one of its cases |
| `Error` | the built-in error class (see Errors) |
| `T?` | a `T`, or `null` |
| `T ! E` | only as a return type: a `T`, or a failure with `E` |
| `! E` | only as a return type: nothing, or a failure with `E` |

A function without `->` returns nothing ("no value"). Its result can't be used in an expression.

`T?` is only allowed for classes, enums and `Error` in 0.1 (**0.1 choice:** not `Int?` etc.). `T??` is not allowed.

### Names and scopes

- Functions, classes and enums can be used before they are declared, because all declarations are known up front.
- Duplicate top-level names are an error. Overloading exists only for `init` (see Classes).
- Each `{ }` block is a scope. Variables are visible from their declaration to the end of the block.
- **(0.1 choice)** Declaring a variable whose name already exists in the same or an outer scope, or that shadows a function, class, enum or parameter, is an error. An `else (e)` binding follows the same rule.
- Parameters behave like `con`: assigning to a parameter is an error.
- Assigning to a `con` variable is an error. Assigning to an undeclared name is an error.
- A `var_decl` with a type annotation requires the value to have exactly that type (or, for `T?`, a `T` or `null`).
- `con` only fixes the variable. The object it refers to can still be changed through its `mut` fields.

### Operators

| Operator | Operands | Result |
|---|---|---|
| `+ - * / %` | `Int, Int` | `Int` |
| `+ - * /` | `Float, Float` | `Float` |
| `+` | `String, String` | `String` (concatenation) |
| `- x` | `Int` or `Float` | same type |
| `== !=` | two values of the same type, or a `T?` and `null` | `Bool` |
| `< <= > >=` | `Int, Int`, `Float, Float`, or `String, String` | `Bool` |
| `and or` | `Bool, Bool` | `Bool`, short-circuit |
| `not` | `Bool` | `Bool` |
| `x is C` | `x` is `Error` or a class that uses `Error`, and `C` is such a class | `Bool` |

**Mixing `Int` and `Float` is an error**: `7 / 2.0` gives "cannot divide Int by Float", with the help "convert explicitly: Float(7) / 2.0". `%` on `Float` is an error.

`==` on classes compares **identity** (the same object). On enums it compares the case.

### Built-ins

| Name | Signature | Meaning |
|---|---|---|
| `print` | `print(x)`, exactly one argument of type `Int`, `Float`, `Bool`, `String` or an enum | prints `x` and a newline |
| `Float` | `Float(Int) -> Float` | conversion |
| `String` | `String(Int)`, `String(Float)`, `String(Bool)` `-> String` | conversion, the same text `print` would show |
| `Error` | `Error(String) -> Error` | makes an error with that message |
| `panic` | `panic(String)` | stops the program (see Panics). It never returns, so it is allowed wherever a value is expected. |
| `assert` | `assert(Bool, String)` | panics with "assertion failed: MESSAGE" if false |

(**0.1 choice:** no `Int(Float)` yet, because rounding rules aren't decided.)

### Returns

- A function with `-> T`, `-> T?` or `-> T ! E` must return on **every path**. "Every path" is checked simply: the last statement of the body is a `return`, a `panic(...)` call, or an `if` with an `else` where every branch ends that way. A `while` never counts. Otherwise: "missing return".
- Functions returning nothing, or `! E`, may fall off the end.
- `return expr` in a function without a value type is an error, unless `expr` is an error and the function is `! E`. A bare `return` in a function that returns a value is an error.

### main

- The program must have `func main() -> Int` with no parameters.
- Its return value is the process exit code.

## Classes

```
class Counter
{
    con name = ""
    mut count = 0

    func init(name: String)
    {
        this.name = name
    }

    func increment()
    {
        this.count = this.count + 1
    }
}

con c = Counter("clicks")
c.increment()
print(c.count)
```

- **Fields** are declared with `con` or `mut` and **must have an initial value** (**0.1 choice**). The type comes from the value, or from an annotation: `mut owner: Account? = null`.
- A `mut` field can be assigned anywhere: `obj.count = 5`. A `con` field can only be assigned inside `init`, through `this`.
- **Methods** are `func` declarations inside the class. Inside them, `this` is the object.
- **Constructors** are methods named `init`. They return `this` automatically. A return type is optional, and if written it must be the class: `func init(name: String) -> Counter`. `return` inside `init` must be bare.
- A class can have **several `init`s**. (**0.1 choice:** they must differ in their number of parameters.)
- A class without any `init` gets an empty `init()`. Writing any `init` removes it.
- **Creating an object**: call the class like a function, `Counter("clicks")`. Field initial values are evaluated first, then `init` runs.
- Objects are **shared**: passing or assigning an object never copies it.
- **(0.1 choice)** Not yet: inheritance (except `use Error`), interfaces, `static`, `private`/`protected`, `override`, `super`, read-only parameters (`a: mut T`) and `mut func`. In 0.1, any method can change `mut` fields.
- Printing an object directly is a type error. Print its fields instead.

## Enums

```
enum Status
{
    Active
    Frozen

    func label() -> String
    {
        return match this
        {
            Status.Active => "active"
            Status.Frozen => "frozen"
        }
    }
}

con s = Status.Frozen
print(s)              // Frozen
print(s.label())      // frozen
```

- Cases are written one per line, and used as `Enum.Case`.
- Enums can have methods. Inside them, `this` is the current case.
- `==` / `!=` compare cases. `print` prints the case name.
- **(0.1 choice)** Not yet: enums with values (`enum Role: String { Admin = "admin" }`).

## match

```
con fee = match status
{
    Status.Active => 0
    Status.Frozen => 25
}
```

- `match` is an **expression**. Every arm's value must have the same type, and that is the type of the `match`.
- Patterns: enum cases, `Int` / `String` / `Bool` literals, and `_` (anything else).
- **Exhaustiveness:** on an enum, every case must be listed, or there must be a `_`. On `Bool`, both `true` and `false`, or `_`. On `Int` and `String`, a `_` is required.
- A duplicate pattern, or an arm after `_`, is an error.
- An arm's value may be `panic(...)`.

## Errors

```
class NotFound use Error
{
    func init(id: Int)
    {
        this.message = "no account with id " + String(id)
    }
}

func find(id: Int) -> Account ! NotFound
{
    ...
    return NotFound(id)
}

con a = find(1) else return
```

- **`Error`** is a built-in class with one field, `mut message = ""`, created with `Error("text")`.
- A class can **`use Error`**. It inherits the `message` field, and can be used wherever an `Error` is expected. (**0.1 choice:** `use` works only with `Error` for now.)
- **`-> T ! E`**: the function returns a `T`, or fails with an `E`. `E` must be `Error` or a class that uses it, and `T` must not be one. Inside, `return x` returns the value if `x` is a `T`, and fails if `x` is an `E`.
- **`-> ! E`**: returns nothing, or fails with an `E`.
- A call to a function that can fail **must be handled with `else`**. Using its result in any other way, or calling it as a plain statement, is an error: "`find` can fail, and the error is not handled", with the help "add `else return`, `else <fallback>` or `else (e) { ... }`".
- **(0.1 choice)** Not yet: the Go form `con x, err = f()`.

### The error `else`

On an expression of type `T ! E`, the result of `expr else handler` is a `T`:

| Handler | Meaning | Checked |
|---|---|---|
| `else return` | fail with the same error | the enclosing function is `! E2` or `T2 ! E2`, and `E` can be passed as `E2` |
| `else return x` | return `x` instead | `x` fits the enclosing function's return type |
| `else (e) return x` | the same, with the error bound to `e` | `e` has type `E` |
| `else (e) { ... }` | run the block, with the error bound to `e` | the block must **leave**: its last statement is a `return` or a `panic(...)` |
| `else fallback` | use this value instead | `fallback` is a `T`, or a `panic(...)` |

As a statement, `save(x) else return` handles a `! E` call, and the value of a `T ! E` call is thrown away.

## Null

- `null` is a value of every `T?` type. A plain `T` is never null.
- Using a `T?` where a `T` is needed (reading a field, calling a method, passing it as a `T`) is an error: "`a` may be null".
- **Narrowing**: for a `con` variable or a parameter `x` of type `T?`:
  - inside the `if` block of `if x != null`, `x` has type `T`
  - inside the `else` block of `if x == null`, `x` has type `T`
  - after `if x == null { ... }` whose block always leaves (ends in `return` or `panic`), `x` has type `T` for the rest of the enclosing block
- **(0.1 choice)** No narrowing through `and` / `or`, fields, or `mut` variables.
- On an expression of type `T?`, `else` handles the null and gives a `T`. The handlers are the same as above, except that there's no error to bind (`else (e)` is an error), and a bare `else return` is only allowed in a function that returns nothing.
- On `T? ! E`, `else` handles **the error first** and gives a `T?`. A second `else` then handles the null: `con a = find(x) else return` and then `con b = a else fallback`.

## Running

- Evaluation is left to right. Arguments are evaluated before the call.
- **Int arithmetic wraps around** on overflow (64-bit two's complement).
- `/` on `Int` rounds toward zero: `-7 / 2` is `-3`. `%` takes the sign of the left operand: `-7 % 2` is `-1`.
- **Int division or `%` by zero is a panic.** Float division by zero gives `+Inf`, `-Inf` or `NaN`.
- Recursion depth: (**0.1 choice**) panic "stack overflow" after 10 000 nested calls, with the stack trace.

### print and String() format

| Type | Printed as |
|---|---|
| `Int` | decimal: `-12` |
| `Float` | shortest form that round-trips, **always with a `.`**: `6.0`, `3.5`, `0.1`. (**0.1 choice:** plain decimals for `1e-7 <= |x| < 1e21`, otherwise exponent form: `1e+21`, `1.5e-08`.) Infinity and NaN print as `inf`, `-inf`, `NaN`. |
| `Bool` | `true` / `false` |
| `String` | the text as is |
| enum | the case name: `Frozen` |

### Panics

A panic prints to stderr and exits with code **101**:

```
panic: division by zero
  at divide (core.brz:12)
  at main   (core.brz:40)
```

The message is the `panic(...)` text, "assertion failed: MESSAGE", "division by zero", or "stack overflow".

## CLI

| Command | Does | Exit code |
|---|---|---|
| `bronze run FILE` | check, then run | `main`'s return value. `1` if checking fails, `101` on panic |
| `bronze check FILE` | parse and check only, print diagnostics | `0` if clean, `1` if there are errors |
| `bronze version` | print the version | `0` |

## Diagnostics format

```
error: cannot add Int and Float
  --> core.brz:14:12
   |
14 |     return a + 2.0
   |            ^^^^^^^ Int + Float
   |
   = help: convert explicitly: Float(a) + 2.0
```

- First line: `error: ` plus a message that starts lowercase and has no trailing period.
- Second line: `  --> FILE:LINE:COLUMN` (1-based. Columns count characters, not bytes).
- Then the source line with a caret underline, and optional `= help:` / `= note:` lines.
- Several errors are separated by a blank line, in source order.
