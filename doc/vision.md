# Vision

## What Bronze is

Bronze is a programming language that is, first of all, for its author's own use, and a place to try out ideas.
The goal is a language that is **enjoyable to program in**.

- File extension: `.brz`
- Main domain: **backend work** (services, APIs, data handling, tooling)

## Big decisions

| Topic | Decision |
|---|---|
| Typing | **Static** types |
| Paradigm | **Imperative at heart**, but it should support OOP, DDD-style modelling and a functional style equally well. The language should not force one style. |
| Memory | **Garbage collected** |
| Execution | **No VM** like the JVM, .NET or BEAM. Implemented in **Go** as a **tree-walking interpreter**, see [implementation](implementation.md). |
| Performance | Not a priority right now. Exploring ideas comes first. |

## Guiding principles

1. **Explore before committing.** Ideas are written down and argued about before they are implemented.
2. **Enjoyment over cleverness.** If a feature is powerful but unpleasant to use, it doesn't belong.
3. **Backend-shaped.** When choosing between two designs, prefer the one that makes typical backend code (HTTP handlers, domain models, DB access, error handling) nicer.

See also [core values](core-values.md).
