# Errors and null

**Decided:** errors as values, handled with `else`, with the Go form for full control. Full examples in [`errors.brz`](../examples/errors.brz).

## Errors are values

- An error is a normal value that is **returned, never thrown**. There are no exceptions.
- An error is any class or data class that `use`s `Error`.
- `-> User ! DbError` means "returns a `User`, or fails with a `DbError`". `-> ! Error` means "returns nothing, can fail".
- A function returns **either** the value **or** the error. No dummy values.

## Handling errors with `else`

| Form | Meaning |
|---|---|
| `f() else return` | pass the error up unchanged |
| `f() else return x` | return `x` from the current function instead (e.g. `else return http.not_found()`) |
| `f() else (e) return Wrap(e)` | pass it up with context |
| `f() else value` | use a fallback value |
| `f() else (e) { ... }` | full block. It must leave (`return`, `break`, `continue`, `panic`). |
| `f() else continue` / `else break` | skip or stop in a loop |
| `f() else panic("...")` | crash: this must not fail |

```
func load_port(path: String) -> Int ! ConfigError
{
    con text = read_file(path) else (e) return ConfigError("could not read {path}", e)
    con port = Int.parse(text) else (e) return ConfigError("{path} is not a number", e)
    return port
}
```

`if` is a statement, so its `else` always follows a closing `}`. The error `else` always follows an expression on the same line, so the two never meet. `or` is only logical or.

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

### Handling null with `else`

`else` handles missing values the same way it handles errors. There is no `??`.

```
con email = find_email(id) else "unknown"
con first = nums[0] else 0
con age = ages["Zed"] else 0
```

### Errors and null together

For `T? ! E` (may fail, and may find nothing), `else` peels off **one layer at a time, the error first**:

```
con maybe = find_by_email(email) else return    // the error: maybe is User?
con user = maybe else User(0, "guest")          // the null: user is User
```

That keeps "the database is down" separate from "no such user".

