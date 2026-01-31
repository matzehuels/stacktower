package ruby

import (
	"testing"
	"time"

	"github.com/matzehuels/stacktower/pkg/cache"
)

func TestNewResolver(t *testing.T) {
	r, err := Language.NewResolver(cache.NewNullCache(), time.Hour)
	if err != nil {
		t.Fatalf("NewResolver failed: %v", err)
	}
	if r == nil {
		t.Error("resolver not initialized")
	}
}
