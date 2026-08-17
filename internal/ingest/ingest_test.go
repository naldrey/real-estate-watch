package ingest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/naldrey/real-estate-watch/internal/listing"
)

type fakeProvider struct {
	uids     map[string][]uint32
	bodies   map[string]map[uint32][]byte
	listErr  error
	fetchErr error

	// calledSinceUID records the sinceUID each mailbox's ListUIDs call was
	// made with, so tests can assert on the watermark/lookback math.
	calledSinceUID map[string]uint32
}

func (f *fakeProvider) ListUIDs(ctx context.Context, mailbox string, sinceUID uint32) ([]uint32, error) {
	if f.calledSinceUID == nil {
		f.calledSinceUID = map[string]uint32{}
	}
	f.calledSinceUID[mailbox] = sinceUID

	if f.listErr != nil {
		return nil, f.listErr
	}

	var uids []uint32
	for _, uid := range f.uids[mailbox] {
		if uid > sinceUID {
			uids = append(uids, uid)
		}
	}
	return uids, nil
}

func (f *fakeProvider) FetchMessage(ctx context.Context, mailbox string, uid uint32) ([]byte, error) {
	if f.fetchErr != nil {
		return nil, f.fetchErr
	}
	return f.bodies[mailbox][uid], nil
}

type fakeStore struct {
	processed map[string]map[uint32]bool
	saved     []listing.Listing
}

func newFakeStore() *fakeStore {
	return &fakeStore{processed: map[string]map[uint32]bool{}}
}

func (f *fakeStore) IsProcessed(ctx context.Context, mailbox string, uid uint32) (bool, error) {
	return f.processed[mailbox][uid], nil
}

func (f *fakeStore) MarkProcessed(ctx context.Context, mailbox string, uid uint32) error {
	if f.processed[mailbox] == nil {
		f.processed[mailbox] = map[uint32]bool{}
	}
	f.processed[mailbox][uid] = true
	return nil
}

func (f *fakeStore) SaveListing(ctx context.Context, l listing.Listing) (bool, error) {
	f.saved = append(f.saved, l)
	return true, nil
}

func (f *fakeStore) MaxProcessedUID(ctx context.Context, mailbox string) (uint32, bool, error) {
	var max uint32
	var ok bool
	for uid, done := range f.processed[mailbox] {
		if done && (!ok || uid > max) {
			max = uid
			ok = true
		}
	}
	return max, ok, nil
}

type fakeNotifier struct {
	notified []listing.Listing
}

func (n *fakeNotifier) NotifyNewListing(ctx context.Context, l listing.Listing) error {
	n.notified = append(n.notified, l)
	return nil
}

type fakeParser struct {
	portal  string
	parseFn func(body []byte) ([]listing.Listing, error)
}

func (p *fakeParser) Portal() string { return p.portal }

func (p *fakeParser) Parse(body []byte) ([]listing.Listing, error) {
	return p.parseFn(body)
}

func TestRun_NewMessageSavedAndMarkedProcessed(t *testing.T) {
	ctx := context.Background()

	provider := &fakeProvider{
		uids: map[string][]uint32{"idealista": {1}},
		bodies: map[string]map[uint32][]byte{
			"idealista": {1: []byte("raw email")},
		},
	}
	s := newFakeStore()
	parser := &fakeParser{
		portal: "idealista",
		parseFn: func(body []byte) ([]listing.Listing, error) {
			return []listing.Listing{{
				Portal:      "idealista",
				ExternalID:  "123",
				URL:         "https://example.com/listing/123?utm_source=email",
				PriceEUR:    250000,
				FirstSeenAt: time.Now(),
			}}, nil
		},
	}

	if err := Run(ctx, provider, s, []MessageParser{parser}, nil); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if len(s.saved) != 1 {
		t.Fatalf("got %d saved listings, want 1", len(s.saved))
	}
	if got, want := s.saved[0].URL, "https://example.com/listing/123"; got != want {
		t.Errorf("saved listing URL = %q, want %q (tracking params should be stripped)", got, want)
	}
	if !s.processed["idealista"][1] {
		t.Error("message uid 1 was not marked processed")
	}
}

func TestRun_NotifiesOnNewListingOnly(t *testing.T) {
	ctx := context.Background()

	provider := &fakeProvider{
		uids: map[string][]uint32{"idealista": {1, 2}},
		bodies: map[string]map[uint32][]byte{
			"idealista": {1: []byte("raw email 1"), 2: []byte("raw email 2")},
		},
	}
	s := newFakeStore()
	s.processed["idealista"] = map[uint32]bool{2: true} // uid 2 already processed

	parser := &fakeParser{
		portal: "idealista",
		parseFn: func(body []byte) ([]listing.Listing, error) {
			return []listing.Listing{{Portal: "idealista", ExternalID: "1", URL: "https://example.com/1"}}, nil
		},
	}
	notifier := &fakeNotifier{}

	if err := Run(ctx, provider, s, []MessageParser{parser}, notifier); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if len(notifier.notified) != 1 {
		t.Fatalf("got %d notifications, want 1 (only for uid 1, not the already-processed uid 2)", len(notifier.notified))
	}
	if notifier.notified[0].ExternalID != "1" {
		t.Errorf("notified listing external_id = %q, want %q", notifier.notified[0].ExternalID, "1")
	}
}

func TestRun_SkipsAlreadyProcessed(t *testing.T) {
	ctx := context.Background()

	provider := &fakeProvider{
		uids: map[string][]uint32{"idealista": {1}},
	}
	s := newFakeStore()
	s.processed["idealista"] = map[uint32]bool{1: true}

	parseCalled := false
	parser := &fakeParser{
		portal: "idealista",
		parseFn: func(body []byte) ([]listing.Listing, error) {
			parseCalled = true
			return nil, nil
		},
	}

	if err := Run(ctx, provider, s, []MessageParser{parser}, nil); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if parseCalled {
		t.Error("Parse was called for an already-processed message")
	}
}

func TestRun_NarrowsListUIDsToWatermarkMinusLookback(t *testing.T) {
	ctx := context.Background()

	provider := &fakeProvider{
		uids: map[string][]uint32{"idealista": {1500}},
		bodies: map[string]map[uint32][]byte{
			"idealista": {1500: []byte("raw email")},
		},
	}
	s := newFakeStore()
	s.processed["idealista"] = map[uint32]bool{1000: true} // watermark = 1000

	parser := &fakeParser{
		portal:  "idealista",
		parseFn: func(body []byte) ([]listing.Listing, error) { return nil, nil },
	}

	if err := Run(ctx, provider, s, []MessageParser{parser}, nil); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	want := uint32(1000 - retryLookback)
	if got := provider.calledSinceUID["idealista"]; got != want {
		t.Errorf("ListUIDs called with sinceUID = %d, want %d (watermark 1000 minus lookback %d)", got, want, retryLookback)
	}
}

func TestRun_WatermarkBelowLookbackListsFromStart(t *testing.T) {
	ctx := context.Background()

	provider := &fakeProvider{uids: map[string][]uint32{"idealista": {}}}
	s := newFakeStore()
	s.processed["idealista"] = map[uint32]bool{5: true} // watermark well under retryLookback

	parser := &fakeParser{
		portal:  "idealista",
		parseFn: func(body []byte) ([]listing.Listing, error) { return nil, nil },
	}

	if err := Run(ctx, provider, s, []MessageParser{parser}, nil); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if got := provider.calledSinceUID["idealista"]; got != 0 {
		t.Errorf("ListUIDs called with sinceUID = %d, want 0 (watermark - lookback must saturate, not underflow)", got)
	}
}

func TestRun_ParseErrorDoesNotMarkProcessedOrAbortRun(t *testing.T) {
	ctx := context.Background()

	provider := &fakeProvider{
		uids: map[string][]uint32{
			"idealista": {1},
			"fotocasa":  {1},
		},
		bodies: map[string]map[uint32][]byte{
			"idealista": {1: []byte("bad email")},
			"fotocasa":  {1: []byte("good email")},
		},
	}
	s := newFakeStore()

	brokenParser := &fakeParser{
		portal: "idealista",
		parseFn: func(body []byte) ([]listing.Listing, error) {
			return nil, errors.New("malformed email")
		},
	}
	workingParser := &fakeParser{
		portal: "fotocasa",
		parseFn: func(body []byte) ([]listing.Listing, error) {
			return []listing.Listing{{Portal: "fotocasa", ExternalID: "9", URL: "https://example.com/9"}}, nil
		},
	}

	if err := Run(ctx, provider, s, []MessageParser{brokenParser, workingParser}, nil); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if s.processed["idealista"][1] {
		t.Error("message with parse error was marked processed")
	}
	if !s.processed["fotocasa"][1] {
		t.Error("message in the working mailbox was not marked processed")
	}
	if len(s.saved) != 1 {
		t.Fatalf("got %d saved listings, want 1 (only from the working mailbox)", len(s.saved))
	}
}

func TestRegister_PanicsOnDuplicatePortal(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Register did not panic on duplicate portal")
		}
	}()

	p := &fakeParser{portal: "duplicate-test-portal", parseFn: func([]byte) ([]listing.Listing, error) { return nil, nil }}
	Register(p)
	Register(p)
}
