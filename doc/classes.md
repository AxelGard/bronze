# Classes

Examples: [`basic.brz`](../exampels/basic.brz), [`namespaces.brz`](../exampels/namespaces.brz).

## Classes

```
class Animal
{
    mut name = ""

    func init(name: String) -> Animal
    {
        this.name = name
    }

    func speak() -> String
    {
        return "..."
    }
}

con a = Animal("Rex")
```

- A class has **fields** and **methods**. Inside a class, the current object is **`this`**.
- Fields are declared in the class body. The type can be inferred from the initial value (`mut x = 0.0`), and **`mut`** marks a field as mutable.
- **`static`** marks a method as belonging to the class rather than an object: `static func create() -> My_Cls`, called as `My_Cls.create()`.

## Constructors

- Constructors are methods named **`init`**.
- A class can have **several `init`s** with different parameters.
- `init` **returns `this` automatically**. No `return` statement is needed.
- A class that defines **no** `init` gets an **empty `init()` automatically**.
  - Every field must then have a value. A field without one (`mut conn: db.Connection`) is a **compile error**, except `T?` fields, which start as `null`.
- If a class defines **any** `init`, the empty `init()` is **not** generated.

## Inheritance

```
class Dog use Animal
{
    func init(name: String) -> Dog
    {
        super(Animal).init(name)
    }

    override func speak() -> String
    {
        return "Woof"
    }
}
```

- `class A use B` means **A inherits from B**.
- Overriding a parent method requires **`override`**.
- `super(A).method()` calls parent `A`'s version of a method, and `super(A).init(...)` calls `A`'s constructor.
- If a class does not call `super(A).init(...)`, then `A`'s `init()` without arguments **runs automatically**, in `use` order. If `A` has no such `init`, it's a **compile error**.

### Multiple inheritance

```
class A { func hello() -> String { return "A" } }
class B { func hello() -> String { return "B" } }
class C use A, B { }

C().hello()   // "B"
```

- **Allowed**: `class C use A, B`.
- Parents are applied left to right, and the **last one wins** when they conflict.
- The compiler/LSP shows a **hint** when a method is inherited from more than one parent. It is not an error or a warning.
- **Diamonds**: if `A` and `B` both inherit from `Base`, a class using `A, B` gets **one shared copy** of `Base`.

## Interfaces

Interfaces exist, and a class takes them on with the same keyword as inheritance: `class C use A, My_Interface`.

## Data classes

Classes that mainly hold data. You get a constructor, equality by value and readable printing for free.

```
// short form: all fields are con
data Money(amount: Int, currency: String)

// long form: each field is con or mut, methods in the same body
data Wallet
{
    mut amount: Int
    con currency: String

    func is_empty() -> Bool
    {
        return this.amount == 0
    }
}
```

## Visibility

Public by default, with `private` and `protected` to hide things. See [namespaces and visibility](namespaces.md).
