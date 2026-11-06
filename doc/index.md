# Bronze docs

Everything decided about Bronze so far.

| Page | Contents |
|---|---|
| [Vision](vision.md) | What Bronze is for, the big decisions, guiding principles |
| [Core values](core-values.md) | Tooling, packaging, batteries included, shell commands |
| [Syntax](syntax.md) | Functions, variables, types, comments, control flow, entry point |
| [Classes](classes.md) | Classes, inheritance, interfaces, constructors, data classes |
| [Namespaces and visibility](namespaces.md) | Grouping code, `public` / `private` / `protected` |
| [Errors and null](errors.md) | Errors as values with `else`, the Go form, null safety |
| [Implementation](implementation.md) | How Bronze will run: current thinking (not decided) |
| [Milestones](milestones.md) | What each compiler milestone must be able to run |
| [Open questions](open-questions.md) | What we still need to decide |

## Examples

| File | Shows |
|---|---|
| [`core.brz`](../examples/core.brz) | The target for the first compiler. Uses only decided syntax. |
| [`basic.brz`](../examples/basic.brz) | Classes, multiple inheritance, diamonds, `super(A)`, `static` |
| [`tour.brz`](../examples/tour.brz) | A tour of the whole language, including a small backend |
| [`errors.brz`](../examples/errors.brz) | Error handling with `else`, many examples |
| [`namespaces.brz`](../examples/namespaces.brz) | Data classes, namespaces, visibility |
| [`lambdas.brz`](../examples/lambdas.brz) | Functions as values, one-expression lambdas, iterators |
| [`references.brz`](../examples/references.brz) | Passing arguments: `T`, `mut T`, `copy T` |

The examples outside `core.brz` also contain syntax that is not decided yet.
