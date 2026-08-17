package ingest

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/naldrey/real-estate-watch/internal/listing"
)

// Run polls each parser's mailbox for unprocessed alert emails, parses new
// listings out of them, and persists the results. A failure scoped to one
// mailbox or message is logged and does not stop processing of the rest.
// n may be nil to skip notifications.
func Run(ctx context.Context, p Provider, s Store, parsers []MessageParser, n Notifier) error {
	for _, parser := range parsers {
		if err := ctx.Err(); err != nil {
			return err
		}

		mailbox := parser.Portal()
		if err := runMailbox(ctx, p, s, n, mailbox, parser); err != nil {
			slog.Error("failed to process mailbox", "mailbox", mailbox, "err", err)
		}
	}
	return nil
}

// retryLookback bounds how far below the highest-ever-processed UID each
// poll re-examines. Without it, a message that failed processing (and so
// was never marked processed) while a later message in the same batch
// succeeded would be silently skipped forever once the watermark moved past
// it. With it, a stuck message is retried for up to retryLookback newer
// messages' worth of polls - long enough in practice to catch transient
// failures and code fixes, without re-listing a mailbox's entire history on
// every single poll as it grows over months.
const retryLookback = 200

func runMailbox(ctx context.Context, p Provider, s Store, n Notifier, mailbox string, parser MessageParser) error {
	var sinceUID uint32
	if maxUID, ok, err := s.MaxProcessedUID(ctx, mailbox); err != nil {
		return fmt.Errorf("max processed uid: %w", err)
	} else if ok && maxUID > retryLookback {
		sinceUID = maxUID - retryLookback
	}

	uids, err := p.ListUIDs(ctx, mailbox, sinceUID)
	if err != nil {
		return fmt.Errorf("list uids: %w", err)
	}

	for _, uid := range uids {
		if err := ctx.Err(); err != nil {
			return err
		}

		processed, err := s.IsProcessed(ctx, mailbox, uid)
		if err != nil {
			return fmt.Errorf("check processed uid %d: %w", uid, err)
		}
		if processed {
			continue
		}

		if err := processMessage(ctx, p, s, n, mailbox, uid, parser); err != nil {
			slog.Error("failed to process message", "mailbox", mailbox, "uid", uid, "err", err)
		}
	}

	return nil
}

// processMessage fetches, parses, and saves a single message. On failure it
// leaves the message unmarked so it is retried on the next run.
func processMessage(ctx context.Context, p Provider, s Store, n Notifier, mailbox string, uid uint32, parser MessageParser) error {
	body, err := p.FetchMessage(ctx, mailbox, uid)
	if err != nil {
		return fmt.Errorf("fetch message: %w", err)
	}

	listings, err := parser.Parse(body)
	if err != nil {
		return fmt.Errorf("parse message: %w", err)
	}

	for _, l := range listings {
		normalized, err := listing.NormalizeURL(l.URL)
		if err != nil {
			return fmt.Errorf("normalize url %q: %w", l.URL, err)
		}
		l.URL = normalized

		isNew, err := s.SaveListing(ctx, l)
		if err != nil {
			return fmt.Errorf("save listing %s/%s: %w", l.Portal, l.ExternalID, err)
		}
		if isNew {
			slog.Info("new listing", "portal", l.Portal, "external_id", l.ExternalID, "price_eur", l.PriceEUR, "url", l.URL)

			if n != nil {
				if err := n.NotifyNewListing(ctx, l); err != nil {
					slog.Error("failed to send notification", "portal", l.Portal, "external_id", l.ExternalID, "err", err)
				}
			}
		}
	}

	if err := s.MarkProcessed(ctx, mailbox, uid); err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}

	return nil
}
