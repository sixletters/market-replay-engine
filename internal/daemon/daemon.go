package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"sixletters/back-to-the-past/internal/marketdata"
	"sixletters/back-to-the-past/internal/model"
	"sixletters/back-to-the-past/internal/orderbook"
	"sixletters/back-to-the-past/internal/recorder"
)

var (
	ErrUnknownMarket = errors.New("daemon: unknown market")
	ErrNotReady      = errors.New("daemon: market not ready")
)

// MarketStatus is the externally visible per-market state.
type MarketStatus struct {
	Market    model.MarketKey    `json:"market"`
	Ready     bool               `json:"ready"`
	Cursor    model.SourceCursor `json:"cursor"`
	Applied   uint64             `json:"applied"`
	Skipped   uint64             `json:"skipped"`
	Gaps      uint64             `json:"gaps"`
	LastEvent time.Time          `json:"lastEvent"`
	LastError string             `json:"lastError,omitempty"`
}

type slot struct {
	mu     sync.RWMutex
	book   orderbook.OrderBook
	status MarketStatus
}

// Daemon owns the book registry and dispatch loop.
type Daemon struct {
	src   marketdata.Source
	sinks []recorder.Sink
	mu    sync.RWMutex
	books map[model.MarketKey]*slot
}

func New(src marketdata.Source, sinks ...recorder.Sink) *Daemon {
	d := &Daemon{
		src:   src,
		sinks: sinks,
		books: make(map[model.MarketKey]*slot),
	}
	for _, m := range src.Markets() {
		d.books[m] = &slot{
			book:   orderbook.NewSimpleOrderBook(m),
			status: MarketStatus{Market: m},
		}
	}
	return d
}

func (d *Daemon) Run(ctx context.Context) error {
	events := make(chan marketdata.Event, 256)
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.src.Run(ctx, events)
		close(events)
	}()

	unknownSeen := make(map[model.MarketKey]struct{})

	for {
		select {
		case <-ctx.Done():
			// Shutdown: flush and close sinks before returning.
			for _, sink := range d.sinks {
				if err := sink.Close(); err != nil {
					return err
				}
			}
			return ctx.Err()
		case ev, ok := <-events:
			if !ok {
				if err := <-errCh; err != nil && !errors.Is(err, context.Canceled) {
					return err
				}
				for _, sink := range d.sinks {
					if err := sink.Close(); err != nil {
						return err
					}
				}
				return nil
			}
			if err := d.apply(ev, unknownSeen); err != nil {
				return err
			}
		}
	}
}

func (d *Daemon) Markets() []MarketStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]MarketStatus, 0, len(d.books))
	for _, s := range d.books {
		s.mu.RLock()
		out = append(out, s.status)
		s.mu.RUnlock()
	}
	return out
}

func (d *Daemon) View(m model.MarketKey, depth int) (model.BookView, error) {
	d.mu.RLock()
	s, ok := d.books[m]
	d.mu.RUnlock()
	if !ok {
		return model.BookView{}, ErrUnknownMarket
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.status.Ready {
		return model.BookView{}, ErrNotReady
	}
	return s.book.View(depth), nil
}

func (d *Daemon) apply(ev marketdata.Event, unknownSeen map[model.MarketKey]struct{}) error {
	m := eventMarket(ev)
	d.mu.RLock()
	s, ok := d.books[m]
	d.mu.RUnlock()
	if !ok {
		if _, seen := unknownSeen[m]; !seen {
			unknownSeen[m] = struct{}{}
			log.Printf("daemon: dropping event for unknown market %s/%s", m.Venue, m.Instrument)
		}
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if ev.Kind == marketdata.KindSnapshot {
		if ev.Snapshot == nil {
			return fmt.Errorf("daemon: nil snapshot")
		}
		if err := s.book.ApplySnapshot(*ev.Snapshot); err != nil {
			s.status.Ready = false
			s.status.LastError = err.Error()
			return err
		}
		s.status.Ready = true
		s.status.Cursor = s.book.Cursor()
		s.status.Applied++
		s.status.LastError = ""
		s.status.LastEvent = ev.ReceivedAt
		return d.record(ev)
	}

	if ev.Update == nil {
		return fmt.Errorf("daemon: nil update")
	}
	if !s.status.Ready {
		// Wait for the next snapshot after a gap.
		return nil
	}
	if ev.Update.Cursor.Sequence <= s.book.Cursor().Sequence {
		s.status.Skipped++
		s.status.LastEvent = ev.ReceivedAt
		return nil
	}
	if err := s.book.ApplyUpdate(*ev.Update); err != nil {
		if errors.Is(err, orderbook.ErrGap) {
			s.status.Gaps++
			s.status.Ready = false
			s.status.LastError = err.Error()
			return nil
		}
		return err
	}
	s.status.Cursor = s.book.Cursor()
	s.status.Applied++
	s.status.LastError = ""
	s.status.LastEvent = ev.ReceivedAt
	return d.record(ev)
}

func (d *Daemon) record(ev marketdata.Event) error {
	for _, sink := range d.sinks {
		if err := sink.Record(ev); err != nil {
			return err
		}
	}
	return nil
}

func eventMarket(ev marketdata.Event) model.MarketKey {
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
