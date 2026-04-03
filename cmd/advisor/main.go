package main

import (
	"context"
	"fmt"
	"os"

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

	// Initialize SDK client.
	sdk := client.New(client.Config{
		APIKey: apiKey,
	})

	// Create provider and fetch player state.
	provider := torn.NewProvider(sdk)
	state, err := provider.FetchPlayerState(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching player state: %v\n", err)
		os.Exit(1)
	}

	// Run the advisor engine with default rules.
	eng := engine.NewEngine(rules.DefaultRules())
	plan := eng.Run(state)

	// Print the action plan.
	if len(plan) == 0 {
		fmt.Println("No actions recommended right now.")
		return
	}

	fmt.Println("=== Torn Advisor — Action Plan ===")
	fmt.Println()
	for i, action := range plan {
		fmt.Printf("%d. [%s] %s (priority %d)\n", i+1, action.Category, action.Name, action.Priority)
		fmt.Printf("   %s\n\n", action.Description)
	}
}
