# Implementation

**Decided:** no VM (JVM, .NET, BEAM), garbage collected, batteries-included standard library.
**Not decided:** compiled or interpreted, and in which language. This page is the current thinking.

## Leaning: written in Go

The compiler/interpreter would be written in Go. That gives Bronze:

| Need | From Go |
|---|---|
| Garbage collector | Go's GC |
| Concurrency | Goroutines |
| Standard library | Wrap Go's: HTTP, JSON, SQL, TLS, crypto, `os/exec` for `shell` |
| Integrations | Postgres (`pgx`), Redis, AWS, ... the whole Go ecosystem |
| LSP | Go has LSP libraries, and the LSP shares code with the compiler |

### Suggested path

1. Lexer and parser with error recovery and source locations
2. Name resolution, then a minimal LSP (go to definition, rename)
3. **Tree-walking interpreter** (no VM: runs the syntax tree directly) until [`core.brz`](../exampels/core.brz) runs
4. Type checker
5. Standard library by wrapping Go
6. Later, if wanted: **transpile to Go** for native single-file binaries that can import any Go package, or emit LLVM IR

## Alternatives considered

| Option | Why not (for now) |
|---|---|
| LLVM | Native and fast, and can use C libraries (`libpq`) through the C ABI. But it needs its own GC and concurrency runtime, and multiple inheritance with shared diamonds means C++-style virtual inheritance. Months of work before talking to Postgres. |
| Transpile to C | The same C interop as LLVM with less work, but still its own GC and standard library. |
| JVM / .NET | Ruled out: no VM. |

## How Bronze maps to Go (if transpiling)

| Bronze | Go |
|---|---|
| Classes, methods | Structs, methods |
| Inheritance, `override`, last wins | Generated dispatch: one Go interface per class, embedded parents, forwarding methods for conflicts |
| Overloading, default parameters | Different generated names, defaults filled in at the call site |
| `con x, err = f()` / `if err != null` | `x, err := f()` / `if err != nil` |
| Source locations | `//line main.brz:12` directives, so Go stack traces point at Bronze lines |
