// Package store provides SQLite-backed persistence for listings and
// IMAP message processing state.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/naldrey/real-estate-watch/internal/listing"
)

const schema = `
CREATE TABLE IF NOT EXISTS listings (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	portal TEXT NOT NULL,
	external_id TEXT NOT NULL,
	url TEXT NOT NULL,
	title TEXT NOT NULL,
	price_eur INTEGER NOT NULL,
	size_m2 INTEGER NOT NULL DEFAULT 0,
	rooms INTEGER NOT NULL DEFAULT 0,
	address TEXT NOT NULL DEFAULT '',
	first_seen_at DATETIME NOT NULL,
	UNIQUE (portal, external_id)
);

CREATE TABLE IF NOT EXISTS processed_messages (
	mailbox TEXT NOT NULL,
	uid INTEGER NOT NULL,
	processed_at DATETIME NOT NULL,
	PRIMARY KEY (mailbox, uid)
);
`

// Store persists listings and IMAP processing state in SQLite.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite database at path and applies the schema.
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// modernc.org/sqlite serializes writes at the connection pool level;
	// a single connection avoids SQLITE_BUSY under concurrent access.
	db.SetMaxOpenConns(1)

	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// SaveListing inserts a listing, returning true if it was new. If a listing
// with the same (portal, external_id) already exists, it returns false and
// leaves the existing row untouched.
func (s *Store) SaveListing(ctx context.Context, l listing.Listing) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO listings (portal, external_id, url, title, price_eur, size_m2, rooms, address, first_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (portal, external_id) DO NOTHING
	`, l.Portal, l.ExternalID, l.URL, l.Title, l.PriceEUR, l.SizeM2, l.Rooms, l.Address, l.FirstSeenAt)
	if err != nil {
		return false, fmt.Errorf("insert listing: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("check rows affected: %w", err)
	}

	return n > 0, nil
}

// IsProcessed reports whether the message identified by (mailbox, uid) has already been ingested.
func (s *Store) IsProcessed(ctx context.Context, mailbox string, uid uint32) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM processed_messages WHERE mailbox = ? AND uid = ?)
	`, mailbox, uid).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check processed message: %w", err)
	}

	return exists, nil
}

// MarkProcessed records that the message identified by (mailbox, uid) has been ingested.
func (s *Store) MarkProcessed(ctx context.Context, mailbox string, uid uint32) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO processed_messages (mailbox, uid, processed_at)
		VALUES (?, ?, ?)
		ON CONFLICT (mailbox, uid) DO NOTHING
	`, mailbox, uid, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("mark message processed: %w", err)
	}

	return nil
}
