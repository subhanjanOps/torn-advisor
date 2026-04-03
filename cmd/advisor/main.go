package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/subhanjanOps/torn-advisor/config"
	"github.com/subhanjanOps/torn-advisor/domain"
	"github.com/subhanjanOps/torn-advisor/engine"
	"github.com/subhanjanOps/torn-advisor/providers/torn"
	"github.com/subhanjanOps/torn-advisor/rules"
	"github.com/subhanjanOps/tornSDK/client"
)

func main() {
	if err := mainRun(os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}

// newSDKProvider creates a real tornSDK-backed provider.
var newSDKProvider = func(apiKey string) domain.StateProvider {
	sdk := client.New(client.Config{APIKey: apiKey})
	return torn.NewProvider(sdk)
}

// mainRun contains all the logic of main() but returns an error instead of calling os.Exit.
func mainRun(stdout, stderr io.Writer) error {
	apiKey := os.Getenv("TORN_API_KEY")
	if apiKey == "" {
		_, _ = fmt.Fprintln(stderr, "TORN_API_KEY environment variable is required")
		return fmt.Errorf("TORN_API_KEY not set")
	}

	provider := newSDKProvider(apiKey)

	cfg := config.DefaultPriorities()
	if path := os.Getenv("ADVISOR_CONFIG"); path != "" {
		var err error
		cfg, err = config.LoadPriorities(path)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "Warning: failed to load config %s: %v (using defaults)\n", path, err)
			cfg = config.DefaultPriorities()
		}
	}

	return run(provider, cfg, stdout)
}

func run(provider domain.StateProvider, cfg config.RulePriorities, w io.Writer) error {
	state, err := provider.FetchPlayerState(context.Background())
	if err != nil {
		return fmt.Errorf("fetching player state: %w", err)
	}

	eng := engine.NewEngine(rules.DefaultRulesWithConfig(cfg))
	plan := eng.Run(state)

	if len(plan) == 0 {
		_, _ = fmt.Fprintln(w, "No actions recommended right now.")
		return nil
	}

	_, _ = fmt.Fprintln(w, "=== Torn Advisor — Action Plan ===")
	_, _ = fmt.Fprintln(w)
	for i, action := range plan {
		_, _ = fmt.Fprintf(w, "%d. [%s] %s (priority %d)\n", i+1, action.Category, action.Name, action.Priority)
		_, _ = fmt.Fprintf(w, "   %s\n\n", action.Description)
	}
	return nil
}
