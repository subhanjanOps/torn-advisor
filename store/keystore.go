package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// KeyStore manages per-user encrypted Torn API keys.
type KeyStore struct {
	mu       sync.RWMutex
	keys     map[string]string // discordUserID -> torn API key (plaintext in memory)
	filePath string
	gcm      cipher.AEAD
}

// filePayload is the JSON structure stored on disk.
type filePayload struct {
	// Keys maps discordUserID -> hex-encoded encrypted API key.
	Keys map[string]string `json:"keys"`
}

// NewKeyStore creates a KeyStore that encrypts keys with the given 32-byte hex secret
// and persists them to filePath.
func NewKeyStore(filePath string, encryptionKeyHex string) (*KeyStore, error) {
	keyBytes, err := hex.DecodeString(encryptionKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decoding encryption key: %w", err)
	}
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (64 hex chars), got %d bytes", len(keyBytes))
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	ks := &KeyStore{
		keys:     make(map[string]string),
		filePath: filePath,
		gcm:      gcm,
	}

	if err := ks.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading key store: %w", err)
	}

	return ks, nil
}

// Set stores a Torn API key for the given Discord user.
func (ks *KeyStore) Set(discordUserID, apiKey string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.keys[discordUserID] = apiKey
	return ks.save()
}

// Get retrieves the Torn API key for the given Discord user.
// Returns empty string and false if not found.
func (ks *KeyStore) Get(discordUserID string) (string, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	key, ok := ks.keys[discordUserID]
	return key, ok
}

// Delete removes the stored key for the given Discord user.
func (ks *KeyStore) Delete(discordUserID string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	delete(ks.keys, discordUserID)
	return ks.save()
}

// UserCount returns the number of registered users.
func (ks *KeyStore) UserCount() int {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return len(ks.keys)
}

func (ks *KeyStore) encrypt(plaintext string) (string, error) {
	nonce := make([]byte, ks.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}
	ciphertext := ks.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

func (ks *KeyStore) decrypt(ciphertextHex string) (string, error) {
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", fmt.Errorf("decoding ciphertext: %w", err)
	}

	nonceSize := ks.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := ks.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypting: %w", err)
	}
	return string(plaintext), nil
}

func (ks *KeyStore) save() error {
	payload := filePayload{Keys: make(map[string]string, len(ks.keys))}
	for userID, apiKey := range ks.keys {
		enc, err := ks.encrypt(apiKey)
		if err != nil {
			return fmt.Errorf("encrypting key for %s: %w", userID, err)
		}
		payload.Keys[userID] = enc
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling store: %w", err)
	}

	return os.WriteFile(ks.filePath, data, 0600)
}

func (ks *KeyStore) load() error {
	data, err := os.ReadFile(ks.filePath)
	if err != nil {
		return err
	}

	var payload filePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("parsing store: %w", err)
	}

	for userID, enc := range payload.Keys {
		plain, err := ks.decrypt(enc)
		if err != nil {
			return fmt.Errorf("decrypting key for %s: %w", userID, err)
		}
		ks.keys[userID] = plain
	}

	return nil
}
