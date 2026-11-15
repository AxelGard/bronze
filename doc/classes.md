# Classes

Examples: [`basic.brz`](explorations/basic.brz), [`namespaces.brz`](explorations/namespaces.brz).

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
- **`static`** marks a method as belonging to the class rather than an object: `static func create() -> My_Cls`, called as `My_Cls.create()`. Static methods have no `this`, and are not inherited. There are no static fields.
- A method that changes `this` is a **`mut func`**. In other methods `this` is read only. See [passing arguments](#passing-arguments).

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
- `override` is strict: it is required to replace a parent method, it is an error on a method that replaces nothing, and the signature must be exactly the parent's.
- A child can't redeclare a parent's field.
- `super(A).init(...)` may only be the **first statement** of an `init`. If a class does not call it, then `A`'s `init()` without arguments **runs automatically**, before the child's `init` body. If `A` has no such `init`, it's a **compile error**.
- Setting up an object: first every field gets its initial value, the parent's fields first, then the child's `init` runs.
- `x is C` works for every class and interface, and narrows `x` to a `C` inside the `if`, like `!= null`.
- No abstract methods or `final`: interfaces cover "must have this method".

### Multiple inheritance

**After 1.0.** Until then a class has at most one class parent, plus any number of interfaces. The design so far:

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

```
interface Shape
{
    func area() -> Float
    func name() -> String
}

interface Solid use Shape
{
    func volume() -> Float
}

class Square use Shape { ... }
```

- An interface has **method signatures only**: no bodies, no fields.
- A class takes it on with the same keyword as inheritance: `class C use A, My_Interface`. It must have each method, with the same types, and doesn't write `override` for them.
- **Nominal**: a class is a `Shape` only if it says `use Shape`.
- An interface can use other interfaces.

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

- The long form's constructor takes the fields **without an initial value**, in order. A data class has no `init` of its own.
- `==` compares by value. It prints as `Money(amount: 5, currency: "EUR")`.
- A data class can `use` interfaces and `Error` (`data NotFound(id: Int) use Error`), but not classes, and nothing can use a data class.
- One whose fields are all `con` and valid keys can be a map key.
- Copy with changes, `m.with(amount: 750)`, comes later, with named arguments.

## Passing arguments

Example: [`references.brz`](explorations/references.brz).

Everything is shared. What a function may do with what it is given is part of the parameter:

| Parameter | Meaning |
|---|---|
| `a: Account` | read only, all the way down |
| `a: mut Account` | may change the caller's object |
| `a: copy Account` | gets its own deep copy (lists, maps, data classes; a class can't be copied) |

The call site doesn't mark it: `deposit(acc, 100)`. A read-only value can't be stored or returned where it could be changed later, except that a method that isn't `mut` may return what it holds; its result is then as read only as the object. See [spec-0.3](spec-0.3.md#parameter-modes) for the exact rules.

## Visibility

Public by default, with `private` and `protected` to hide things. See [namespaces and visibility](namespaces.md).
