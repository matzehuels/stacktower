package metadata

import "time"

// NodeMetadata provides typed access to common node metadata fields.
// This is an optional helper that provides compile-time safety for frequently
// accessed metadata. The underlying dag.Metadata remains map[string]any for
// flexibility with arbitrary registry data.
//
// Usage:
//
//	typed := metadata.FromMap(node.Meta)
//	if typed.RepoURL != "" {
//	    fmt.Println("Repository:", typed.RepoURL)
//	}
//
// To convert back to map form:
//
//	node.Meta = typed.ToMap()
type NodeMetadata struct {
	// Version is the package version (e.g., "2.31.0").
	Version string

	// Description is a short summary of the package.
	Description string

	// RepoURL is the canonical repository URL (e.g., "https://github.com/owner/repo").
	RepoURL string

	// RepoOwner is the repository owner/organization name.
	RepoOwner string

	// RepoStars is the GitHub/GitLab star count.
	RepoStars int

	// RepoArchived indicates whether the repository is archived.
	RepoArchived bool

	// RepoLanguage is the primary programming language.
	RepoLanguage string

	// RepoTopics are the repository topic tags.
	RepoTopics []string

	// RepoMaintainers are the top contributors/maintainers.
	RepoMaintainers []string

	// RepoLastCommit is the date of the most recent commit.
	RepoLastCommit string

	// RepoLastRelease is the date of the most recent release.
	RepoLastRelease string

	// RepoLicense is the SPDX license identifier (e.g., "MIT").
	RepoLicense string

	// Extra holds additional metadata not covered by typed fields.
	// This preserves arbitrary registry data.
	Extra map[string]any
}

// FromMap converts a dag.Metadata map to typed NodeMetadata.
// Unknown fields are preserved in the Extra map.
//
// This function is safe to call with nil input - it returns a zero NodeMetadata.
func FromMap(m map[string]any) NodeMetadata {
	if m == nil {
		return NodeMetadata{}
	}

	typed := NodeMetadata{
		Extra: make(map[string]any),
	}

	for k, v := range m {
		switch k {
		case "version":
			typed.Version, _ = v.(string)
		case "description":
			typed.Description, _ = v.(string)
		case RepoURL:
			typed.RepoURL, _ = v.(string)
		case RepoOwner:
			typed.RepoOwner, _ = v.(string)
		case RepoStars:
			if stars, ok := v.(int); ok {
				typed.RepoStars = stars
			} else if stars, ok := v.(float64); ok {
				typed.RepoStars = int(stars)
			}
		case RepoArchived:
			typed.RepoArchived, _ = v.(bool)
		case RepoLanguage:
			typed.RepoLanguage, _ = v.(string)
		case RepoTopics:
			if topics, ok := v.([]string); ok {
				typed.RepoTopics = topics
			} else if topics, ok := v.([]any); ok {
				typed.RepoTopics = make([]string, 0, len(topics))
				for _, t := range topics {
					if s, ok := t.(string); ok {
						typed.RepoTopics = append(typed.RepoTopics, s)
					}
				}
			}
		case RepoMaintainers:
			if maintainers, ok := v.([]string); ok {
				typed.RepoMaintainers = maintainers
			} else if maintainers, ok := v.([]any); ok {
				typed.RepoMaintainers = make([]string, 0, len(maintainers))
				for _, m := range maintainers {
					if s, ok := m.(string); ok {
						typed.RepoMaintainers = append(typed.RepoMaintainers, s)
					}
				}
			}
		case RepoLastCommit:
			typed.RepoLastCommit, _ = v.(string)
		case RepoLastRelease:
			typed.RepoLastRelease, _ = v.(string)
		case RepoLicense:
			typed.RepoLicense, _ = v.(string)
		default:
			// Preserve unknown fields
			typed.Extra[k] = v
		}
	}

	return typed
}

// ToMap converts typed NodeMetadata back to a dag.Metadata map.
// Only non-zero fields are included in the output.
// Extra fields are merged into the result.
func (n NodeMetadata) ToMap() map[string]any {
	m := make(map[string]any)

	if n.Version != "" {
		m["version"] = n.Version
	}
	if n.Description != "" {
		m["description"] = n.Description
	}
	if n.RepoURL != "" {
		m[RepoURL] = n.RepoURL
	}
	if n.RepoOwner != "" {
		m[RepoOwner] = n.RepoOwner
	}
	if n.RepoStars > 0 {
		m[RepoStars] = n.RepoStars
	}
	if n.RepoArchived {
		m[RepoArchived] = n.RepoArchived
	}
	if n.RepoLanguage != "" {
		m[RepoLanguage] = n.RepoLanguage
	}
	if len(n.RepoTopics) > 0 {
		m[RepoTopics] = n.RepoTopics
	}
	if len(n.RepoMaintainers) > 0 {
		m[RepoMaintainers] = n.RepoMaintainers
	}
	if n.RepoLastCommit != "" {
		m[RepoLastCommit] = n.RepoLastCommit
	}
	if n.RepoLastRelease != "" {
		m[RepoLastRelease] = n.RepoLastRelease
	}
	if n.RepoLicense != "" {
		m[RepoLicense] = n.RepoLicense
	}

	// Merge extra fields
	for k, v := range n.Extra {
		m[k] = v
	}

	return m
}

// ParseTime attempts to parse a time string in common formats.
// Returns nil if the string is empty or cannot be parsed.
//
// Supported formats:
//   - RFC3339 (ISO 8601): "2006-01-02T15:04:05Z07:00"
//   - RFC3339Nano: "2006-01-02T15:04:05.999999999Z07:00"
//   - Date only: "2006-01-02"
func ParseTime(s string) *time.Time {
	if s == "" {
		return nil
	}

	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return &t
		}
	}

	return nil
}
