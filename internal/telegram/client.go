// Package telegram sends listing notifications to a Telegram chat via the
// Bot API.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/naldrey/real-estate-watch/internal/listing"
)

// Client sends messages to a single Telegram chat. It implements
// ingest.Notifier.
type Client struct {
	botToken string
	chatID   string
	apiBase  string // overridable in tests; defaults to the real Bot API
	http     *http.Client
}

// NewClient creates a Client for the given bot token and destination chat ID.
func NewClient(botToken, chatID string) *Client {
	return &Client{
		botToken: botToken,
		chatID:   chatID,
		apiBase:  "https://api.telegram.org",
		http:     &http.Client{},
	}
}

// NotifyNewListing sends a formatted message describing l to the chat.
func (c *Client) NotifyNewListing(ctx context.Context, l listing.Listing) error {
	return c.SendMessage(ctx, formatListing(l))
}

func formatListing(l listing.Listing) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<b>%s</b>\n%d €", html.EscapeString(l.Title), l.PriceEUR)

	var details []string
	if l.Rooms > 0 {
		details = append(details, fmt.Sprintf("%d hab.", l.Rooms))
	}
	if l.SizeM2 > 0 {
		details = append(details, fmt.Sprintf("%d m²", l.SizeM2))
	}
	if len(details) > 0 {
		fmt.Fprintf(&b, "\n%s", strings.Join(details, " · "))
	}

	fmt.Fprintf(&b, "\n%s\n%s", l.Portal, l.URL)
	return b.String()
}

type sendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// SendMessage sends text as an HTML-formatted message to the configured chat.
func (c *Client) SendMessage(ctx context.Context, text string) error {
	body, err := json.Marshal(sendMessageRequest{
		ChatID:    c.chatID,
		Text:      text,
		ParseMode: "HTML",
	})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", c.apiBase, c.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram api returned status %d", resp.StatusCode)
	}

	return nil
}
