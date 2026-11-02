# Errors and null

Example: [`errors.brz`](../examples/errors.brz).

## Errors are values

- An error is a normal object that is **returned, never thrown**. There are no exceptions.
- The caller gets the value **and** the error, and checks it **Go style** with a plain `if`. Easy to read and follow.
- `con x, err = ...` may **reuse** an existing `err` in the same function (like Go's `:=`), so you don't need `err2`, `err3` and so on.

```
con user, err = find_user(id)
if err != null
{
    return err
}

con email, err = find_email(user)   // err reused
if err != null
{
    return err
}
```

Because errors are ordinary values, they can be stored, put in lists and passed around:

```
mut errors: List<DbError> = []

for id in ids
{
    con user, err = find_user(id)
    if err != null
    {
        errors.add(err)
    }
}
```

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
