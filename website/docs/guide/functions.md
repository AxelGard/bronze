# Functions

```bronze
func add(a: Int, b: Int) -> Int
{
    return a + b
}

func average(a: Int, b: Int) -> Float
{
    return Float(a + b) / 2.0
}

func greet(name: String)
{
    print("hello, " + name)
}

func main() -> Int
{
    print(add(7, 5))
    print(average(7, 5))
    greet("Ada")
    return 0
}
```

```text title="Output"
12
6.0
hello, Ada
```

- Every parameter has a type.
- `-> T` gives the return type. A function without `->` returns nothing, and its call can't be used as a value.
- Calls must pass exactly as many arguments as there are parameters, with the right types.
- Parameters can't be assigned to. Copy one into a `mut` variable to change it.
- An object, list or map parameter is read only, unless it is declared `mut` (`nums: mut List<Int>`), or `copy` to get its own copy. See [parameters are read only](classes.md#parameters-are-read-only).
- Functions can be called before they are declared, and can call themselves.

## Returning on every path

A function that returns a value must end in a `return` on every path. The checker looks at the last statement of the body: it must be a `return`, a [`panic(...)`](panics.md), or an `if` with an `else` (or a [`match`](enums-and-match.md)) whose branches all end that way.

```bronze
func abs(n: Int) -> Int
{
    if n < 0
    {
        return -n
    }
}
```

```text title="Output"
error: missing return at the end of `abs`
  --> main.brz:7:1
  |
7 | }
  | ^
  |
  = note: `abs` returns Int, so every path must end with `return`
```

A `while` never counts as returning, even `while true`. Add a `return` (or a `panic`) after the loop.

## Recursion

```bronze
func fib(n: Int) -> Int
{
    if n < 2
    {
        return n
    }
    return fib(n - 1) + fib(n - 2)
}

func main() -> Int
{
    print(fib(20))
    return 0
}
```

```text title="Output"
6765
```

Calls can nest 10 000 deep. Deeper than that, the program stops with a "stack overflow" [panic](panics.md).

## Next

[Classes](classes.md).
