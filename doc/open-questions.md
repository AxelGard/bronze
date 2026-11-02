# Open questions

What we still need to decide. When something is settled, it moves into the other docs and is removed here.

## Implementation
- Compiled or interpreted? Written in Go? (See [implementation](implementation.md).)
- Do users need Go installed, or does `bronze` bundle or download it?

## Syntax
- Can block comments be nested?
- Are `if` and `match` expressions (do they produce values)?
- Naming conventions, and how strictly they are enforced
- `for` loops: `for i in 1..=n`, `for x in list`?
- String interpolation: `"{name}"`?
- Lambdas: `(n) => n * n`?
- Creating objects: `Dog("Rex")`, or `new Dog("Rex")`?

## Functions
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
- Namespace = folder, or free?
- Import syntax: `import billing`, `import shop.catalog as cat`, `import billing.{Invoice}`?
- `private` on a top-level function outside any namespace: file or folder?

## Errors
- Return type syntax: `-> User ! DbError` or `-> User, DbError`?
- Return one thing (`return NotFound(id)`) or Go-style two (`return null, NotFound(id)`)?
- Should `Error` be a built-in base class, an interface, or its own `error` declaration? Or are errors simply data classes that `use Error`?
- Checking the kind: `if err is NotFound`?
- Should the compiler refuse unchecked errors, or only warn?
- An optional shortcut for "if error, return it"?

## Other areas
- **Type inference and generics**: how much do you write? Bounded generics?
- **Mutability**: does `con` freeze only the variable, or the object too? Mutable vs immutable collections?
- **Concurrency**: green threads (goroutines) plus structured concurrency? Async/await? Actors?
- **Shell commands**: return type, non-zero exit codes, safe interpolation, which shell
- **Standard library**: how big, and can Bronze call Go packages directly?
- **Packaging**: git URLs or a registry? TOML or a Bronze project file? Versioning?
