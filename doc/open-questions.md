# Open questions

What we still need to decide. When something is settled, it moves into the other docs and is removed here.

## Implementation
- Later: transpile to Go for native binaries? Then: do users need Go installed, or does `bronze` bundle or download it?
- Can block comments nest? (0.1.0 says no.)
- Shadowing: is declaring a name that exists in an outer scope allowed? (0.1.0 says no.)
- `Int(Float)`: how does it round?

## Syntax
- Naming conventions, and how strictly they are enforced
- Creating objects: `Dog("Rex")`, or `new Dog("Rex")`?

## Functions
- Multiple return values together with errors: `-> Int, Int ! Error`? How does `else` work then?
- Can some returned values be ignored: `con low, _ = min_max(a, b)`?
- Overloading for all functions, or only `init`? What about clashes with default parameters?
- Named arguments: `connect(host: "db", timeout: 5)`?
- Static fields? Can static methods be inherited?

## Classes
- Plain `super.method()` without naming a parent?
- The return type on `init`: required, optional, or not allowed?
- Abstract methods? `final`?
- Interfaces: declaration syntax, default method bodies, fields?
- Can `super(A)` reach a grandparent (e.g. a shared diamond base)?

## Data classes
- Copy with changes: `a.with(amount: 750)`?
- Destructuring: `con Money(amount, cur) = a`?
- Can a data class `use` other classes, or only interfaces?
- Validation in constructors (`Email` must contain `@`)?
- Mutable data classes as map keys?
- Mixing short and long form?

## Namespaces
- Block form `namespace x { }`, file form (`namespace x` at the top), or both?
- `private` on a top-level function outside any namespace: file or folder?

## Errors
- Can an `else (e) { }` block produce a fallback value, or must it always leave?
- Is `-> ! Error` the right way to write "returns nothing, can fail"?
- Checking the kind: `if e is NotFound`?

## Generics
Explored in [`generics.brz`](../examples/generics.brz).
- `<T>` or `[T]`?
- Constraints with `use`: `<T use Comparable<T>>`?
- Checked once against constraints (Rust, Swift, C#, Go), or C++-style templates?
- Explicit type arguments in calls: `parse<Int>("42")`?
- Static methods on interfaces and type parameters (`T.table()`)?
- Default type parameters?

## Lambdas and iterators
Explored in [`lambdas.brz`](../examples/lambdas.brz).
- Is matching a signature both automatic and explicit (`use Handler`) the right call, or should one go?
- Parameter names in signatures: `type Handler = func(req: http.Request) -> ...`?
- Writing iterators: generators with `yield`, an `Iterator` class with `next()`, or both?
- Chains: eager, lazy, or lazy on `Iter` and eager on `List`?
- Names: `where` or `filter`, `fold` or `reduce`?
- Ranges `0..10` / `0..=10`, and `for i, x in list`?

## Passing arguments
Trying out: everything shared, with the mode on the type: `a: T` (read only), `a: mut T`, `a: copy T`, plus `mut func` for methods that change `this`. No `ref`: functions return new values instead. See [`references.brz`](../examples/references.brz).
- Mark `mut` at the call site too?
- Can you write `x, y = y, x` directly (without a function)?
- Does `con` on a local variable also freeze the object?
- Can `mut T` / `copy T` be used for fields and return types?
- Is `copy` shallow or deep?
- Objects like DB connections: are their methods `mut`?

## Other areas
- **Mutability**: does `con` freeze only the variable, or the object too? Mutable vs immutable collections?
- **Concurrency**: green threads (goroutines) plus structured concurrency? Async/await? Actors?
- **Shell commands**: return type, non-zero exit codes, safe interpolation, which shell
- **Standard library**: how big, and can Bronze call Go packages directly?
- **Packaging**: git URLs or a registry? TOML or a Bronze project file? Versioning?
