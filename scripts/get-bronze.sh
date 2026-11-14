#!/bin/sh
# Downloads a released bronze binary for macOS or Linux and installs it:
#
#   curl -fsSL https://raw.githubusercontent.com/AxelGard/bronze/master/scripts/get-bronze.sh | sh
#
# Settings (environment variables):
#   BRONZE_VERSION   a release tag like v0.1.0 (default: the latest release)
#   BRONZE_BIN       where to install (default: ~/.local/bin)
#
# (To build from a clone instead, use `make install`.)
set -eu

repo=AxelGard/bronze
version=${BRONZE_VERSION:-latest}
bindir=${BRONZE_BIN:-$HOME/.local/bin}
base=${BRONZE_BASE_URL:-https://github.com/$repo/releases} # for testing

fail() { printf 'error: %s\n' "$*" >&2; exit 1; }

case $(uname -s) in
Darwin) os=darwin ;;
Linux) os=linux ;;
*) fail "there is no bronze release for $(uname -s); build it from source (see the README)" ;;
esac
case $(uname -m) in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) fail "there is no bronze release for $(uname -m); build it from source (see the README)" ;;
esac

if [ "$version" = latest ]; then
    url=$base/latest/download
else
    url=$base/download/$version
fi
name=bronze-$os-$arch

command -v curl >/dev/null 2>&1 || fail "curl is needed to download bronze"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

printf 'downloading %s (%s)\n' "$name" "$version"
curl -fsSL "$url/$name.tar.gz" -o "$tmp/$name.tar.gz" || fail "could not download $url/$name.tar.gz"
curl -fsSL "$url/checksums.txt" -o "$tmp/checksums.txt" || fail "could not download $url/checksums.txt"

# Check the archive against the published checksum.
want=$(grep " $name.tar.gz\$" "$tmp/checksums.txt" | cut -d' ' -f1)
if command -v sha256sum >/dev/null 2>&1; then
    got=$(sha256sum "$tmp/$name.tar.gz" | cut -d' ' -f1)
else
    got=$(shasum -a 256 "$tmp/$name.tar.gz" | cut -d' ' -f1)
fi
[ -n "$want" ] && [ "$want" = "$got" ] || fail "the checksum of $name.tar.gz does not match; not installing"

tar -xzf "$tmp/$name.tar.gz" -C "$tmp"
mkdir -p "$bindir"
mv "$tmp/$name/bronze" "$bindir/bronze"
chmod +x "$bindir/bronze"
printf 'installed %s\n' "$("$bindir/bronze" version) to $bindir/bronze"

case ":$PATH:" in
*":$bindir:"*) ;;
*) printf '\n%s is not on your PATH. Add it, for example in ~/.zshrc or ~/.bashrc:\n    export PATH="%s:$PATH"\n' "$bindir" "$bindir" ;;
esac
