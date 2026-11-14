# Values and variables

## Types

| Type | Values | Literals |
|---|---|---|
| `Int` | 64-bit signed integers | `0`, `42`, `-7`, `1_000_000` |
| `Float` | 64-bit floating point | `2.0`, `3.14159`, `1.5e-3` |
| `Bool` | `true` and `false` | `true`, `false` |
| `String` | UTF-8 text; see [Strings](strings.md) | `"hello"` |

A `Float` literal needs digits on both sides of the `.`: write `2.0`, not `2.` or `.5`. `_` can separate digits in numbers.

Strings can hold the escapes `\n` (newline), `\t` (tab), `\"`, `\\` and `\{`. A `{` in a string starts an [interpolation](strings.md#interpolation); write `\{` for a literal brace.

```bronze
print("one\ttwo")
print("say \"hi\"")
print("a brace: \{")
```

```text title="Output"
one	two
say "hi"
a brace: {
```

## Variables

`con` declares a variable that can't be changed. `mut` declares one that can:

```bronze
con name = "Ada"
mut count = 0
count = count + 1
print(count)
```

```text title="Output"
1
```

`+=`, `-=`, `*=`, `/=` and `%=` change a variable in place: `count += 2` is the same as `count = count + 2`, with the same type rules as the operator.

```bronze
mut total = 10
total += 5
total *= 2
mut word = "bron"
word += "ze"
print(total)
print(word)
```

```text title="Output"
30
bronze
```

The type comes from the value. You can also write it:

```bronze
con limit: Int = 10
mut ratio: Float = 0.5
```

Assigning to a `con` is an error:

```bronze
func main() -> Int
{
    con name = "Ada"
    name = "Grace"
    return 0
}
```

```text title="Output"
error: cannot assign to `name`: it is declared with `con`
  --> main.brz:4:5
  |
4 |     name = "Grace"
  |     ^^^^
  |
  = help: declare it with `mut` if it needs to change
  = note: `name` is declared at line 3
```

A variable is visible from where it is declared to the end of its block (`{ ... }`). A name can't be declared again while it is visible, even in an inner block, and a variable can't have the same name as a parameter, function, class or enum.

## Arithmetic

| Operator | On | Result |
|---|---|---|
| `+ - * /` | two `Int`s, or two `Float`s | the same type |
| `%` | two `Int`s | `Int` |
| `-x` | an `Int` or a `Float` | the same type |
| `+` | two `String`s | the strings joined |

```bronze
print(7 / 2)
print(-7 / 2)
print(-7 % 2)
print(7.0 / 2.0)
print("bron" + "ze")
```

```text title="Output"
3
-3
-1
3.5
bronze
```

`Int` division rounds toward zero, and `%` takes the sign of the left side. Dividing an `Int` by zero stops the program with a [panic](panics.md). `Float` division by zero gives infinity, or NaN for `0.0 / 0.0`. `Int` arithmetic that overflows wraps around.

**`Int` and `Float` never mix.** `7 / 2.0` is an error. Convert explicitly with `Float(...)`:

```bronze
con n = 7
print(Float(n) / 2.0)
```

```text title="Output"
3.5
```

There is no conversion from `Float` to `Int` yet.

## Comparisons and logic

`==` and `!=` work on two values of the same type. `<`, `<=`, `>` and `>=` work on two `Int`s, two `Float`s, or two `String`s. `and`, `or` and `not` work on `Bool`s, and `and` and `or` only evaluate the right side when they need to.

```bronze
con age = 36
print(age >= 18 and age < 65)
print(not (age == 36))
print("apple" < "banana")
```

```text title="Output"
true
false
true
```

Comparisons can't be chained: write `0 < x and x < 10`, not `0 < x < 10`.

## Printing and converting to text

`print` takes exactly one `Int`, `Float`, `Bool`, `String` or enum value (or a [list](lists.md) or [map](maps.md) of them), and prints it on its own line. `String(...)` converts the same values to the text `print` would show:

```bronze
print(6.0)
print(0.1 + 0.2)
print(1.0 / 0.0)
print("total: " + String(42))
print("ok: " + String(true))
```

```text title="Output"
6.0
0.30000000000000004
inf
total: 42
ok: true
```

`{...}` in a string does the same conversion: `"total: {42}"` is `"total: 42"` (see [interpolation](strings.md#interpolation)).

A `Float` always prints with a `.`, in the shortest form that reads back as the same number. Very large and very small numbers use an exponent: `1e+21`, `1.5e-08`. Infinity and NaN print as `inf`, `-inf` and `NaN`.

## Next

[Control flow](control-flow.md).
