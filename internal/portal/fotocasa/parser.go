// Package fotocasa parses fotocasa.es alert emails into listings.
package fotocasa

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/naldrey/real-estate-watch/internal/ingest"
	"github.com/naldrey/real-estate-watch/internal/listing"
	"github.com/naldrey/real-estate-watch/internal/mimepart"
)

const portalName = "fotocasa"

func init() {
	ingest.Register(parser{})
}

type parser struct{}

func (parser) Portal() string { return portalName }

var (
	// pairRe matches a "label ( url )" line from the alert's plain-text body.
	// Every element of a listing card (photo caption, badge, price, address,
	// rooms/size, CTA) is rendered as one such line linking to the same
	// listing URL.
	pairRe = regexp.MustCompile(`(?m)^(.+?)\s*\(\s*(https?://[^\s)]+)\s*\)\s*$`)
	// listingIDRe extracts a listing's numeric id from its detail-page URL
	// (".../190415349/d?..."). Non-listing links (header, footer, app store,
	// unsubscribe, ...) don't match and are ignored.
	listingIDRe = regexp.MustCompile(`/(\d+)/d(?:[/?]|$)`)
	priceRe     = regexp.MustCompile(`^([\d.,]+)\s*€$`)
	roomsSizeRe = regexp.MustCompile(`(\d+)\s*habs?\s*·\s*(\d+)\s*m`)
)

// nonTitleLabels are lines that appear once per listing card but aren't the
// listing's title/address: a badge, photo caption, or call-to-action.
var nonTitleLabels = map[string]bool{
	"fotografía de la oferta": true,
	"novedad":                 true,
	"rebajado":                true,
	"actualizado":             true,
	"ver anuncio":             true,
}

// Parse only handles the "novedades" style alert digest, which is sent as
// multipart/alternative with a text/plain part. fotocasa also sends other
// mail to this mailbox - e.g. a "Hemos contactado por ti..." transactional
// notification, sent as HTML-only - which this skips.
func (parser) Parse(body []byte) ([]listing.Listing, error) {
	content, err := mimepart.ExtractPlainText(body)
	if errors.Is(err, mimepart.ErrPartNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("extract plain text body: %w", err)
	}

	byID := map[string]*listing.Listing{}
	var order []string

	for _, m := range pairRe.FindAllStringSubmatch(content, -1) {
		label := strings.TrimSpace(html.UnescapeString(m[1]))
		url := html.UnescapeString(m[2])

		idMatch := listingIDRe.FindStringSubmatch(url)
		if idMatch == nil {
			continue
		}
		id := idMatch[1]

		l, seen := byID[id]
		if !seen {
			l = &listing.Listing{Portal: portalName, ExternalID: id, URL: url}
			byID[id] = l
			order = append(order, id)
		}

		switch {
		case priceRe.MatchString(label):
			if price, err := parsePrice(priceRe.FindStringSubmatch(label)[1]); err == nil {
				l.PriceEUR = price
			}
		case roomsSizeRe.MatchString(label):
			rm := roomsSizeRe.FindStringSubmatch(label)
			if rooms, err := strconv.Atoi(rm[1]); err == nil {
				l.Rooms = rooms
			}
			if size, err := strconv.Atoi(rm[2]); err == nil {
				l.SizeM2 = size
			}
		case l.Title == "" && !nonTitleLabels[strings.ToLower(label)]:
			l.Title = label
		}
	}

	listings := make([]listing.Listing, 0, len(order))
	for _, id := range order {
		listings = append(listings, *byID[id])
	}

	return listings, nil
}

// parsePrice converts a Spanish-formatted price ("1.400" or "1.400,50") into
// whole euros, discarding any decimal remainder.
func parsePrice(raw string) (int, error) {
	cleaned := strings.ReplaceAll(raw, ".", "")
	if idx := strings.Index(cleaned, ","); idx != -1 {
		cleaned = cleaned[:idx]
	}
	return strconv.Atoi(cleaned)
}
