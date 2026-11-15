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

    mut func increment()
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

A method that changes the object is a **`mut func`**, like `increment` above. In any other method, `this` is read only: it can read fields and call other methods that aren't `mut func`, but not change anything. `init` may always change `this`, and is never written `mut func`.

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

## Static methods

A **`static func`** belongs to the class instead of an object. It has no `this`, and is called on the class's name, also inside the class. It is handy for other ways to create an object:

```bronze
class Temperature
{
    con celsius = 0.0

    func init(celsius: Float)
    {
        this.celsius = celsius
    }

    static func from_fahrenheit(f: Float) -> Temperature
    {
        return Temperature((f - 32.0) * 5.0 / 9.0)
    }
}

func main() -> Int
{
    con t = Temperature.from_fahrenheit(212.0)
    print(t.celsius)
    return 0
}
```

```text title="Output"
100.0
```

A static method can't be called on an object, and there are no static fields.

## Objects are shared

Assigning an object or passing it to a function never copies it. Both names refer to the same object:

```bronze
class Box
{
    mut value = 0
}

func fill(b: mut Box)
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

## Parameters are read only

A function can only change an object it is given if the parameter says so, with **`mut`**: `fill(b: mut Box)` above. A plain parameter is read only. The function can read fields and call methods that aren't `mut func`, but changing anything is an error:

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
    fill(Box())
    return 0
}
```

```text title="Output"
error: `b` is read only
  --> main.brz:8:5
  |
8 |     b.value = 42
  |     ^ b is a read-only Box
  |
  = help: declare the parameter as `b: mut Box`
```

So a signature tells you what a function does to what you pass it. The call doesn't change: `fill(b)`.

- Read only goes all the way down: through a read-only object, its fields' objects and lists are read only too.
- A read-only value can't be stored in a field or a list, or passed as a `mut` parameter, since it could then be changed later.
- A function can't return a read-only parameter. A method that isn't a `mut func` may return what it read out of `this`, like a `find` that returns one of its objects. The result is then read only wherever the object it was called on is.
- `mut` only goes on types that can change: objects, lists and maps. `n: mut Int` is an error.

## Not yet

Inheritance, interfaces and `private`/`protected` are planned for later versions. The one kind of inheritance that works now is [`use Error`](errors.md).

## Next

[Enums and match](enums-and-match.md).
