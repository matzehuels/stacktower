package errors

import (
	"testing"
)

func TestValidatePackageName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "requests", false},
		{"valid with dash", "my-package", false},
		{"valid with underscore", "my_package", false},
		{"valid with dot", "my.package", false},
		{"valid scoped npm", "@scope/package", false},

		{"empty", "", true},
		{"too long", string(make([]byte, 300)), true},
		{"path traversal ..", "foo/../bar", true},
		{"path traversal //", "foo//bar", true},
		{"null byte", "foo\x00bar", true},
		{"backslash", "foo\\bar", true},
		{"control char", "foo\x01bar", true},
		{"newline", "foo\nbar", true},
		{"carriage return", "foo\rbar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePackageName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePackageName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateManifestFilename(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid poetry.lock", "poetry.lock", false},
		{"valid requirements.txt", "requirements.txt", false},
		{"valid package.json", "package.json", false},
		{"valid .env", ".env", false},

		{"empty", "", true},
		{"with path /", "path/to/file", true},
		{"with path \\", "path\\to\\file", true},
		{"hidden file", ".hidden", true},
		{"hidden file long", ".secret.json", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateManifestFilename(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateManifestFilename(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"https", "https://example.com/path", false},
		{"http", "http://example.com/path", false},

		{"empty", "", true},
		{"ftp", "ftp://example.com", true},
		{"file", "file:///etc/passwd", true},
		{"javascript", "javascript:alert(1)", true},
		{"no scheme", "example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePythonPackageName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple", "requests", false},
		{"with dash", "my-package", false},
		{"with underscore", "my_package", false},
		{"with dot", "my.package", false},
		{"with numbers", "package123", false},
		{"uppercase", "MyPackage", false},

		{"empty", "", true},
		{"starts with dash", "-package", true},
		{"starts with dot", ".package", true},
		{"ends with dash", "package-", true},
		{"ends with dot", "package.", true},
		{"special chars", "my@package", true},
		{"spaces", "my package", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePythonPackageName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePythonPackageName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateNpmPackageName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple", "express", false},
		{"with dash", "my-package", false},
		{"with underscore", "my_package", false},
		{"scoped", "@scope/package", false},
		{"with tilde", "~package", false},

		{"empty", "", true},
		{"uppercase", "Express", true},
		{"starts with dot", ".package", true},
		{"spaces", "my package", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNpmPackageName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNpmPackageName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCratesPackageName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple", "serde", false},
		{"with dash", "my-crate", false},
		{"with underscore", "my_crate", false},

		{"empty", "", true},
		{"starts with number", "123crate", true},
		{"starts with dash", "-crate", true},
		{"with dot", "my.crate", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCratesPackageName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCratesPackageName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateGoModulePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"github module", "github.com/user/repo", false},
		{"simple", "mymodule", false},
		{"with version", "example.com/v2", false},

		{"empty", "", true},
		{"starts with dot", ".module", true},
		{"starts with slash", "/module", true},
		{"special chars", "module@latest", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGoModulePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGoModulePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "src/main.go", false},
		{"valid nested", "pkg/internal/util/helpers.go", false},
		{"valid filename only", "README.md", false},
		{"valid with dots", "v1.2.3/package.json", false},

		{"empty", "", true},
		{"too long", string(make([]byte, 600)), true},
		{"absolute path", "/etc/passwd", true},
		{"path traversal", "../../../etc/passwd", true},
		{"path traversal middle", "foo/../bar", true},
		{"null byte", "foo\x00bar", true},
		{"backslash", "foo\\bar", true},
		{"control char", "foo\x01bar", true},
		{"newline", "foo\nbar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err != nil && !Is(err, ErrCodeInvalidPath) {
				t.Errorf("ValidatePath(%q) returned wrong error code: %v", tt.input, err)
			}
		})
	}
}

func TestErrorCodesAreUnique(t *testing.T) {
	codes := []Code{
		ErrCodeInvalidInput,
		ErrCodeInvalidLanguage,
		ErrCodeInvalidPackage,
		ErrCodeInvalidFormat,
		ErrCodeInvalidStyle,
		ErrCodeInvalidVizType,
		ErrCodeInvalidManifest,
		ErrCodeInvalidPath,
		ErrCodeNotFound,
		ErrCodePackageNotFound,
		ErrCodeFileNotFound,
		ErrCodeSessionNotFound,
		ErrCodeNetwork,
		ErrCodeTimeout,
		ErrCodeRateLimited,
		ErrCodeUnauthorized,
		ErrCodeForbidden,
		ErrCodeSessionExpired,
		ErrCodeInternal,
		ErrCodeUnsupported,
	}

	seen := make(map[Code]bool)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("Duplicate error code: %s", code)
		}
		seen[code] = true
	}
}
