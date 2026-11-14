.PHONY: build test run-core fmt vet tree-sitter nvim-install

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

# tree-sitter grammar (editors/tree-sitter-bronze); needs the tree-sitter CLI.
# On macOS, build against Xcode's SDK: an older linker can't read a newer
# Command Line Tools SDK.
TS_DIR = editors/tree-sitter-bronze
TS_ENV = $(if $(shell xcrun --sdk macosx --show-sdk-path 2>/dev/null),SDKROOT=$(shell xcrun --sdk macosx --show-sdk-path))
TS_VALID = examples/core.brz examples/projects/counter.brz examples/projects/bank.brz $(wildcard tests/run/*.brz tests/panics/*.brz)
NVIM_SITE = $(HOME)/.local/share/nvim/site

tree-sitter:
	cd $(TS_DIR) && tree-sitter generate && $(TS_ENV) tree-sitter test
	$(TS_ENV) tree-sitter parse -p $(TS_DIR) -q $(TS_VALID)

nvim-install:
	cd $(TS_DIR) && tree-sitter generate
	mkdir -p $(NVIM_SITE)/parser $(NVIM_SITE)/queries/bronze
	$(TS_ENV) tree-sitter build -o $(NVIM_SITE)/parser/bronze.so $(TS_DIR)
	cp $(TS_DIR)/queries/highlights.scm $(NVIM_SITE)/queries/bronze/highlights.scm
