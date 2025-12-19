package cli

import (
	"testing"
)

func TestSetVersion(t *testing.T) {
	// Test that SetVersion updates the package-level variables
	SetVersion("1.0.0", "abc123", "2024-01-01")

	if version != "1.0.0" {
		t.Errorf("version = %q, want %q", version, "1.0.0")
	}
	if commit != "abc123" {
		t.Errorf("commit = %q, want %q", commit, "abc123")
	}
	if date != "2024-01-01" {
		t.Errorf("date = %q, want %q", date, "2024-01-01")
	}
}

func TestSetVersionEmpty(t *testing.T) {
	SetVersion("", "", "")

	if version != "" {
		t.Errorf("version should be empty, got %q", version)
	}
	if commit != "" {
		t.Errorf("commit should be empty, got %q", commit)
	}
	if date != "" {
		t.Errorf("date should be empty, got %q", date)
	}
}
