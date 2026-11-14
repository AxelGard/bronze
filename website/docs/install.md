# Install

## A release

On macOS or Linux (Intel or ARM), this downloads the latest release, checks it against the published checksum, and installs `bronze` into `~/.local/bin`:

```sh
curl -fsSL https://raw.githubusercontent.com/AxelGard/bronze/master/scripts/get-bronze.sh | sh
```

Two environment variables change what it does:

| Variable | Default | Meaning |
|---|---|---|
| `BRONZE_VERSION` | the latest release | a release tag, like `v0.1.0` |
| `BRONZE_BIN` | `~/.local/bin` | where to put `bronze` |

```sh
curl -fsSL https://raw.githubusercontent.com/AxelGard/bronze/master/scripts/get-bronze.sh | BRONZE_VERSION=v0.1.0 sh
```

Make sure the directory is on your `PATH`, then check:

```sh
bronze version
```

The archives are also on the [releases page](https://github.com/AxelGard/bronze/releases).

!!! note "macOS"
    A binary downloaded with a browser is blocked by Gatekeeper until you run `xattr -d com.apple.quarantine bronze`. The install script doesn't need that.

## With Go

With Go installed:

```sh
go install github.com/AxelGard/bronze/cmd/bronze@latest
```

## From a clone

```sh
git clone https://github.com/AxelGard/bronze
cd bronze
make build        # bin/bronze
make install      # bronze into ~/.local/bin, and the tree-sitter parser into Neovim
```

`make install` is mostly for working on Bronze itself; see [Editors](reference/editors.md) for what it sets up.

## Next

Write [a first program](guide/first-program.md).
