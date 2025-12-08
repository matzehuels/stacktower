package manifest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/integrations/npm"
	"github.com/matzehuels/stacktower/pkg/integrations/pypi"
	"github.com/matzehuels/stacktower/pkg/source"
)

type Parser struct {
	npmClient   *npm.Client
	pypiClient  *pypi.Client
	manifestPath string
	parsedRoot   *packageInfo
}

func NewParser(cacheTTL time.Duration) (*Parser, error) {
	npmC, err := npm.NewClient(cacheTTL)
	if err != nil {
		return nil, err
	}
	pypiC, err := pypi.NewClient(cacheTTL)
	if err != nil {
		return nil, err
	}
	return &Parser{npmClient: npmC, pypiClient: pypiC}, nil
}

func (p *Parser) Parse(ctx context.Context, manifestPath string, opts source.Options) (*dag.DAG, error) {
	// 1. Parse manifest based on file type and determine which registry client to use
	var root *packageInfo
	var err error
	var registryClient interface{} // Can be *npm.Client or *pypi.Client

	filename := filepath.Base(manifestPath)
	switch filename {
	case "package.json":
		root, err = parsePackageJSON(manifestPath)
		registryClient = p.npmClient
	case "pyproject.toml":
		root, err = parsePyprojectTOML(manifestPath)
		registryClient = p.pypiClient
	default:
		return nil, fmt.Errorf("unsupported manifest file: %s", filename)
	}

	if err != nil {
		return nil, err
	}
	p.parsedRoot = root

	// 2. Create a fetch function that returns manifest data for root,
	//    but delegates to registry for everything else
	return source.Parse(ctx, root.Name, opts, func(ctx context.Context, name string, refresh bool) (*packageInfo, error) {
		if name == root.Name {
			return root, nil
		}
		// For transitive dependencies, use the appropriate registry client based on manifest type
		fmt.Printf("[DEBUG] Get from reg: %s\n", name)
		return p.fetchFromRegistry(ctx, name, refresh, registryClient)
	})
}

func (p *Parser) fetchFromRegistry(ctx context.Context, name string, refresh bool, registryClient interface{}) (*packageInfo, error) {
	switch client := registryClient.(type) {
	case *pypi.Client:
		registryInfo, err := client.FetchPackage(ctx, name, refresh)
		if err != nil {
			return nil, err
		}
		return &packageInfo{PyPI: registryInfo}, nil
	case *npm.Client:

		registryInfo, err := client.FetchPackage(ctx, name, refresh)
		if err != nil {
			return nil, err
		}
		return &packageInfo{NPM: registryInfo}, nil
	default:
		return nil, fmt.Errorf("unknown registry client type")
	}
}

type packageInfo struct {
	Name           string
	Version        string
	Dependencies   []string
	NPM            *npm.PackageInfo
	PyPI           *pypi.PackageInfo
}

func (pi *packageInfo) GetName() string {
	if pi.Name != "" {
		return pi.Name
	}
	if pi.NPM != nil && pi.NPM.Name != "" {
		return pi.NPM.Name
	}
	if pi.PyPI != nil && pi.PyPI.Name != "" {
		return pi.PyPI.Name
	}
	return ""
}

func (pi *packageInfo) GetVersion() string {
	if pi.Version != "" {
		return pi.Version
	}
	if pi.NPM != nil && pi.NPM.Version != "" {
		return pi.NPM.Version
	}
	if pi.PyPI != nil && pi.PyPI.Version != "" {
		return pi.PyPI.Version
	}
	return ""
}

func (pi *packageInfo) GetDependencies() []string {
	if len(pi.Dependencies) > 0 {
		return pi.Dependencies
	}
	if pi.NPM != nil && len(pi.NPM.Dependencies) > 0 {
		return pi.NPM.Dependencies
	}
	if pi.PyPI != nil && len(pi.PyPI.Dependencies) > 0 {
		return pi.PyPI.Dependencies
	}
	return nil
}

func (pi *packageInfo) ToMetadata() map[string]any {
	m := map[string]any{"version": pi.GetVersion()}

	// summary: prefer PyPI summary, then NPM description
	if pi.PyPI != nil && pi.PyPI.Summary != "" {
		m["summary"] = pi.PyPI.Summary
	} else if pi.NPM != nil && pi.NPM.Description != "" {
		m["summary"] = pi.NPM.Description
	}

	// license
	if pi.PyPI != nil && pi.PyPI.License != "" {
		m["license"] = pi.PyPI.License
	} else if pi.NPM != nil && pi.NPM.License != "" {
		m["license"] = pi.NPM.License
	}

	// author
	if pi.PyPI != nil && pi.PyPI.Author != "" {
		m["author"] = pi.PyPI.Author
	} else if pi.NPM != nil && pi.NPM.Author != "" {
		m["author"] = pi.NPM.Author
	}
	return m
}

func (pi *packageInfo) ToRepoInfo() *source.RepoInfo {
	urls := make(map[string]string)
	var homepage string

	if pi.PyPI != nil {
		for k, v := range pi.PyPI.ProjectURLs {
			urls[k] = v
		}
		homepage = pi.PyPI.HomePage
	}
	if pi.NPM != nil {
		if pi.NPM.Repository != "" {
			urls["repository"] = pi.NPM.Repository
		}
		if homepage == "" && pi.NPM.HomePage != "" {
			homepage = pi.NPM.HomePage
		}
	}

	return &source.RepoInfo{
		Name:        pi.GetName(),
		Version:     pi.GetVersion(),
		ProjectURLs: urls,
		HomePage:    homepage,
	}
}

// parsePackageJSON parses a package.json file and returns the package info
func parsePackageJSON(path string) (*packageInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Name         string            `json:"name"`
		Version      string            `json:"version"`
		Dependencies map[string]string `json:"dependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	// Extract dependency names
	deps := make([]string, 0, len(pkg.Dependencies))
	for name := range pkg.Dependencies {
		deps = append(deps, name)
	}

	return &packageInfo{
		Name:         pkg.Name,
		Version:      pkg.Version,
		Dependencies: deps,
	}, nil
}

// parsePyprojectTOML parses a pyproject.toml file and returns the package info
func parsePyprojectTOML(path string) (*packageInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to read pyproject.toml: %v\n", err)
		return nil, err
	}

	content := string(data)
	fmt.Printf("[DEBUG] Read pyproject.toml, length: %d bytes\n", len(content))

	// Extract name and version
	name := extractTOMLString(content, `name\s*=\s*["']([^"']+)["']`)

	version := extractTOMLString(content, `version\s*=\s*["']([^"']+)["']`)

	// Extract dependencies from [project] dependencies (PEP 621 format)
	deps := extractTOMLArrayPEP621(content)

	if name == "" {
		return nil, fmt.Errorf("project name not found in pyproject.toml")
	}
	if version == "" {
		version = "0.0.0"
	}

	return &packageInfo{
		Name:         name,
		Version:      version,
		Dependencies: deps,
	}, nil
}

// extractTOMLString extracts a string value from TOML content using a regex pattern
func extractTOMLString(content, pattern string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractTOMLArrayPEP621 extracts dependencies from PEP 621 [project] section
func extractTOMLArrayPEP621(content string) []string {
	fmt.Println("[DEBUG] Starting extractTOMLArrayPEP621")

	// Find [project] section - stop at blank line or another section
	projectRe := regexp.MustCompile(`\[project\]\s*\n([\s\S]*?)(?:\n\s*\n|$)`)
	projectMatches := projectRe.FindStringSubmatch(content)
	if len(projectMatches) < 2 {
		fmt.Println("[DEBUG] Could not find [project] section")
		return []string{}
	}

	projectContent := projectMatches[1]

	// Find dependencies array - match everything between = [ and the closing ]
	// Need to handle nested brackets in package specs like "inboard[fastapi]"
	depsRe := regexp.MustCompile(`dependencies\s*=\s*\[([\s\S]*?)\]\s*(?:\n|$)`)
	depsMatches := depsRe.FindStringSubmatch(projectContent)

	if len(depsMatches) < 2 {
		fmt.Println("[DEBUG] Could not find dependencies array in [project] section")
		return []string{}
	}

	fmt.Printf("[DEBUG] Found dependencies array, content length: %d\n", len(depsMatches[1]))
	depsContent := depsMatches[1]

	// Extract quoted strings (package names with version specs)
	itemRe := regexp.MustCompile(`["']([^"']+)["']`)
	items := itemRe.FindAllStringSubmatch(depsContent, -1)

	var deps []string
	seen := make(map[string]bool)

	for _, item := range items {
		if len(item) > 1 {
			// Extract just the package name (before any operators)
			pkgSpec := strings.TrimSpace(item[1])

			// Handle extras like "inboard[fastapi]" or version specifiers like "requests>=2.0"
			pkgName := pkgSpec

			// First, remove extras (the part in square brackets)
			if idx := strings.Index(pkgName, "["); idx != -1 {
				pkgName = pkgName[:idx]
			}

			// Then remove version specifiers
			for _, op := range []string{">", "<", "=", "!", "~", "@"} {
				if idx := strings.Index(pkgName, op); idx != -1 {
					pkgName = pkgName[:idx]
					break
				}
			}
			pkgName = strings.TrimSpace(pkgName)

			// Skip empty or duplicate entries
			if pkgName != "" && !seen[pkgName] {
				deps = append(deps, pkgName)
				seen[pkgName] = true
			}
		}
	}

	return deps
}