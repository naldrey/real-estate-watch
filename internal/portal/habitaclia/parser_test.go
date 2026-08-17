package habitaclia

import (
	"os"
	"strings"
	"testing"
)

func TestParse_SixListingsAlert(t *testing.T) {
	body, err := os.ReadFile("testdata/six_listings.eml")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	listings, err := parser{}.Parse(body)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}

	if len(listings) != 6 {
		t.Fatalf("got %d listings, want 6: %+v", len(listings), listings)
	}

	// First listing: "new listing" badge, has both a title and a non-empty subtitle.
	first := listings[0]
	if first.Portal != "habitaclia" {
		t.Errorf("first.Portal = %q, want %q", first.Portal, "habitaclia")
	}
	if first.ExternalID != "41348000000162" {
		t.Errorf("first.ExternalID = %q, want %q", first.ExternalID, "41348000000162")
	}
	if first.PriceEUR != 1275 {
		t.Errorf("first.PriceEUR = %d, want 1275", first.PriceEUR)
	}
	if first.Rooms != 2 {
		t.Errorf("first.Rooms = %d, want 2", first.Rooms)
	}
	if first.SizeM2 != 75 {
		t.Errorf("first.SizeM2 = %d, want 75", first.SizeM2)
	}
	wantTitle := "Piso en Barcelona - L´Antiga Esquerra... - Carrer d'Aragó Piso Eixample Izquierd..."
	if first.Title != wantTitle {
		t.Errorf("first.Title = %q, want %q", first.Title, wantTitle)
	}
	if !strings.Contains(first.Title, "ó") {
		t.Errorf("first.Title = %q, want it to contain 'ó' (iso-8859-1 must be decoded to UTF-8)", first.Title)
	}
	wantURL := "https://www.habitaclia.com/i41348000000162/28090374/express28090374202608051610299400/alertas/email/lo-34/20260805-e_nuevo_fastmail-com_titulo.htm"
	if first.URL != wantURL {
		t.Errorf("first.URL = %q, want %q", first.URL, wantURL)
	}

	// Fifth listing: "price dropped" badge instead of "new listing" - same
	// extraction path should still work, and its subtitle starts with "N/A".
	fifth := listings[4]
	if fifth.ExternalID != "52795000012419" {
		t.Errorf("fifth.ExternalID = %q, want %q", fifth.ExternalID, "52795000012419")
	}
	if fifth.PriceEUR != 1500 {
		t.Errorf("fifth.PriceEUR = %d, want 1500", fifth.PriceEUR)
	}
	if fifth.Rooms != 2 {
		t.Errorf("fifth.Rooms = %d, want 2", fifth.Rooms)
	}
	if fifth.SizeM2 != 60 {
		t.Errorf("fifth.SizeM2 = %d, want 60", fifth.SizeM2)
	}

	// All external ids must be distinct.
	seen := map[string]bool{}
	for _, l := range listings {
		if seen[l.ExternalID] {
			t.Errorf("duplicate external_id %q across listings", l.ExternalID)
		}
		seen[l.ExternalID] = true
	}
}

func TestParse_NoMatchesReturnsEmpty(t *testing.T) {
	body := []byte("MIME-Version: 1.0\r\nContent-Type: text/html; charset=utf-8\r\nSubject: 1 novedades en Barcelona\r\n\r\n<p>no listings here</p>")

	listings, err := (parser{}).Parse(body)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}
	if len(listings) != 0 {
		t.Errorf("got %d listings, want 0", len(listings))
	}
}

func TestParse_SkipsNonNovedadesSubjects(t *testing.T) {
	// A body that would otherwise parse as a listing (matches the id pattern
	// habitaclia.com/i{digits}/), but under an unrelated subject - e.g. a
	// welcome or "primera alerta" email, not a novedades digest.
	body := []byte("MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"Subject: Bienvenido a tu primera alerta de habitaclia\r\n" +
		"\r\n" +
		`<a href="https://www.habitaclia.com/i12345000000001/x/y/z_titulo.htm">1.000 &euro;</a>`)

	listings, err := (parser{}).Parse(body)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}
	if len(listings) != 0 {
		t.Errorf("got %d listings, want 0 (subject doesn't contain \"novedades\"): %+v", len(listings), listings)
	}
}

func TestPortal(t *testing.T) {
	if got := (parser{}).Portal(); got != "habitaclia" {
		t.Errorf("Portal() = %q, want %q", got, "habitaclia")
	}
}
