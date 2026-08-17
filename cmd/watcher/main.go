package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/naldrey/real-estate-watch/internal/config"
	"github.com/naldrey/real-estate-watch/internal/ingest"
	"github.com/naldrey/real-estate-watch/internal/mail"
	"github.com/naldrey/real-estate-watch/internal/monitor"
	"github.com/naldrey/real-estate-watch/internal/store"
	"github.com/naldrey/real-estate-watch/internal/telegram"

	_ "github.com/naldrey/real-estate-watch/internal/portal/fotocasa"   // registers the fotocasa parser
	_ "github.com/naldrey/real-estate-watch/internal/portal/habitaclia" // registers the habitaclia parser
	_ "github.com/naldrey/real-estate-watch/internal/portal/idealista"  // registers the idealista parser
	_ "github.com/naldrey/real-estate-watch/internal/portal/yaencontre" // registers the yaencontre parser
	// TODO: blank-import pisos.com's parser package here once it exists.
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("starting up", "app", "real-estate-watch")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

// run loads configuration once, then polls for new listings on
// cfg.PollInterval until ctx is canceled (SIGINT/SIGTERM).
func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := store.Open(ctx, cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer db.Close()

	var notifier ingest.Notifier
	var alerter monitor.Alerter
	if cfg.TelegramBotToken != "" {
		tg := telegram.NewClient(cfg.TelegramBotToken, cfg.TelegramChatID)
		notifier = tg
		alerter = tg
		slog.Info("telegram notifications enabled")
	}
	health := monitor.New(alerter)

	slog.Info("polling for new listings", "interval", cfg.PollInterval)

	poll(ctx, cfg, db, notifier, health)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("shutting down")
			return nil
		case <-ticker.C:
			poll(ctx, cfg, db, notifier, health)
		}
	}
}

// poll runs a single ingest cycle and reports its outcome to health, which
// alerts (via Telegram, if configured) only when polling starts failing or
// recovers - not on every cycle - so a stopped-working watcher doesn't fail
// silently.
func poll(ctx context.Context, cfg config.Config, db *store.Store, notifier ingest.Notifier, health *monitor.PollHealth) {
	if err := pollOnce(ctx, cfg, db, notifier); err != nil {
		slog.Error("poll failed", "err", err)
		health.Failure(ctx, err)
		return
	}

	health.Success(ctx)
	slog.Info("ingest complete")
}

func pollOnce(ctx context.Context, cfg config.Config, db *store.Store, notifier ingest.Notifier) error {
	mailClient := mail.NewClient(cfg.IMAPHost, cfg.IMAPUsername, cfg.IMAPPassword)
	if err := mailClient.Connect(ctx); err != nil {
		return fmt.Errorf("connect to imap: %w", err)
	}
	defer mailClient.Close()

	if err := ingest.Run(ctx, mailClient, db, ingest.Registered(), notifier); err != nil {
		return fmt.Errorf("run ingest: %w", err)
	}

	return nil
}
