package main

import (
	"testing"
	"time"
)

func TestPercentage(t *testing.T) {
	now := time.Now()
	h := &appUptime{
		Name:      "app",
		FirstSeen: now.Add(-2 * time.Hour),
		LastUp:    true,
		Trans: []utTransition{
			{T: now.Add(-2 * time.Hour), Up: true},
			{T: now.Add(-90 * time.Minute), Up: false},
			{T: now.Add(-30 * time.Minute), Up: true},
		},
	}

	// Fully covered window: 30m up, 1h down, 30m up over 2h => 50%.
	got := h.percentage(now.Add(-2*time.Hour), now)
	if got == nil {
		t.Fatal("expected a percentage for a covered window")
	}
	if *got < 49 || *got > 51 {
		t.Fatalf("expected ~50%%, got %.1f", *got)
	}

	// A window the container has not existed through yields no figure.
	if p := h.percentage(now.Add(-24*time.Hour), now); p != nil {
		t.Fatalf("expected nil for uncovered window, got %v", *p)
	}
}

func TestRestarts(t *testing.T) {
	now := time.Now()
	h := &appUptime{
		FirstSeen: now.Add(-3 * time.Hour),
		LastUp:    true,
		Trans: []utTransition{
			{T: now.Add(-3 * time.Hour), Up: true},
			{T: now.Add(-2 * time.Hour), Up: false},
			{T: now.Add(-90 * time.Minute), Up: true},
			{T: now.Add(-30 * time.Minute), Up: false},
			{T: now.Add(-10 * time.Minute), Up: true},
		},
	}
	if n := h.restarts(now.Add(-time.Hour), now); n != 1 {
		t.Fatalf("expected 1 restart in the last hour, got %d", n)
	}
	if n := h.restarts(now.Add(-24*time.Hour), now); n != 2 {
		t.Fatalf("expected 2 restarts in the last day, got %d", n)
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	first := newUptimeLedger(dir)
	first.mu.Lock()
	first.observeLocked("web", true, now.Add(-2*time.Hour))
	first.observeLocked("web", false, now.Add(-time.Hour))
	first.observeLocked("web", true, now.Add(-30*time.Minute))
	first.mu.Unlock()
	first.flush()

	second := newUptimeLedger(dir)
	h := second.apps["web"]
	if h == nil {
		t.Fatal("ledger did not reload the app")
	}
	if len(h.Trans) != 3 {
		t.Fatalf("expected 3 replayed transitions, got %d", len(h.Trans))
	}
	info := second.summary("web", now)
	if info.Day != nil {
		t.Fatalf("expected no 24h figure for a 2h-old container, got %v", *info.Day)
	}
	if info.Restarts != 1 {
		t.Fatalf("expected 1 restart in the last day, got %d", info.Restarts)
	}
	if p := h.percentage(now.Add(-2*time.Hour), now); p == nil || *p < 74 || *p > 76 {
		t.Fatalf("expected ~75%% over the covered window, got %v", p)
	}
}

func TestPruneKeepsBoundaryState(t *testing.T) {
	now := time.Now()
	h := &appUptime{
		FirstSeen: now.Add(-40 * 24 * time.Hour),
		LastUp:    true,
		Trans: []utTransition{
			{T: now.Add(-40 * 24 * time.Hour), Up: true},
			{T: now.Add(-20 * 24 * time.Hour), Up: false},
			{T: now.Add(-5 * 24 * time.Hour), Up: true},
		},
	}
	h.prune(now.Add(-uptimeRetention))
	// The transition that defines the state at the cutoff (the 40d-old up) is
	// kept so windows starting after it are still computed correctly.
	if len(h.Trans) != 3 || !h.Trans[0].Up {
		t.Fatalf("expected the boundary transition to be kept, got %+v", h.Trans)
	}

	// State before an even older transition is not needed: only the covering
	// one is retained.
	older := &appUptime{Trans: []utTransition{
		{T: now.Add(-50 * 24 * time.Hour), Up: true},
		{T: now.Add(-45 * 24 * time.Hour), Up: false},
		{T: now.Add(-40 * 24 * time.Hour), Up: true},
	}}
	older.prune(now.Add(-uptimeRetention))
	if len(older.Trans) != 1 || !older.Trans[0].Up {
		t.Fatalf("expected only the covering transition to remain, got %+v", older.Trans)
	}
}
