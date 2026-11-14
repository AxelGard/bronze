.PHONY: build test run-core fmt vet install tree-sitter

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
