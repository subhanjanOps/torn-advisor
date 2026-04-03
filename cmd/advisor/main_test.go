package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/subhanjanOps/torn-advisor/config"
	"github.com/subhanjanOps/torn-advisor/domain"
)

// mockProvider implements domain.StateProvider for testing.
type mockProvider struct {
	state domain.PlayerState
	err   error
}

func (m *mockProvider) FetchPlayerState(_ context.Context) (domain.PlayerState, error) {
	return m.state, m.err
}

func TestRun_WithActions(t *testing.T) {
	provider := &mockProvider{
		state: domain.PlayerState{
			Energy:        100,
			Happy:         5000,
			Nerve:         60,
			NerveMax:      60,
			Life:          7500,
			LifeMax:       7500,
			XanaxCooldown: 300,
		},
	}

	var buf bytes.Buffer
	err := run(provider, config.DefaultPriorities(), &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Action Plan") {
		t.Error("expected output to contain 'Action Plan'")
	}
	if !strings.Contains(output, "Train at Gym") {
		t.Error("expected output to contain 'Train at Gym'")
	}
}

func TestRun_NoActions(t *testing.T) {
	provider := &mockProvider{
		state: domain.PlayerState{
			XanaxCooldown:   300,
			BoosterCooldown: 300,
			TravelCooldown:  600,
			Life:            7500,
			LifeMax:         7500,
		},
	}

	var buf bytes.Buffer
	err := run(provider, config.DefaultPriorities(), &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No actions recommended") {
		t.Errorf("expected 'No actions recommended', got: %s", output)
	}
}

func TestRun_ProviderError(t *testing.T) {
	provider := &mockProvider{
		err: errors.New("api down"),
	}

	var buf bytes.Buffer
	err := run(provider, config.DefaultPriorities(), &buf)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "api down") {
		t.Errorf("expected error to contain 'api down', got: %v", err)
	}
}

// --- mainRun tests ---

func TestMainRun_NoAPIKey(t *testing.T) {
	t.Setenv("TORN_API_KEY", "")
	t.Setenv("ADVISOR_CONFIG", "")

	var stdout, stderr bytes.Buffer
	err := mainRun(&stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when TORN_API_KEY is empty")
	}
	if !strings.Contains(stderr.String(), "TORN_API_KEY") {
		t.Errorf("expected stderr to mention TORN_API_KEY, got: %s", stderr.String())
	}
}

func TestMainRun_Success(t *testing.T) {
	t.Setenv("TORN_API_KEY", "test-key")
	t.Setenv("ADVISOR_CONFIG", "")

	origProvider := newSDKProvider
	defer func() { newSDKProvider = origProvider }()

	newSDKProvider = func(_ string) domain.StateProvider {
		return &mockProvider{
			state: domain.PlayerState{
				Energy:  100,
				Happy:   5000,
				Life:    7500,
				LifeMax: 7500,
			},
		}
	}

	var stdout, stderr bytes.Buffer
	err := mainRun(&stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Train at Gym") {
		t.Errorf("expected output to contain 'Train at Gym', got: %s", stdout.String())
	}
}

func TestMainRun_WithValidConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	err := os.WriteFile(cfgPath, []byte(`{"gym": 99}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("TORN_API_KEY", "test-key")
	t.Setenv("ADVISOR_CONFIG", cfgPath)

	origProvider := newSDKProvider
	defer func() { newSDKProvider = origProvider }()

	newSDKProvider = func(_ string) domain.StateProvider {
		return &mockProvider{
			state: domain.PlayerState{
				Energy:  100,
				Happy:   5000,
				Life:    7500,
				LifeMax: 7500,
			},
		}
	}

	var stdout, stderr bytes.Buffer
	err = mainRun(&stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "priority 99") {
		t.Errorf("expected custom priority 99 in output, got: %s", stdout.String())
	}
}

func TestMainRun_WithInvalidConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "bad.json")
	err := os.WriteFile(cfgPath, []byte(`{bad`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("TORN_API_KEY", "test-key")
	t.Setenv("ADVISOR_CONFIG", cfgPath)

	origProvider := newSDKProvider
	defer func() { newSDKProvider = origProvider }()

	newSDKProvider = func(_ string) domain.StateProvider {
		return &mockProvider{
			state: domain.PlayerState{
				Energy:  100,
				Happy:   5000,
				Life:    7500,
				LifeMax: 7500,
			},
		}
	}

	var stdout, stderr bytes.Buffer
	err = mainRun(&stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should warn on stderr but still work with defaults.
	if !strings.Contains(stderr.String(), "Warning") {
		t.Errorf("expected warning on stderr, got: %s", stderr.String())
	}
	// Should still produce output using defaults.
	if !strings.Contains(stdout.String(), "Train at Gym") {
		t.Errorf("expected output with defaults, got: %s", stdout.String())
	}
}

func TestMainRun_WithMissingConfigFile(t *testing.T) {
	t.Setenv("TORN_API_KEY", "test-key")
	t.Setenv("ADVISOR_CONFIG", "/nonexistent/config.json")

	origProvider := newSDKProvider
	defer func() { newSDKProvider = origProvider }()

	newSDKProvider = func(_ string) domain.StateProvider {
		return &mockProvider{
			state: domain.PlayerState{Life: 7500, LifeMax: 7500},
		}
	}

	var stdout, stderr bytes.Buffer
	err := mainRun(&stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Warning") {
		t.Errorf("expected warning on stderr, got: %s", stderr.String())
	}
}

func TestMainRun_ProviderError(t *testing.T) {
	t.Setenv("TORN_API_KEY", "test-key")
	t.Setenv("ADVISOR_CONFIG", "")

	origProvider := newSDKProvider
	defer func() { newSDKProvider = origProvider }()

	newSDKProvider = func(_ string) domain.StateProvider {
		return &mockProvider{err: errors.New("connection refused")}
	}

	var stdout, stderr bytes.Buffer
	err := mainRun(&stdout, &stderr)
	if err == nil {
		t.Fatal("expected error from provider")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("expected 'connection refused' in error, got: %v", err)
	}
}

func TestNewSDKProvider_ReturnsProvider(t *testing.T) {
	// Exercise the default newSDKProvider — it creates a real SDK client
	// but doesn't make any HTTP calls.
	provider := newSDKProvider("dummy-key")
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}
