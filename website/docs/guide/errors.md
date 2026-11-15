# Errors

A function that can fail says so in its return type, and every call to it must say what happens when it does. Errors are ordinary values, not exceptions.

```bronze
func parse_port(n: Int) -> Int ! Error
{
    if n < 1 or n > 65535
    {
        return Error("port out of range: " + String(n))
    }
    return n
}

func main() -> Int
{
    con port = parse_port(8080) else 80
    print(port)

    con other = parse_port(99999) else 80
    print(other)
    return 0
}
```

```text title="Output"
8080
80
```

- `-> Int ! Error` means: returns an `Int`, or fails with an `Error`.
- Inside the function, `return n` returns the value, and `return Error(...)` fails.
- `-> ! Error` is a function that returns nothing, or fails.

## Handling an error: else

A call to a function that can fail must be handled with `else`. Leaving it out is an error:

```bronze
func parse_port(n: Int) -> Int ! Error
{
    return n
}

func main() -> Int
{
    con port = parse_port(8080)
    return 0
}
```

```text title="Output"
error: `parse_port` can fail, and the error is not handled
  --> main.brz:8:16
  |
8 |     con port = parse_port(8080)
  |                ^^^^^^^^^^^^^^^^
  |
  = help: add `else return`, `else <fallback>` or `else (e) { ... }`
```

There are several kinds of `else`:

| Handler | What happens when the call fails |
|---|---|
| `else 80` | use this value instead |
| `else return` | the current function fails too, with the same error |
| `else return x` | the current function returns `x` |
| `else (e) return x` | the same, with the error available as `e` |
| `else (e) { ... }` | run the block, with the error as `e`. The block must end in a `return` or a `panic` |
| `else panic("...")` | stop the program |

The `else` must be on the same line as the call. When the call succeeds, the whole expression is its value, so `con port = parse_port(8080) else 80` is always an `Int`.

```bronze
func parse_port(n: Int) -> Int ! Error
{
    if n < 1 or n > 65535
    {
        return Error("port out of range: " + String(n))
    }
    return n
}

// passes the error on to its own caller
func double_port(n: Int) -> Int ! Error
{
    con port = parse_port(n) else return
    return port * 2
}

func main() -> Int
{
    con a = double_port(100) else (e)
    {
        print("failed: " + e.message)
        return 1
    }
    print(a)

    double_port(0) else (e)
    {
        print("failed: " + e.message)
        return 1
    }
    return 0
}
```

```text title="Output"
200
failed: port out of range: 0
```

That program exits with code 1, from the second `else`.

A call to a `-> ! E` function, or a call whose value you don't need, can stand on its own line as long as it has an `else`: `save(x) else return`.

## Your own errors

`Error` is a built-in class with one field, `message`, made with `Error("text")`. A class that says `use Error` is an error too: it gets the `message` field, and can be used wherever an `Error` is expected. It can add fields of its own. Error classes can [use each other](classes.md#inheritance) too: a `class NotFound use DbError` is a `DbError` and an `Error`, and its `init` can set the message with `super(DbError).init(...)`.

```bronze
class InsufficientFunds use Error
{
    con missing = 0

    func init(missing: Int)
    {
        this.missing = missing
        this.message = "need " + String(missing) + " more"
    }
}

class InvalidAmount use Error
{
    func init(amount: Int)
    {
        this.message = "invalid amount: " + String(amount)
    }
}

func withdraw(balance: Int, amount: Int) -> Int ! Error
{
    if amount <= 0
    {
        return InvalidAmount(amount)
    }
    if amount > balance
    {
        return InsufficientFunds(amount - balance)
    }
    return balance - amount
}

func try_withdraw(balance: Int, amount: Int)
{
    con left = withdraw(balance, amount) else (e)
    {
        if e is InsufficientFunds
        {
            print("too little money: " + e.message)
        }
        else
        {
            print("error: " + e.message)
        }
        return
    }
    print("left: " + String(left))
}

func main() -> Int
{
    try_withdraw(100, 30)
    try_withdraw(100, 130)
    try_withdraw(100, -5)
    return 0
}
```

```text title="Output"
left: 70
too little money: need 30 more
error: invalid amount: -5
```

`e is InsufficientFunds` checks which kind of error `e` is.

A function can name a specific error type, `-> Int ! InsufficientFunds`, and then it can only fail with that one. With `else return`, the caller's error type must be able to hold the error: a function that fails with `Error` can pass on any error, but one that fails with `InvalidAmount` can't pass on an `InsufficientFunds`.

## Next

[Null and T?](null.md).
