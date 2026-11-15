# Spec for 0.3.0

The exact language that version 0.3.0 implements. It is [spec-0.2](spec-0.2.md) plus the changes on this page: everything in 0.1 and 0.2 still holds unless this page says otherwise.

0.3.0 adds read-only parameters by default with `mut T`, `copy T` and `mut func`, inheritance with `override` and `super`, `is` with narrowing for all classes, interfaces, data classes, `static` methods, `private` and `protected`, and enums with values. The choices come from [`worksheet-0.3.brz`](explorations/worksheet-0.3.brz).

Where this page makes a choice the worksheet didn't, it says **(0.3 choice)**. Those can change later.

**This version breaks 0.2 programs.** A parameter is now read only, so a function that changes an object or list it was given needs `mut` on that parameter, and a method that changes `this` must be a `mut func`. The checker says where (see Parameter modes).

## Tokens

- `data`, `interface`, `override`, `static`, `private`, `protected`, `super` and `copy` become keywords. They are no longer only reserved. (Each becomes a keyword with the step of the work order that implements it.)
- `namespace`, `import`, `test` and `type` stay reserved.

## Grammar changes

Only the changed and new rules. Everything else is as in 0.1 and 0.2.

```
decl        = func_decl | class_decl | enum_decl | interface_decl | data_decl

param       = IDENT ":" [ "mut" | "copy" ] type

class_decl  = "class" IDENT [ uses ] "{" { NEWLINE } { member { NEWLINE } } "}"
uses        = "use" IDENT { "," IDENT }
member      = field | method
field       = [ visibility ] ( "con" | "mut" ) IDENT [ ":" type ] "=" expr
method      = [ visibility ] [ "static" | "override" ] [ "mut" ] func_decl
visibility  = "private" | "protected"

interface_decl = "interface" IDENT [ uses ] "{" { NEWLINE } { signature { NEWLINE } } "}"
signature   = [ "mut" ] "func" IDENT "(" [ params ] ")" [ "->" ret_type ]

data_decl   = "data" IDENT "(" [ data_param { "," data_param } ] ")" [ uses ]
              [ "{" { NEWLINE } { method { NEWLINE } } "}" ]                 // short form
            | "data" IDENT [ uses ] "{" { NEWLINE } { ( data_field | method ) { NEWLINE } } "}"   // long form
data_param  = IDENT ":" type
data_field  = [ visibility ] ( "con" | "mut" ) IDENT ( ":" type [ "=" expr ] | "=" expr )

enum_decl   = "enum" IDENT [ ":" ( "Int" | "String" ) ] "{" { NEWLINE } { ( case | func_decl ) { NEWLINE } } "}"
case        = IDENT [ "=" ( INT | "-" INT | STRING ) ]

primary     = ... | "super" "(" IDENT ")"      // only before "." IDENT, see super
```

Notes:
- In a class body, `mut` followed by `func` starts a method, and `mut` followed by a name starts a field.
- `mut` in `param` is the parameter mode. A parameter is still like `con`: assigning to the parameter itself is an error, with any mode.
- After `use`, a `,` continues the list on the next line, like anywhere else.

## Parameter modes

Objects, lists and maps are shared, as in 0.1 and 0.2. What a function may do with the ones it is given is now part of the parameter:

| Parameter | The function may |
|---|---|
| `a: Account` | only read: read fields, call methods that aren't `mut func`, loop, index |
| `a: mut Account` | change the caller's object |
| `a: copy Account` | change its own deep copy, made at the call |

```
func total(items: List<Int>) -> Int           // reads only
func add_defaults(items: mut List<Int>)       // changes the caller's list
func sorted(items: copy List<Int>) -> List<Int>
{
    items.sort()                              // the caller's list is untouched
    return items
}
```

**Changeable types** are classes (including `Error`), lists, maps, and data classes that have a `mut` field or a field of a changeable type. The others (`Int`, `Float`, `Bool`, `String`, enums, and data classes with only `con` fields of unchangeable types) can't change, so for them read only is simply what they are. `mut` or `copy` on a parameter of an unchangeable type is an error: "`mut` does nothing on Int, which can't be changed". Read-only is checked only; it costs nothing at runtime.

The call site doesn't mark anything: `deposit(acc, 100)`. The signature says it, and hover in the language server shows it.

### mut func

A method that changes `this` is marked `mut func`:

```
class Account
{
    mut balance = 0

    mut func deposit(amount: Int)
    {
        this.balance += amount
    }

    func describe() -> String
    {
        return "balance: {this.balance}"
    }
}
```

- Inside a method that isn't `mut`, `this` is read only.
- `init` always has a changeable `this`, and is never written `mut func`. `static mut func` and a top-level `mut func` are errors.
- Enums can't have `mut func` methods, since a case can't change.

### What read only means

A value is **read only** when it is:
- a plain parameter of a changeable type
- `this` in a method that isn't a `mut func`
- anything read out of a read-only value: a field, an element, a loop variable, a narrowed `is` or `!= null` form of it. Read only goes **all the way down**: through a read-only `List<Account>`, the accounts are read only too.
- a slice, `keys()` or `values()` of a read-only list or map, **if its elements can change**. These are new lists, so a slice of a read-only `List<Int>` is an ordinary changeable `List<Int>`; but a slice of a read-only `List<Account>` holds the same accounts, and is read only as a whole. **(0.3 choice)**
- the result of calling a method that isn't a `mut func` on a read-only object (see below)
- a `con` or `mut` variable whose initial value is read only, and a list or map literal with a read-only element **(0.3 choice)**

Everything else is changeable: a new object, a list or map literal, a function's result, a `mut` or `copy` parameter, `this` in `init` or a `mut func`, and what is read out of a changeable value.

A read-only value **can't**:
- have a field or element assigned, with `=` or `op=`: "`a` is read only"
- call a `mut func`, or a built-in that changes it: `append`, `insert`, `remove_at`, `pop`, `reverse` and `sort` on a list, `remove` and `m[k] = v` on a map
- be passed as a `mut T`
- be **stored** where it could be changed later: assigned to a field or an element, appended or inserted into a list, or assigned to a `mut` variable that was declared changeable. "`a` is read only, and can't be stored here", with the help "declare the parameter as `mut Account`"
- be **returned**, with one exception: a method that isn't a `mut func` may return what it read out of `this`. **(0.3 choice)**

A read-only value **can** be passed as a plain `T` or as a `copy T`, compared, printed and looped over.

### Results of methods

A method that isn't a `mut func` may return objects it holds, like a `find` that returns one of its accounts:

```
class Bank
{
    mut first: Account? = null

    func find(owner: String) -> Account?
    {
        con a = this.first
        ...
        return a
    }
}
```

**The result of a method that isn't a `mut func` is as read only as the object it is called on.** On a changeable `bank`, `bank.find("Ada")` is a changeable `Account`; on a read-only `bank` parameter, it is read only. **(0.3 choice)** This also covers methods that build a new value, whose result is then read only on a read-only object, even though nothing else holds it.

A plain function can't return a read-only parameter: `func same(b: Box) -> Box { return b }` is an error, with the help "declare the parameter as `b: mut Box`".

These rules don't apply to values of unchangeable types: a `String` field of a read-only object is just a `String`.

### copy

- `a: copy T` makes a **deep copy** at the call, before the function's body runs: lists, maps and data classes are copied all the way down.
- A class can't be copied: `copy T` where `T` is a class, or contains one (`copy List<Account>`), is an error: "a class can't be copied; use a data class". A class may hold things like connections, and copying those silently would be wrong.
- The copy is changeable, and belongs to the function.

### What changes in 0.2 programs

A plain parameter that is changed, and a method that changes `this`, are errors now, with a help that says what to add:

```
error: `xs` is read only
  --> algorithms.brz:27:13
   |
27 |             xs[j + 1] = y
   |             ^^^^^^^^^ xs is a read-only List<Int>
   |
   = help: declare the parameter as `xs: mut List<Int>`
```

## Inheritance

```
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
```

- `class Dog use Animal` makes `Dog` a **child** of `Animal`: it has every field and method of `Animal`, and a `Dog` can be used wherever an `Animal` is expected, also as a `T?`. `Animal` is its **parent**, and the parent's parents are its ancestors.
- A class has **at most one class parent**, plus any number of interfaces, in any order: `class Dog use Animal, Named`. Two classes after `use` is an error: "a class can use only one class in 0.3". `Error` counts as a class, so error classes can form a family: `class DbError use Error`, `class NotFound use DbError`.
- A class can't use itself, directly or through its ancestors: "`A` uses itself".
- Data classes can't be parents: "`Money` is a data class, and can't be used".
- `List<Dog>` is still not a `List<Animal>` (see spec-0.2).

### Fields

- A child can't declare a field or method with the name of an ancestor's field or method, `private` ones included, except to override a method: "`name` is already a field of `Animal`".
- **(0.3 choice)** A `con` field can only be assigned in an `init` of the class that declares it. A child sets it through `super(Animal).init(...)`.

### Constructors

- A child's `init` may call **`super(Animal).init(...)`**, and only as its **first statement**: "`super(Animal).init` must be the first statement of `init`".
- If it doesn't, `Animal`'s `init()` without arguments runs before the child's `init` body. If `Animal` has no such `init`: "`Dog` must call `super(Animal).init(...)`: `Animal` has no `init()`".
- **(0.3 choice)** `init`s are not inherited. A child without an `init` gets an empty `init()`, as in 0.1, which runs the parent's `init()`; if the parent has none, it is the same error, at the class.

Setting up `Dog("Rex")`:
1. Every field gets its initial value, the ancestors' fields first (the oldest first), each class in the order its fields are written.
2. `Dog`'s `init` runs. If its first statement is `super(Animal).init(...)`, that runs `Animal`'s `init` (which may in turn call its parent's). Otherwise `Animal`'s `init()` runs first, by itself.
3. The rest of `Dog`'s `init` body runs.

### override

- Replacing a parent method requires **`override`**: "`speak` replaces `Animal.speak`, and needs `override`".
- `override` on a method that replaces nothing is an error: "`speak` doesn't override anything".
- The overriding method must have **exactly** the parent's parameter types, modes, result type, `mut`, and visibility: "`speak` must return String, like `Animal.speak`".
- `init`, `static` methods and `private` methods can't be overridden. Methods that come from an interface aren't written with `override` (see Interfaces).
- Calls are **dynamic**: calling `speak()` on an `Animal` that is a `Dog` runs `Dog`'s `speak`. That includes calls through `this` in `Animal`'s own methods.

### super

- `super(Animal).speak()` calls `Animal`'s version of `speak`, which may be one `Animal` inherited. It doesn't dispatch dynamically.
- `super(X)` must name the class's own class parent: "`Shape` is not the class parent of `Square`". A grandparent or an interface is an error. **(0.3 choice)**
- `super(X)` is only allowed directly before `.method(...)`, in a method or `init` that has a `this`, and not in a `static` method. On its own, it is an error: "`super(Animal)` is not a value".
- `super(Animal).m()` in a method that isn't `mut` can only call methods that aren't `mut` either, as for `this`.

### is and narrowing

`x is C` now works for **any class or interface `C`**:

```
func describe(pet: Animal) -> String
{
    if pet is Dog
    {
        return pet.fetch()          // pet is a Dog here
    }
    return pet.speak()
}
```

- `x is C` is true when `x` is a `C` or a child of `C` (for an interface: when its class uses `C`, directly or through a parent or another interface). On a `T?`, it is false for `null`.
- **Narrowing**: inside the `if` block of `if x is C`, a `con` variable, a parameter or an `else (e)` binding `x` has type `C`, like for `!= null`. On a `T?`, that also means not null. No narrowing through `not`, `and`, `or`, fields or `mut` variables, and none after the `if` **(0.3 choice)**.
- `x is C` where `x` can never be a `C` is an error: "`pet` is an Animal, and can never be a Square". That is the case when both are classes and neither is a child of the other. With an interface on either side it is allowed, since a child class could use the interface.
- Narrowing keeps read only: a read-only `pet` is a read-only `Dog`.

## Interfaces

```
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
```

- An interface has **method signatures only**: no bodies, no fields, no `static`, no visibility. A signature can be a `mut func`, and can fail (`-> T ! E`).
- An interface is a type: for parameters, fields, variables, lists, maps (as values) and `T?`. It can't be created: "`Shape` is an interface, and can't be created".
- A class takes it on with **`use`** (nominal). Having the methods without `use Shape` doesn't make a class a `Shape`.
- A class that uses an interface must have each of its methods, its own or inherited, public, with exactly the same parameter types, modes, result type and `mut`: "`Square` uses `Shape`, but has no method `name`". It doesn't write `override` for them.
- A child of a class that uses an interface is also that interface.
- An interface can use other interfaces: `interface Solid use Shape { func volume() -> Float }`. A class that uses `Solid` must have the methods of both, and is a `Shape` too. An interface can't use a class, and can't use itself.
- Two interfaces of a class may ask for the same method, if the signatures are the same. Different signatures are an error.
- `==` on interface values compares identity, as for classes.
- An interface counts as a changeable type for parameter modes, and `copy Shape` is an error, since the value may be a class.

## Data classes

```
data Money(amount: Int, currency: String)

data Wallet
{
    mut amount: Int
    con currency: String
    mut history: List<Int> = []

    func is_empty() -> Bool
    {
        return this.amount == 0
    }
}

con m = Money(5, "EUR")
con w = Wallet(100, "EUR")
```

- **Short form**: `data Money(amount: Int, currency: String)`. Every field is `con`, and the constructor takes them in order. **(0.3 choice)** It may have a body with methods (no fields): `data Money(amount: Int, currency: String) { func ... }`.
- **Long form**: `data Wallet { ... }`, with `con` and `mut` fields and methods. Here a field may leave out its initial value (`mut amount: Int`), and then it needs a type. The constructor takes **the fields without an initial value, in the order they are written**. Fields with a value keep it.
- A data class has no `init` of its own: "a data class gets its constructor from its fields".
- **`==`** compares **by value**: the same data class, and every field equal by its own `==` (objects in fields by identity). `!=` likewise.
- `print`, `String(...)` and interpolation take a data class when they take all of its field types. It prints as **`Money(amount: 5, currency: "EUR")`**: the name, then every field with its name and value, in order, with `String`s quoted as inside a list. **(0.3 choice)** `private` fields are printed too.
- A data class can **`use` interfaces and `Error`**, but not other classes. `data NotFound(id: Int) use Error` is an error class. **(0.3 choice)** Its `message` is set to how it prints, `NotFound(id: 5)`, when it is created. `message` is not part of `==` or printing.
- A data class can't be a parent of anything.
- **Map keys**: a data class whose fields are all `con` and each a valid key type (`Int`, `String`, `Bool`, an enum, or such a data class) can be a map key. Others: "a map key can be an Int, String, Bool, enum or data class with only con key fields, not Wallet".
- `private` and `protected` work on long-form fields and methods, and on methods in the short form's body. `protected` means the same as `private` there, since a data class has no children.
- `static func` works in data classes.
- Not in 0.3: copy with changes (`m.with(amount: 750)`), destructuring.

## static

```
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

con t = Temperature.from_fahrenheit(212.0)
```

- `static func` belongs to the class, not an object. It is called as `Temperature.from_fahrenheit(...)`, also inside the class. **(0.3 choice)**
- It has no `this`: "`this` can't be used in a static method". Calling it on an object is an error: "`from_fahrenheit` is static: call it as `Temperature.from_fahrenheit(...)`".
- **Static methods are not inherited**: `Dog.create()` exists only if `Dog` declares it.
- A static method and an ordinary method can't have the same name.
- Only methods: there are no static fields in 0.3. Enums and interfaces can't have static methods.

## private and protected

```
class Account
{
    mut balance = 0
    private mut pin = ""

    protected func fee() -> Int
    {
        return 1
    }
}
```

- Fields and methods are **public by default**.
- **`private`**: only usable in the methods and `init`s of the class that declares it, on **any object of that class**, not only `this`: `other.pin` works inside `Account`. A child class can't use its parent's `private` members.
- **`protected`**: usable in the class that declares it and in its descendants, on any object. **(0.3 choice)**
- Outside, using one is an error: "`pin` is private to `Account`".
- `private` and `protected` go only on fields and methods (including `init` and `static func`). On a top-level function, class, enum or interface they are an error: "private is for class members in 0.3". (Namespaces come in 0.5.)
- A `private init` means only the class itself can create objects, for example in a `static func`. A child can still call a `protected init` with `super(...).init(...)`, but not a `private` one.
- A method used for an interface must be public.

## Enums with values

```
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
```

- The value type is **`Int` or `String`**. Others are an error: "an enum's values can be Int or String, not Float".
- In an enum with values, **every case has a value**, written as a literal (an `Int` may be negative). Two cases with the same value are an error. In an enum without a value type, a case can't have one.
- **`r.value`** gives the value: a `String` for `Role`.
- **`Role.from(x)`** takes a value of the value type and gives a **`Role?`**: the case with that value, or `null`.
- `print(r)`, `String(r)` and interpolation still show the **case name**: `Admin`. `match`, `==` and map keys work on cases, as before.
- An enum with values can't have a method named `value` or `from`. Enums without values have neither.

## print and String()

New in the table of spec-0.1 and spec-0.2:

| Type | Printed as |
|---|---|
| data class | `Money(amount: 5, currency: "EUR")`, nested data and lists as they print inside a list: `LineItem(name: "tea", price: Money(amount: 3, currency: "EUR"), count: 2)`. With no fields: `Empty()` |

Printing a plain class or an interface value is still a type error.

## Language server

The existing features cover the new declarations:

- **hover**: `mut func deposit(amount: Int)`, `func show(a: mut Account)`, `class Dog use Animal`, `interface Shape`, `data Money(amount: Int, currency: String)`, `enum Role: String`, `Role.Admin = "admin"`. On an inherited member, hover says where it comes from: `func speak() -> String` (in `Animal`).
- **go to definition**: on `super(Animal)`, an inherited member, an overridden method's `override` (to the parent's method), and a class name in `use`.
- **completion**: after `.`, inherited members, and not the `private` and `protected` members that can't be used there. After `Cls.`, static methods.
- **rename**: renaming a method renames its overrides and the interface methods it fills in, and the other way round.

## Not in 0.3

Still for later: abstract methods and `final`, default bodies and fields in interfaces, static fields, `with` on data classes and named arguments, destructuring, read-only result types written out and `mut T` / `copy T` outside parameters, several class parents, and everything the [milestones](milestones.md) put in 0.4 and after.
