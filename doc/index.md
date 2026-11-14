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
| [Milestones](milestones.md) | Version plan, and when 0.1.0 is done |
| [Open questions](open-questions.md) | What we still need to decide |

## Examples

| File | Shows |
|---|---|
| [`core.brz`](../examples/core.brz) | Target for 0.1.0: functions, variables, control flow |
| [`projects/counter.brz`](../examples/projects/counter.brz) | Target for 0.1.0: a simple class |
| [`projects/bank.brz`](../examples/projects/bank.brz) | Target for 0.1.0: classes, enums, match, errors, null |
| [`basics.brz`](../examples/basics.brz) | Worksheet: 24 basic topics to decide |
| [`basic.brz`](../examples/basic.brz) | Classes, multiple inheritance, diamonds, `super(A)`, `static` |
| [`tour.brz`](../examples/tour.brz) | A tour of the whole language, including a small backend |
| [`errors.brz`](../examples/errors.brz) | Error handling with `else`, many examples |
| [`namespaces.brz`](../examples/namespaces.brz) | Data classes, namespaces, visibility |
| [`generics.brz`](../examples/generics.brz) | Generics with `<T>`, constraints, a generic repository (exploration) |
| [`lambdas.brz`](../examples/lambdas.brz) | Functions as values, one-expression lambdas, iterators |
| [`references.brz`](../examples/references.brz) | Passing arguments: `T`, `mut T`, `copy T` |

The examples outside `core.brz` and `projects/` also contain syntax that is not decided yet.
