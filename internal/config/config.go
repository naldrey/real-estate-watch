// Package config loads runtime configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds runtime configuration.
type Config struct {
	IMAPHost     string
	IMAPUsername string
	IMAPPassword string
	DBPath       string

	// TelegramBotToken and TelegramChatID are both empty if Telegram
	// notifications are disabled.
	TelegramBotToken string
	TelegramChatID   string

	PollInterval time.Duration
}

// Load reads configuration from environment variables, applying defaults
// where sensible and failing if a required secret is missing. A .env file in
// the current directory, if present, is loaded first; real environment
// variables always take priority over it.
func Load() (Config, error) {
	if err := loadDotEnv(); err != nil {
		return Config{}, err
	}

	cfg := Config{
		IMAPHost:         envOrDefault("IMAP_HOST", "imap.fastmail.com:993"),
		DBPath:           envOrDefault("DB_PATH", "real-estate-watch.db"),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:   os.Getenv("TELEGRAM_CHAT_ID"),
	}

	cfg.IMAPUsername = os.Getenv("IMAP_USERNAME")
	if cfg.IMAPUsername == "" {
		return Config{}, fmt.Errorf("IMAP_USERNAME is required")
	}

	cfg.IMAPPassword = os.Getenv("IMAP_APP_PASSWORD")
	if cfg.IMAPPassword == "" {
		return Config{}, fmt.Errorf("IMAP_APP_PASSWORD is required")
	}

	if (cfg.TelegramBotToken == "") != (cfg.TelegramChatID == "") {
		return Config{}, fmt.Errorf("TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID must be set together")
	}

	rawInterval := envOrDefault("POLL_INTERVAL", "5m")
	interval, err := time.ParseDuration(rawInterval)
	if err != nil {
		return Config{}, fmt.Errorf("invalid POLL_INTERVAL %q: %w", rawInterval, err)
	}
	cfg.PollInterval = interval

	return cfg, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// loadDotEnv reads KEY=VALUE pairs from a .env file in the current
// directory, if present, and sets them via os.Setenv without overriding
// variables already set in the real environment. A missing file is not an
// error.
func loadDotEnv() error {
	data, err := os.ReadFile(".env")
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read .env: %w", err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "export ")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)

		if _, set := os.LookupEnv(key); !set {
			os.Setenv(key, value)
		}
	}

	return nil
}
