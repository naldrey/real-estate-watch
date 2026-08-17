// Package monitor tracks whether the watcher's poll cycle is healthy and
// alerts an operator on failure/recovery, so a stopped-working watcher
// doesn't fail silently.
package monitor

import (
	"context"
	"fmt"
	"log/slog"
)

// Alerter sends a plain text message somewhere the operator will see it
// (e.g. Telegram).
type Alerter interface {
	SendMessage(ctx context.Context, text string) error
}

// PollHealth tracks whether the most recent poll cycle succeeded. It alerts
// only on failing/healthy transitions rather than on every cycle, so a
// prolonged outage produces one alert instead of one per poll interval.
type PollHealth struct {
	alerter Alerter // nil disables alerting; transitions are still logged
	failing bool
}

// New creates a PollHealth that alerts via alerter. alerter may be nil, in
// which case transitions are logged but no alert is sent.
func New(alerter Alerter) *PollHealth {
	return &PollHealth{alerter: alerter}
}

// Failure reports that a poll cycle failed with cause. It alerts only the
// first time this happens after a healthy (or initial) state.
func (h *PollHealth) Failure(ctx context.Context, cause error) {
	if h.failing {
		return
	}
	h.failing = true

	slog.Warn("poll health: started failing", "cause", cause)
	h.notify(ctx, fmt.Sprintf("real-estate-watch: polling started failing: %v", cause))
}

// Success reports that a poll cycle succeeded. It alerts only if the
// previous cycle had failed, i.e. on recovery.
func (h *PollHealth) Success(ctx context.Context) {
	if !h.failing {
		return
	}
	h.failing = false

	slog.Info("poll health: recovered")
	h.notify(ctx, "real-estate-watch: polling has recovered")
}

func (h *PollHealth) notify(ctx context.Context, text string) {
	if h.alerter == nil {
		return
	}
	if err := h.alerter.SendMessage(ctx, text); err != nil {
		slog.Error("failed to send poll health alert", "err", err)
	}
}
