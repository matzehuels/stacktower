package java

import "testing"

func TestNormalizeCoordinate(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Already has colon - unchanged
		{"com.google.guava:guava", "com.google.guava:guava"},
		{"org.apache.commons:commons-lang3", "org.apache.commons:commons-lang3"},

		// Underscore converted to colon
		{"com.google.guava_guava", "com.google.guava:guava"},
		{"org.apache.commons_commons-lang3", "org.apache.commons:commons-lang3"},

		// No colon or underscore - unchanged
		{"simple-name", "simple-name"},
		{"", ""},

		// Multiple underscores - only last one converted
		{"com.example_foo_bar", "com.example_foo:bar"},

		// Edge case: underscore at start or end
		{"_test", ":test"},
		{"test_", "test:"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizeCoordinate(tt.input); got != tt.want {
				t.Errorf("NormalizeCoordinate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
