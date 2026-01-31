package errors

import (
	"regexp"
	"strings"
	"unicode"
)

// ValidatePackageName validates a package name for safety and correctness.
// It rejects names that could be used for path traversal or injection attacks.
//
// The validation rules are intentionally conservative:
//   - No empty names
//   - No control characters
//   - No path traversal sequences (.., //, etc.)
//   - No null bytes
//   - Maximum length of 256 characters
//
// Language-specific validation should be done separately by the language parsers.
func ValidatePackageName(name string) error {
	if name == "" {
		return New(ErrCodeInvalidPackage, "package name cannot be empty")
	}

	if len(name) > 256 {
		return New(ErrCodeInvalidPackage, "package name too long (max 256 characters)")
	}

	// Check for control characters and null bytes
	for _, r := range name {
		if unicode.IsControl(r) {
			return New(ErrCodeInvalidPackage, "package name contains invalid control characters")
		}
	}

	// Check for path traversal patterns
	dangerousPatterns := []string{
		"..",   // Parent directory
		"//",   // Double slash
		"\x00", // Null byte
		"\\",   // Backslash (Windows path)
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(name, pattern) {
			return New(ErrCodeInvalidPackage, "package name contains invalid characters: %q", pattern)
		}
	}

	return nil
}

// ValidateManifestFilename validates a manifest filename for safety.
// It ensures the filename is a simple basename without path components.
func ValidateManifestFilename(filename string) error {
	if filename == "" {
		return New(ErrCodeInvalidManifest, "manifest filename cannot be empty")
	}

	// Must be a simple filename, not a path
	if strings.ContainsAny(filename, "/\\") {
		return New(ErrCodeInvalidManifest, "manifest filename cannot contain path separators")
	}

	// No hidden files (starting with .)
	if strings.HasPrefix(filename, ".") && filename != ".env" {
		// Allow .env but reject other hidden files for security
		return New(ErrCodeInvalidManifest, "manifest filename cannot be a hidden file")
	}

	return nil
}

// ValidatePath validates a file path within a repository for safety.
// It prevents path traversal attacks and ensures reasonable path length.
//
// Validation rules:
//   - Path cannot be empty
//   - Maximum length of 500 characters
//   - No null bytes or control characters
//   - No absolute paths (must be relative)
//   - No path traversal sequences (..)
//   - No backslashes (Windows-style paths)
func ValidatePath(path string) error {
	if path == "" {
		return New(ErrCodeInvalidPath, "path cannot be empty")
	}

	const maxPathLength = 500
	if len(path) > maxPathLength {
		return New(ErrCodeInvalidPath, "path too long (max %d characters)", maxPathLength)
	}

	// Check for null bytes and control characters
	for _, r := range path {
		if r == '\x00' || unicode.IsControl(r) {
			return New(ErrCodeInvalidPath, "path contains invalid characters")
		}
	}

	// Must not be absolute path
	if strings.HasPrefix(path, "/") {
		return New(ErrCodeInvalidPath, "path must be relative (cannot start with /)")
	}

	// Check for path traversal
	if strings.Contains(path, "..") {
		return New(ErrCodeInvalidPath, "path cannot contain path traversal sequences (..)")
	}

	// No backslashes (potential Windows path injection)
	if strings.Contains(path, "\\") {
		return New(ErrCodeInvalidPath, "path cannot contain backslashes")
	}

	return nil
}

// ValidateURL validates a URL string for safety.
// It ensures the URL has a safe scheme (http or https).
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return New(ErrCodeInvalidInput, "URL cannot be empty")
	}

	// Simple scheme validation without full URL parsing
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return New(ErrCodeInvalidInput, "URL must use http or https scheme")
	}

	return nil
}

// pythonPackageNameRegex matches valid Python package names (PEP 508).
var pythonPackageNameRegex = regexp.MustCompile(`^([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9._-]*[A-Za-z0-9])$`)

// ValidatePythonPackageName validates a Python package name per PEP 508.
func ValidatePythonPackageName(name string) error {
	if err := ValidatePackageName(name); err != nil {
		return err
	}

	if !pythonPackageNameRegex.MatchString(name) {
		return New(ErrCodeInvalidPackage, "invalid Python package name: %q", name)
	}

	return nil
}

// npmPackageNameRegex matches valid npm package names.
var npmPackageNameRegex = regexp.MustCompile(`^(@[a-z0-9-~][a-z0-9-._~]*/)?[a-z0-9-~][a-z0-9-._~]*$`)

// ValidateNpmPackageName validates an npm package name.
func ValidateNpmPackageName(name string) error {
	if err := ValidatePackageName(name); err != nil {
		return err
	}

	// npm names must be lowercase
	if strings.ToLower(name) != name {
		return New(ErrCodeInvalidPackage, "npm package names must be lowercase: %q", name)
	}

	if !npmPackageNameRegex.MatchString(name) {
		return New(ErrCodeInvalidPackage, "invalid npm package name: %q", name)
	}

	return nil
}

// cratesPackageNameRegex matches valid crates.io package names.
var cratesPackageNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// ValidateCratesPackageName validates a crates.io package name.
func ValidateCratesPackageName(name string) error {
	if err := ValidatePackageName(name); err != nil {
		return err
	}

	if !cratesPackageNameRegex.MatchString(name) {
		return New(ErrCodeInvalidPackage, "invalid crates.io package name: %q", name)
	}

	return nil
}

// goModulePathRegex matches valid Go module paths.
var goModulePathRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)

// ValidateGoModulePath validates a Go module path.
func ValidateGoModulePath(path string) error {
	if err := ValidatePackageName(path); err != nil {
		return err
	}

	if !goModulePathRegex.MatchString(path) {
		return New(ErrCodeInvalidPackage, "invalid Go module path: %q", path)
	}

	return nil
}
