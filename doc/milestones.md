# Milestones

Each version is a working language you can use.

| Version | Contents | Proof |
|---|---|---|
| **0.1.0** | functions, variables, control flow, simple classes, enums with methods, `match`, errors with `else`, `T?` and null | `core.brz`, `projects/counter.brz` and `projects/bank.brz` run |
| **0.2.0** | `T?` for all types, `for`, ranges, `break`/`continue`, lists, maps, string interpolation, `+=`, slicing; **LSP**: diagnostics, go to definition, rename, hover | small scripts and algorithms |
| 0.3.0 | inheritance, `override`, `super`, interfaces, data classes, `static`, `private`/`protected`, enums with values, parameter modes (`mut T`, `copy T`) | domain models |
| 0.4.0 | the Go error form `con x, err = f()`, `else continue`/`else break`, multiple return values, lambdas and function types | real error handling |
| 0.5.0 | multiple files, namespaces, imports, `bronze test` | a real project layout |
| 0.6.0 | standard library: files, `shell`, env, JSON | useful command-line tools |
| 1.0.0 | HTTP server and Postgres | **a small backend service** |

After 1.0: user-defined generics, multiple inheritance with diamonds, `yield` generators, concurrency, transpiling to Go.

## 0.1.0

The language is exactly [spec-0.1](spec-0.1.md):
- functions, recursion, `con`/`mut`, `Int`/`Float`/`Bool`/`String`, arithmetic, comparisons, `and`/`or`/`not`, `if`/`else if`/`else`, `while`
- built-ins: `print`, `Float(x)`, `String(x)`, `Error(msg)`, `panic`, `assert`
- simple classes: fields, methods, `this`, several `init`s, objects shared by reference
- enums with methods, and `match` as an expression
- errors: `use Error`, `-> T ! E`, `-> ! E`, the error `else` handlers, `is`
- `T?` for classes, enums and `Error`, `null`, narrowing, and `else` for null

### Done when

1. These programs print exactly the expected output at the bottom of each file, and exit with 0:
   - `bronze run examples/core.brz`
   - `bronze run examples/projects/counter.brz`
   - `bronze run examples/projects/bank.brz`
2. `bronze check` reports clear errors (line, column, caret) for at least: syntax errors (several per file), unknown names, wrong argument count or types, wrong return type, missing return, assigning to `con` or a parameter, mixing `Int` and `Float`, an unhandled error, using a `T?` without handling null, and a `match` that doesn't cover every case.
3. Runtime panics (integer division by zero, `panic(...)`, failed `assert`, stack overflow) print a Bronze stack trace and exit with 101.
4. `go test ./...` runs the golden tests in `tests/` and unit tests for each package, and passes.
5. Neovim highlights `.brz` files with a tree-sitter grammar in `editors/tree-sitter-bronze/`.

### Work order

Get `core.brz` working end to end first, then add the rest feature by feature:

1. `diag`: positions, spans, diagnostic formatting
2. `lexer`, with unit tests (including the newline rules)
3. `ast` + `parser` for the `core.brz` subset, with error recovery and unit tests
4. `check` for the `core.brz` subset: scopes, names, types, returns
5. `interp` for the `core.brz` subset
6. `cmd/bronze`: wire up `run` and `check`. **`core.brz` runs.**
7. Golden-test runner over `tests/run`, `tests/errors` and `tests/panics`
8. Classes (parser, checker, interpreter). **`counter.brz` runs.**
9. Enums and `match`
10. Errors and the error `else`
11. `T?`, `null`, narrowing, `else` for null. **`bank.brz` runs.**
12. tree-sitter grammar (independent, can be done anytime)

## 0.2.0

The language is exactly [spec-0.2](spec-0.2.md): spec-0.1, plus
- `T?` for all types
- ranges, `for`, `break`, `continue`
- `List<T>` and `Map<K, V>`: literals, indexing, slicing, methods, `==`, printing
- string indexing, slicing and methods
- string interpolation
- `+=`, `-=`, `*=`, `/=`, `%=`
- hover in the language server

### Done when

1. Two new programs in `examples/` print their expected output and exit with 0: `algorithms.brz` (sorting, searching, primes, with lists and ranges) and `projects/words.brz` (counting words in a text, with maps, string methods and interpolation). `core.brz`, `counter.brz` and `bank.brz` still run unchanged.
2. `bronze check` reports clear errors for at least: a `T?` used as a `T` for the new types, mixed element types in a list or map literal, `[]` or `{}` without a type, a bad map key type, `List<Int?>`, indexing a non-list, assigning to a string index, assigning to a loop variable, `break`/`continue` outside a loop, an interpolated value that can't be converted, and wrong types in compound assignment.
3. The new panics (index out of range on assignment and `insert`, a missing key with `op=`, changing a list or map while looping over it) print a stack trace and exit with 101.
4. `bronze lsp` answers hover with the type or signature of the name under the cursor.
5. The tree-sitter grammar parses every valid program, and the website documents everything new, with every example run by `tests/docs_test.go`.

### Work order

1. `+=` and friends, `break`/`continue` (with `while`, before `for` exists)
2. `T?` for `Int`, `Float`, `Bool` and `String`
3. Ranges as `List<Int>` and `for` over ranges
4. Lists: types, literals, indexing, assignment, `for` over lists, methods, `==`, printing, the "changed while looping" panic
5. Maps: the same
6. Slicing, then string indexing and methods, and `for` over strings
7. String interpolation (lexer and parser)
8. Hover in the LSP
9. The two example programs

With each step: golden tests for its diagnostics and panics, the tree-sitter grammar, and the website.
