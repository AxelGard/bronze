.PHONY: build test run-core fmt vet

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
