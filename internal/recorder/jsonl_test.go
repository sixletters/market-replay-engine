package recorder

import (
	"os"
	"path/filepath"
	"testing"

	"sixletters/back-to-the-past/internal/marketdata"
	"sixletters/back-to-the-past/internal/model"
	"sixletters/back-to-the-past/internal/replayio"
)

func TestJSONLInterface(t *testing.T) {
	t.Parallel()

	var _ Sink = (*JSONL)(nil)
}

func TestJSONLRecordRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	j, err := NewJSONL(dir)
	if err != nil {
		t.Fatalf("NewJSONL() error = %v", err)
	}

	market := model.MarketKey{Venue: "sim", Instrument: "BTC-USD"}
	events := []marketdata.Event{
		{Kind: marketdata.KindSnapshot, Snapshot: &model.BookState{
			Market:            market,
			LastAppliedCursor: model.SourceCursor{Sequence: 1},
			Bids:              []model.PriceLevel{{Price: 100, Quantity: 5}},
			Asks:              []model.PriceLevel{{Price: 101, Quantity: 4}},
		}},
		{Kind: marketdata.KindUpdate, Update: &model.BookStateUpdate{
			Market:  market,
			Cursor:  model.SourceCursor{Sequence: 2},
			Changes: []model.LevelChange{{Side: model.SideBid, Price: 100, QuantityAfter: 3}},
		}},
	}
	for _, ev := range events {
		if err := j.Record(ev); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
	}
	if err := j.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	path := filepath.Join(dir, "venue=sim", "instrument=BTC-USD", "events.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open recorded file: %v", err)
	}
	defer f.Close()

	got, err := replayio.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ReadAll() decoded %d events, want 2", len(got))
	}
	if got[0].Kind != replayio.KindSnapshot {
		t.Fatalf("event 0 kind = %v, want snapshot", got[0].Kind)
	}
	if got[1].Kind != replayio.KindUpdate {
		t.Fatalf("event 1 kind = %v, want update", got[1].Kind)
	}
	if got[1].Update.Cursor.Sequence != 2 {
		t.Fatalf("event 1 cursor = %d, want 2", got[1].Update.Cursor.Sequence)
	}
	if got[1].Update.Changes[0].QuantityAfter != 3 {
		t.Fatalf("event 1 change qty = %d, want 3", got[1].Update.Changes[0].QuantityAfter)
	}
}
