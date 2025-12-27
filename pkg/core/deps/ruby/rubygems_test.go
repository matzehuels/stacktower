package ruby

import (
	"testing"
	"time"

	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

func TestNewResolver(t *testing.T) {
	r, err := Language.NewResolver(storage.NullBackend{}, time.Hour)
	if err != nil {
		t.Fatalf("NewResolver failed: %v", err)
	}
	if r == nil {
		t.Error("resolver not initialized")
	}
}
