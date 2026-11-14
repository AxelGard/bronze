# Panics

[Errors](errors.md) are for things that can go wrong and should be handled. A panic is for bugs: it stops the program right away, prints what happened and where, and exits with code **101**.

```bronze
func divide(a: Int, b: Int) -> Int
{
    return a / b
}

func main() -> Int
{
    print(divide(10, 2))
    print(divide(1, 0))
    return 0
}
```

```text title="Output"
5
panic: division by zero
  at divide (main.brz:3)
  at main   (main.brz:9)
```

The panic message and the stack trace go to stderr.

## What panics

| Cause | Message |
|---|---|
| `panic("text")` | `text` |
| `assert(condition, "text")` with a false condition | `assertion failed: text` |
| `Int` division or `%` by zero | `division by zero` |
| calls nested more than 10 000 deep | `stack overflow` |
| `xs[i] = v` or `xs[i] += v` out of range | `index 5 out of range for a list of length 3` |
| `xs.insert(i, x)` out of range | `index 5 out of range for insert into a list of length 3` |
| changing a list while a `for` loops over it | `list changed while looping over it` |
| `m[k] += v` with a key that isn't there | `key "Zed" is not in the map` |
| adding or removing a key while a `for` loops over the map | `map changed while looping over it` |

## panic and assert

`panic(message)` never returns, so it can be used wherever a value is expected: in a `match` arm, as an error `else`, or as the last statement of a function that returns a value.

```bronze
func percent(n: Int) -> Int
{
    assert(n >= 0, "n must not be negative")
    if n > 100
    {
        panic("more than 100%")
    }
    return n
}

func main() -> Int
{
    print(percent(42))
    print(percent(-1))
    return 0
}
```

```text title="Output"
42
panic: assertion failed: n must not be negative
  at percent (main.brz:3)
  at main    (main.brz:14)
```

## Next

That is the whole language as it is today. The [language reference](../reference/language.md) has the details in one place.
