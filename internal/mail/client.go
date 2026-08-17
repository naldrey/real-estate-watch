// Package mail implements ingest.Provider over IMAP.
package mail

import (
	"context"
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// Client is an IMAP client for a single mail account.
type Client struct {
	addr     string
	username string
	password string
	conn     *imapclient.Client
}

// NewClient creates a Client for the IMAP server at addr (host:port),
// authenticating with username and password. Call Connect before use.
func NewClient(addr, username, password string) *Client {
	return &Client{addr: addr, username: username, password: password}
}

// Connect dials the IMAP server over TLS and logs in.
func (c *Client) Connect(ctx context.Context) error {
	conn, err := imapclient.DialTLS(c.addr, nil)
	if err != nil {
		return fmt.Errorf("dial imap server %s: %w", c.addr, err)
	}

	if err := conn.Login(c.username, c.password).Wait(); err != nil {
		conn.Close()
		return fmt.Errorf("login as %s: %w", c.username, err)
	}

	c.conn = conn
	return nil
}

// Close logs out and closes the connection.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// ListUIDs returns the UIDs of every message in mailbox with UID greater
// than sinceUID (0 returns every message in the mailbox).
func (c *Client) ListUIDs(ctx context.Context, mailbox string, sinceUID uint32) ([]uint32, error) {
	mbox, err := c.conn.Select(mailbox, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("select mailbox %q: %w", mailbox, err)
	}
	if mbox.NumMessages == 0 {
		return nil, nil
	}

	var set imap.UIDSet
	set.AddRange(imap.UID(sinceUID+1), 0) // "sinceUID+1:*"

	messages, err := c.conn.Fetch(set, &imap.FetchOptions{UID: true}).Collect()
	if err != nil {
		return nil, fmt.Errorf("list uids in mailbox %q: %w", mailbox, err)
	}

	uids := make([]uint32, len(messages))
	for i, msg := range messages {
		uids[i] = uint32(msg.UID)
	}
	return uids, nil
}

// ListMailboxes returns the names of every mailbox (folder) in the account.
// It exists to help map each portal to its actual IMAP folder name, which
// may not match the portal identifier exactly.
func (c *Client) ListMailboxes(ctx context.Context) ([]string, error) {
	mailboxes, err := c.conn.List("", "*", nil).Collect()
	if err != nil {
		return nil, fmt.Errorf("list mailboxes: %w", err)
	}

	names := make([]string, len(mailboxes))
	for i, mbox := range mailboxes {
		names[i] = mbox.Mailbox
	}
	return names, nil
}

// FetchMessage returns the raw source of the message identified by uid in mailbox.
func (c *Client) FetchMessage(ctx context.Context, mailbox string, uid uint32) ([]byte, error) {
	if _, err := c.conn.Select(mailbox, nil).Wait(); err != nil {
		return nil, fmt.Errorf("select mailbox %q: %w", mailbox, err)
	}

	bodySection := &imap.FetchItemBodySection{}
	fetchOptions := &imap.FetchOptions{BodySection: []*imap.FetchItemBodySection{bodySection}}

	messages, err := c.conn.Fetch(imap.UIDSetNum(imap.UID(uid)), fetchOptions).Collect()
	if err != nil {
		return nil, fmt.Errorf("fetch message uid %d in mailbox %q: %w", uid, mailbox, err)
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("message uid %d not found in mailbox %q", uid, mailbox)
	}

	body := messages[0].FindBodySection(bodySection)
	if body == nil {
		return nil, fmt.Errorf("message uid %d in mailbox %q has no body section", uid, mailbox)
	}

	return body, nil
}
