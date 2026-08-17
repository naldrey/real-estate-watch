package idealista

import (
	"os"
	"strings"
	"testing"

	"github.com/naldrey/real-estate-watch/internal/listing"
)

func TestParse_NewListingAlert(t *testing.T) {
	body, err := os.ReadFile("testdata/new_listing.eml")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	listings, err := parser{}.Parse(body)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	want := listing.Listing{
		Portal:     "idealista",
		ExternalID: "112208779",
		URL:        "https://www.idealista.com/inmueble/112208779/",
		Title:      "Piso en Calle de Pallars, 294, El Poblenou, Barcelona",
		PriceEUR:   1300,
		Rooms:      2,
		SizeM2:     67,
	}

	if len(listings) != 1 {
		t.Fatalf("got %d listings, want 1: %+v", len(listings), listings)
	}

	got := listings[0]
	if got.Portal != want.Portal || got.ExternalID != want.ExternalID || got.PriceEUR != want.PriceEUR ||
		got.Rooms != want.Rooms || got.SizeM2 != want.SizeM2 || got.Title != want.Title {
		t.Errorf("listing = %+v, want %+v", got, want)
	}

	// The parser returns the raw href with tracking params attached;
	// normalization is an ingest-level concern.
	if !strings.HasPrefix(got.URL, want.URL+"?") {
		t.Errorf("URL = %q, want it to start with %q", got.URL, want.URL+"?")
	}
	normalized, err := listing.NormalizeURL(got.URL)
	if err != nil {
		t.Fatalf("NormalizeURL(%q) returned error: %v", got.URL, err)
	}
	if normalized != want.URL {
		t.Errorf("NormalizeURL(%q) = %q, want %q", got.URL, normalized, want.URL)
	}
}

func TestParse_NoMatchesReturnsEmpty(t *testing.T) {
	listings, err := (parser{}).Parse([]byte("MIME-Version: 1.0\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<p>no listings here</p>"))
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}
	if len(listings) != 0 {
		t.Errorf("got %d listings, want 0", len(listings))
	}
}

func TestPortal(t *testing.T) {
	if got := (parser{}).Portal(); got != "idealista" {
		t.Errorf("Portal() = %q, want %q", got, "idealista")
	}
}
