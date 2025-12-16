# Contributing to Stacktower

Thanks for your interest in contributing!

## Getting Started

```bash
git clone https://github.com/matzehuels/stacktower.git
cd stacktower
make install-tools  # Install golangci-lint, goimports, govulncheck
make check          # Run all CI checks locally
```

## Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/amazing-feature`)
3. Make your changes
4. Run checks: `make check`
5. Commit with [Conventional Commits](https://www.conventionalcommits.org/) format:
   - `feat: add new feature`
   - `fix: resolve bug`
   - `docs: update readme`
   - `refactor: restructure code`
   - `test: add tests`
   - `ci: update workflows`
6. Push and open a Pull Request

## Code Style

- Run `make fmt` before committing
- Run `make lint` to check for issues
- Keep changes focused and minimal

## Running Tests

```bash
make test       # Unit tests
make e2e        # End-to-end tests
make cover      # Tests with coverage
```

## Questions?

Open an [issue](https://github.com/matzehuels/stacktower/issues) — we're happy to help!

