package api

import (
	"net/http"
	"sort"
	"strings"

	"github.com/matzehuels/stacktower/pkg/core/deps/languages"
)

// IntegrationsResponse is the response for GET /api/v1/integrations.
type IntegrationsResponse struct {
	Languages []LanguageInfo `json:"languages"`
}

// LanguageInfo describes a supported programming language and its ecosystem.
type LanguageInfo struct {
	Name      string         `json:"name"`
	Registry  RegistryInfo   `json:"registry"`
	Manifests []ManifestInfo `json:"manifests"`
}

// RegistryInfo describes a package registry.
type RegistryInfo struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases,omitempty"`
}

// ManifestInfo describes a supported manifest file.
type ManifestInfo struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
}

// handleIntegrations handles GET /api/v1/integrations.
// Returns information about supported languages, registries, and manifest files.
// This endpoint is public (no auth required).
func (s *Server) handleIntegrations(w http.ResponseWriter, r *http.Request) {
	var response IntegrationsResponse

	for _, lang := range languages.All {
		// Build registry info
		registry := RegistryInfo{
			Name: lang.DefaultRegistry,
		}

		// Collect unique aliases (excluding the canonical name)
		aliasSet := make(map[string]struct{})
		for alias, canonical := range lang.RegistryAliases {
			if alias != canonical {
				aliasSet[alias] = struct{}{}
			}
		}
		for alias := range aliasSet {
			registry.Aliases = append(registry.Aliases, alias)
		}
		sort.Strings(registry.Aliases)

		// Build manifest info, deduplicating case-insensitive filenames
		// (e.g., Cargo.toml and cargo.toml are the same manifest).
		// Prefer the properly-cased version (e.g., "Cargo.toml" over "cargo.toml").
		var manifests []ManifestInfo
		seenLower := make(map[string]ManifestInfo)
		for filename, manifestType := range lang.ManifestAliases {
			lower := strings.ToLower(filename)
			existing, seen := seenLower[lower]
			if seen {
				// Prefer the version that starts with uppercase (proper casing)
				if filename[0] >= 'A' && filename[0] <= 'Z' && (existing.Filename[0] < 'A' || existing.Filename[0] > 'Z') {
					seenLower[lower] = ManifestInfo{Filename: filename, Type: manifestType}
				}
				continue
			}
			seenLower[lower] = ManifestInfo{Filename: filename, Type: manifestType}
		}
		for _, info := range seenLower {
			manifests = append(manifests, info)
		}
		// Sort for consistent ordering
		sort.Slice(manifests, func(i, j int) bool {
			return manifests[i].Filename < manifests[j].Filename
		})

		response.Languages = append(response.Languages, LanguageInfo{
			Name:      lang.Name,
			Registry:  registry,
			Manifests: manifests,
		})
	}

	// Sort languages alphabetically
	sort.Slice(response.Languages, func(i, j int) bool {
		return response.Languages[i].Name < response.Languages[j].Name
	})

	s.jsonResponse(w, http.StatusOK, response)
}
