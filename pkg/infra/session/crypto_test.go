package session

import (
	"testing"
)

func TestTokenEncryptor_RoundTrip(t *testing.T) {
	// Generate a valid key
	encodedKey, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	key, err := ParseKey(encodedKey)
	if err != nil {
		t.Fatalf("ParseKey: %v", err)
	}

	encryptor, err := NewTokenEncryptor(key)
	if err != nil {
		t.Fatalf("NewTokenEncryptor: %v", err)
	}

	testCases := []string{
		"gho_abc123xyz",
		"",
		"a",
		"very-long-token-" + string(make([]byte, 1000)),
		"token with special chars: !@#$%^&*()",
		"unicode: 你好世界 🚀",
	}

	for _, plaintext := range testCases {
		t.Run("", func(t *testing.T) {
			encrypted, err := encryptor.Encrypt(plaintext)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}

			// Encrypted should be different from plaintext (unless empty)
			if plaintext != "" && encrypted == plaintext {
				t.Error("encrypted should differ from plaintext")
			}

			decrypted, err := encryptor.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}

			if decrypted != plaintext {
				t.Errorf("round-trip failed: got %q, want %q", decrypted, plaintext)
			}
		})
	}
}

func TestTokenEncryptor_DifferentNonces(t *testing.T) {
	key, _ := ParseKey(mustGenerateKey(t))
	encryptor, _ := NewTokenEncryptor(key)

	plaintext := "same-token"

	// Encrypt the same plaintext twice
	enc1, _ := encryptor.Encrypt(plaintext)
	enc2, _ := encryptor.Encrypt(plaintext)

	// Should produce different ciphertexts (different nonces)
	if enc1 == enc2 {
		t.Error("encrypting same plaintext should produce different ciphertext")
	}

	// Both should decrypt to the same plaintext
	dec1, _ := encryptor.Decrypt(enc1)
	dec2, _ := encryptor.Decrypt(enc2)

	if dec1 != plaintext || dec2 != plaintext {
		t.Error("both ciphertexts should decrypt to same plaintext")
	}
}

func TestTokenEncryptor_WrongKey(t *testing.T) {
	key1, _ := ParseKey(mustGenerateKey(t))
	key2, _ := ParseKey(mustGenerateKey(t))

	encryptor1, _ := NewTokenEncryptor(key1)
	encryptor2, _ := NewTokenEncryptor(key2)

	encrypted, _ := encryptor1.Encrypt("secret-token")

	// Decrypting with wrong key should fail
	_, err := encryptor2.Decrypt(encrypted)
	if err == nil {
		t.Error("decrypting with wrong key should fail")
	}
}

func TestTokenEncryptor_InvalidKey(t *testing.T) {
	// Too short
	_, err := NewTokenEncryptor([]byte("short"))
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey for short key, got %v", err)
	}

	// Too long
	_, err = NewTokenEncryptor(make([]byte, 64))
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey for long key, got %v", err)
	}
}

func TestParseKey_Invalid(t *testing.T) {
	// Invalid base64
	_, err := ParseKey("not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}

	// Valid base64 but wrong length
	_, err = ParseKey("c2hvcnQ=") // "short"
	if err != ErrInvalidKey {
		t.Errorf("expected ErrInvalidKey, got %v", err)
	}
}

func TestDecrypt_InvalidData(t *testing.T) {
	key, _ := ParseKey(mustGenerateKey(t))
	encryptor, _ := NewTokenEncryptor(key)

	// Invalid base64
	_, err := encryptor.Decrypt("not-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}

	// Valid base64 but corrupted ciphertext
	_, err = encryptor.Decrypt("YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=")
	if err != ErrDecryptionFailed {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}

	// Too short (less than nonce size)
	_, err = encryptor.Decrypt("YWJj") // "abc"
	if err != ErrDecryptionFailed {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}
}

func mustGenerateKey(t *testing.T) string {
	t.Helper()
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return key
}
