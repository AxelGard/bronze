# Data classes

A **data class** holds data. Bronze gives it a constructor, `==` by value, and printing:

```bronze
data Money(amount: Int, currency: String)

func main() -> Int
{
    con price = Money(5, "EUR")
    print(price)
    print(price.amount)
    print(price == Money(5, "EUR"))
    return 0
}
```

```text title="Output"
Money(amount: 5, currency: "EUR")
5
true
```

## The short form

`data Money(amount: Int, currency: String)` declares the fields in `( )`. They are all `con`, and the constructor takes them in that order. A body after it can add methods:

```bronze
data Money(amount: Int, currency: String)
{
    func plus(other: Money) -> Money
    {
        return Money(this.amount + other.amount, this.currency)
    }
}

func main() -> Int
{
    print(Money(5, "EUR").plus(Money(2, "EUR")))
    return 0
}
```

```text title="Output"
Money(amount: 7, currency: "EUR")
```

## The long form

`data Wallet { ... }` declares fields like a class does, `con` or `mut`, but a field may leave out its initial value. **The constructor takes the fields without an initial value, in the order they are written.** The others start with their value.

```bronze
data Wallet
{
    mut amount: Int
    con currency: String
    mut history: List<Int> = []

    mut func spend(n: Int)
    {
        this.amount -= n
        this.history.append(n)
    }
}

func main() -> Int
{
    con w = Wallet(100, "EUR")
    w.spend(30)
    print(w)
    return 0
}
```

```text title="Output"
Wallet(amount: 70, currency: "EUR", history: [30])
```

A data class has no `init` of its own: its constructor comes from its fields.

## Equality and printing

- `==` compares two values of a data class field by field, each with its own `==`. Objects of plain classes in fields are compared by identity, as always.
- `print`, `String(...)` and `{...}` in strings show the name and every field, with strings quoted: `LineItem(name: "tea", count: 2)`. This works when every field can be printed.

## Map keys

A data class whose fields are all `con`, and themselves keys (`Int`, `String`, `Bool`, enums, or such data classes), can be a map key:

```bronze
data Point(x: Int, y: Int)

func main() -> Int
{
    con names: Map<Point, String> = {}
    names[Point(0, 0)] = "origin"
    print(names[Point(0, 0)] else "?")
    return 0
}
```

```text title="Output"
origin
```

A data class with a `mut` field can't be a key, since changing the key would lose its entry.

## Using interfaces and Error

A data class can use [interfaces](classes.md#interfaces), and `Error`, but not other classes, and nothing can use a data class. An error as a data class is short; its `message` is how it prints:

```bronze
data NotFound(id: Int) use Error

func find(id: Int) -> String ! NotFound
{
    return NotFound(id)
}

func main() -> Int
{
    con name = find(7) else (e)
    {
        print(e.message)
        print(e.id)
        return 0
    }
    print(name)
    return 1
}
```

```text title="Output"
NotFound(id: 7)
7
```

## Passing them around

A data class whose fields are all `con` and can't change themselves, like `Money`, can't change at all. It is passed around like an `Int`, and [`mut`](classes.md#parameters-are-read-only) on such a parameter is an error. A long-form data class with `mut` fields is read only as a parameter, like an object.

## Next

[Enums and match](enums-and-match.md).
