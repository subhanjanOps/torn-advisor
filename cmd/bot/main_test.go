package main

import (
	"context"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/subhanjanOps/torn-advisor/config"
)

func clearEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		old := os.Getenv(k)
		_ = os.Unsetenv(k)
		t.Cleanup(func() {
			if old != "" {
				_ = os.Setenv(k, old)
			}
		})
	}
}

func validEncKey() string {
	return hex.EncodeToString([]byte("01234567890123456789012345678901"))
}

func TestRun_MissingBotToken(t *testing.T) {
	clearEnv(t, "DISCORD_BOT_TOKEN", "DISCORD_APP_ID", "ENCRYPTION_KEY")

	err := run()
	if err == nil || !strings.Contains(err.Error(), "DISCORD_BOT_TOKEN") {
		t.Fatalf("expected DISCORD_BOT_TOKEN error, got: %v", err)
	}
}

func TestRun_MissingAppID(t *testing.T) {
	clearEnv(t, "DISCORD_BOT_TOKEN", "DISCORD_APP_ID", "ENCRYPTION_KEY")
	t.Setenv("DISCORD_BOT_TOKEN", "test-token")

	err := run()
	if err == nil || !strings.Contains(err.Error(), "DISCORD_APP_ID") {
		t.Fatalf("expected DISCORD_APP_ID error, got: %v", err)
	}
}

func TestRun_MissingEncryptionKey(t *testing.T) {
	clearEnv(t, "DISCORD_BOT_TOKEN", "DISCORD_APP_ID", "ENCRYPTION_KEY")
	t.Setenv("DISCORD_BOT_TOKEN", "test-token")
	t.Setenv("DISCORD_APP_ID", "test-app-id")

	err := run()
	if err == nil || !strings.Contains(err.Error(), "ENCRYPTION_KEY") {
		t.Fatalf("expected ENCRYPTION_KEY error, got: %v", err)
	}
}

func TestRun_InvalidEncryptionKey(t *testing.T) {
	clearEnv(t, "DISCORD_BOT_TOKEN", "DISCORD_APP_ID", "ENCRYPTION_KEY", "KEY_STORE_PATH")
	t.Setenv("DISCORD_BOT_TOKEN", "test-token")
	t.Setenv("DISCORD_APP_ID", "test-app-id")
	t.Setenv("ENCRYPTION_KEY", "tooshort")
	t.Setenv("KEY_STORE_PATH", t.TempDir()+"/keys.json")

	err := run()
	if err == nil || !strings.Contains(err.Error(), "key store") {
		t.Fatalf("expected key store error, got: %v", err)
	}
}

func TestParseConfig_AllSet(t *testing.T) {
	clearEnv(t, "DISCORD_BOT_TOKEN", "DISCORD_APP_ID", "ENCRYPTION_KEY", "KEY_STORE_PATH", "ADVISOR_CONFIG")
	t.Setenv("DISCORD_BOT_TOKEN", "tok")
	t.Setenv("DISCORD_APP_ID", "app")
	t.Setenv("ENCRYPTION_KEY", validEncKey())

	bc, key, err := parseConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bc.botToken != "tok" || bc.appID != "app" || key != validEncKey() {
		t.Errorf("wrong config: %+v", bc)
	}
	if bc.storePath != "keys.json" {
		t.Errorf("expected default store path, got %s", bc.storePath)
	}
}

func TestParseConfig_CustomStorePath(t *testing.T) {
	clearEnv(t, "DISCORD_BOT_TOKEN", "DISCORD_APP_ID", "ENCRYPTION_KEY", "KEY_STORE_PATH", "ADVISOR_CONFIG")
	t.Setenv("DISCORD_BOT_TOKEN", "tok")
	t.Setenv("DISCORD_APP_ID", "app")
	t.Setenv("ENCRYPTION_KEY", validEncKey())
	t.Setenv("KEY_STORE_PATH", "/custom/keys.json")

	bc, _, err := parseConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bc.storePath != "/custom/keys.json" {
		t.Errorf("expected custom store path, got %s", bc.storePath)
	}
}

func TestParseConfig_InvalidConfig(t *testing.T) {
	clearEnv(t, "DISCORD_BOT_TOKEN", "DISCORD_APP_ID", "ENCRYPTION_KEY", "KEY_STORE_PATH", "ADVISOR_CONFIG")
	t.Setenv("DISCORD_BOT_TOKEN", "tok")
	t.Setenv("DISCORD_APP_ID", "app")
	t.Setenv("ENCRYPTION_KEY", validEncKey())
	t.Setenv("ADVISOR_CONFIG", "/nonexistent/config.json")

	bc, _, err := parseConfig()
	if err != nil {
		t.Fatalf("unexpected error (should use defaults): %v", err)
	}
	// Should fall back to defaults.
	if bc.cfg.Hospital != 98 {
		t.Errorf("expected default hospital priority 98, got %d", bc.cfg.Hospital)
	}
}

func TestSetupBot_Success(t *testing.T) {
	bc := botConfig{
		botToken:  "test-token",
		appID:     "test-app",
		storePath: t.TempDir() + "/keys.json",
		cfg:       config.DefaultPriorities(),
	}
	b, ks, err := setupBot(bc, validEncKey())
	if err != nil {
		t.Fatalf("setupBot: %v", err)
	}
	defer func() { _ = b.Stop() }()
	if ks == nil {
		t.Fatal("expected non-nil keystore")
	}
	if ks.UserCount() != 0 {
		t.Errorf("expected 0 users, got %d", ks.UserCount())
	}
}

func TestSetupBot_BadEncKey(t *testing.T) {
	bc := botConfig{
		botToken:  "test-token",
		appID:     "test-app",
		storePath: t.TempDir() + "/keys.json",
		cfg:       config.DefaultPriorities(),
	}
	_, _, err := setupBot(bc, "tooshort")
	if err == nil || !strings.Contains(err.Error(), "key store") {
		t.Fatalf("expected key store error, got: %v", err)
	}
}

func TestSetupBot_InvalidToken(t *testing.T) {
	// discordgo.New accepts any string, so this should succeed
	// even with an empty token — testing that the session is created.
	bc := botConfig{
		botToken:  "",
		appID:     "app",
		storePath: t.TempDir() + "/keys.json",
		cfg:       config.DefaultPriorities(),
	}
	b, _, err := setupBot(bc, validEncKey())
	if err != nil {
		t.Fatalf("setupBot with empty token: %v", err)
	}
	defer func() { _ = b.Stop() }()
}

func TestSetupBot_FactoryCreatesProvider(t *testing.T) {
	bc := botConfig{
		botToken:  "test-token",
		appID:     "test-app",
		storePath: t.TempDir() + "/keys.json",
		cfg:       config.DefaultPriorities(),
	}
	b, ks, err := setupBot(bc, validEncKey())
	if err != nil {
		t.Fatalf("setupBot: %v", err)
	}
	defer b.Stop()

	// Register a user so the factory can be triggered.
	if err := ks.Set("u1", "fake-api-key"); err != nil {
		t.Fatalf("registering user: %v", err)
	}

	// BuildAdviseResponse triggers getProvider → factory call.
	// The actual API call will fail (fake key), but the factory itself still executes.
	resp := b.BuildAdviseResponse("u1")
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestRunBot(t *testing.T) {
	bc := botConfig{
		botToken:  "test-token",
		appID:     "test-app",
		storePath: t.TempDir() + "/keys.json",
		cfg:       config.DefaultPriorities(),
	}
	b, ks, err := setupBot(bc, validEncKey())
	if err != nil {
		t.Fatalf("setupBot: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so runBot returns right away

	if err := runBot(ctx, b, ks); err != nil {
		t.Fatalf("runBot: %v", err)
	}
}
