# Classes

```bronze
class Counter
{
    con name = ""
    mut count = 0

    func init(name: String)
    {
        this.name = name
    }

    func increment()
    {
        this.count = this.count + 1
    }

    func report() -> String
    {
        return this.name + ": " + String(this.count)
    }
}

func main() -> Int
{
    con clicks = Counter("clicks")
    clicks.increment()
    clicks.increment()
    print(clicks.report())
    return 0
}
```

```text title="Output"
clicks: 2
```

## Fields

Fields are declared with `con` or `mut`, and always have an initial value. Their type comes from that value, or from a written type:

```bronze
class Player
{
    con name = ""
    mut score: Int = 0
}
```

- A `mut` field can be assigned anywhere: `player.score = 10`.
- A `con` field can only be assigned in `init`, through `this`.

## Methods and this

Methods are functions declared inside the class. Inside a method, `this` is the object it was called on. Fields are always reached through `this`: `this.count`, not `count`.

## Constructors

You create an object by calling the class like a function: `Counter("clicks")`. First every field gets its initial value, then the matching `init` runs. `init` returns the new object by itself; a `return` inside it must be bare.

A class can have several `init`s, as long as they take different numbers of parameters:

```bronze
class Point
{
    mut x = 0
    mut y = 0

    func init()
    {
    }

    func init(x: Int, y: Int)
    {
        this.x = x
        this.y = y
    }
}

func main() -> Int
{
    con origin = Point()
    con p = Point(3, 4)
    print(origin.x)
    print(p.x + p.y)
    return 0
}
```

```text title="Output"
0
7
```

A class without any `init` gets an empty one, so `Player()` works for the class above.

## Objects are shared

Assigning an object or passing it to a function never copies it. Both names refer to the same object:

```bronze
class Box
{
    mut value = 0
}

func fill(b: Box)
{
    b.value = 42
}

func main() -> Int
{
    con a = Box()
    con b = a
    fill(b)
    print(a.value)
    print(a == b)
    print(a == Box())
    return 0
}
```

```text title="Output"
42
true
false
```

`con` only fixes the variable: `a` above always refers to the same box, but the box's `mut` fields can still change. `==` on objects asks whether they are the same object.

An object can't be printed directly; print its fields instead.

## Not yet

Inheritance, interfaces, `static`, and `private`/`protected` are planned for later versions. The one kind of inheritance that works now is [`use Error`](errors.md).

## Next

[Enums and match](enums-and-match.md).
