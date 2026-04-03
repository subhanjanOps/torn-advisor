//go:build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/subhanjanOps/torn-advisor/engine"
	"github.com/subhanjanOps/torn-advisor/providers/torn"
	"github.com/subhanjanOps/torn-advisor/rules"
	"github.com/subhanjanOps/tornSDK/client"
)

func TestIntegration_FullPipeline(t *testing.T) {
	apiKey := os.Getenv("TORN_API_KEY")
	if apiKey == "" {
		t.Skip("TORN_API_KEY not set, skipping integration test")
	}

	sdk := client.New(client.Config{
		APIKey: apiKey,
	})

	// Fetch real player state.
	provider := torn.NewProvider(sdk)
	state, err := provider.FetchPlayerState(context.Background())
	if err != nil {
		t.Fatalf("FetchPlayerState failed: %v", err)
	}

	// Verify state has reasonable values.
	if state.EnergyMax == 0 {
		t.Error("EnergyMax should not be 0")
	}
	if state.NerveMax == 0 {
		t.Error("NerveMax should not be 0")
	}
	if state.LifeMax == 0 {
		t.Error("LifeMax should not be 0")
	}

	// Run engine and verify it produces a plan without panicking.
	eng := engine.NewEngine(rules.DefaultRules())
	plan := eng.Run(state)

	t.Logf("Player state: Energy=%d/%d, Nerve=%d/%d, Happy=%d, Life=%d/%d",
		state.Energy, state.EnergyMax,
		state.Nerve, state.NerveMax,
		state.Happy,
		state.Life, state.LifeMax,
	)
	t.Logf("Actions recommended: %d", len(plan))
	for i, a := range plan {
		t.Logf("  %d. [%s] %s (priority %d)", i+1, a.Category, a.Name, a.Priority)
	}
}
