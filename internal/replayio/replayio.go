// Package replayio decodes normalized market events from a JSONL stream.
//
// One event per line, discriminated by a "type" field:
//
//	{"type":"snapshot","market":{...},"cursor":{"sequence":100},"bids":[...],"asks":[...]}
//	{"type":"update","market":{...},"cursor":{"sequence":101},"changes":[...]}
//
// The decoder is deliberately ignorant of order books — it only parses.
// Other formats (Parquet, live feeds) can produce the same Event values.
package replayio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"sixletters/back-to-the-past/internal/model"
)

// Kind identifies the type of a decoded Event.
type Kind uint8

const (
	KindSnapshot Kind = iota + 1
	KindUpdate
)

// Event is a single decoded market event. Exactly one of Snapshot or Update
// is non-nil, according to Kind.
type Event struct {
	Kind     Kind
	Line     int // 1-based line number in the source stream
	Snapshot *model.BookState
	Update   *model.BookStateUpdate
}

// rawEvent mirrors the on-disk JSONL envelope.
type rawEvent struct {
	Type    string              `json:"type"`
	Market  model.MarketKey     `json:"market"`
	Cursor  model.SourceCursor  `json:"cursor"`
	Bids    []model.PriceLevel  `json:"bids"`
	Asks    []model.PriceLevel  `json:"asks"`
	Changes []model.LevelChange `json:"changes"`
}

// Decoder streams Events from an io.Reader, one JSON object per line.
type Decoder struct {
	scanner *bufio.Scanner
	line    int
	event   Event
	err     error
}

func NewDecoder(r io.Reader) *Decoder {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 64*1024), 4*1024*1024) // allow large snapshot lines
	return &Decoder{scanner: s}
}

// Next advances to the next event. It returns false at EOF or on error;
// check Err after the loop.
func (d *Decoder) Next() bool {
	if d.err != nil {
		return false
	}
	if !d.scanner.Scan() {
		d.err = d.scanner.Err() // nil at clean EOF
		return false
	}
	d.line++
	ev, err := parse(d.scanner.Bytes(), d.line)
	if err != nil {
		d.err = err
		return false
	}
	d.event = ev
	return true
}

// Event returns the most recently decoded event.
func (d *Decoder) Event() Event {
	return d.event
}

// Err returns the first error encountered, if any.
func (d *Decoder) Err() error {
	return d.err
}

// ReadAll decodes an entire stream into memory.
func ReadAll(r io.Reader) ([]Event, error) {
	d := NewDecoder(r)
	var events []Event
	for d.Next() {
		events = append(events, d.Event())
	}
	return events, d.Err()
}

func parse(line []byte, lineNo int) (Event, error) {
	var raw rawEvent
	if err := json.Unmarshal(line, &raw); err != nil {
		return Event{}, fmt.Errorf("replayio: line %d: invalid JSON: %w", lineNo, err)
	}
	switch raw.Type {
	case "snapshot":
		return Event{
			Kind: KindSnapshot,
			Line: lineNo,
			Snapshot: &model.BookState{
				Market:            raw.Market,
				LastAppliedCursor: raw.Cursor,
				Bids:              raw.Bids,
				Asks:              raw.Asks,
			},
		}, nil
	case "update":
		return Event{
			Kind: KindUpdate,
			Line: lineNo,
			Update: &model.BookStateUpdate{
				Market:  raw.Market,
				Cursor:  raw.Cursor,
				Changes: raw.Changes,
			},
		}, nil
	default:
		return Event{}, fmt.Errorf("replayio: line %d: unknown event type %q", lineNo, raw.Type)
	}
}
