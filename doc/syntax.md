# Syntax

The basics. Everything here is used in [`core.brz`](../examples/core.brz).

## Blocks and statements

- Blocks use **braces**, and putting the opening brace on its own line is allowed.
- **No semicolons**: a newline ends a statement. The parser must allow a newline before `{`.

## Comments

```
// line comment
/* block comment */
```

## Functions

```
func add(a: Float, b: Float) -> Float
{
    return a + b
}

func greet(name: String, greeting: String = "Hello") -> String
{
    return greeting + ", " + name
}
```

- Declared with `func`.
- Parameters are `name: Type`, and every parameter has a type.
- The return type comes after `->`.
- Parameters can have **default values**.
- Functions can live outside classes.

### Multiple return values

```
func min_max(a: Int, b: Int) -> Int, Int
{
    if a < b
    {
        return a, b
    }
    return b, a
}

con low, high = min_max(7, 3)      // receive into new variables
x, y = min_max(x, y)               // or assign to existing mut variables
```

- List the return types after `->`, separated by commas.
- `return a, b` returns several values.
- The caller receives them with `con a, b = f()` or `mut a, b = f()`, or assigns them to existing `mut` variables.

## Functions as values

Example: [`lambdas.brz`](../examples/lambdas.brz). The goal is **no callback soup**: named functions are the normal thing to pass around.

```
type Check = func(User) -> Bool                 // a named signature

func is_active(user: User) -> Bool { return user.active }

func get_user(req: http.Request) -> http.Response use Handler   // explicit match
{
    ...
}

con active = users.where(is_active)             // pass a named function
con names  = users.map((u) => u.name)           // or a one-expression lambda
```

- **Function types**: `func(User) -> Bool`.
- **Named signatures**: `type Check = func(User) -> Bool`.
- A function **matches a signature automatically** if it has the right shape. It can also say so **explicitly** with `use Handler`, and then the compiler checks it at the declaration.
- **Local functions**: a function declared inside another function can see the variables around it.
- **Lambdas** are **one expression only**: `(u) => u.active`. Anything bigger must be a named function.

## Variables

```
con limit = 10           // constant: cannot be reassigned
mut total = 0            // mutable
con ratio: Float = 0.5   // explicit type
```

- **`con`** declares a constant, and **`mut`** declares a mutable variable.
- The type is inferred from the value, or written as `name: Type`.
- Assigning to a `con` is a compile error.

## Built-in types

See [types](types.md) for numbers, strings, bytes and collections.

- Conversions are **explicit**: `Float(x)`. Mixing `Int` and `Float` in arithmetic is a compile error.
- `print(...)` is a built-in that accepts any built-in type.

## Operators

| Kind | Operators |
|---|---|
| Arithmetic | `+ - * / %` |
| Compound assignment | `+= -= *= /= %=` |
| Comparison | `== != < <= > >=` |
| Logic | **`and`**, **`or`**, **`not`** (words, not symbols) |
| Ranges | `0..10` (0 to 9), `0..=10` (0 to 10) |

### Precedence (highest first)

1. `()` `.` `[]` calls
2. `not` `-x`
3. `*` `/` `%`
4. `+` `-`
5. `..` `..=`
6. `==` `!=` `<` `<=` `>` `>=` `is`
7. `and`
8. `or`
9. `else` (error handling)

So `not done and count > 3 or forced` means `((not done) and (count > 3)) or forced`. Because that is easy to misread, the formatter/LSP **suggests parentheses** when `and` and `or` are mixed.

### Equality

- **Classes** compare by **identity**: `a == b` is true only if they are the same object.
- **Data classes** compare by **value**: `Money(5, "EUR") == Money(5, "EUR")` is true.

## Control flow

No parentheses around conditions, and braces are always required. **`if` is a statement**, not an expression: it does not produce a value.

```
if n < 0
{
    return "negative"
}
else if n == 0
{
    return "zero"
}
else
{
    return "positive"
}

while i <= n
{
    i += 1
}
```

### for loops

```
for n in nums               // each element
for i, n in nums            // index and element (index first, like Go)
for name, age in ages       // map: key and value
for i in 0..10              // 0 to 9
for i in 0..=10             // 0 to 10
```

`break` and `continue` work in `for` and `while`.

### match

`match` is an **expression**: it produces a value. It must cover every case, and `_` means "anything else".

```
con label = match status
{
    Status.Pending   => "waiting"
    Status.Paid      => "paid"
    Status.Shipped   => "on its way"
    Status.Cancelled => "cancelled"
}

con size = match n
{
    0       => "none"
    1..=9   => "a few"
    10..=99 => "many"
    _       => "lots"
}
```

`match` can also be a **statement**, when only its effects matter. Then an arm can be a block, and a block can `return`:

```
match status
{
    Status.Pending =>
    {
        print("still waiting")
        return
    }
    Status.Cancelled => refund(order)
    _ => { }
}
```

## Top-level declarations

Only **`con`** is allowed outside functions. There is no global `mut`.

```
con MAX_USERS = 1000
con API_VERSION = "v2"
```

## Panics and assert

A panic stops the program with a message and a stack trace pointing at Bronze lines.

```
assert(age >= 0, "age can't be negative")
panic("unrealistic age: {age}")
```

```
panic: unrealistic age: 200
  at checked_age (src/users.brz:12)
  at main        (src/main.brz:5)
```

## Tests

Tests use the **`test`** keyword and `assert`. They live in the `tests/` folder, and can also be written next to the code. Run them with `bronze test`.

```
test "add works"
{
    assert(add(2, 2) == 4)
    assert(add(-1, 1) == 0)
}
```

## Entry point

```
func main() -> Int
{
    print("hello, world")
    return 0
}
```

`main` returns an `Int`, which becomes the process exit code.

Command-line arguments and environment variables come from the standard library:

```
con args = os.args()                      // List<String>, without the program name
con port = env.get("PORT") else "8080"    // can fail: not set
```
