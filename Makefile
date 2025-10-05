.PHONY: all build clean fmt lint test cover e2e e2e-test e2e-real e2e-parse blog blog-diagrams blog-showcase install-tools snapshot release help

BINARY := stacktower

all: check build

check: fmt lint test

fmt:
	@gofmt -s -w .
	@goimports -w -local stacktower .

lint:
	@go vet ./...
	@staticcheck ./...

test:
	@go test -race -timeout=2m ./...

cover:
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

build:
	@go build -o bin/$(BINARY) .

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
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest

vuln:
	@govulncheck ./...

snapshot:
	@goreleaser release --snapshot --clean --skip=publish

release:
	@goreleaser release --clean

clean:
	@rm -rf bin/ dist/ coverage.out

help:
	@echo "make              - Run checks and build"
	@echo "make check        - Format, lint, test"
	@echo "make fmt          - Format code"
	@echo "make lint         - Run go vet and staticcheck"
	@echo "make test         - Run tests"
	@echo "make cover        - Run tests with coverage"
	@echo "make build        - Build binary"
	@echo "make e2e          - Run all end-to-end tests"
	@echo "make e2e-test     - Render examples/test/*.json"
	@echo "make e2e-real     - Render examples/real/*.json"
	@echo "make e2e-parse    - Parse packages to examples/real/"
	@echo "make blog          - Generate all blogpost diagrams"
	@echo "make blog-diagrams - Generate blogpost example diagrams"
	@echo "make blog-showcase - Generate blogpost showcase diagrams"
	@echo "make vuln         - Check for vulnerabilities"
	@echo "make clean        - Remove build artifacts"
