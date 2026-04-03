package store

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func testEncKey() string {
	// Fixed 32-byte key for testing (not used in production).
	return hex.EncodeToString([]byte("01234567890123456789012345678901"))
}

func TestNewKeyStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.json")

	ks, err := NewKeyStore(path, testEncKey())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ks.UserCount() != 0 {
		t.Errorf("expected 0 users, got %d", ks.UserCount())
	}
}

func TestNewKeyStore_BadKeyLength(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.json")

	_, err := NewKeyStore(path, "aabbcc")
	if err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestNewKeyStore_BadKeyHex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.json")

	_, err := NewKeyStore(path, "not-hex-at-all!")
	if err == nil {
		t.Fatal("expected error for invalid hex")
	}
}

func TestSetGetDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.json")

	ks, err := NewKeyStore(path, testEncKey())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Set a key.
	if err := ks.Set("user123", "torn-api-key-abc"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if ks.UserCount() != 1 {
		t.Errorf("expected 1 user, got %d", ks.UserCount())
	}

	// Get the key back.
	key, ok := ks.Get("user123")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if key != "torn-api-key-abc" {
		t.Errorf("got %q, want %q", key, "torn-api-key-abc")
	}

	// Get non-existent.
	_, ok = ks.Get("unknown")
	if ok {
		t.Error("expected key not to exist")
	}

	// Delete.
	if err := ks.Delete("user123"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, ok = ks.Get("user123")
	if ok {
		t.Error("key should be deleted")
	}
}

func TestPersistenceAcrossRestarts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.json")
	encKey := testEncKey()

	// Create store and add a key.
	ks1, err := NewKeyStore(path, encKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ks1.Set("user1", "key1"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := ks1.Set("user2", "key2"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Create new store from same file — should load existing keys.
	ks2, err := NewKeyStore(path, encKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ks2.UserCount() != 2 {
		t.Errorf("expected 2 users, got %d", ks2.UserCount())
	}

	key, ok := ks2.Get("user1")
	if !ok || key != "key1" {
		t.Errorf("user1 key: got %q, ok=%v", key, ok)
	}
	key, ok = ks2.Get("user2")
	if !ok || key != "key2" {
		t.Errorf("user2 key: got %q, ok=%v", key, ok)
	}
}

func TestFileIsEncrypted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.json")

	ks, err := NewKeyStore(path, testEncKey())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ks.Set("user1", "my-secret-api-key"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Read raw file — should NOT contain the plaintext key.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	contents := string(data)
	if contains(contents, "my-secret-api-key") {
		t.Error("plaintext API key found in stored file — encryption failed")
	}
}

func TestWrongEncryptionKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.json")

	ks, err := NewKeyStore(path, testEncKey())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ks.Set("user1", "secret"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Try to load with a different key.
	differentKey := hex.EncodeToString([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ012345"))
	_, err = NewKeyStore(path, differentKey)
	if err == nil {
		t.Fatal("expected error when loading with wrong key")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
