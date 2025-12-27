.PHONY: all build build-server clean fmt lint test cover e2e e2e-test e2e-real e2e-parse blog blog-diagrams blog-showcase install-tools snapshot release server-local server-api server-worker help

BINARY := stacktower
SERVER_BINARY := stacktowerd

all: check build

check: fmt lint test vuln

fmt:
	@gofmt -s -w .
	@goimports -w -local stacktower .

lint:
	@golangci-lint run

test:
	@go test -race -timeout=2m ./...

cover:
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

build:
	@go build -ldflags "-X github.com/matzehuels/stacktower/pkg/buildinfo.Version=$(shell git describe --tags --always --dirty) -X github.com/matzehuels/stacktower/pkg/buildinfo.Commit=$(shell git rev-parse --short HEAD) -X github.com/matzehuels/stacktower/pkg/buildinfo.Date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/$(BINARY) ./cmd/stacktower

build-server:
	@go build -ldflags "-X github.com/matzehuels/stacktower/pkg/buildinfo.Version=$(shell git describe --tags --always --dirty) -X github.com/matzehuels/stacktower/pkg/buildinfo.Commit=$(shell git rev-parse --short HEAD) -X github.com/matzehuels/stacktower/pkg/buildinfo.Date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/$(SERVER_BINARY) ./cmd/server

install:
	@go install -ldflags "-X github.com/matzehuels/stacktower/pkg/buildinfo.Version=$(shell git describe --tags --always --dirty) -X github.com/matzehuels/stacktower/pkg/buildinfo.Commit=$(shell git rev-parse --short HEAD) -X github.com/matzehuels/stacktower/pkg/buildinfo.Date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" ./cmd/stacktower

e2e: build
	@./scripts/test_e2e.sh all

e2e-test: build
	@./scripts/test_e2e.sh test

e2e-real: build
	@./scripts/test_e2e.sh real

e2e-parse: build
	@./scripts/test_e2e.sh parse

blog: blog-diagrams blog-showcase

blog-diagrams: build
	@./scripts/blog_diagrams.sh

blog-showcase: build
	@./scripts/blog_showcase.sh

install-tools:
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest

vuln:
	@govulncheck ./...

snapshot:
	@goreleaser release --snapshot --clean --skip=publish

release:
	@goreleaser release --clean

clean:
	@rm -rf bin/ dist/ coverage.out data/

# Server targets
server-local: build-server
	@echo "Starting server in local mode (API + Worker)..."
	@./bin/$(SERVER_BINARY) --local --port 8080 --concurrency 2

server-api: build-server
	@echo "Starting API server only..."
	@./bin/$(SERVER_BINARY) --port 8080

server-worker: build-server
	@echo "Starting worker only..."
	@./bin/$(SERVER_BINARY) --worker --concurrency 4

help:
	@echo "Build targets:"
	@echo "  make              - Run checks and build"
	@echo "  make build        - Build CLI binary (bin/stacktower)"
	@echo "  make build-server - Build daemon binary (bin/stacktowerd)"
	@echo ""
	@echo "Quality checks:"
	@echo "  make check        - Format, lint, test, vulncheck (same as CI)"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Run golangci-lint"
	@echo "  make test         - Run tests"
	@echo "  make cover        - Run tests with coverage"
	@echo "  make vuln         - Check for vulnerabilities"
	@echo ""
	@echo "Server modes:"
	@echo "  make server-local   - Run API + worker in local mode"
	@echo "  make server-api     - Run API server only"
	@echo "  make server-worker  - Run worker only"
	@echo ""
	@echo "End-to-end tests:"
	@echo "  make e2e          - Run all end-to-end tests"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean        - Remove build artifacts"
