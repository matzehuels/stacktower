package deps

import (
	"fmt"
	"path/filepath"
)

type ManifestParser interface {
	Parse(path string, opts Options) (*ManifestResult, error)
	Supports(filename string) bool
	Type() string
	IncludesTransitive() bool
}

type ManifestResult struct {
	Graph              any
	Type               string
	IncludesTransitive bool
	RootPackage        string
}

func DetectManifest(path string, parsers ...ManifestParser) (ManifestParser, error) {
	name := filepath.Base(path)
	for _, p := range parsers {
		if p.Supports(name) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("unsupported manifest: %s", name)
}
