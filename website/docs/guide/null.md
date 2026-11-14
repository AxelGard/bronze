# Null and T?

A plain type is never null. A value that may be missing has the type `T?`: an `Account?` is an `Account`, or `null`.

```bronze
class User
{
    con name = ""

    func init(name: String)
    {
        this.name = name
    }
}

func find(id: Int) -> User?
{
    if id == 1
    {
        return User("Ada")
    }
    return null
}

func main() -> Int
{
    con user = find(1)
    if user != null
    {
        print(user.name)
    }

    con nobody = find(2)
    print(nobody == null)
    return 0
}
```

```text title="Output"
Ada
true
```

`T?` works for every type: `Int?`, `String?`, `Bool?`, classes, enums and `Error`. A `T?` can be compared with a `T`: `n == 5` is true when `n` is not null and is `5`.

```bronze
func parse_digit(c: String) -> Int?
{
    if c == "1"
    {
        return 1
    }
    if c == "2"
    {
        return 2
    }
    return null
}

func main() -> Int
{
    con one = parse_digit("1")
    print(one == 1)
    print(parse_digit("x") == null)
    con n = parse_digit("2") else 0
    print(n * 10)
    return 0
}
```

```text title="Output"
true
true
20
```

`print`, `String(...)` and arithmetic need a `T`, so handle the null first.

## Using a T?

Using a `T?` where a `T` is needed, such as reading a field or calling a method, is an error:

```bronze
class User
{
    con name = ""
}

func find(id: Int) -> User?
{
    return null
}

func main() -> Int
{
    con user = find(1)
    print(user.name)
    return 0
}
```

```text title="Output"
error: `user` may be null
  --> main.brz:14:11
   |
14 |     print(user.name)
   |           ^^^^
   |
   = help: check it first with `if user != null { ... }`, or give a fallback: `user else ...`
```

There are two ways to get a `T` out of it.

### Check it with if

After a check, a `con` variable or a parameter has the plain type:

- inside `if x != null { ... }`
- inside the `else` block of `if x == null { ... } else { ... }`
- after `if x == null { ... }`, when that block always ends in a `return` or a `panic`: then `x` is a `T` for the rest of the block

```bronze
class User
{
    con name = ""

    func init(name: String)
    {
        this.name = name
    }
}

func greet(user: User?)
{
    if user == null
    {
        print("nobody here")
        return
    }
    print("hello, " + user.name)
}

func main() -> Int
{
    greet(User("Ada"))
    greet(null)
    return 0
}
```

```text title="Output"
hello, Ada
nobody here
```

This only works on `con` variables and parameters, since a `mut` variable could change after the check. It also doesn't look inside `and`/`or` or at fields.

### Give a fallback with else

`else` handles the null, just like it handles an [error](errors.md):

```bronze
class User
{
    con name = ""

    func init(name: String)
    {
        this.name = name
    }
}

func find(id: Int) -> User?
{
    if id == 1
    {
        return User("Ada")
    }
    return null
}

func name_of(id: Int) -> String
{
    con user = find(id) else return "unknown"
    return user.name
}

func main() -> Int
{
    print(name_of(1))
    print(name_of(2))
    con guest = find(2) else User("guest")
    print(guest.name)
    return 0
}
```

```text title="Output"
Ada
unknown
guest
```

The handlers are the same as for errors, except that there is no error to bind, so there's no `else (e)`. A bare `else return` only works in a function that returns nothing.

## Declaring with null

`null` alone has no type, so a variable or field that starts as `null` needs a written type:

```bronze
class Account
{
    mut owner: User? = null
}

class User
{
    con name = ""
}
```

## Errors and null together

A function can return `T? ! E`: a `T`, `null`, or a failure. The first `else` handles the error and gives a `T?`; a second step handles the null:

```bronze
class User
{
    con name = ""

    func init(name: String)
    {
        this.name = name
    }
}

func lookup(id: Int) -> User? ! Error
{
    if id < 0
    {
        return Error("bad id")
    }
    if id == 1
    {
        return User("Ada")
    }
    return null
}

func main() -> Int
{
    con maybe = lookup(2) else return 1
    con user = maybe else User("guest")
    print(user.name)
    return 0
}
```

```text title="Output"
guest
```

## Next

[Panics](panics.md).
