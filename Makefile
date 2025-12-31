.PHONY: all build build-cli build-server clean fmt lint test cover vuln e2e e2e-test e2e-real e2e-parse blog blog-diagrams blog-showcase install-tools snapshot release help
.PHONY: standalone standalone-noauth dev dev-down dev-logs dev-wipe docker-build

# =============================================================================
# Variables
# =============================================================================

CLI_BINARY := stacktower
SERVER_BINARY := stacktowerd
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/matzehuels/stacktower/pkg/buildinfo.Version=$(VERSION) \
           -X github.com/matzehuels/stacktower/pkg/buildinfo.Commit=$(COMMIT) \
           -X github.com/matzehuels/stacktower/pkg/buildinfo.Date=$(DATE)

# =============================================================================
# Default Target
# =============================================================================

all: check build

# =============================================================================
# Build Targets
# =============================================================================

build: build-cli build-server

build-cli:
	@echo "Building CLI (bin/$(CLI_BINARY))..."
	@go build -ldflags "$(LDFLAGS)" -o bin/$(CLI_BINARY) ./cmd/stacktower

build-server:
	@echo "Building server (bin/$(SERVER_BINARY))..."
	@go build -ldflags "$(LDFLAGS)" -o bin/$(SERVER_BINARY) ./cmd/stacktowerd

install:
	@echo "Installing CLI..."
	@go install -ldflags "$(LDFLAGS)" ./cmd/stacktower

clean:
	@rm -rf bin/ dist/ coverage.out data/ tmp/

# =============================================================================
# Quality Checks
# =============================================================================

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

vuln:
	@govulncheck ./...

install-tools:
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/air-verse/air@v1.61.5

# =============================================================================
# Standalone Mode (No Docker)
# =============================================================================
# Run the server locally with in-memory storage. Perfect for quick development.

standalone: build-server
	@echo ""
	@echo "Starting stacktowerd in STANDALONE mode..."
	@echo "  API + Worker running in single process"
	@echo "  In-memory storage (data will not persist)"
	@echo "  Server: http://localhost:8080"
	@echo ""
	@echo "Test with:"
	@echo "  curl http://localhost:8080/health"
	@echo "  curl http://localhost:8080/api/v1/auth/me"
	@echo ""
	@./bin/$(SERVER_BINARY) standalone

standalone-noauth: build-server
	@echo ""
	@echo "Starting stacktowerd in STANDALONE mode (no auth)..."
	@echo "  API + Worker running in single process"
	@echo "  In-memory storage (data will not persist)"
	@echo "  Authentication DISABLED (mock 'local' user)"
	@echo "  Server: http://localhost:8080"
	@echo ""
	@echo "Test with:"
	@echo "  curl http://localhost:8080/health"
	@echo "  curl http://localhost:8080/api/v1/auth/me"
	@echo ""
	@echo "Render a package to PNG:"
	@echo '  curl -X POST http://localhost:8080/api/v1/render -H "Content-Type: application/json" \'
	@echo '    -d '\''{"language":"python","package":"requests","formats":["png"]}'\'''
	@echo ""
	@./bin/$(SERVER_BINARY) standalone --no-auth

# =============================================================================
# Development Mode (Docker with Hot Reloading)
# =============================================================================
# Full hot-reload experience: both frontend (Vite HMR) and backend (Air).
# Edit Go code → saves → rebuilds in ~1-2s → server restarts automatically.

dev:
	@echo ""
	@echo "Starting stacktowerd..."
	@echo ""
	@echo "  Frontend:         http://localhost:3000"
	@echo "  API:              http://localhost:8080"
	@echo "  Mongo Express:    http://localhost:8081"
	@echo "  Redis Commander:  http://localhost:8082"
	@echo ""
	@echo "  Edit Go files → auto-rebuild (~1-2s)"
	@echo "  Edit React files → instant HMR"
	@echo ""
	@echo "To view logs:  make dev-logs"
	@echo "To stop:       make dev-down"
	@echo ""
	@docker compose up -d --build

dev-down:
	@echo "Stopping stack..."
	@docker compose down --remove-orphans

dev-logs:
	@docker compose logs -f api worker frontend

dev-wipe:
	@echo "Wiping stack (containers, volumes, images)..."
	@docker compose down --volumes --remove-orphans --rmi local
	@echo "Stack wiped. Run 'make dev' to start fresh."

docker-build:
	@echo "Building Docker image: stacktower:$(VERSION)"
	@docker build --target prod -t stacktower:$(VERSION) -t stacktower:latest .

# =============================================================================
# End-to-End Tests
# =============================================================================

e2e-cli: build
	@./scripts/test_cli_e2e.sh all

e2e-cli-test: build
	@./scripts/test_cli_e2e.sh test

e2e-cli-real: build
	@./scripts/test_cli_e2e.sh real

e2e-cli-parse: build
	@./scripts/test_cli_e2e.sh parse

e2e-api: build-server
	@./scripts/test_api_e2e.sh all

e2e-api-quick: build-server
	@./scripts/test_api_e2e.sh quick

# =============================================================================
# Blog Assets
# =============================================================================

blog: blog-diagrams blog-showcase

blog-diagrams: build
	@./scripts/blog_diagrams.sh

blog-showcase: build
	@./scripts/blog_showcase.sh

# =============================================================================
# Release
# =============================================================================

snapshot:
	@goreleaser release --snapshot --clean --skip=publish

release:
	@goreleaser release --clean

# =============================================================================
# Help
# =============================================================================

help:
	@echo "Stacktower Makefile"
	@echo ""
	@echo "DEVELOPMENT:"
	@echo "  make dev              - Start full stack with hot-reload"
	@echo "  make dev-logs         - View logs"
	@echo "  make dev-down         - Stop stack"
	@echo "  make dev-wipe         - Wipe everything (volumes, images)"
	@echo ""
	@echo "STANDALONE (no Docker):"
	@echo "  make standalone       - Run locally with in-memory storage"
	@echo "  make standalone-noauth- Same but skip authentication"
	@echo ""
	@echo "BUILDING:"
	@echo "  make build            - Build both CLI and server binaries"
	@echo "  make build-cli        - Build CLI only (bin/stacktower)"
	@echo "  make build-server     - Build server only (bin/stacktowerd)"
	@echo "  make docker-build     - Build production Docker image"
	@echo "  make install          - Install CLI to GOPATH"
	@echo ""
	@echo "QUALITY:"
	@echo "  make check            - Run all checks (fmt, lint, test, vuln)"
	@echo "  make fmt              - Format code"
	@echo "  make lint             - Run golangci-lint"
	@echo "  make test             - Run tests"
	@echo "  make cover            - Run tests with coverage"
	@echo "  make vuln             - Check for vulnerabilities"
	@echo ""
	@echo "TESTING:"
	@echo "  make e2e-cli          - Run all CLI end-to-end tests"
	@echo "  make e2e-api          - Run API server end-to-end tests"
	@echo ""
	@echo "RELEASE:"
	@echo "  make snapshot         - Build release locally (no publish)"
	@echo "  make release          - Build and publish release"
	@echo ""
	@echo "OTHER:"
	@echo "  make clean            - Remove build artifacts"
	@echo "  make install-tools    - Install development tools"
	@echo "  make help             - Show this help"
