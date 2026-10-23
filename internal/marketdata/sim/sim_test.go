package sim

import (
	"context"
	"testing"
	"time"

	"sixletters/back-to-the-past/internal/marketdata"
	"sixletters/back-to-the-past/internal/model"
)

func TestSimDeterministic(t *testing.T) {
	t.Parallel()

	markets := []model.MarketKey{{Venue: "sim", Instrument: "BTC-USD"}}
	run := func() []marketdata.Event {
		src := New(markets, Options{Seed: 42, Interval: 0, UpdatesPerTick: 2})
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		out := make(chan marketdata.Event, 32)
		go func() { _ = src.Run(ctx, out) }()
		var events []marketdata.Event
		for ev := range out {
			events = append(events, ev)
			if len(events) == 7 {
				break
			}
		}
		return events
	}

	a, b := run(), run()
	if len(a) != len(b) {
		t.Fatalf("len mismatch: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Kind != b[i].Kind {
			t.Fatalf("event %d kind mismatch", i)
		}
		if a[i].Snapshot != nil && b[i].Snapshot != nil {
			if a[i].Snapshot.LastAppliedCursor != b[i].Snapshot.LastAppliedCursor {
				t.Fatalf("snapshot %d cursor mismatch", i)
			}
		}
		if a[i].Update != nil && b[i].Update != nil {
			if a[i].Update.Cursor != b[i].Update.Cursor {
				t.Fatalf("update %d cursor mismatch", i)
			}
		}
	}
}

func TestSimGapEvery(t *testing.T) {
	t.Parallel()

	markets := []model.MarketKey{{Venue: "sim", Instrument: "BTC-USD"}}
	src := New(markets, Options{Seed: 1, Interval: 0, UpdatesPerTick: 1, GapEvery: 2})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	out := make(chan marketdata.Event, 16)
	go func() { _ = src.Run(ctx, out) }()

	var seqs []int64
	for len(seqs) < 6 {
		select {
		case ev := <-out:
			if ev.Kind == marketdata.KindUpdate {
				seqs = append(seqs, ev.Update.Cursor.Sequence)
			}
		case <-ctx.Done():
			t.Fatal("timed out waiting for events")
		}
	}

	// With GapEvery=2 the second update should skip a sequence.
	if seqs[0] != 2 {
		t.Fatalf("first update seq = %d, want 2", seqs[0])
	}
	if seqs[1] <= seqs[0]+1 {
		t.Fatalf("expected a gap, got seqs %v", seqs)
	}
}
