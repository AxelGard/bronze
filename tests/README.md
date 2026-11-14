# Tests

Golden tests, run by `go test ./...` (the runner is `golden_test.go`). Tests that need features not implemented yet are listed in its `pending` map and skipped. File names in messages are compared as the base name (`bank.brz`, not a full path).

| Folder | Files | The test passes when |
|---|---|---|
| `run/` | `NAME.brz`, `NAME.out` | the program checks, runs, prints exactly `NAME.out` to stdout, and exits with 0 |
| `errors/` | `NAME.brz`, `NAME.err` | `bronze check` exits with 1, and its output **starts with** the lines in `NAME.err` (the error message and its `-->` location) |
| `panics/` | `NAME.brz`, `NAME.out`, `NAME.err` | the program prints exactly `NAME.out` to stdout, exits with 101, and stderr **starts with** `NAME.err` |

Only the start of a diagnostic is compared, so the rest (source excerpt, help, stack trace) can be improved freely.

`run/core.brz`, `run/counter.brz` and `run/bank.brz` are copies of the example programs in `examples/` that 0.1.0 must run. Keep them in sync.
