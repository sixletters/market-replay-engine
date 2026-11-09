package replayio

import (
	"strings"
	"testing"

	"sixletters/back-to-the-past/internal/model"
)

const sampleStream = `{"type":"snapshot","market":{"venue":"coinbase","instrument":"BTC-USD"},"cursor":{"sequence":100},"bids":[{"price":100,"quantity":5}],"asks":[{"price":102,"quantity":4}]}
{"type":"update","market":{"venue":"coinbase","instrument":"BTC-USD"},"cursor":{"sequence":101},"changes":[{"side":0,"price":100,"quantityAfter":3}]}
`

func TestDecoderSnapshotThenUpdate(t *testing.T) {
	t.Parallel()

	d := NewDecoder(strings.NewReader(sampleStream))

	if !d.Next() {
		t.Fatalf("Next() = false, want true (err = %v)", d.Err())
	}
	ev := d.Event()
	if ev.Kind != KindSnapshot || ev.Snapshot == nil {
		t.Fatalf("event 1 Kind = %v, want snapshot", ev.Kind)
	}
	if ev.Line != 1 {
		t.Fatalf("event 1 Line = %d, want 1", ev.Line)
	}
	if got := ev.Snapshot.Market; got != (model.MarketKey{Venue: "coinbase", Instrument: "BTC-USD"}) {
		t.Fatalf("snapshot market = %+v", got)
	}
	if got := ev.Snapshot.LastAppliedCursor.Sequence; got != 100 {
		t.Fatalf("snapshot sequence = %d, want 100", got)
	}
	if got := ev.Snapshot.Bids[0]; got != (model.PriceLevel{Price: 100, Quantity: 5}) {
		t.Fatalf("snapshot bid[0] = %+v", got)
	}

	if !d.Next() {
		t.Fatalf("Next() = false, want true (err = %v)", d.Err())
	}
	ev = d.Event()
	if ev.Kind != KindUpdate || ev.Update == nil {
		t.Fatalf("event 2 Kind = %v, want update", ev.Kind)
	}
	ch := ev.Update.Changes[0]
	if ch.Side != model.SideBid || ch.Price != 100 || ch.QuantityAfter != 3 {
		t.Fatalf("update change = %+v", ch)
	}

	if d.Next() {
		t.Fatal("Next() = true, want false at EOF")
	}
	if err := d.Err(); err != nil {
		t.Fatalf("Err() = %v, want nil", err)
	}
}

func TestDecoderMalformedLine(t *testing.T) {
	t.Parallel()

	stream := `{"type":"snapshot","market":{"venue":"coinbase","instrument":"BTC-USD"},"cursor":{"sequence":100},"bids":[],"asks":[]}
{not json}
`
	d := NewDecoder(strings.NewReader(stream))
	if !d.Next() {
		t.Fatalf("Next() = false on first line, want true (err = %v)", d.Err())
	}
	if d.Next() {
		t.Fatal("Next() = true on malformed line, want false")
	}
	if err := d.Err(); err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("Err() = %v, want error mentioning line 2", err)
	}
}

func TestDecoderUnknownType(t *testing.T) {
	t.Parallel()

	d := NewDecoder(strings.NewReader(`{"type":"trade"}` + "\n"))
	if d.Next() {
		t.Fatal("Next() = true, want false")
	}
	if err := d.Err(); err == nil || !strings.Contains(err.Error(), `unknown event type "trade"`) {
		t.Fatalf("Err() = %v, want unknown type error", err)
	}
}

func TestReadAll(t *testing.T) {
	t.Parallel()

	events, err := ReadAll(strings.NewReader(sampleStream))
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("ReadAll() decoded %d events, want 2", len(events))
	}
	if events[0].Kind != KindSnapshot || events[1].Kind != KindUpdate {
		t.Fatalf("event kinds = %v, %v", events[0].Kind, events[1].Kind)
	}
}
