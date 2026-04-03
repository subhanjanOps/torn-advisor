package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

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

func TestRun_MissingPublicKey(t *testing.T) {
	clearEnv(t, "DISCORD_PUBLIC_KEY", "ENCRYPTION_KEY")

	err := run()
	if err == nil || !strings.Contains(err.Error(), "DISCORD_PUBLIC_KEY") {
		t.Fatalf("expected DISCORD_PUBLIC_KEY error, got: %v", err)
	}
}

func TestRun_MissingEncryptionKey(t *testing.T) {
	clearEnv(t, "DISCORD_PUBLIC_KEY", "ENCRYPTION_KEY")
	t.Setenv("DISCORD_PUBLIC_KEY", "aabbcc")

	err := run()
	if err == nil || !strings.Contains(err.Error(), "ENCRYPTION_KEY") {
		t.Fatalf("expected ENCRYPTION_KEY error, got: %v", err)
	}
}

func TestRun_InvalidEncryptionKey(t *testing.T) {
	clearEnv(t, "DISCORD_PUBLIC_KEY", "ENCRYPTION_KEY", "KEY_STORE_PATH")
	t.Setenv("DISCORD_PUBLIC_KEY", "aabbcc")
	t.Setenv("ENCRYPTION_KEY", "tooshort")
	t.Setenv("KEY_STORE_PATH", t.TempDir()+"/keys.json")

	err := run()
	if err == nil || !strings.Contains(err.Error(), "key store") {
		t.Fatalf("expected key store error, got: %v", err)
	}
}

func TestRun_InvalidPublicKey(t *testing.T) {
	clearEnv(t, "DISCORD_PUBLIC_KEY", "ENCRYPTION_KEY", "KEY_STORE_PATH")
	t.Setenv("DISCORD_PUBLIC_KEY", "not-valid-hex")
	t.Setenv("ENCRYPTION_KEY", validEncKey())
	t.Setenv("KEY_STORE_PATH", t.TempDir()+"/keys.json")

	err := run()
	if err == nil || !strings.Contains(err.Error(), "webhook handler") {
		t.Fatalf("expected webhook handler error, got: %v", err)
	}
}

func TestRun_WrongPublicKeyLength(t *testing.T) {
	clearEnv(t, "DISCORD_PUBLIC_KEY", "ENCRYPTION_KEY", "KEY_STORE_PATH")
	t.Setenv("DISCORD_PUBLIC_KEY", "aabb")
	t.Setenv("ENCRYPTION_KEY", validEncKey())
	t.Setenv("KEY_STORE_PATH", t.TempDir()+"/keys.json")

	err := run()
	if err == nil || !strings.Contains(err.Error(), "webhook handler") {
		t.Fatalf("expected webhook handler error, got: %v", err)
	}
}

func TestParseConfig_AllSet(t *testing.T) {
	clearEnv(t, "DISCORD_PUBLIC_KEY", "ENCRYPTION_KEY", "KEY_STORE_PATH", "WEBHOOK_PORT", "ADVISOR_CONFIG")
	t.Setenv("DISCORD_PUBLIC_KEY", "aabb")
	t.Setenv("ENCRYPTION_KEY", validEncKey())

	wc, key, err := parseConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wc.publicKey != "aabb" || key != validEncKey() {
		t.Error("wrong config values")
	}
	if wc.storePath != "keys.json" {
		t.Errorf("expected default store path, got %s", wc.storePath)
	}
	if wc.port != "8080" {
		t.Errorf("expected default port 8080, got %s", wc.port)
	}
}

func TestParseConfig_CustomValues(t *testing.T) {
	clearEnv(t, "DISCORD_PUBLIC_KEY", "ENCRYPTION_KEY", "KEY_STORE_PATH", "WEBHOOK_PORT", "ADVISOR_CONFIG")
	t.Setenv("DISCORD_PUBLIC_KEY", "aabb")
	t.Setenv("ENCRYPTION_KEY", validEncKey())
	t.Setenv("KEY_STORE_PATH", "/data/keys.json")
	t.Setenv("WEBHOOK_PORT", "9090")

	wc, _, err := parseConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wc.storePath != "/data/keys.json" {
		t.Errorf("expected custom store path, got %s", wc.storePath)
	}
	if wc.port != "9090" {
		t.Errorf("expected port 9090, got %s", wc.port)
	}
}

func TestParseConfig_InvalidConfig(t *testing.T) {
	clearEnv(t, "DISCORD_PUBLIC_KEY", "ENCRYPTION_KEY", "KEY_STORE_PATH", "WEBHOOK_PORT", "ADVISOR_CONFIG")
	t.Setenv("DISCORD_PUBLIC_KEY", "aabb")
	t.Setenv("ENCRYPTION_KEY", validEncKey())
	t.Setenv("ADVISOR_CONFIG", "/nonexistent/config.json")

	wc, _, err := parseConfig()
	if err != nil {
		t.Fatalf("unexpected error (should use defaults): %v", err)
	}
	if wc.cfg.Hospital != 98 {
		t.Errorf("expected default hospital priority 98, got %d", wc.cfg.Hospital)
	}
}

func TestSetupServer_Success(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	wc := webhookConfig{
		publicKey: hex.EncodeToString(pub),
		storePath: t.TempDir() + "/keys.json",
		port:      "0",
		cfg:       defaultCfg(),
	}
	srv, b, err := setupServer(wc, validEncKey())
	if err != nil {
		t.Fatalf("setupServer: %v", err)
	}
	defer b.Stop()
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.Addr != ":0" {
		t.Errorf("expected addr :0, got %s", srv.Addr)
	}
}

func TestSetupServer_BadEncKey(t *testing.T) {
	wc := webhookConfig{
		publicKey: "aabb",
		storePath: t.TempDir() + "/keys.json",
		port:      "0",
		cfg:       defaultCfg(),
	}
	_, _, err := setupServer(wc, "tooshort")
	if err == nil || !strings.Contains(err.Error(), "key store") {
		t.Fatalf("expected key store error, got: %v", err)
	}
}

func TestSetupServer_BadPublicKey(t *testing.T) {
	wc := webhookConfig{
		publicKey: "not-valid-hex",
		storePath: t.TempDir() + "/keys.json",
		port:      "0",
		cfg:       defaultCfg(),
	}
	_, _, err := setupServer(wc, validEncKey())
	if err == nil || !strings.Contains(err.Error(), "webhook handler") {
		t.Fatalf("expected webhook handler error, got: %v", err)
	}
}

func TestSetupServer_WrongKeyLength(t *testing.T) {
	wc := webhookConfig{
		publicKey: "aabb",
		storePath: t.TempDir() + "/keys.json",
		port:      "0",
		cfg:       defaultCfg(),
	}
	_, _, err := setupServer(wc, validEncKey())
	if err == nil || !strings.Contains(err.Error(), "webhook handler") {
		t.Fatalf("expected webhook handler error, got: %v", err)
	}
}

func TestSetupServer_ListenAndShutdown(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	wc := webhookConfig{
		publicKey: hex.EncodeToString(pub),
		storePath: t.TempDir() + "/keys.json",
		port:      "0",
		cfg:       defaultCfg(),
	}
	srv, b, err := setupServer(wc, validEncKey())
	if err != nil {
		t.Fatalf("setupServer: %v", err)
	}
	defer b.Stop()

	// Use a listener so we get an ephemeral port.
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	// Verify the server responds.
	resp, err := http.Get("http://" + ln.Addr().String() + "/")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	_ = resp.Body.Close()

	// Graceful shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	if srvErr := <-errCh; srvErr != http.ErrServerClosed {
		t.Fatalf("expected ErrServerClosed, got: %v", srvErr)
	}
}

func TestStartAndServe(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	wc := webhookConfig{
		publicKey: hex.EncodeToString(pub),
		storePath: t.TempDir() + "/keys.json",
		port:      "0",
		cfg:       defaultCfg(),
	}
	srv, b, err := setupServer(wc, validEncKey())
	if err != nil {
		t.Fatalf("setupServer: %v", err)
	}

	// Use an ephemeral port via a real listener.
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	// Override the server's Addr so ListenAndServe won't be called.
	// We'll use Serve directly after starting startAndServe with a pre-cancelled context.

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		// Use Serve on the listener; startAndServe calls ListenAndServe which needs a real addr.
		// Instead, we'll directly call startAndServe with the srv configured to our listener port.
		srv.Addr = ln.Addr().String()
		_ = ln.Close() // free the port so ListenAndServe can bind
		errCh <- startAndServe(ctx, srv, b)
	}()

	// Wait for server to start.
	addr := "http://" + srv.Addr + "/"
	for i := 0; i < 30; i++ {
		time.Sleep(50 * time.Millisecond)
		resp, reqErr := http.Get(addr)
		if reqErr == nil {
			_ = resp.Body.Close()
			break
		}
	}

	// Cancel context to trigger shutdown.
	cancel()

	srvErr := <-errCh
	if srvErr != nil {
		t.Fatalf("startAndServe: %v", srvErr)
	}
}

func defaultCfg() config.RulePriorities {
	return config.DefaultPriorities()
}

func TestStartAndServe_ListenError(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	wc := webhookConfig{
		publicKey: hex.EncodeToString(pub),
		storePath: t.TempDir() + "/keys.json",
		port:      "0",
		cfg:       defaultCfg(),
	}
	srv, b, err := setupServer(wc, validEncKey())
	if err != nil {
		t.Fatalf("setupServer: %v", err)
	}

	// Bind a port so the server's ListenAndServe fails.
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	srv.Addr = ln.Addr().String()
	ctx := context.Background()

	srvErr := startAndServe(ctx, srv, b)
	if srvErr == nil || !strings.Contains(srvErr.Error(), "server error") {
		t.Fatalf("expected server error, got: %v", srvErr)
	}
}
