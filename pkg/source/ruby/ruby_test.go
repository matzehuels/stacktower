package ruby

import (
	"testing"
	"time"
)

func TestNewParser(t *testing.T) {
	p, err := NewParser(time.Hour)
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	if p.client == nil {
		t.Error("client not initialized")
	}
}

