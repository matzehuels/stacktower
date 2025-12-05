# StackTower

Inspired by [XKCD #2347](https://xkcd.com/2347/), StackTower renders dependency graphs as **physical towers** where blocks rest on what they depend on. Your application sits at the top, supported by libraries below—all the way down to that one critical package maintained by *some dude in Nebraska*.


📖 **[Read the full story at stacktower.io](https://www.stacktower.io)**

## Quick Start

```bash
go install github.com/matzehuels/stacktower@latest

# Render the included Flask example
stacktower render examples/real/flask.json -t tower -o flask.svg
```

Or build from source:

```bash
git clone https://github.com/matzehuels/stacktower.git
cd stacktower
go build -o stacktower .
```

## Usage

StackTower works in two stages: **parse** dependency data from package registries, then **render** visualizations.

### Parsing Dependencies

```bash
# Python (PyPI)
stacktower parse python fastapi -o fastapi.json

# Rust (crates.io)
stacktower parse rust serde -o serde.json

# JavaScript (npm)
stacktower parse javascript yup -o yup.json

# manifest package.json + npm
stacktower parse manifest package.json -o tktm.json
```

Add `--enrich` with a `GITHUB_TOKEN` to pull repository metadata (stars, maintainers, last commit) for richer visualizations.

Exemple from source to svg using manifest and a bunch of options
```
go build -o stacktower .

./stacktower parse manifest package.json -o tktm.json --enrich --max-depth 2

./stacktower render tktm.json -t tower -o tktm.svg --ordering barycentric --popups --nebraska --style handdrawn
```

### Rendering

```bash
# Tower visualization (recommended)
stacktower render fastapi.json -t tower -o fastapi.svg

# Hand-drawn style with hover popups
stacktower render serde.json -t tower --style handdrawn --popups -o serde.svg

# Traditional node-link diagram
stacktower render yup.json -t nodelink -o yup.svg
```

### Included Examples

The repository ships with pre-parsed graphs so you can experiment immediately:

```bash
# Real packages with full metadata
stacktower render examples/real/flask.json -t tower --style handdrawn --merge -o flask.svg
stacktower render examples/real/serde.json -t tower --popups -o serde.svg
stacktower render examples/real/express.json -t tower --ordering barycentric -o express.svg

# Synthetic test cases
stacktower render examples/test/diamond.json -t tower -o diamond.svg
```

## Options Reference

### Parse Options

| Flag | Description |
|------|-------------|
| `--max-depth N` | Maximum dependency depth (default: 10) |
| `--max-nodes N` | Maximum packages to fetch (default: 100) |
| `--enrich` | Add repository metadata (requires `GITHUB_TOKEN`) |
| `--refresh` | Bypass cache |

### Render Options (Tower)

| Flag | Description |
|------|-------------|
| `--style simple\|handdrawn` | Visual style |
| `--width`, `--height` | Frame dimensions (default: 800×600) |
| `--edges` | Show dependency edges |
| `--merge` | Merge subdivider blocks |
| `--ordering optimal\|barycentric` | Crossing minimization algorithm |
| `--ordering-timeout N` | Timeout for optimal search in seconds (default: 60) |
| `--nebraska` | Show "Nebraska guy" maintainer ranking |
| `--popups` | Enable hover popups with metadata |

### Render Options (Node-link)

| Flag | Description |
|------|-------------|
| `--detailed` | Show node metadata in labels |

## How It Works

1. **Parse** — Fetch package metadata from registries (PyPI, crates.io, npm)
2. **Reduce** — Remove transitive edges to show only direct dependencies
3. **Layer** — Assign each package to a row based on its depth
4. **Order** — Minimize edge crossings using branch-and-bound with PQ-tree pruning
5. **Layout** — Compute block widths proportional to downstream dependents
6. **Render** — Generate clean SVG output

The ordering step is where the magic happens. StackTower uses an optimal search algorithm that guarantees minimum crossings for small-to-medium graphs. For larger graphs, it gracefully falls back after a configurable timeout.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | GitHub API token for `--enrich` metadata |
| `GITLAB_TOKEN` | GitLab API token for `--enrich` metadata |

## Caching

HTTP responses are cached in `~/.cache/stacktower/` with a 24-hour TTL. Use `--refresh` to bypass.

## Adding New Languages

To add support for a new package manager (e.g., Ruby/RubyGems):

1. **Create a registry client** in `pkg/integrations/rubygems/client.go` — parse the registry API, extract dependencies, use `integrations.BaseClient` for HTTP + caching

2. **Create a source parser** in `pkg/source/ruby/ruby.go` — implement the `source.PackageInfo` interface (`GetName`, `GetVersion`, `GetDependencies`, `ToMetadata`, `ToRepoInfo`)

3. **Wire into CLI** in `internal/cli/parse.go`:
   ```go
   cmd.AddCommand(newParserCmd("ruby <gem>", "Parse Ruby dependencies",
       func() (source.Parser, error) { return ruby.NewParser(source.DefaultCacheTTL) }, &opts))
   ```

The generic `source.Parse()` handles concurrent fetching, depth limits, and graph construction automatically.

## Learn More

- 📖 **[stacktower.io](https://www.stacktower.io)** — Interactive examples and the full story behind tower visualizations
- 🐛 **[Issues](https://github.com/matzehuels/stacktower/issues)** — Bug reports and feature requests

## License

MIT
