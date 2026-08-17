package yaencontre

import (
	"os"
	"strings"
	"testing"

	"github.com/naldrey/real-estate-watch/internal/listing"
)

func TestParse_SimilarListingsAlert(t *testing.T) {
	body, err := os.ReadFile("testdata/similar_listings.eml")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	listings, err := parser{}.Parse(body)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	want := []listing.Listing{
		{
			Portal:     "yaencontre",
			ExternalID: "80475-112204056",
			URL:        "https://www.yaencontre.com/alquiler/piso/inmueble-80475-112204056",
			Title:      "Piso en alquiler en calle D'enric Granados, L'Antiga Esquerra de l'Eixample en Barcelona",
			PriceEUR:   1458,
			Rooms:      3,
			SizeM2:     110,
		},
		{
			Portal:     "yaencontre",
			ExternalID: "64012-112165622",
			URL:        "https://www.yaencontre.com/alquiler/piso/inmueble-64012-112165622",
			Title:      "Piso en alquiler en calle Cantabria, Sant Martí de Provençals en Barcelona",
			PriceEUR:   1500,
			Rooms:      3,
			SizeM2:     150,
		},
		{
			Portal:     "yaencontre",
			ExternalID: "56071-111854849",
			URL:        "https://www.yaencontre.com/alquiler/piso/inmueble-56071-111854849",
			Title:      "Piso en alquiler en calle De Badajoz, El Parc i la Llacuna del Poblenou en Barcelona",
			PriceEUR:   1500,
			Rooms:      3,
			SizeM2:     90,
		},
	}

	if len(listings) != len(want) {
		t.Fatalf("got %d listings, want %d: %+v", len(listings), len(want), listings)
	}

	for i, w := range want {
		got := listings[i]
		if got.Portal != w.Portal || got.ExternalID != w.ExternalID || got.PriceEUR != w.PriceEUR ||
			got.Rooms != w.Rooms || got.SizeM2 != w.SizeM2 || got.Title != w.Title {
			t.Errorf("listing %d = %+v, want %+v", i, got, w)
		}

		// The parser returns the raw href (unescaped, tracking params still
		// attached) — normalization is an ingest-level concern, not the
		// parser's. Check the base path and that normalizing it recovers the
		// clean URL, rather than hand-transcribing the exact tracking query.
		if !strings.HasPrefix(got.URL, w.URL+"?") {
			t.Errorf("listing %d URL = %q, want it to start with %q", i, got.URL, w.URL+"?")
		}
		normalized, err := listing.NormalizeURL(got.URL)
		if err != nil {
			t.Errorf("listing %d: NormalizeURL(%q) returned error: %v", i, got.URL, err)
		} else if normalized != w.URL {
			t.Errorf("listing %d: NormalizeURL(%q) = %q, want %q", i, got.URL, normalized, w.URL)
		}
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
	if got := (parser{}).Portal(); got != "yaencontre" {
		t.Errorf("Portal() = %q, want %q", got, "yaencontre")
	}
}
