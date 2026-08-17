package ingest

import (
	"fmt"

	"github.com/naldrey/real-estate-watch/internal/listing"
)

// MessageParser extracts listings from a single portal alert email's raw
// source. Each portal has exactly one MessageParser, registered via Register
// from the parser's init(). The parser's Portal() also names the IMAP
// mailbox it reads from, since there is one folder per portal.
type MessageParser interface {
	Portal() string
	Parse(body []byte) ([]listing.Listing, error)
}

var registry = map[string]MessageParser{}

// Register adds a parser to the registry, keyed by its portal. It panics if
// a parser for the same portal is already registered, since that indicates
// a programming error, not a runtime condition.
func Register(p MessageParser) {
	portal := p.Portal()
	if _, exists := registry[portal]; exists {
		panic(fmt.Sprintf("ingest: parser already registered for portal %q", portal))
	}
	registry[portal] = p
}

// Registered returns every parser registered so far.
func Registered() []MessageParser {
	parsers := make([]MessageParser, 0, len(registry))
	for _, p := range registry {
		parsers = append(parsers, p)
	}
	return parsers
}
