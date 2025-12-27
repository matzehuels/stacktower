package php

import (
	"testing"
	"time"

	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

func TestNewResolver(t *testing.T) {
	r, err := Language.NewResolver(storage.NullBackend{}, time.Minute)
	if err != nil {
		t.Fatalf("NewResolver failed: %v", err)
	}
	if r == nil {
		t.Fatal("resolver is nil")
	}
}
