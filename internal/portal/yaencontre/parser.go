// Package yaencontre parses yaencontre.com alert emails into listings.
package yaencontre

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/naldrey/real-estate-watch/internal/ingest"
	"github.com/naldrey/real-estate-watch/internal/listing"
	"github.com/naldrey/real-estate-watch/internal/mimepart"
)

const portalName = "yaencontre"

func init() {
	ingest.Register(parser{})
}

type parser struct{}

func (parser) Portal() string { return portalName }

// titleLinkRe matches a listing's title anchor: the only one of the three
// anchors pointing at a given listing (image, title, "Ver detalles" CTA)
// whose inner content is plain text rather than a nested <img>/<span>.
var (
	titleLinkRe = regexp.MustCompile(`<a[^>]+href="([^"]*inmueble-(\d+-\d+)[^"]*)"[^>]*>([^<]+)</a>`)
	priceRe     = regexp.MustCompile(`>([\d.,]+)\s*€`)
	roomsSizeRe = regexp.MustCompile(`(\d+)\s*hab\.\s*\|\s*\d+\s*ba[ñn]os\s*\|\s*(\d+)\s*m`)
)

func (parser) Parse(body []byte) ([]listing.Listing, error) {
	content, err := mimepart.ExtractHTML(body)
	if err != nil {
		return nil, fmt.Errorf("extract html body: %w", err)
	}

	matches := titleLinkRe.FindAllStringSubmatchIndex(content, -1)

	listings := make([]listing.Listing, 0, len(matches))
	for i, m := range matches {
		end := len(content)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		card := content[m[0]:end]

		l := listing.Listing{
			Portal:     portalName,
			ExternalID: content[m[4]:m[5]],
			URL:        html.UnescapeString(content[m[2]:m[3]]),
			Title:      strings.TrimSpace(html.UnescapeString(content[m[6]:m[7]])),
		}

		if pm := priceRe.FindStringSubmatch(card); pm != nil {
			if price, err := parsePrice(pm[1]); err == nil {
				l.PriceEUR = price
			}
		}

		if rm := roomsSizeRe.FindStringSubmatch(card); rm != nil {
			if rooms, err := strconv.Atoi(rm[1]); err == nil {
				l.Rooms = rooms
			}
			if size, err := strconv.Atoi(rm[2]); err == nil {
				l.SizeM2 = size
			}
		}

		listings = append(listings, l)
	}

	return listings, nil
}

// parsePrice converts a Spanish-formatted price ("1.458" or "1.458,50") into
// whole euros, discarding any decimal remainder.
func parsePrice(raw string) (int, error) {
	cleaned := strings.ReplaceAll(raw, ".", "")
	if idx := strings.Index(cleaned, ","); idx != -1 {
		cleaned = cleaned[:idx]
	}
	return strconv.Atoi(cleaned)
}
