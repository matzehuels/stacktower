package ruby

import (
	"testing"
	"time"
)

func TestNewResolver(t *testing.T) {
	r, err := Language.NewResolver(time.Hour)
	if err != nil {
		t.Fatalf("NewResolver failed: %v", err)
	}
	if r == nil {
		t.Error("resolver not initialized")
	}
}
