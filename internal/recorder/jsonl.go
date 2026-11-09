package recorder

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"sixletters/back-to-the-past/internal/marketdata"
	"sixletters/back-to-the-past/internal/model"
	"sixletters/back-to-the-past/internal/replayio"
)

// JSONL records events to replayio-compatible JSONL files, partitioned by
// venue and instrument:
//
//	<dir>/venue=sim/instrument=BTC-USD/events.jsonl
type JSONL struct {
	dir     string
	mu      sync.Mutex
	writers map[model.MarketKey]*jsonlWriter
}

type jsonlWriter struct {
	f   *os.File
	enc *replayio.Encoder
}

func NewJSONL(dir string) (*JSONL, error) {
	if dir == "" {
		return nil, fmt.Errorf("recorder: record dir is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("recorder: create dir: %w", err)
	}
	return &JSONL{dir: dir, writers: make(map[model.MarketKey]*jsonlWriter)}, nil
}

func (j *JSONL) Record(ev marketdata.Event) error {
	m := marketOf(ev)
	j.mu.Lock()
	defer j.mu.Unlock()

	w, err := j.writerFor(m)
	if err != nil {
		return err
	}
	return w.enc.Encode(marketdata.ToReplayIO(ev))
}

func (j *JSONL) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	var firstErr error
	for _, w := range j.writers {
		if err := w.enc.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := w.f.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	j.writers = make(map[model.MarketKey]*jsonlWriter)
	return firstErr
}

func (j *JSONL) writerFor(m model.MarketKey) (*jsonlWriter, error) {
	if w, ok := j.writers[m]; ok {
		return w, nil
	}
	dir := filepath.Join(j.dir, "venue="+m.Venue, "instrument="+m.Instrument)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("recorder: create market dir: %w", err)
	}
	f, err := os.Create(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("recorder: create events file: %w", err)
	}
	w := &jsonlWriter{f: f, enc: replayio.NewEncoder(f)}
	j.writers[m] = w
	return w, nil
}

func marketOf(ev marketdata.Event) model.MarketKey {
	switch ev.Kind {
	case marketdata.KindSnapshot:
		if ev.Snapshot != nil {
			return ev.Snapshot.Market
		}
	case marketdata.KindUpdate:
		if ev.Update != nil {
			return ev.Update.Market
		}
	}
	return model.MarketKey{}
}
