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

## Inheritance

`class Dog use Animal` makes `Dog` a **child** of `Animal`: it has all of `Animal`'s fields and methods, and a `Dog` can be used wherever an `Animal` is expected.

```bronze
class Animal
{
    con name = ""

    func init(name: String)
    {
        this.name = name
    }

    func speak() -> String
    {
        return "..."
    }

    func introduce() -> String
    {
        return this.name + " says " + this.speak()
    }
}

class Dog use Animal
{
    func init(name: String)
    {
        super(Animal).init(name)
    }

    override func speak() -> String
    {
        return "Woof"
    }
}

func main() -> Int
{
    con pets: List<Animal> = [Animal("Cat"), Dog("Rex")]
    for pet in pets
    {
        print(pet.introduce())
    }
    return 0
}
```

```text title="Output"
Cat says ...
Rex says Woof
```

- A class has at most one class parent.
- Replacing a parent's method needs **`override`**, and the method must have exactly the parent's parameters and result. Calls use the object's own version: `introduce` above calls `Dog`'s `speak` for a dog.
- **`super(Animal).speak()`** calls the parent's version of a method.
- A child doesn't get its parent's `init`s. Its `init` sets up the parent with **`super(Animal).init(...)`**, which must be its first statement. Without that call, the parent's `init()` runs by itself first; if the parent has no `init()`, the call is required.
- Setting up an object gives every field its initial value, the parent's first, then runs the child's `init`.
- A child can't declare a field its parent already has, and a parent's `con` field can only be set in the parent's own `init`.
- Static methods aren't inherited.

## is

`x is C` asks whether an object is a `C`, or a child of `C`. Inside `if x is C`, a `con` variable or a parameter `x` is a `C`, so `C`'s own methods can be used:

```bronze
class Animal
{
}

class Dog use Animal
{
    func fetch() -> String
    {
        return "fetches the ball"
    }
}

func play(pet: Animal) -> String
{
    if pet is Dog
    {
        return pet.fetch()
    }
    return "sleeps"
}

func main() -> Int
{
    print(play(Dog()))
    print(play(Animal()))
    return 0
}
```

```text title="Output"
fetches the ball
sleeps
```

On a `T?`, `is` is false for `null`, and inside the `if` the value is not null. Like `!= null`, it doesn't narrow a `mut` variable or a field, and only inside the `if` itself. `x is C` where `x` could never be a `C` is an error.

Error classes use inheritance too: `class NotFound use DbError` is a `DbError`, and through it an `Error`. See [errors](errors.md).

## private and protected

Fields and methods are public by default. **`private`** hides one from everything but the class's own methods, and **`protected`** also lets the class's children use it:

```bronze
class Account
{
    con owner = ""
    private mut pin = ""
    protected mut balance = 0

    func init(owner: String, pin: String)
    {
        this.owner = owner
        this.pin = pin
    }

    func same_pin(other: Account) -> Bool
    {
        return this.pin == other.pin
    }
}

class Savings use Account
{
    func init(owner: String)
    {
        super(Account).init(owner, "1234")
    }

    mut func add_interest()
    {
        this.balance += 10
    }
}

func main() -> Int
{
    con s = Savings("Ada")
    s.add_interest()
    print(s.same_pin(Account("Bob", "1234")))
    print(s.pin)
    return 0
}
```

```text title="Output"
error: `pin` is private to `Account`
  --> main.brz:37:13
   |
37 |     print(s.pin)
   |             ^^^
   |
   = note: it is declared at line 4; only `Account`'s own methods can use it
```

- A private member works on any object of the class, not only `this`: `same_pin` reads `other.pin`.
- A child can't use its parent's private members, and can't declare a method with the name of one.
- An override keeps the visibility of the method it replaces.
- A `private init` means only the class itself can create objects that way, for example in a static method. A child can call a `protected init` with `super(...).init(...)`.
- `private` and `protected` are only for fields and methods, until namespaces come.

## Interfaces

An **interface** lists methods, without bodies. A class that says `use` with it must have each of them, with the same types, and can then be used wherever the interface is expected:

```bronze
interface Shape
{
    func area() -> Float
    func name() -> String
}

class Square use Shape
{
    con side = 0.0

    func init(side: Float)
    {
        this.side = side
    }

    func area() -> Float
    {
        return this.side * this.side
    }

    func name() -> String
    {
        return "square"
    }
}

class Circle use Shape
{
    con radius = 0.0

    func init(radius: Float)
    {
        this.radius = radius
    }

    func area() -> Float
    {
        return 3.0 * this.radius * this.radius
    }

    func name() -> String
    {
        return "circle"
    }
}

func main() -> Int
{
    con shapes: List<Shape> = [Square(2.0), Circle(1.0)]
    for s in shapes
    {
        print(s.name() + ": " + String(s.area()))
    }
    return 0
}
```

```text title="Output"
square: 4.0
circle: 3.0
```

- A class takes on an interface only by saying so: `class Square use Shape`. Having the methods isn't enough.
- A class can use one class and any number of interfaces: `class Dog use Animal, Named, Printable`. Its children are those interfaces too.
- The methods for an interface are public, and written without `override`. A method may be a `mut func` in the interface; then it must be one in the class too.
- An interface can use other interfaces: `interface Solid use Shape { func volume() -> Float }`. A class that uses `Solid` needs both sets of methods.
- An interface can't be created, has no fields, and can't be printed. `==` asks whether two values are the same object, and `is` works with interfaces too.

## Next

[Enums and match](enums-and-match.md).
