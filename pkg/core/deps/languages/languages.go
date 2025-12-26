// Package languages provides the complete list of supported language ecosystems.
//
// This package exists to break import cycles: the individual language packages
// (python, rust, etc.) import pkg/deps, so pkg/deps cannot import them back.
// Instead, consumers that need the full language list import this package.
//
// Usage:
//
//	import "github.com/matzehuels/stacktower/pkg/core/deps/languages"
//
//	for _, lang := range languages.All {
//	    fmt.Println(lang.Name)
//	}
package languages

import (
	"github.com/matzehuels/stacktower/pkg/core/deps"
	"github.com/matzehuels/stacktower/pkg/core/deps/golang"
	"github.com/matzehuels/stacktower/pkg/core/deps/java"
	"github.com/matzehuels/stacktower/pkg/core/deps/javascript"
	"github.com/matzehuels/stacktower/pkg/core/deps/php"
	"github.com/matzehuels/stacktower/pkg/core/deps/python"
	"github.com/matzehuels/stacktower/pkg/core/deps/ruby"
	"github.com/matzehuels/stacktower/pkg/core/deps/rust"
)

// All is the canonical list of supported package ecosystems.
// Each language provides resolvers for package registries and manifest parsers.
var All = []*deps.Language{
	python.Language,
	rust.Language,
	javascript.Language,
	ruby.Language,
	php.Language,
	java.Language,
	golang.Language,
}

// Find returns the Language with the given name, or nil if not found.
func Find(name string) *deps.Language {
	return deps.FindLanguage(name, All)
}
