package monitor

import (
	"context"
	"errors"
	"testing"
)

type fakeAlerter struct {
	messages []string
}

func (f *fakeAlerter) SendMessage(ctx context.Context, text string) error {
	f.messages = append(f.messages, text)
	return nil
}

func TestPollHealth_AlertsOnceOnFirstFailure(t *testing.T) {
	ctx := context.Background()
	alerter := &fakeAlerter{}
	h := New(alerter)

	h.Failure(ctx, errors.New("connect to imap: boom"))
	h.Failure(ctx, errors.New("connect to imap: boom again"))
	h.Failure(ctx, errors.New("connect to imap: boom a third time"))

	if len(alerter.messages) != 1 {
		t.Fatalf("got %d alerts, want 1 (only the first failure should alert): %v", len(alerter.messages), alerter.messages)
	}
}

func TestPollHealth_SuccessDoesNotAlertWhenAlreadyHealthy(t *testing.T) {
	ctx := context.Background()
	alerter := &fakeAlerter{}
	h := New(alerter)

	h.Success(ctx)
	h.Success(ctx)

	if len(alerter.messages) != 0 {
		t.Errorf("got %d alerts, want 0 (never failed, so no recovery to report): %v", len(alerter.messages), alerter.messages)
	}
}

func TestPollHealth_AlertsOnRecoveryThenReAlertsOnNextFailure(t *testing.T) {
	ctx := context.Background()
	alerter := &fakeAlerter{}
	h := New(alerter)

	h.Failure(ctx, errors.New("first outage"))
	h.Success(ctx)
	h.Failure(ctx, errors.New("second outage"))

	want := []string{
		"real-estate-watch: polling started failing: first outage",
		"real-estate-watch: polling has recovered",
		"real-estate-watch: polling started failing: second outage",
	}
	if len(alerter.messages) != len(want) {
		t.Fatalf("got %d alerts, want %d: %v", len(alerter.messages), len(want), alerter.messages)
	}
	for i, msg := range want {
		if alerter.messages[i] != msg {
			t.Errorf("alert %d = %q, want %q", i, alerter.messages[i], msg)
		}
	}
}

func TestPollHealth_NilAlerterDoesNotPanic(t *testing.T) {
	ctx := context.Background()
	h := New(nil)

	h.Failure(ctx, errors.New("boom"))
	h.Success(ctx)
}
