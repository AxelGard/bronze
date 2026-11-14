# The bronze command

```text
bronze run <file.brz>     check, then run a program
bronze check <file.brz>   check syntax and types only
bronze lsp                run the language server on stdin and stdout
bronze version            print the version
```

`bronze help` (or `-h`, `--help`) prints the same list.

## Exit codes

| Command | Exit code |
|---|---|
| `bronze run` | the value `main` returns; `1` if the program has errors; `101` if it panics |
| `bronze check` | `0` if the program has no errors, `1` if it has |
| any | `2` for a wrong command or missing argument |

`main` returns an `Int`, and its value goes to the operating system as it is. The system keeps only the low 8 bits, so values outside 0 to 255 don't come out as written: `return 256` exits with 0.

## Diagnostics

`bronze check` and `bronze run` print every error they find, to stderr, in source order:

```text
error: cannot divide Int by Float
  --> main.brz:3:17
  |
3 |     con total = 7 / 2.0
  |                 ^^^^^^^ Int / Float
  |
  = help: convert explicitly: Float(7) / 2.0
```

- The first line is `error: ` and the message.
- `-->` gives the file, line and column. Lines and columns start at 1, and columns count characters, not bytes.
- Then comes the source line with the place underlined, and sometimes `= help:` (what to do) and `= note:` (more context) lines.
- Several errors are separated by a blank line.

The parser recovers after a syntax error and the checker keeps going after a type error, so one run reports as many errors as it can find.

## Panics

A panic prints its message and a stack trace to stderr, innermost call first, and exits with 101:

```text
panic: division by zero
  at divide (main.brz:3)
  at main   (main.brz:9)
```

See [Panics](../guide/panics.md).

## bronze lsp

Runs the language server, talking the Language Server Protocol over stdin and stdout. Editors start it themselves; see [Editors](editors.md).
