package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/naldrey/real-estate-watch/internal/listing"
)

func TestFormatListing(t *testing.T) {
	tests := []struct {
		name string
		l    listing.Listing
		want string
	}{
		{
			name: "full details",
			l: listing.Listing{
				Portal:   "idealista",
				Title:    "Piso en Calle de Pallars, 294, El Poblenou, Barcelona",
				PriceEUR: 1300,
				Rooms:    2,
				SizeM2:   67,
				URL:      "https://www.idealista.com/inmueble/112208779/",
			},
			want: "<b>Piso en Calle de Pallars, 294, El Poblenou, Barcelona</b>\n1300 €\n2 hab. · 67 m²\nidealista\nhttps://www.idealista.com/inmueble/112208779/",
		},
		{
			name: "missing rooms and size omits the details line",
			l: listing.Listing{
				Portal:   "yaencontre",
				Title:    "Piso en alquiler",
				PriceEUR: 900,
				URL:      "https://www.yaencontre.com/x",
			},
			want: "<b>Piso en alquiler</b>\n900 €\nyaencontre\nhttps://www.yaencontre.com/x",
		},
		{
			name: "title is html-escaped",
			l: listing.Listing{
				Title:    "3 & 4 <rooms>",
				PriceEUR: 500,
			},
			want: "<b>3 &amp; 4 &lt;rooms&gt;</b>\n500 €\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatListing(tt.l); got != tt.want {
				t.Errorf("formatListing() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSendMessage_PostsExpectedRequest(t *testing.T) {
	var gotPath string
	var gotBody sendMessageRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := &Client{botToken: "TEST-TOKEN", chatID: "12345", apiBase: server.URL, http: server.Client()}

	if err := c.SendMessage(context.Background(), "hello"); err != nil {
		t.Fatalf("SendMessage() returned error: %v", err)
	}

	if want := "/botTEST-TOKEN/sendMessage"; gotPath != want {
		t.Errorf("request path = %q, want %q", gotPath, want)
	}
	if gotBody.ChatID != "12345" || gotBody.Text != "hello" || gotBody.ParseMode != "HTML" {
		t.Errorf("request body = %+v, want chat_id=12345 text=hello parse_mode=HTML", gotBody)
	}
}

func TestSendMessage_NonOKStatusReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	c := &Client{botToken: "TEST-TOKEN", chatID: "12345", apiBase: server.URL, http: server.Client()}

	err := c.SendMessage(context.Background(), "hello")
	if err == nil {
		t.Fatal("SendMessage() returned nil error for a non-200 response")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("SendMessage() error = %q, want it to mention status 403", err.Error())
	}
}
