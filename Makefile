.PHONY: build test run-core fmt vet install tree-sitter dist

build:
	go build -o bin/bronze ./cmd/bronze

test:
	go test ./...

run-core: build
	./bin/bronze run examples/core.brz

fmt:
	gofmt -w cmd internal

vet:
	go vet ./...

# Install bronze into ~/.local/bin, and the tree-sitter parser into Neovim.
install:
	scripts/install.sh

# The tree-sitter grammar (editors/tree-sitter-bronze); needs the tree-sitter
# CLI. Every valid example program must parse without errors.
TS_DIR = editors/tree-sitter-bronze
TS_VALID = examples/core.brz examples/projects/counter.brz examples/projects/bank.brz $(wildcard tests/run/*.brz tests/panics/*.brz)

tree-sitter:
	cd $(TS_DIR) && tree-sitter generate && tree-sitter test
	tree-sitter parse -p $(TS_DIR) -q $(TS_VALID)

# Release archives for macOS and Linux, into dist/. The release workflow
# (.github/workflows/release.yml) runs this with VERSION set to the tag.
VERSION ?= $(shell git describe --tags --always --dirty)
PLATFORMS = darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

dist:
	rm -rf dist
	mkdir -p dist
	for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; name=bronze-$$os-$$arch; \
		mkdir -p dist/$$name; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath \
			-ldflags "-s -w -X github.com/AxelGard/bronze/internal/cli.Version=$(patsubst v%,%,$(VERSION))" \
			-o dist/$$name/bronze ./cmd/bronze || exit 1; \
		cp LICENSE README.md dist/$$name/; \
		tar -C dist -czf dist/$$name.tar.gz $$name; \
		rm -rf dist/$$name; \
	done
	cd dist && shasum -a 256 *.tar.gz > checksums.txt
