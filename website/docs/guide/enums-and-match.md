# Enums and match

## Enums

An enum lists its cases, one per line, and can have methods:

```bronze
enum Status
{
    Active
    Frozen
    Closed

    func is_open() -> Bool
    {
        return this != Status.Closed
    }
}

func main() -> Int
{
    con s = Status.Frozen
    print(s)
    print(s.is_open())
    print(s == Status.Active)
    print("status: " + String(s))
    return 0
}
```

```text title="Output"
Frozen
true
false
status: Frozen
```

- A case is written `Enum.Case`: `Status.Frozen`.
- Inside a method, `this` is the current case.
- `==` and `!=` compare cases. `print` and `String(...)` give the case name.

## Enums with values

An enum can give each case a value, an `Int` or a `String`, for storing a case or sending it somewhere, as in a database or JSON:

```bronze
enum Role: String
{
    Admin = "admin"
    Member = "member"
}

enum Priority: Int
{
    Low = 1
    Normal = 5
    High = 10
}

func parse_role(s: String) -> Role
{
    return Role.from(s) else Role.Member
}

func main() -> Int
{
    con r = Role.Admin
    print(r)
    print(r.value)
    print(parse_role("admin"))
    print(parse_role("nobody"))
    print(Priority.High.value - Priority.Low.value)
    return 0
}
```

```text title="Output"
Admin
admin
Admin
Member
9
```

- The value type comes after the name: `enum Role: String`. It is `Int` or `String`.
- Every case has a value, written out as a literal. Two cases can't have the same value.
- `r.value` gives the value. `Role.from(x)` gives the case with that value as a `Role?`, or `null` if there is none.
- `print` and `String(...)` still give the case name, and `match` and `==` work on cases as before.

## match

`match` picks a value by comparing against each pattern in turn:

```bronze
enum Status
{
    Active
    Frozen
    Closed
}

func fee(s: Status) -> Int
{
    return match s
    {
        Status.Active => 0
        Status.Frozen => 25
        Status.Closed => 100
    }
}

func main() -> Int
{
    print(fee(Status.Frozen))
    return 0
}
```

```text title="Output"
25
```

`match` is an expression: every arm gives a value of the same type, and that is the type of the whole `match`.

Patterns can be enum cases, `Int`, `String` or `Bool` literals, or `_`, which matches anything:

```bronze
func describe(n: Int) -> String
{
    return match n
    {
        0 => "zero"
        1 => "one"
        -1 => "minus one"
        _ => "many"
    }
}

func main() -> Int
{
    print(describe(1))
    print(describe(7))
    return 0
}
```

```text title="Output"
one
many
```

### Covering every case

A `match` must cover every possible value:

- On an enum: every case, or a `_`.
- On a `Bool`: `true` and `false`, or a `_`.
- On an `Int` or a `String`: a `_` is always needed.

The checker tells you which cases are missing:

```bronze
enum Light
{
    Red
    Yellow
    Green
}

func next(l: Light) -> Light
{
    return match l
    {
        Light.Red => Light.Green
        Light.Green => Light.Yellow
    }
}
```

```text title="Output"
error: match does not cover `Light.Yellow`
  --> main.brz:10:12
   |
10 |     return match l
   |            ^^^^^^^
   |
   = help: add an arm for each, or `_ => ...` for all other values
```

So when you add a case to an enum, every `match` that needs to handle it is reported. A pattern that appears twice, or an arm after `_`, is also an error.

An arm can [`panic`](panics.md) instead of giving a value: `_ => panic("unexpected")`.

### match as a statement

A `match` can also stand on its own, to run different code for each case. Its arms are then calls or blocks, and don't need to have a type in common. A `return` in a block returns from the function:

```bronze
enum Suit
{
    Hearts
    Diamonds
    Clubs
    Spades
}

func color(s: Suit) -> String
{
    match s
    {
        Suit.Hearts =>
        {
            print("hearts")
            return "red"
        }
        Suit.Diamonds => print("diamonds")
        _ => { }
    }
    return "red or black"
}

func main() -> Int
{
    print(color(Suit.Hearts))
    print(color(Suit.Spades))
    return 0
}
```

```text title="Output"
hearts
red
red or black
```

It must still cover every case; write `_ => { }` to ignore the rest.

## Next

[Errors](errors.md).
