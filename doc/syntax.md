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

- **`Int`**, **`Float`**, **`Bool`** (`true` / `false`) and **`String`**.
- Conversions are **explicit**: `Float(x)`. An `Int` is never silently turned into a `Float`.
- Strings are joined with `+`: `"hello, " + name`.
- `print(...)` is a built-in that accepts any built-in type.

## Operators

| Kind | Operators |
|---|---|
| Arithmetic | `+ - * / %` |
| Comparison | `== != < <= > >=` |
| Logic | **`and`**, **`or`**, **`not`** (words, not symbols) |

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
    i = i + 1
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
