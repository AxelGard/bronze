# bronze

Bronze is a statically typed, garbage-collected programming language for backend work, built to be enjoyable to program in. Files end in `.brz`.

```
func main() -> Int
{
    print("hello, world")
    return 0
}
```

It is early days: the language design lives in [`doc/`](doc/index.md), examples in [`examples/`](examples/), and the implementation is a tree-walking interpreter written in Go. See the [milestones](doc/milestones.md) for what each version adds.

## Install

On macOS or Linux, download the latest release into `~/.local/bin`:

```
curl -fsSL https://raw.githubusercontent.com/AxelGard/bronze/master/scripts/get-bronze.sh | sh
```

Set `BRONZE_VERSION=v0.1.0` for a specific version, or `BRONZE_BIN=...` for another directory. The archives are also on the [releases page](https://github.com/AxelGard/bronze/releases). On macOS, a binary downloaded with a browser is blocked until you run `xattr -d com.apple.quarantine bronze`; the script doesn't need that.

With Go 1.21 or later, you can also build it yourself:

```
go install github.com/AxelGard/bronze/cmd/bronze@latest
```

Then try it:

```
bronze run examples/projects/bank.brz
bronze check my_program.brz
```

For editor support (highlighting and the language server), see [editors/nvim](editors/nvim/README.md).

## Working on Bronze

```
make build              # builds bin/bronze
make test               # go test ./...
make install            # bronze into ~/.local/bin, and the Neovim parser
bin/bronze run examples/core.brz
```
