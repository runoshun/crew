// Package crypto provides encryption utilities for gitstore.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	// NonceSize is the size of the nonce for AES-GCM (12 bytes).
	NonceSize = 12
	// KeySize is the size of the AES-256 key (32 bytes).
	KeySize = 32
)

var (
	// ErrInvalidKey is returned when the encryption key is invalid.
	ErrInvalidKey = errors.New("invalid encryption key: must be 32 bytes (64 hex characters)")
	// ErrDecryptionFailed is returned when decryption fails.
	ErrDecryptionFailed = errors.New("decryption failed: invalid ciphertext or key")
	// ErrCiphertextTooShort is returned when the ciphertext is too short.
	ErrCiphertextTooShort = errors.New("ciphertext too short")
)

// Encryptor handles AES-256-GCM encryption with IV caching.
type Encryptor struct {
	gcm       cipher.AEAD
	cache     map[string][]byte // plaintext hash -> ciphertext (nonce + encrypted)
	cachePath string
	mu        sync.RWMutex
}

// NewEncryptor creates a new Encryptor with the given hex-encoded key.
// The key must be 64 hex characters (32 bytes).
// cachePath is the directory to store IV cache (e.g., ".git/crew-cache").
func NewEncryptor(hexKey, cachePath string) (*Encryptor, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil || len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	e := &Encryptor{
		gcm:       gcm,
		cachePath: cachePath,
		cache:     make(map[string][]byte),
	}

	// Load existing cache
	if cachePath != "" {
		_ = e.loadCache()
	}

	return e, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// If the same plaintext was encrypted before, returns the cached ciphertext
// to ensure deterministic blob hashes.
// Returns: nonce (12 bytes) + ciphertext + auth tag
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	// Hash plaintext to use as cache key
	hash := sha256.Sum256(plaintext)
	hashKey := hex.EncodeToString(hash[:])

	// Check cache
	e.mu.RLock()
	if cached, ok := e.cache[hashKey]; ok {
		e.mu.RUnlock()
		return cached, nil
	}
	e.mu.RUnlock()

	// Generate random nonce
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := e.gcm.Seal(nonce, nonce, plaintext, nil)

	// Cache the result
	e.mu.Lock()
	e.cache[hashKey] = ciphertext
	e.mu.Unlock()

	// Persist cache
	if e.cachePath != "" {
		_ = e.SaveCache()
	}

	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM.
// Expects: nonce (12 bytes) + ciphertext + auth tag
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < NonceSize {
		return nil, ErrCiphertextTooShort
	}

	nonce := ciphertext[:NonceSize]
	encrypted := ciphertext[NonceSize:]

	plaintext, err := e.gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// cacheFilePath returns the path to the cache file.
func (e *Encryptor) cacheFilePath() string {
	return filepath.Join(e.cachePath, "iv-cache")
}

// loadCache loads the IV cache from disk.
func (e *Encryptor) loadCache() error {
	data, err := os.ReadFile(e.cacheFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Format: each entry is hashKey (64 hex) + ciphertext length (4 bytes) + ciphertext
	offset := 0
	for offset < len(data) {
		if offset+64+4 > len(data) {
			break
		}
		hashKey := string(data[offset : offset+64])
		offset += 64

		length := int(data[offset])<<24 | int(data[offset+1])<<16 | int(data[offset+2])<<8 | int(data[offset+3])
		offset += 4

		if offset+length > len(data) {
			break
		}
		ciphertext := make([]byte, length)
		copy(ciphertext, data[offset:offset+length])
		offset += length

		e.cache[hashKey] = ciphertext
	}

	return nil
}

// SaveCache saves the IV cache to disk.
func (e *Encryptor) SaveCache() error {
	if err := os.MkdirAll(e.cachePath, 0o700); err != nil {
		return err
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Pre-allocate with estimated size
	data := make([]byte, 0, len(e.cache)*(64+4+128))
	for hashKey, ciphertext := range e.cache {
		data = append(data, []byte(hashKey)...)
		length := len(ciphertext)
		data = append(data, byte(length>>24), byte(length>>16), byte(length>>8), byte(length))
		data = append(data, ciphertext...)
	}

	return os.WriteFile(e.cacheFilePath(), data, 0o600)
}

// ClearCache clears the in-memory and on-disk cache.
func (e *Encryptor) ClearCache() error {
	e.mu.Lock()
	e.cache = make(map[string][]byte)
	e.mu.Unlock()

	if e.cachePath != "" {
		return os.Remove(e.cacheFilePath())
	}
	return nil
}
