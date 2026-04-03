package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/subhanjanOps/torn-advisor/bot"
	"github.com/subhanjanOps/torn-advisor/config"
	"github.com/subhanjanOps/torn-advisor/domain"
	"github.com/subhanjanOps/torn-advisor/providers/torn"
	"github.com/subhanjanOps/torn-advisor/store"
	"github.com/subhanjanOps/torn-advisor/webhook"
	"github.com/subhanjanOps/tornSDK/client"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// webhookConfig holds validated webhook configuration.
type webhookConfig struct {
	publicKey string
	storePath string
	port      string
	cfg       config.RulePriorities
}

// parseConfig reads and validates environment variables.
func parseConfig() (webhookConfig, string, error) {
	publicKey := os.Getenv("DISCORD_PUBLIC_KEY")
	if publicKey == "" {
		return webhookConfig{}, "", fmt.Errorf("DISCORD_PUBLIC_KEY environment variable is required")
	}

	encKey := os.Getenv("ENCRYPTION_KEY")
	if encKey == "" {
		return webhookConfig{}, "", fmt.Errorf("ENCRYPTION_KEY environment variable is required (64 hex chars = 32 bytes)")
	}

	storePath := os.Getenv("KEY_STORE_PATH")
	if storePath == "" {
		storePath = "keys.json"
	}

	port := os.Getenv("WEBHOOK_PORT")
	if port == "" {
		port = "8080"
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

	return webhookConfig{
		publicKey: publicKey,
		storePath: storePath,
		port:      port,
		cfg:       cfg,
	}, encKey, nil
}

func run() error {
	wc, encKey, err := parseConfig()
	if err != nil {
		return err
	}

	srv, b, err := setupServer(wc, encKey)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("Webhook server listening on :%s", wc.port)
	return startAndServe(ctx, srv, b)
}

// startAndServe runs the HTTP server and shuts it down when ctx is cancelled.
func startAndServe(ctx context.Context, srv *http.Server, b *bot.Bot) error {
	defer b.Stop()

	go func() {
		<-ctx.Done()
		log.Println("Shutting down webhook server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// setupServer creates the keystore, bot, webhook handler, and HTTP server.
func setupServer(wc webhookConfig, encKey string) (*http.Server, *bot.Bot, error) {
	ks, err := store.NewKeyStore(wc.storePath, encKey)
	if err != nil {
		return nil, nil, fmt.Errorf("initializing key store: %w", err)
	}

	factory := func(apiKey string) domain.StateProvider {
		sdk := client.New(client.Config{APIKey: apiKey})
		return torn.NewProvider(sdk)
	}

	b := bot.New(nil, ks, factory, wc.cfg, 30*time.Second)

	h, err := webhook.NewHandler(b, wc.publicKey)
	if err != nil {
		b.Stop()
		return nil, nil, fmt.Errorf("creating webhook handler: %w", err)
	}

	srv := &http.Server{
		Addr:              ":" + wc.port,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return srv, b, nil
}
