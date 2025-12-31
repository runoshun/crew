package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func testKey() string {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return hex.EncodeToString(key)
}

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	enc, err := NewEncryptor(testKey(), "")
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	plaintext := []byte("Hello, World! This is a test message.")

	// Encrypt
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Ciphertext should be different from plaintext
	if bytes.Equal(ciphertext, plaintext) {
		t.Error("Ciphertext should not equal plaintext")
	}

	// Ciphertext should be longer (IV + tag)
	if len(ciphertext) <= len(plaintext) {
		t.Error("Ciphertext should be longer than plaintext")
	}

	// Decrypt
	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypted text mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptor_DeterministicWithCache(t *testing.T) {
	cacheDir := t.TempDir()
	enc, err := NewEncryptor(testKey(), cacheDir)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	plaintext := []byte("Same content should produce same ciphertext")

	// First encryption
	ciphertext1, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("First Encrypt failed: %v", err)
	}

	// Second encryption of same content should be identical (cached IV)
	ciphertext2, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Second Encrypt failed: %v", err)
	}

	if !bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("Same plaintext should produce same ciphertext when cache is used")
	}

	// Both should decrypt correctly
	decrypted, err := enc.Decrypt(ciphertext1)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypted text mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptor_DifferentWithoutCache(t *testing.T) {
	// No cache directory - but in-memory cache still works
	// To truly get different ciphertext, we need new encryptor instances
	enc1, err := NewEncryptor(testKey(), "")
	if err != nil {
		t.Fatalf("NewEncryptor 1 failed: %v", err)
	}

	enc2, err := NewEncryptor(testKey(), "")
	if err != nil {
		t.Fatalf("NewEncryptor 2 failed: %v", err)
	}

	plaintext := []byte("Same content, different ciphertext")

	ciphertext1, err := enc1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("First Encrypt failed: %v", err)
	}

	ciphertext2, err := enc2.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Second Encrypt failed: %v", err)
	}

	// Different encryptor instances without shared cache should produce different ciphertext
	if bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("Different encryptor instances should produce different ciphertext (random IV)")
	}

	// Both should still decrypt correctly
	decrypted1, err := enc1.Decrypt(ciphertext1)
	if err != nil {
		t.Fatalf("Decrypt ciphertext1 failed: %v", err)
	}
	decrypted2, err := enc2.Decrypt(ciphertext2)
	if err != nil {
		t.Fatalf("Decrypt ciphertext2 failed: %v", err)
	}

	if !bytes.Equal(decrypted1, plaintext) || !bytes.Equal(decrypted2, plaintext) {
		t.Error("Both ciphertexts should decrypt to the same plaintext")
	}
}

func TestEncryptor_InvalidKey(t *testing.T) {
	// Key too short
	_, err := NewEncryptor("0102030405060708", "")
	if err == nil {
		t.Error("Expected error for short key")
	}

	// Invalid hex
	_, err = NewEncryptor("not-a-hex-string-at-all-1234567890abcdef1234567890abcdef", "")
	if err == nil {
		t.Error("Expected error for invalid hex key")
	}
}

func TestEncryptor_InvalidCiphertext(t *testing.T) {
	enc, err := NewEncryptor(testKey(), "")
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	// Too short (less than IV size)
	_, err = enc.Decrypt([]byte("short"))
	if err == nil {
		t.Error("Expected error for short ciphertext")
	}

	// Corrupted ciphertext
	plaintext := []byte("test")
	ciphertext, _ := enc.Encrypt(plaintext)
	ciphertext[len(ciphertext)-1] ^= 0xFF // Corrupt last byte
	_, err = enc.Decrypt(ciphertext)
	if err == nil {
		t.Error("Expected error for corrupted ciphertext")
	}
}

func TestEncryptor_EmptyPlaintext(t *testing.T) {
	enc, err := NewEncryptor(testKey(), "")
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	plaintext := []byte{}
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt empty failed: %v", err)
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt empty failed: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("Expected empty decrypted, got %d bytes", len(decrypted))
	}
}

func TestEncryptor_CachePersistence(t *testing.T) {
	cacheDir := t.TempDir()
	plaintext := []byte("Persistent cache test")

	// First encryptor
	enc1, err := NewEncryptor(testKey(), cacheDir)
	if err != nil {
		t.Fatalf("NewEncryptor 1 failed: %v", err)
	}
	ciphertext1, err := enc1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt 1 failed: %v", err)
	}

	// Save cache (already done automatically, but explicit call to ensure)
	if saveErr := enc1.SaveCache(); saveErr != nil {
		t.Fatalf("SaveCache failed: %v", saveErr)
	}

	// New encryptor with same cache dir should produce same ciphertext
	enc2, err := NewEncryptor(testKey(), cacheDir)
	if err != nil {
		t.Fatalf("NewEncryptor 2 failed: %v", err)
	}
	ciphertext2, err := enc2.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt 2 failed: %v", err)
	}

	if !bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("Cache should persist across encryptor instances")
	}
}

func TestEncryptor_CrossDecrypt(t *testing.T) {
	// Encrypt with one encryptor, decrypt with another (same key)
	cacheDir := t.TempDir()

	enc1, err := NewEncryptor(testKey(), cacheDir)
	if err != nil {
		t.Fatalf("NewEncryptor 1 failed: %v", err)
	}

	plaintext := []byte("Cross-decrypt test")
	ciphertext, err := enc1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Different encryptor with same key should decrypt
	enc2, err := NewEncryptor(testKey(), "")
	if err != nil {
		t.Fatalf("NewEncryptor 2 failed: %v", err)
	}

	decrypted, err := enc2.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Cross-decrypt should produce original plaintext")
	}
}
