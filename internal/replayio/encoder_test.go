package replayio

import (
	"bytes"
	"strings"
	"testing"

	"sixletters/back-to-the-past/internal/model"
)

func TestEncoderRoundTrip(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	ev := Event{Kind: KindSnapshot, Snapshot: &model.BookState{
		Market:            model.MarketKey{Venue: "sim", Instrument: "BTC-USD"},
		LastAppliedCursor: model.SourceCursor{Sequence: 1},
		Bids:              []model.PriceLevel{{Price: 100, Quantity: 5}},
		Asks:              []model.PriceLevel{{Price: 101, Quantity: 4}},
	}}
	if err := enc.Encode(ev); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if err := enc.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	d := NewDecoder(strings.NewReader(buf.String()))
	if !d.Next() {
		t.Fatalf("Next() = false, err=%v", d.Err())
	}
	got := d.Event()
	if got.Kind != KindSnapshot || got.Snapshot == nil {
		t.Fatalf("decoded event = %+v", got)
	}
	if got.Snapshot.Market.Venue != "sim" || got.Snapshot.Market.Instrument != "BTC-USD" {
		t.Fatalf("decoded market = %+v", got.Snapshot.Market)
	}
	if got.Snapshot.LastAppliedCursor.Sequence != 1 {
		t.Fatalf("decoded cursor = %+v", got.Snapshot.LastAppliedCursor)
	}
}
