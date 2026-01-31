package errors

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(ErrCodeInvalidInput, "test message: %s", "value")

	if err.Code != ErrCodeInvalidInput {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeInvalidInput)
	}

	if err.Message != "test message: value" {
		t.Errorf("Message = %v, want %v", err.Message, "test message: value")
	}

	expected := "INVALID_INPUT: test message: value"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := Wrap(ErrCodeNetwork, cause, "failed to fetch")

	if err.Code != ErrCodeNetwork {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeNetwork)
	}

	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}

	// Test Unwrap
	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	// Test errors.Is with wrapped error
	if !errors.Is(err, cause) {
		t.Error("errors.Is(err, cause) = false, want true")
	}
}

func TestIs(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		code     Code
		expected bool
	}{
		{
			name:     "matching code",
			err:      New(ErrCodeInvalidInput, "test"),
			code:     ErrCodeInvalidInput,
			expected: true,
		},
		{
			name:     "non-matching code",
			err:      New(ErrCodeInvalidInput, "test"),
			code:     ErrCodeNetwork,
			expected: false,
		},
		{
			name:     "wrapped error",
			err:      Wrap(ErrCodeNetwork, New(ErrCodeInvalidInput, "inner"), "outer"),
			code:     ErrCodeNetwork,
			expected: true,
		},
		{
			name:     "non-Error type",
			err:      errors.New("plain error"),
			code:     ErrCodeInvalidInput,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			code:     ErrCodeInvalidInput,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Is(tt.err, tt.code); got != tt.expected {
				t.Errorf("Is() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected Code
	}{
		{
			name:     "Error type",
			err:      New(ErrCodeInvalidPackage, "test"),
			expected: ErrCodeInvalidPackage,
		},
		{
			name:     "plain error",
			err:      errors.New("plain"),
			expected: "",
		},
		{
			name:     "nil",
			err:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCode(tt.err); got != tt.expected {
				t.Errorf("GetCode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUserMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "Error type",
			err:      New(ErrCodeInvalidInput, "friendly message"),
			expected: "friendly message",
		},
		{
			name:     "plain error",
			err:      errors.New("plain error"),
			expected: "plain error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UserMessage(tt.err); got != tt.expected {
				t.Errorf("UserMessage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRateLimitedError(t *testing.T) {
	t.Run("with retry after", func(t *testing.T) {
		err := &RateLimitedError{RetryAfter: 60}
		expected := "rate limited: retry after 60 seconds"
		if err.Error() != expected {
			t.Errorf("Error() = %v, want %v", err.Error(), expected)
		}
	})

	t.Run("without retry after", func(t *testing.T) {
		err := &RateLimitedError{}
		expected := "rate limited"
		if err.Error() != expected {
			t.Errorf("Error() = %v, want %v", err.Error(), expected)
		}
	})

	t.Run("code method", func(t *testing.T) {
		err := &RateLimitedError{}
		if err.Code() != ErrCodeRateLimited {
			t.Errorf("Code() = %v, want %v", err.Code(), ErrCodeRateLimited)
		}
	})
}
