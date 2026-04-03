package main

import (
	"bytes"
	"context"
	"errors"
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
