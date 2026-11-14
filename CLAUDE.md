# Bronze

Bronze is a programming language (`.brz` files) designed in this repo by its author, for their own use and for backend work. It is statically typed, garbage collected, imperative at heart, and supports OOP, DDD and functional styles. It is implemented in **Go** as a **tree-walking interpreter**.

## Where things are

| Path | What |
|---|---|
| `doc/index.md` | Start here: map of all design docs |
| `doc/spec-0.1.md` | **The exact language for 0.1.0**: tokens, newline rules, grammar, type rules, runtime semantics, print format, CLI, diagnostics format |
| `doc/milestones.md` | Version plan, the "done when" list and work order for 0.1.0 |
| `doc/implementation.md` | Architecture and implementation principles |
| `doc/open-questions.md` | Everything not decided yet |
| `doc/*.md` (others) | Decided design: syntax, types, classes, errors, namespaces, projects, core values |
| `examples/core.brz`, `examples/projects/*.brz` | **The 0.1.0 targets**: each has its expected output at the bottom |
| `examples/*.brz` (others) | Design explorations with later features and undecided syntax, marked `[decided]` / `[proposal]` / `[exploration]` |
| `cmd/bronze/` | CLI (`run`, `check`, `version`); currently a stub |
| `internal/{diag,lexer,ast,parser,check,interp}/` | The pipeline; currently only package docs (`doc.go`) |
| `tests/run/`, `tests/errors/`, `tests/panics/` | Golden tests; see `tests/README.md` |
| `std/algo.bz` | Old leftover from before the redesign; ignore |

## Current state

Design for the basics is done. Implementation of 0.1.0 has **not started**: the Go module, package skeleton, CLI stub, Makefile and golden tests exist. Follow the work order in `doc/milestones.md`: get `core.brz` running end to end first, then add classes, enums/match, errors and null.

## Commands

```
make build       # go build -o bin/bronze ./cmd/bronze
make test        # go test ./...
make run-core    # build, then run examples/core.brz
bin/bronze run examples/projects/bank.brz
make fmt vet
```

Go 1.26, module `github.com/AxelGard/bronze`, standard library only.

## Implementation rules

- Implement exactly `doc/spec-0.1.md` for 0.1.0. If something isn't covered there, check the other docs, then ask. Don't add later features early.
- Every AST node carries a source span. The parser recovers from errors and reports several. The checker reports all errors. These matter for the LSP in 0.2.
- The interpreter assumes a checked program. Bronze runtime failures are Bronze panics (stack trace, exit code 101), never Go panics.
- Tests: unit tests per package, plus golden tests in `tests/`. Add a golden test for every new diagnostic.
- `gofmt` and `go vet` clean.

## Working with the author

- Design is explored by writing example `.brz` files with options side by side, or worksheets with `// CHOICE:` lines the author fills in. Decisions then go into `doc/*.md`, and open items into `doc/open-questions.md`.
- Docs are Markdown in `doc/`. Keep them in sync when a decision changes.
- Only commit when asked. Commit messages: a short summary line, then details if needed.
- Bronze code style in examples: braces on their own line (Allman), 4-space indent, `//` comments.
