package php

import (
	"testing"
	"time"
)

func TestNewResolver(t *testing.T) {
	r, err := Language.NewResolver(time.Minute)
	if err != nil {
		t.Fatalf("NewResolver failed: %v", err)
	}
	if r == nil {
		t.Fatal("resolver is nil")
	}
}
