# A first program

Save this as `hello.brz`:

```bronze
func main() -> Int
{
    print("hello, world")
    return 0
}
```

and run it:

```sh
bronze run hello.brz
```

```text title="Output"
hello, world
```

Every program has a `main` function that takes no parameters and returns an `Int`. That number is the program's exit code, so `return 0` means success.

A program is one `.brz` file. At the top level it holds functions, [classes](classes.md) and [enums](enums-and-match.md), in any order: everything can be used before it is declared.

## Checking

`bronze run` checks the whole program before it runs any of it. `bronze check` only checks:

```sh
bronze check hello.brz
```

It prints nothing when the program is fine. Otherwise it lists every error it found, each with the line and column:

```bronze
func main() -> Int
{
    con total = 7 / 2.0
    print(totl)
    return 0
}
```

```text title="Output"
error: cannot divide Int by Float
  --> main.brz:3:17
  |
3 |     con total = 7 / 2.0
  |                 ^^^^^^^ Int / Float
  |
  = help: convert explicitly: Float(7) / 2.0

error: unknown name `totl`
  --> main.brz:4:11
  |
4 |     print(totl)
  |           ^^^^
```

A program with errors never starts. The exit code is then `1`. See [the bronze command](../reference/cli.md) for all exit codes.

## Style

Bronze code is written with braces on their own line, 4 spaces of indentation, and `snake_case` names for functions and variables. Type names start with a capital letter: `Int`, `Account`.

Comments are `// to the end of the line` or `/* a block */`. Block comments don't nest.

Statements end at the end of the line; there are no semicolons. A line continues onto the next one when it ends in an operator, a `,`, `=`, `->` or `.`, or inside parentheses:

```bronze
con total = 1 +
    2 +
    3
print(total)
print(
    total * 2
)
```

## Next

[Values and variables](values.md).
