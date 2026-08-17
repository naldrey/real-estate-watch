// Package habitaclia parses habitaclia.com alert emails into listings.
package habitaclia

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

const portalName = "habitaclia"

func init() {
	ingest.Register(parser{})
}

type parser struct{}

func (parser) Portal() string { return portalName }

var (
	// idOccurrenceRe matches every occurrence of a listing's id in the HTML.
	// Each listing's id repeats ~4 times consecutively (photo, price, title,
	// favorite-heart links), so consecutive matches sharing an id delimit one
	// listing's HTML chunk.
	idOccurrenceRe = regexp.MustCompile(`habitaclia\.com/i(\d+)/`)
	titleURLRe     = regexp.MustCompile(`href="([^"]*_titulo\.htm[^"]*)"`)
	priceRe        = regexp.MustCompile(`>\s*([\d.,]+)\s*&euro;`)
	// roomsSizeRe: habitaclia lists size before rooms, unlike other portals.
	roomsSizeRe = regexp.MustCompile(`<b>(\d+)m<sup>2</sup></b>\s*\|\s*<b>(\d+)\s*hab\.`)
	titleRe     = regexp.MustCompile(`<p[^>]*>([^<]+)<br><font[^>]*>([^<]*)</font>`)
)

// Parse only handles "novedades" digest emails (new listings and price
// changes for a saved search). habitaclia also sends other mail to this
// mailbox - e.g. a welcome/"primera alerta" email - which this skips.
func (parser) Parse(body []byte) ([]listing.Listing, error) {
	subject, err := mimepart.Subject(body)
	if err != nil {
		return nil, fmt.Errorf("read subject: %w", err)
	}
	if !strings.Contains(strings.ToLower(subject), "novedades") {
		return nil, nil
	}

	content, err := mimepart.ExtractHTML(body)
	if err != nil {
		return nil, fmt.Errorf("extract html body: %w", err)
	}

	var listings []listing.Listing
	for _, c := range splitByListingID(content) {
		l := listing.Listing{Portal: portalName, ExternalID: c.id}

		if um := titleURLRe.FindStringSubmatch(c.text); um != nil {
			l.URL = html.UnescapeString(um[1])
		}

		if pm := priceRe.FindStringSubmatch(c.text); pm != nil {
			if price, err := parsePrice(pm[1]); err == nil {
				l.PriceEUR = price
			}
		}

		if rm := roomsSizeRe.FindStringSubmatch(c.text); rm != nil {
			if size, err := strconv.Atoi(rm[1]); err == nil {
				l.SizeM2 = size
			}
			if rooms, err := strconv.Atoi(rm[2]); err == nil {
				l.Rooms = rooms
			}
		}

		if tm := titleRe.FindStringSubmatch(c.text); tm != nil {
			title := strings.TrimSpace(html.UnescapeString(tm[1]))
			subtitle := strings.TrimSpace(html.UnescapeString(tm[2]))
			if subtitle != "" {
				title = title + " - " + subtitle
			}
			l.Title = title
		}

		listings = append(listings, l)
	}

	return listings, nil
}

type listingChunk struct {
	id   string
	text string
}

// splitByListingID groups the HTML into chunks, one per distinct listing id,
// based on runs of consecutive same-id occurrences of idOccurrenceRe.
func splitByListingID(content string) []listingChunk {
	matches := idOccurrenceRe.FindAllStringSubmatchIndex(content, -1)

	var chunks []listingChunk
	var starts []int
	for _, m := range matches {
		id := content[m[2]:m[3]]
		if len(chunks) == 0 || chunks[len(chunks)-1].id != id {
			chunks = append(chunks, listingChunk{id: id})
			starts = append(starts, m[0])
		}
	}

	for i := range chunks {
		end := len(content)
		if i+1 < len(starts) {
			end = starts[i+1]
		}
		chunks[i].text = content[starts[i]:end]
	}

	return chunks
}

// parsePrice converts a Spanish-formatted price ("1.275" or "1.275,50") into
// whole euros, discarding any decimal remainder.
func parsePrice(raw string) (int, error) {
	cleaned := strings.ReplaceAll(raw, ".", "")
	if idx := strings.Index(cleaned, ","); idx != -1 {
		cleaned = cleaned[:idx]
	}
	return strconv.Atoi(cleaned)
}
