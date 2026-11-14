# Bronze

Bronze is a statically typed, garbage-collected programming language for backend work, built to be enjoyable to program in. It is imperative at heart, with classes, enums, `match`, and errors that the type checker makes you handle. Files end in `.brz`.

```bronze
enum Status
{
    Active
    Frozen
}

class Account
{
    con owner = ""
    mut balance = 0
    mut status = Status.Active

    func init(owner: String)
    {
        this.owner = owner
    }

    func withdraw(amount: Int) -> Int ! Error
    {
        if this.status == Status.Frozen
        {
            return Error(this.owner + "'s account is frozen")
        }
        if amount > this.balance
        {
            return Error("not enough money")
        }
        this.balance = this.balance - amount
        return this.balance
    }
}

func main() -> Int
{
    con a = Account("Ada")
    a.balance = 100

    con left = a.withdraw(30) else 0
    print(left)

    a.status = Status.Frozen
    a.withdraw(10) else (e)
    {
        print(e.message)
        return 1
    }
    return 0
}
```

```text title="Output"
70
Ada's account is frozen
```

## Status

Bronze is young. Version **0.1.0** is out: one-file programs with functions, `Int`/`Float`/`Bool`/`String`, `if` and `while`, classes, enums, `match`, errors and null safety. It runs on a tree-walking interpreter written in Go, and comes with a language server and a tree-sitter grammar for editors.

Version **0.2.0** adds `for` loops and ranges, `break` and `continue`, lists, maps, string methods and interpolation, `+=`, `T?` for every type, and hover in the language server. This site describes 0.2.0, and only what works.

## Where to go next

- [Install](install.md) `bronze` on macOS or Linux.
- Read the [guide](guide/first-program.md), from a first program to errors and null.
- Look things up in the [reference](reference/language.md).
- See [the source on GitHub](https://github.com/AxelGard/bronze), including larger [example programs](https://github.com/AxelGard/bronze/tree/master/examples).
