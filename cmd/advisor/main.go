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
	apiKey := os.Getenv("TORN_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "TORN_API_KEY environment variable is required")
		os.Exit(1)
	}

	sdk := client.New(client.Config{
		APIKey: apiKey,
	})

	provider := torn.NewProvider(sdk)

	cfg := config.DefaultPriorities()
	if path := os.Getenv("ADVISOR_CONFIG"); path != "" {
		var err error
		cfg, err = config.LoadPriorities(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load config %s: %v (using defaults)\n", path, err)
			cfg = config.DefaultPriorities()
		}
	}

	if err := run(provider, cfg, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
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
