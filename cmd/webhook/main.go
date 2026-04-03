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

func run() error {
	publicKey := os.Getenv("DISCORD_PUBLIC_KEY")
	if publicKey == "" {
		return fmt.Errorf("DISCORD_PUBLIC_KEY environment variable is required")
	}

	encKey := os.Getenv("ENCRYPTION_KEY")
	if encKey == "" {
		return fmt.Errorf("ENCRYPTION_KEY environment variable is required (64 hex chars = 32 bytes)")
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

	ks, err := store.NewKeyStore(storePath, encKey)
	if err != nil {
		return fmt.Errorf("initializing key store: %w", err)
	}

	factory := func(apiKey string) domain.StateProvider {
		sdk := client.New(client.Config{APIKey: apiKey})
		return torn.NewProvider(sdk)
	}

	// In webhook mode no Discord gateway session is needed.
	b := bot.New(nil, ks, factory, cfg, 30*time.Second)
	defer b.Stop()

	h, err := webhook.NewHandler(b, publicKey)
	if err != nil {
		return fmt.Errorf("creating webhook handler: %w", err)
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		log.Println("Shutting down webhook server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Printf("Webhook server listening on :%s", port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
