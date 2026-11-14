# Bronze docs

Everything decided about Bronze so far.

| Page | Contents |
|---|---|
| [Vision](vision.md) | What Bronze is for, the big decisions, guiding principles |
| [Core values](core-values.md) | Tooling, packaging, batteries included, shell commands |
| [Syntax](syntax.md) | Functions, variables, operators, control flow, match, tests, panics |
| [Types](types.md) | Numbers, bytes, strings, collections, enums |
| [Classes](classes.md) | Classes, inheritance, interfaces, constructors, data classes |
| [Namespaces and visibility](namespaces.md) | Grouping code, `public` / `private` / `protected` |
| [Errors and null](errors.md) | Errors as values with `else`, the Go form, null safety |
| [Projects](projects.md) | Folder layout, imports |
| [Implementation](implementation.md) | Go, tree-walking interpreter, architecture |
| [Spec 0.1](spec-0.1.md) | The exact language version 0.1.0 implements |
| [Spec 0.2](spec-0.2.md) | The changes for 0.2.0: `T?` for all types, lists, maps, `for`, interpolation |
| [Milestones](milestones.md) | Version plan, and when 0.1.0 and 0.2.0 are done |
| [Open questions](open-questions.md) | What we still need to decide |

## Examples

Programs that run with `bronze run`. Each has its expected output at the bottom.

| File | Shows |
|---|---|
| [`core.brz`](../examples/core.brz) | Functions, variables, control flow |
| [`projects/counter.brz`](../examples/projects/counter.brz) | A simple class |
| [`projects/bank.brz`](../examples/projects/bank.brz) | Classes, enums, match, errors, null |

## Explorations

Design files in [`explorations/`](explorations/): options side by side, worksheets with `// CHOICE:` lines, and tours of the language as designed. They use features and syntax that aren't implemented, or aren't decided, so they don't run.

| File | Shows |
|---|---|
| [`worksheet-0.2.brz`](explorations/worksheet-0.2.brz) | Worksheet: the choices for 0.2.0 (answered; see spec-0.2) |
| [`basics.brz`](explorations/basics.brz) | Worksheet: 24 basic topics to decide |
| [`basic.brz`](explorations/basic.brz) | Classes, multiple inheritance, diamonds, `super(A)`, `static` |
| [`tour.brz`](explorations/tour.brz) | A tour of the whole language, including a small backend |
| [`errors.brz`](explorations/errors.brz) | Error handling with `else`, many examples |
| [`namespaces.brz`](explorations/namespaces.brz) | Data classes, namespaces, visibility |
| [`generics.brz`](explorations/generics.brz) | Generics with `<T>`, constraints, a generic repository (exploration) |
| [`lambdas.brz`](explorations/lambdas.brz) | Functions as values, one-expression lambdas, iterators |
| [`references.brz`](explorations/references.brz) | Passing arguments: `T`, `mut T`, `copy T` |

