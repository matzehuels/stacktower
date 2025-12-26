.PHONY: all build build-api clean fmt lint test cover e2e e2e-test e2e-real e2e-parse blog blog-diagrams blog-showcase install-tools snapshot release api-local api-server api-worker help

BINARY := stacktower
API_BINARY := stacktower-api

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
	@go build -o bin/$(BINARY) .

build-api:
	@go build -o bin/$(API_BINARY) ./cmd/api

install:
	@go install .

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

# API server targets
api-local: build-api
	@echo "Starting API in local mode (API + Worker)..."
	@./bin/$(API_BINARY) --local --port 8080 --concurrency 2

api-server: build-api
	@echo "Starting API server only..."
	@./bin/$(API_BINARY) --port 8080

api-worker: build-api
	@echo "Starting worker only..."
	@./bin/$(API_BINARY) --worker --concurrency 4

help:
	@echo "Build targets:"
	@echo "  make              - Run checks and build"
	@echo "  make build        - Build CLI binary"
	@echo "  make build-api    - Build API binary"
	@echo ""
	@echo "Quality checks:"
	@echo "  make check        - Format, lint, test, vulncheck (same as CI)"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Run golangci-lint"
	@echo "  make test         - Run tests"
	@echo "  make cover        - Run tests with coverage"
	@echo "  make vuln         - Check for vulnerabilities"
	@echo ""
	@echo "API server:"
	@echo "  make api-local    - Run API + worker in local mode (port 8080)"
	@echo "  make api-server   - Run API server only (port 8080)"
	@echo "  make api-worker   - Run worker only"
	@echo ""
	@echo "End-to-end tests:"
	@echo "  make e2e          - Run all end-to-end tests"
	@echo "  make e2e-test     - Render examples/test/*.json"
	@echo "  make e2e-real     - Render examples/real/*.json"
	@echo "  make e2e-parse    - Parse packages to examples/real/"
	@echo ""
	@echo "Blog content:"
	@echo "  make blog         - Generate all blogpost diagrams"
	@echo "  make blog-diagrams - Generate blogpost example diagrams"
	@echo "  make blog-showcase - Generate blogpost showcase diagrams"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean        - Remove build artifacts"
