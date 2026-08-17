package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/naldrey/real-estate-watch/internal/listing"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")

	s, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close() returned error: %v", err)
		}
	})

	return s
}

func TestSaveListing_NewAndDuplicate(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	l := listing.Listing{
		Portal:      "idealista",
		ExternalID:  "12345",
		URL:         "https://www.idealista.com/inmueble/12345/",
		Title:       "Piso en Gràcia",
		PriceEUR:    250000,
		SizeM2:      70,
		Rooms:       2,
		Address:     "Gràcia, Barcelona",
		FirstSeenAt: time.Now().UTC(),
	}

	isNew, err := s.SaveListing(ctx, l)
	if err != nil {
		t.Fatalf("SaveListing() returned error: %v", err)
	}
	if !isNew {
		t.Fatal("SaveListing() = false on first insert, want true")
	}

	isNew, err = s.SaveListing(ctx, l)
	if err != nil {
		t.Fatalf("SaveListing() returned error on duplicate: %v", err)
	}
	if isNew {
		t.Fatal("SaveListing() = true on duplicate insert, want false")
	}
}

func TestSaveListing_SamePortalDifferentExternalID(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	base := listing.Listing{
		Portal:      "idealista",
		URL:         "https://www.idealista.com/inmueble/1/",
		Title:       "Piso",
		PriceEUR:    100000,
		FirstSeenAt: time.Now().UTC(),
	}

	first := base
	first.ExternalID = "1"
	second := base
	second.ExternalID = "2"

	for _, l := range []listing.Listing{first, second} {
		isNew, err := s.SaveListing(ctx, l)
		if err != nil {
			t.Fatalf("SaveListing() returned error: %v", err)
		}
		if !isNew {
			t.Fatalf("SaveListing() = false for external_id %q, want true", l.ExternalID)
		}
	}
}

func TestProcessedMessages(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	mailbox := "idealista"
	var uid uint32 = 42

	processed, err := s.IsProcessed(ctx, mailbox, uid)
	if err != nil {
		t.Fatalf("IsProcessed() returned error: %v", err)
	}
	if processed {
		t.Fatal("IsProcessed() = true before marking, want false")
	}

	if err := s.MarkProcessed(ctx, mailbox, uid); err != nil {
		t.Fatalf("MarkProcessed() returned error: %v", err)
	}

	processed, err = s.IsProcessed(ctx, mailbox, uid)
	if err != nil {
		t.Fatalf("IsProcessed() returned error: %v", err)
	}
	if !processed {
		t.Fatal("IsProcessed() = false after marking, want true")
	}

	// Marking twice must not error.
	if err := s.MarkProcessed(ctx, mailbox, uid); err != nil {
		t.Fatalf("MarkProcessed() returned error on duplicate: %v", err)
	}
}

func TestProcessedMessages_DistinctMailboxes(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	var uid uint32 = 1
	if err := s.MarkProcessed(ctx, "idealista", uid); err != nil {
		t.Fatalf("MarkProcessed() returned error: %v", err)
	}

	processed, err := s.IsProcessed(ctx, "fotocasa", uid)
	if err != nil {
		t.Fatalf("IsProcessed() returned error: %v", err)
	}
	if processed {
		t.Fatal("IsProcessed() = true for different mailbox with same uid, want false")
	}
}

func TestMaxProcessedUID_NoneProcessedYet(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	_, ok, err := s.MaxProcessedUID(ctx, "idealista")
	if err != nil {
		t.Fatalf("MaxProcessedUID() returned error: %v", err)
	}
	if ok {
		t.Fatal("MaxProcessedUID() ok = true for a mailbox with no processed messages, want false")
	}
}

func TestMaxProcessedUID_ReturnsHighest(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	for _, uid := range []uint32{5, 42, 17} {
		if err := s.MarkProcessed(ctx, "idealista", uid); err != nil {
			t.Fatalf("MarkProcessed() returned error: %v", err)
		}
	}

	got, ok, err := s.MaxProcessedUID(ctx, "idealista")
	if err != nil {
		t.Fatalf("MaxProcessedUID() returned error: %v", err)
	}
	if !ok {
		t.Fatal("MaxProcessedUID() ok = false, want true")
	}
	if got != 42 {
		t.Errorf("MaxProcessedUID() = %d, want 42", got)
	}
}

func TestMaxProcessedUID_DistinctMailboxes(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	if err := s.MarkProcessed(ctx, "idealista", 100); err != nil {
		t.Fatalf("MarkProcessed() returned error: %v", err)
	}
	if err := s.MarkProcessed(ctx, "fotocasa", 5); err != nil {
		t.Fatalf("MarkProcessed() returned error: %v", err)
	}

	got, ok, err := s.MaxProcessedUID(ctx, "fotocasa")
	if err != nil {
		t.Fatalf("MaxProcessedUID() returned error: %v", err)
	}
	if !ok || got != 5 {
		t.Errorf("MaxProcessedUID(%q) = (%d, %v), want (5, true)", "fotocasa", got, ok)
	}
}
