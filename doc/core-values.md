# Core values

What matters most when *using* Bronze. These guide every other decision.

## 1. Good editor tooling from day one

- **Syntax highlighting** in Neovim (required) and VS Code (wanted).
  - Neovim: a **tree-sitter grammar** (also works in Helix, Zed and Emacs).
  - VS Code: a **TextMate grammar** in a small extension.
  - Both: **semantic tokens** from the LSP, so the compiler itself decides what is a type, a variable and so on.
- **LSP support**: go to definition, rename, find references, hover, completion and diagnostics.

What this requires of the compiler:
- Every syntax node keeps its exact **source location**.
- The parser **recovers from errors** and still produces a tree, because code is broken while you type.
- **Name resolution** is its own step, since rename and go-to-definition are built on it.
- The type checker **keeps going after errors**.

What this requires of the language:
- The syntax should be **easy and unambiguous to parse**, with no constructs whose meaning depends on context.

Order of work: parser, then name resolution, then a basic LSP (go to definition), *before* running code, so Bronze has tooling from the start.

## 2. Easy to share and reuse code

- **One tool for everything**: `bronze new`, `bronze add`, `bronze run`, `bronze test`, `bronze build`, `bronze fmt`.
- A **project file** (e.g. `bronze.toml`) and a **lockfile** for reproducible builds.
- **Modules follow the folder layout** (file = module, folder = package).

## 3. Batteries included

You should be able to build real backend software with **only the standard library**. Other people's packages are a bonus, not a requirement.

For backend work this means roughly: HTTP server and client, JSON, SQL databases (Postgres, SQLite), strings and collections, time, files, env and CLI args, logging, testing, and basic crypto/UUIDs.

Writing all of this from scratch is a lot of work, so the standard library should **wrap a mature existing ecosystem where possible**. See [implementation](implementation.md).

## 4. Shell commands are easy to run

Running a shell command is a single function call that takes the command as a string:

```
con hash, err = shell("git rev-parse HEAD")
```
