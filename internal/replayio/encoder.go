package replayio

import (
	"bufio"
	"encoding/json"
	"io"
)

// Encoder writes the JSONL wire format understood by Decoder.
type Encoder struct {
	w *bufio.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: bufio.NewWriter(w)}
}

// Encode writes one event as a single JSON line.
func (e *Encoder) Encode(ev Event) error {
	raw := rawEvent{}
	switch ev.Kind {
	case KindSnapshot:
		raw.Type = "snapshot"
		if ev.Snapshot != nil {
			raw.Market = ev.Snapshot.Market
			raw.Cursor = ev.Snapshot.LastAppliedCursor
			raw.Bids = ev.Snapshot.Bids
			raw.Asks = ev.Snapshot.Asks
		}
	case KindUpdate:
		raw.Type = "update"
		if ev.Update != nil {
			raw.Market = ev.Update.Market
			raw.Cursor = ev.Update.Cursor
			raw.Changes = ev.Update.Changes
		}
	default:
		return io.ErrUnexpectedEOF
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	if _, err := e.w.Write(data); err != nil {
		return err
	}
	if err := e.w.WriteByte('\n'); err != nil {
		return err
	}
	return nil
}

func (e *Encoder) Flush() error { return e.w.Flush() }

func (e *Encoder) Close() error { return e.w.Flush() }
