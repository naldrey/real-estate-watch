// Command listmailboxes prints every IMAP folder name in the configured
// account, to help map each portal to its actual mailbox name.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/naldrey/real-estate-watch/internal/config"
	"github.com/naldrey/real-estate-watch/internal/mail"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	client := mail.NewClient(cfg.IMAPHost, cfg.IMAPUsername, cfg.IMAPPassword)
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connect to imap: %w", err)
	}
	defer client.Close()

	mailboxes, err := client.ListMailboxes(ctx)
	if err != nil {
		return fmt.Errorf("list mailboxes: %w", err)
	}

	for _, m := range mailboxes {
		fmt.Println(m)
	}
	return nil
}
