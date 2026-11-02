# Syntax

The basics. Everything here is used in [`core.brz`](../exampels/core.brz).

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

No parentheses around conditions, and braces are always required.

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
