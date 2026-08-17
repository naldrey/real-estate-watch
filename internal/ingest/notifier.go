package ingest

import (
	"context"

	"github.com/naldrey/real-estate-watch/internal/listing"
)

// Notifier is notified whenever a new listing is saved. It is optional:
// pass nil to Run to skip notifications entirely.
type Notifier interface {
	NotifyNewListing(ctx context.Context, l listing.Listing) error
}
