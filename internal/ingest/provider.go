// Package ingest polls IMAP mailboxes for new portal alert emails, parses
// listings out of them, and persists the results.
package ingest

import (
	"context"

	"github.com/naldrey/real-estate-watch/internal/listing"
)

// Provider is the mail-fetching capability ingest needs. It is implemented
// by the IMAP client package; declaring it here, in the consumer, keeps
// ingest decoupled from any specific mail transport.
type Provider interface {
	// ListUIDs returns the UIDs of every message in mailbox with UID
	// greater than sinceUID (0 returns every message in the mailbox).
	ListUIDs(ctx context.Context, mailbox string, sinceUID uint32) ([]uint32, error)
	// FetchMessage returns the raw source of the message identified by uid in mailbox.
	FetchMessage(ctx context.Context, mailbox string, uid uint32) ([]byte, error)
}

// Store is the persistence capability ingest needs.
type Store interface {
	IsProcessed(ctx context.Context, mailbox string, uid uint32) (bool, error)
	MarkProcessed(ctx context.Context, mailbox string, uid uint32) error
	SaveListing(ctx context.Context, l listing.Listing) (bool, error)
	// MaxProcessedUID returns the highest UID marked processed for mailbox.
	// ok is false if no message in mailbox has been processed yet.
	MaxProcessedUID(ctx context.Context, mailbox string) (uid uint32, ok bool, err error)
}
