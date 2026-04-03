package main

import (
	"context"
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

// botConfig holds validated configuration for the bot.
type botConfig struct {
	botToken  string
	appID     string
	storePath string
	cfg       config.RulePriorities
}

// parseConfig reads and validates environment variables.
func parseConfig() (botConfig, string, error) {
	botToken := os.Getenv("DISCORD_BOT_TOKEN")
	if botToken == "" {
		return botConfig{}, "", fmt.Errorf("DISCORD_BOT_TOKEN environment variable is required")
	}

	appID := os.Getenv("DISCORD_APP_ID")
	if appID == "" {
		return botConfig{}, "", fmt.Errorf("DISCORD_APP_ID environment variable is required")
	}

	encKey := os.Getenv("ENCRYPTION_KEY")
	if encKey == "" {
		return botConfig{}, "", fmt.Errorf("ENCRYPTION_KEY environment variable is required (64 hex chars = 32 bytes)")
	}

	storePath := os.Getenv("KEY_STORE_PATH")
	if storePath == "" {
		storePath = "keys.json"
	}

	cfg := config.DefaultPriorities()
	if path := os.Getenv("ADVISOR_CONFIG"); path != "" {
		var err error
		cfg, err = config.LoadPriorities(path)
		if err != nil {
			log.Printf("Warning: failed to load config %s: %v (using defaults)", path, err)
			cfg = config.DefaultPriorities()
		}
	}

	return botConfig{
		botToken:  botToken,
		appID:     appID,
		storePath: storePath,
		cfg:       cfg,
	}, encKey, nil
}

func run() error {
	bc, encKey, err := parseConfig()
	if err != nil {
		return err
	}

	b, ks, err := setupBot(bc, encKey)
	if err != nil {
		return err
	}

	if err := b.RegisterAndStart(bc.appID); err != nil {
		return fmt.Errorf("starting bot: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return runBot(ctx, b, ks)
}

// runBot starts the scheduler, logs status, and waits for context cancellation.
func runBot(ctx context.Context, b *bot.Bot, ks *store.KeyStore) error {
	defer func() { _ = b.Stop() }()
	b.StartScheduler(15 * time.Minute)

	log.Printf("Torn Advisor bot is running (%d registered users). Press Ctrl+C to exit.", ks.UserCount())

	<-ctx.Done()
	log.Println("Shutting down...")
	return nil
}

// setupBot creates the keystore, discord session, and bot instance.
func setupBot(bc botConfig, encKey string) (*bot.Bot, *store.KeyStore, error) {
	ks, err := store.NewKeyStore(bc.storePath, encKey)
	if err != nil {
		return nil, nil, fmt.Errorf("initializing key store: %w", err)
	}

	factory := func(apiKey string) domain.StateProvider {
		sdk := client.New(client.Config{APIKey: apiKey})
		return torn.NewProvider(sdk)
	}

	dg, err := discordgo.New("Bot " + bc.botToken)
	if err != nil {
		return nil, nil, fmt.Errorf("creating discord session: %w", err)
	}

	b := bot.New(dg, ks, factory, bc.cfg, 30*time.Second)
	return b, ks, nil
}
