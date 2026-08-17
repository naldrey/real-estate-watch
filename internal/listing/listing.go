// Package listing defines the core domain model for property listings.
package listing

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Listing is a single property listing observed from a portal alert email.
type Listing struct {
	ID          int64
	Portal      string
	ExternalID  string
	URL         string
	Title       string
	PriceEUR    int
	SizeM2      int // 0 if unknown
	Rooms       int // 0 if unknown
	Address     string
	FirstSeenAt time.Time
}

// Key is the dedupe key for a listing: a listing is unique per portal and
// the portal's own identifier for it.
type Key struct {
	Portal     string
	ExternalID string
}

// Key returns the dedupe key for the listing.
func (l Listing) Key() Key {
	return Key{Portal: l.Portal, ExternalID: l.ExternalID}
}

// trackingParams are non-"utm_"-prefixed query parameters known to vary
// between otherwise identical listing links (analytics/tracking), stripped
// during normalization so such links dedupe to the same URL. Every "utm_"
// param is stripped regardless of its specific name, since portals each use
// their own variants (utm_link, utm_recipient_id, utm_notification_id, ...).
var trackingParams = map[string]bool{
	"xtor": true, "gclid": true, "fbclid": true, "sid": true,
	"pr":  true, // yaencontre: constant per-alert id, not listing-specific
	"stc": true, // fotocasa: click-tracking id, not listing-specific
}

// NormalizeURL strips tracking query parameters and the fragment from a
// listing URL so that equivalent links compare equal.
func NormalizeURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}

	q := u.Query()
	for key := range q {
		if strings.HasPrefix(key, "utm_") || trackingParams[key] {
			q.Del(key)
		}
	}
	u.RawQuery = q.Encode()
	u.Fragment = ""

	return u.String(), nil
}

// ParsePriceEUR converts a Spanish-formatted price ("1.400" or "1.400,50")
// into whole euros, discarding any decimal remainder. "." is treated as a
// thousands separator and "," as a decimal separator.
func ParsePriceEUR(raw string) (int, error) {
	cleaned := strings.ReplaceAll(raw, ".", "")
	if idx := strings.Index(cleaned, ","); idx != -1 {
		cleaned = cleaned[:idx]
	}
	return strconv.Atoi(cleaned)
}
