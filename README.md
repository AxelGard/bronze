# bronze

Bronze is a statically typed, garbage-collected programming language for backend work, built to be enjoyable to program in. Files end in `.brz`.

```
func main() -> Int
{
    print("hello, world")
    return 0
}
```

It is early days: the language design lives in [`doc/`](doc/index.md), examples in [`examples/`](examples/), and the implementation (Go, a tree-walking interpreter) is working towards [version 0.1.0](doc/milestones.md).

```
make build              # builds bin/bronze
make test               # go test ./...
bin/bronze run examples/core.brz
```
