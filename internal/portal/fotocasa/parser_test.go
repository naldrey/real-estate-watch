package fotocasa

import (
	"os"
	"strings"
	"testing"

	"github.com/naldrey/real-estate-watch/internal/listing"
)

func TestParse_NewListingAlert(t *testing.T) {
	tests := []struct {
		name     string
		testdata string
		want     listing.Listing
	}{
		{
			name:     "sample 1",
			testdata: "testdata/new_listing.eml",
			want: listing.Listing{
				Portal:     "fotocasa",
				ExternalID: "190415349",
				URL:        "https://www.fotocasa.es/es/alquiler/vivienda/badalona/centre/190415349/d",
				Title:      "piso · Carrer de la Creu91, Badalona",
				PriceEUR:   1400,
				Rooms:      3,
				SizeM2:     85,
			},
		},
		{
			// A second real sample, in the same digest format but with a
			// different listing (4 habs, 98 m², 1.200 €), to guard against
			// overfitting the parser to the first fixture's exact numbers.
			name:     "sample 2",
			testdata: "testdata/new_listing_2.eml",
			want: listing.Listing{
				Portal:     "fotocasa",
				ExternalID: "190493200",
				URL:        "https://www.fotocasa.es/es/alquiler/vivienda/barcelona-capital/aire-acondicionado-calefaccion-parking-terraza-ascensor/190493200/d",
				PriceEUR:   1200,
				Rooms:      4,
				SizeM2:     98,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := os.ReadFile(tt.testdata)
			if err != nil {
				t.Fatalf("failed to read testdata: %v", err)
			}

			listings, err := parser{}.Parse(body)
			if err != nil {
				t.Fatalf("Parse() returned error: %v", err)
			}

			if len(listings) != 1 {
				t.Fatalf("got %d listings, want 1: %+v", len(listings), listings)
			}

			got := listings[0]
			if got.Portal != tt.want.Portal || got.ExternalID != tt.want.ExternalID || got.PriceEUR != tt.want.PriceEUR ||
				got.Rooms != tt.want.Rooms || got.SizeM2 != tt.want.SizeM2 {
				t.Errorf("listing = %+v, want %+v", got, tt.want)
			}
			if tt.want.Title != "" && got.Title != tt.want.Title {
				t.Errorf("Title = %q, want %q", got.Title, tt.want.Title)
			}

			// The parser returns the raw href with tracking params attached;
			// normalization is an ingest-level concern.
			if !strings.HasPrefix(got.URL, tt.want.URL+"?") {
				t.Errorf("URL = %q, want it to start with %q", got.URL, tt.want.URL+"?")
			}
			normalized, err := listing.NormalizeURL(got.URL)
			if err != nil {
				t.Fatalf("NormalizeURL(%q) returned error: %v", got.URL, err)
			}
			if normalized != tt.want.URL {
				t.Errorf("NormalizeURL(%q) = %q, want %q", got.URL, normalized, tt.want.URL)
			}
		})
	}
}

func TestParse_SkipsHTMLOnlyTransactionalEmail(t *testing.T) {
	// "Hemos contactado por ti..." is a transactional notification, not the
	// alert digest: single-part HTML with no text/plain part. The parser
	// must skip it (no listings, no error) rather than fail.
	body, err := os.ReadFile("testdata/contacted_notification.eml")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	listings, err := parser{}.Parse(body)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}
	if len(listings) != 0 {
		t.Errorf("got %d listings, want 0", len(listings))
	}
}

func TestParse_NoMatchesReturnsEmpty(t *testing.T) {
	body := []byte("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nno listings here\r\n")

	listings, err := (parser{}).Parse(body)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}
	if len(listings) != 0 {
		t.Errorf("got %d listings, want 0", len(listings))
	}
}

func TestParse_IgnoresNonListingLinks(t *testing.T) {
	body := []byte("MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" +
		"Ver anuncios ( https://www.fotocasa.es/es/alertemail?userId=102392165 )\r\n" +
		"Modificar ( https://www.fotocasa.es/es/user/alerts?stc=x )\r\n" +
		"Google Play ( https://fotocasa.onelink.me/Wagw/78xx33xs )\r\n")

	listings, err := (parser{}).Parse(body)
	if err != nil {
		t.Fatalf("Parse() returned error: %v", err)
	}
	if len(listings) != 0 {
		t.Errorf("got %d listings, want 0 (none of these links point at a listing): %+v", len(listings), listings)
	}
}

func TestPortal(t *testing.T) {
	if got := (parser{}).Portal(); got != "fotocasa" {
		t.Errorf("Portal() = %q, want %q", got, "fotocasa")
	}
}
