# Implementation

## Decided

- Bronze is implemented in **Go**, in this repository.
- It starts as a **tree-walking interpreter**: the program is parsed, checked, and then the syntax tree is run directly. No bytecode, no VM.
- Bronze uses Go's garbage collector. Bronze values are Go values.
- Later, if wanted: **transpile to Go** for native single-file binaries. The front end (lexer, parser, checker) is shared, so this stays cheap.

## Architecture

```
source.brz
  -> lexer    tokens, each with a position
  -> parser   ast, with error recovery
  -> check    names and types; all errors reported before running
  -> interp   runs the checked ast
```

| Path | Responsibility |
|---|---|
| `cmd/bronze/` | the CLI: `run`, `check`, `version` |
| `internal/lexer/` | text to tokens; emits `NEWLINE` tokens by the rules in [spec-0.1](spec-0.1.md) |
| `internal/ast/` | syntax tree node types, each with a source span |
| `internal/parser/` | recursive descent + precedence climbing; recovers from errors |
| `internal/check/` | scopes, name resolution, types; annotates the ast |
| `internal/interp/` | tree-walking interpreter, panics with Bronze stack traces |
| `internal/lsp/` | the language server (`bronze lsp`): diagnostics, go to definition, hover, rename. Uses the checker's symbols and references (`check.Info.Refs`) |
| `internal/diag/` | positions, spans, diagnostics and their terminal formatting |
| `tests/run/`, `tests/errors/` | golden tests (see [tests/README.md](../tests/README.md)) |
| `editors/tree-sitter-bronze/` | tree-sitter grammar and highlight queries |
| `editors/nvim/` | Neovim plugin: filetype, parser registration with nvim-treesitter, `bronze lsp` |

## Principles

- **Every node has a source span.** The LSP (from 0.2) and good error messages depend on it.
- **Never stop at the first error.** The parser recovers (skips to the next statement or `}`), and the checker keeps going. Error recovery shouldn't produce follow-up noise: an expression with an error gets an "error type" that is silently accepted everywhere.
- **The checker is the only gatekeeper.** The interpreter assumes a checked program. If the interpreter hits a type problem, that is a bug in the checker (Go `panic`, not a Bronze panic).
- **Bronze panics are not Go panics.** A Bronze runtime failure (division by zero) is reported as a Bronze panic with a Bronze stack trace and exit code 101. Internal Go panics mean a bug in `bronze`.
- **Standard Go only** for 0.1.0: no third-party dependencies.
- Code is formatted with `gofmt` and checked with `go vet`.

## Interpreter notes

- Values: a small Go type per Bronze type (`int64`, `float64`, `bool`, `string`), wrapped in a `Value` interface or tagged struct.
- Environments: one per block scope, linked to the parent. The checker could resolve each variable to (depth, index) so the interpreter doesn't need map lookups, but maps are fine for 0.1.
- `return` is implemented by unwinding: statement execution returns a "control" result (normal / return-with-value) rather than using Go panics.
- The exception is a `return` inside an error or null handler (`f() else return`, `else (e) { ... }`). That can sit deep inside an expression, so it unwinds with a Go `panic` of a private signal value, recovered by the function call it returns from.
- A call to a function that can fail returns its error wrapped in a `*Failure`. Only an `else` unwraps it; the checker makes sure no other code sees one.
- The call stack is tracked explicitly (function name + call position) so a Bronze panic can print a Bronze stack trace.
- The program runs in its own goroutine. A Bronze panic records the trace and stops that goroutine with `runtime.Goexit`, so it doesn't unwind through every call as a Go panic.
- Built-in declarations that can be written in Bronze, like the `Error` class, are in a small prelude (`internal/parser/prelude.go`) that the checker and the interpreter add to every program.

## Alternatives considered

| Option | Why not (for now) |
|---|---|
| LLVM | Native and fast, and can use C libraries through the C ABI. But it needs its own GC and concurrency runtime, and multiple inheritance with shared diamonds means C++-style virtual inheritance. Months of work before talking to Postgres. |
| Transpile to C | The same C interop as LLVM with less work, but still its own GC and standard library. |
| JVM / .NET | Ruled out: no VM. |

## How Bronze maps to Go (if transpiling later)

| Bronze | Go |
|---|---|
| Classes, methods | Structs, methods |
| Inheritance, `override`, last wins | Generated dispatch: one Go interface per class, embedded parents, forwarding methods for conflicts |
| Overloading, default parameters | Different generated names, defaults filled in at the call site |
| `con x, err = f()` / `if err != null` | `x, err := f()` / `if err != nil` |
| Source locations | `//line main.brz:12` directives, so Go stack traces point at Bronze lines |
