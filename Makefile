BINARY      := bbx
PKG         := github.com/rahadiangg/bbx
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE        := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -s -w \
               -X $(PKG)/internal/version.Version=$(VERSION) \
               -X $(PKG)/internal/version.Commit=$(COMMIT) \
               -X $(PKG)/internal/version.Date=$(DATE)

.PHONY: build install test lint tidy vet clean run docs

build:
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY) ./cmd/bbx

install:
	go install -trimpath -ldflags '$(LDFLAGS)' ./cmd/bbx

test:
	go test ./...

vet:
	go vet ./...

lint: vet
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed; ran go vet only"

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)

run: build
	./$(BINARY) $(ARGS)

docs: build
	./$(BINARY) docs gen --output docs/COMMANDS.md || true
