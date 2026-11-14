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
| `cmd/bronze/` | CLI entry point; the logic is in `internal/cli` |
| `internal/{diag,lexer,ast,parser,check,interp}/` | The pipeline |
| `internal/cli/` | `run`, `check`, `version`; `cli.Main` is also what the golden tests call |
| `tests/run/`, `tests/errors/`, `tests/panics/` | Golden tests; see `tests/README.md` |
| `editors/tree-sitter-bronze/` | Tree-sitter grammar and highlight queries for Neovim; see its README |
| `std/algo.bz` | Old leftover from before the redesign; ignore |

## Current state

All 12 steps of the 0.1.0 work order in `doc/milestones.md` are done: `core.brz`, `counter.brz` and `bank.brz` run, every golden test passes (the `pending` map in `tests/golden_test.go` is empty; use it again for tests of features not built yet), and the tree-sitter grammar is in `editors/tree-sitter-bronze/` (`make tree-sitter` to test it, `make nvim-install` to install it for Neovim). Open: exit codes outside 0–255 (`doc/open-questions.md`). Next is 0.2.0.

## Commands

```
make build       # go build -o bin/bronze ./cmd/bronze
make test        # go test ./...
make run-core    # build, then run examples/core.brz
bin/bronze run examples/projects/bank.brz
make fmt vet
make tree-sitter  # generate and test the tree-sitter grammar (needs the tree-sitter CLI)
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
