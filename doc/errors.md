# Errors and null

**Decided:** errors as values, handled with `or`, with the Go form for full control. Full examples in [`errors.brz`](../examples/errors.brz).

## Errors are values

- An error is a normal value that is **returned, never thrown**. There are no exceptions.
- An error is any class or data class that `use`s `Error`.
- `-> User ! DbError` means "returns a `User`, or fails with a `DbError`". `-> ! Error` means "returns nothing, can fail".
- A function returns **either** the value **or** the error. No dummy values.

## Handling errors with `or`

| Form | Meaning |
|---|---|
| `f() or return` | pass the error up unchanged |
| `f() or return x` | return `x` from the current function instead (e.g. `or return http.not_found()`) |
| `f() or (e) return Wrap(e)` | pass it up with context |
| `f() or value` | use a fallback value |
| `f() or (e) { ... }` | full block. It must leave (`return`, `break`, `continue`, `panic`). |
| `f() or continue` / `or break` | skip or stop in a loop |
| `f() or panic("...")` | crash: this must not fail |

```
func load_port(path: String) -> Int ! ConfigError
{
    con text = read_file(path) or (e) return ConfigError("could not read {path}", e)
    con port = Int.parse(text) or (e) return ConfigError("{path} is not a number", e)
    return port
}
```

`or` is also logical or. The compiler decides by type: if the left side can fail (`T ! E`), `or` handles the error.

## The Go form

For full control, the Go form still works, and the error is an ordinary value:

```
con user, err = find_user(id)
if err != null
{
    errors.add(err)
}
```

`con x, err = ...` may **reuse** an existing `err` in the same function.

## Null

- The keyword is **`null`**.
- **Only `T?` types can be null.** A plain `User` is never null.
- After `if x != null`, the compiler knows `x` is not null.

```
func find_email(id: Int) -> String?
{
    if id == 1
    {
        return "ada@example.com"
    }
    return null
}
```
