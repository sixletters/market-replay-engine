package daemon

import (
	"context"
	"errors"
	"testing"
	"time"

	"sixletters/back-to-the-past/internal/marketdata"
	"sixletters/back-to-the-past/internal/model"
)

// scriptSource is a deterministic in-memory Source for tests.
type scriptSource struct {
	markets []model.MarketKey
	events  []marketdata.Event
}

func (s *scriptSource) Markets() []model.MarketKey { return s.markets }

func (s *scriptSource) Run(ctx context.Context, out chan<- marketdata.Event) error {
	for _, ev := range s.events {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- ev:
		}
	}
	return nil
}

var testMarket = model.MarketKey{Venue: "sim", Instrument: "BTC-USD"}

func snapshotEvent(seq int64) marketdata.Event {
	return marketdata.Event{
		Kind:       marketdata.KindSnapshot,
		ReceivedAt: time.Now(),
		Snapshot: &model.BookState{
			Market:            testMarket,
			LastAppliedCursor: model.SourceCursor{Sequence: seq},
			Bids:              []model.PriceLevel{{Price: 100, Quantity: 5}},
			Asks:              []model.PriceLevel{{Price: 102, Quantity: 4}},
		},
	}
}

func updateEvent(seq int64, changes ...model.LevelChange) marketdata.Event {
	return marketdata.Event{
		Kind:       marketdata.KindUpdate,
		ReceivedAt: time.Now(),
		Update: &model.BookStateUpdate{
			Market:  testMarket,
			Cursor:  model.SourceCursor{Sequence: seq},
			Changes: changes,
		},
	}
}

func runDaemon(t *testing.T, src marketdata.Source) *Daemon {
	t.Helper()
	d := New(src)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := d.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v", err)
	}
	return d
}

func TestDaemonAppliesStream(t *testing.T) {
	t.Parallel()

	src := &scriptSource{
		markets: []model.MarketKey{testMarket},
		events: []marketdata.Event{
			snapshotEvent(1),
			updateEvent(2, model.LevelChange{Side: model.SideBid, Price: 100, QuantityAfter: 3}),
			updateEvent(3, model.LevelChange{Side: model.SideAsk, Price: 101, QuantityAfter: 8}),
		},
	}
	d := runDaemon(t, src)

	view, err := d.View(testMarket, 10)
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}
	if got := view.Cursor.Sequence; got != 3 {
		t.Fatalf("cursor = %d, want 3", got)
	}
	if got := view.Bids[0]; got != (model.PriceLevel{Price: 100, Quantity: 3}) {
		t.Fatalf("best bid = %+v", got)
	}
	if got := view.Asks[0]; got != (model.PriceLevel{Price: 101, Quantity: 8}) {
		t.Fatalf("best ask = %+v", got)
	}

	statuses := d.Markets()
	if len(statuses) != 1 {
		t.Fatalf("Markets() len = %d, want 1", len(statuses))
	}
	st := statuses[0]
	if !st.Ready || st.Applied != 3 || st.Gaps != 0 {
		t.Fatalf("status = %+v, want ready, 3 applied, 0 gaps", st)
	}
}

func TestDaemonGapThenResync(t *testing.T) {
	t.Parallel()

	src := &scriptSource{
		markets: []model.MarketKey{testMarket},
		events: []marketdata.Event{
			snapshotEvent(1),
			updateEvent(2, model.LevelChange{Side: model.SideBid, Price: 100, QuantityAfter: 3}),
			updateEvent(4, model.LevelChange{Side: model.SideBid, Price: 99, QuantityAfter: 9}), // gap: 3 missing
			updateEvent(5, model.LevelChange{Side: model.SideBid, Price: 98, QuantityAfter: 1}), // dropped: not ready
			snapshotEvent(6), // resync
			updateEvent(7, model.LevelChange{Side: model.SideBid, Price: 100, QuantityAfter: 2}),
		},
	}
	d := runDaemon(t, src)

	st := d.Markets()[0]
	if !st.Ready {
		t.Fatalf("Ready = false after resync, want true (status %+v)", st)
	}
	if st.Gaps != 1 {
		t.Fatalf("Gaps = %d, want 1", st.Gaps)
	}
	if got := st.Cursor.Sequence; got != 7 {
		t.Fatalf("cursor = %d, want 7", got)
	}

	view, err := d.View(testMarket, 10)
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}
	if got := view.Bids[0]; got != (model.PriceLevel{Price: 100, Quantity: 2}) {
		t.Fatalf("best bid = %+v, want {100 2}", got)
	}
}

func TestDaemonViewUnknownMarket(t *testing.T) {
	t.Parallel()

	src := &scriptSource{markets: []model.MarketKey{testMarket}}
	d := runDaemon(t, src)

	_, err := d.View(model.MarketKey{Venue: "sim", Instrument: "ETH-USD"}, 10)
	if !errors.Is(err, ErrUnknownMarket) {
		t.Fatalf("View() error = %v, want ErrUnknownMarket", err)
	}
}

func TestDaemonViewNotReady(t *testing.T) {
	t.Parallel()

	src := &scriptSource{markets: []model.MarketKey{testMarket}}
	d := runDaemon(t, src)

	_, err := d.View(testMarket, 10)
	if !errors.Is(err, ErrNotReady) {
		t.Fatalf("View() error = %v, want ErrNotReady", err)
	}
}
