# Bronze

Bronze is a programming language (`.brz` files) designed in this repo by its author, for their own use and for backend work. It is statically typed, garbage collected, imperative at heart, and supports OOP, DDD and functional styles. It is implemented in **Go** as a **tree-walking interpreter**.

## Where things are

| Path | What |
|---|---|
| `doc/index.md` | Start here: map of all design docs |
| `doc/spec-0.1.md` | **The exact language for 0.1.0**: tokens, newline rules, grammar, type rules, runtime semantics, print format, CLI, diagnostics format |
| `doc/spec-0.2.md` | **The changes for 0.2.0**, on top of spec-0.1 |
| `doc/milestones.md` | Version plan, the "done when" list and work order for 0.1.0 |
| `doc/implementation.md` | Architecture and implementation principles |
| `doc/open-questions.md` | Everything not decided yet |
| `doc/*.md` (others) | Decided design: syntax, types, classes, errors, namespaces, projects, core values |
| `examples/` | Programs that run (`core.brz`, `algorithms.brz`, `projects/`); each has its expected output at the bottom, and a test checks it (`tests/golden_test.go`). Only programs that run go here |
| `doc/explorations/*.brz` | Design explorations and worksheets with later features and undecided syntax, marked `[decided]` / `[proposal]` / `[exploration]` / `[worksheet]` |
| `cmd/bronze/` | CLI entry point; the logic is in `internal/cli` |
| `internal/{diag,lexer,ast,parser,check,interp}/` | The pipeline |
| `internal/cli/` | `run`, `check`, `lsp`, `version`; `cli.Main` is also what the golden tests call |
| `internal/lsp/` | Language server: diagnostics, go to definition, hover, completion, rename |
| `tests/run/`, `tests/errors/`, `tests/panics/` | Golden tests; see `tests/README.md` |
| `website/` | User docs (MkDocs Material), published to GitHub Pages by `.github/workflows/docs.yml`. **Only what is implemented and working.** Every ```` ```bronze ```` block is run by `tests/docs_test.go`, and an ```` ```text title="Output" ```` block right after it must match exactly (stdout then stderr, file named `main.brz`). Update it when a feature lands |
| `editors/tree-sitter-bronze/` | Tree-sitter grammar and highlight queries; see its README |
| `editors/nvim/` | Neovim plugin: filetype, tree-sitter registration, `bronze lsp` |
| `scripts/install.sh` | Installs `bronze` and the Neovim parser (`make install`) |
| `std/algo.bz` | Old leftover from before the redesign; ignore |

## Current state

0.1.0 is complete: all 12 steps of the work order in `doc/milestones.md`. `core.brz`, `counter.brz` and `bank.brz` run, every golden test passes (the `pending` map in `tests/golden_test.go` is empty; use it again for tests of features not built yet), and the tree-sitter grammar is in `editors/tree-sitter-bronze/` (it stays in this repo so it changes with the language; the author decided against a separate repo for now). 0.2.0 is implemented too: all 9 steps of its work order in `doc/milestones.md`, as `doc/spec-0.2.md` describes (compound assignment, `break`/`continue`, `T?` for all types, ranges and `for`, lists, maps, slicing, string methods, interpolation, LSP hover). `examples/algorithms.brz` and `examples/projects/words.brz` run, and the website documents it. Tagged `v0.2.0`. Next: 0.3, whose choices are in `doc/explorations/worksheet-0.3.brz` for the author to fill in; then write `doc/spec-0.3.md` and a work order. Open: exit codes outside 0–255 (`doc/open-questions.md`).

## Commands

```
make build       # go build -o bin/bronze ./cmd/bronze
make test        # go test ./...
make run-core    # build, then run examples/core.brz
bin/bronze run examples/projects/bank.brz
make fmt vet
make tree-sitter  # generate and test the tree-sitter grammar (needs the tree-sitter CLI)
make install      # bronze into ~/.local/bin, and the parser into Neovim (scripts/install.sh)
make dist VERSION=v0.1.0   # release archives for macOS and Linux into dist/
cd website && pip install -r requirements.txt && mkdocs serve   # the website, locally
```

CI (`.github/workflows/ci.yml`) runs gofmt, vet, `go test` and `make tree-sitter` on every push to master and every pull request, and fails if `editors/tree-sitter-bronze/src` doesn't match `grammar.js` (regenerate with tree-sitter CLI 0.27.2). Releases: pushing a tag `vX.Y.Z` runs `.github/workflows/release.yml`, which tests, runs `make dist`, and publishes a GitHub Release. `scripts/get-bronze.sh` downloads from `releases/latest`, so only tags with a suffix (`v0.2.0-rc1`) are prereleases. Release builds set `cli.Version` from the tag.

Go 1.26, module `github.com/AxelGard/bronze`, standard library only.

## Implementation rules

- Implement exactly `doc/spec-0.1.md` for 0.1.0, and `doc/spec-0.2.md` on top of it for 0.2.0. If something isn't covered there, check the other docs, then ask. Don't add later features early.
- Every AST node carries a source span. The parser recovers from errors and reports several. The checker reports all errors. These matter for the LSP in 0.2.
- The interpreter assumes a checked program. Bronze runtime failures are Bronze panics (stack trace, exit code 101), never Go panics.
- Tests: unit tests per package, plus golden tests in `tests/`. Add a golden test for every new diagnostic.
- `gofmt` and `go vet` clean.

## Working with the author

- Design is explored by writing `.brz` files in `doc/explorations/` with options side by side, or worksheets with `// CHOICE:` lines the author fills in. Decisions then go into `doc/*.md`, and open items into `doc/open-questions.md`.
- Docs are Markdown in `doc/`. Keep them in sync when a decision changes.
- Only commit when asked. Commit messages: a short summary line, then details if needed.
- Bronze code style in examples: braces on their own line (Allman), 4-space indent, `//` comments.
