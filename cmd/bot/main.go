package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/subhanjanOps/torn-advisor/bot"
	"github.com/subhanjanOps/torn-advisor/config"
	"github.com/subhanjanOps/torn-advisor/domain"
	"github.com/subhanjanOps/torn-advisor/providers/torn"
	"github.com/subhanjanOps/torn-advisor/store"
	"github.com/subhanjanOps/tornSDK/client"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	botToken := os.Getenv("DISCORD_BOT_TOKEN")
	if botToken == "" {
		return fmt.Errorf("DISCORD_BOT_TOKEN environment variable is required")
	}

	appID := os.Getenv("DISCORD_APP_ID")
	if appID == "" {
		return fmt.Errorf("DISCORD_APP_ID environment variable is required")
	}

	encKey := os.Getenv("ENCRYPTION_KEY")
	if encKey == "" {
		return fmt.Errorf("ENCRYPTION_KEY environment variable is required (64 hex chars = 32 bytes)")
	}

	// Key store path (default: ./keys.json).
	storePath := os.Getenv("KEY_STORE_PATH")
	if storePath == "" {
		storePath = "keys.json"
	}

	// Load optional config.
	cfg := config.DefaultPriorities()
	if path := os.Getenv("ADVISOR_CONFIG"); path != "" {
		var err error
		cfg, err = config.LoadPriorities(path)
		if err != nil {
			log.Printf("Warning: failed to load config %s: %v (using defaults)", path, err)
			cfg = config.DefaultPriorities()
		}
	}

	// Create encrypted key store.
	ks, err := store.NewKeyStore(storePath, encKey)
	if err != nil {
		return fmt.Errorf("initializing key store: %w", err)
	}

	// Provider factory: creates a Torn provider for each user's API key.
	factory := func(apiKey string) domain.StateProvider {
		sdk := client.New(client.Config{APIKey: apiKey})
		return torn.NewProvider(sdk)
	}

	// Create Discord session.
	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return fmt.Errorf("creating discord session: %w", err)
	}

	// Create and start the bot.
	b := bot.New(dg, ks, factory, cfg, 30*time.Second)
	if err := b.RegisterAndStart(appID); err != nil {
		return fmt.Errorf("starting bot: %w", err)
	}
	defer b.Stop()

	// Start the scheduler (checks every 15 minutes).
	b.StartScheduler(15 * time.Minute)

	log.Printf("Torn Advisor bot is running (%d registered users). Press Ctrl+C to exit.", ks.UserCount())

	// Wait for interrupt signal.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down...")
	return nil
}
