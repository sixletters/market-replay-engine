// Package sim provides a deterministic simulated market-data source.
// It emits a snapshot per market followed by seeded random-walk deltas.
package sim

import (
	"context"
	"math/rand/v2"
	"strconv"
	"time"

	"sixletters/back-to-the-past/internal/marketdata"
	"sixletters/back-to-the-past/internal/model"
)

// Options configures the simulated feed.
type Options struct {
	Seed           int64
	Interval       time.Duration // time between update ticks (0 = as fast as possible)
	UpdatesPerTick int           // level changes per update event
	StartPrice     model.Price   // mid price at snapshot time
	GapEvery       int           // inject a sequence gap every N updates (0 = never; for testing resync)
}

// Source is a deterministic simulated marketdata.Source.
type Source struct {
	markets []model.MarketKey
	opts    Options
}

func New(markets []model.MarketKey, opts Options) *Source {
	if opts.UpdatesPerTick <= 0 {
		opts.UpdatesPerTick = 3
	}
	if opts.StartPrice <= 0 {
		opts.StartPrice = 10000
	}
	return &Source{markets: markets, opts: opts}
}

func (s *Source) Markets() []model.MarketKey {
	out := make([]model.MarketKey, len(s.markets))
	copy(out, s.markets)
	return out
}

// Run streams events until ctx is cancelled. Deterministic for a given seed.
func (s *Source) Run(ctx context.Context, out chan<- marketdata.Event) error {
	rng := rand.New(rand.NewPCG(uint64(s.opts.Seed), uint64(s.opts.Seed)>>32))

	for _, m := range s.markets {
		if err := emitSnapshot(ctx, out, m, s.opts); err != nil {
			return err
		}
	}

	seq := map[model.MarketKey]int64{}
	for _, m := range s.markets {
		seq[m] = 1 // snapshot used sequence 1
	}

	var ticker *time.Ticker
	var tick <-chan time.Time
	if s.opts.Interval > 0 {
		ticker = time.NewTicker(s.opts.Interval)
		defer ticker.Stop()
		tick = ticker.C
	}

	updates := 0
	for {
		if s.opts.Interval > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-tick:
			}
		} else {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		for _, m := range s.markets {
			updates++
			next := seq[m] + 1
			if s.opts.GapEvery > 0 && updates%s.opts.GapEvery == 0 {
				next++ // skip a sequence number to simulate a feed gap
			}

			ev := marketdata.Event{
				Kind:       marketdata.KindUpdate,
				ReceivedAt: time.Now(),
				Update: &model.BookStateUpdate{
					Market:  m,
					Cursor:  model.SourceCursor{Sequence: next},
					Changes: randomChanges(rng, s.opts, s.opts.UpdatesPerTick),
				},
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- ev:
			}

			// After a gap, the adapter contract is to resync via a fresh snapshot.
			if next != seq[m]+1 {
				if err := emitSnapshotAt(ctx, out, m, s.opts, next); err != nil {
					return err
				}
			}
			seq[m] = next
		}
	}
}

func emitSnapshot(ctx context.Context, out chan<- marketdata.Event, m model.MarketKey, opts Options) error {
	return emitSnapshotAt(ctx, out, m, opts, 1)
}

func emitSnapshotAt(ctx context.Context, out chan<- marketdata.Event, m model.MarketKey, opts Options, seq int64) error {
	mid := opts.StartPrice
	snap := &model.BookState{
		Market:            m,
		LastAppliedCursor: model.SourceCursor{Sequence: seq},
		Bids: []model.PriceLevel{
			{Price: mid - 1, Quantity: 5},
			{Price: mid - 2, Quantity: 10},
			{Price: mid - 3, Quantity: 4},
		},
		Asks: []model.PriceLevel{
			{Price: mid + 1, Quantity: 4},
			{Price: mid + 2, Quantity: 6},
			{Price: mid + 3, Quantity: 7},
		},
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case out <- marketdata.Event{Kind: marketdata.KindSnapshot, ReceivedAt: time.Now(), Snapshot: snap}:
		return nil
	}
}

// randomChanges produces level changes around the start price: a mix of
// inserts/updates (qty > 0) and removals (qty = 0).
func randomChanges(rng *rand.Rand, opts Options, n int) []model.LevelChange {
	changes := make([]model.LevelChange, 0, n)
	for i := 0; i < n; i++ {
		side := model.SideBid
		offset := model.Price(rng.IntN(5) + 1)
		if rng.IntN(2) == 1 {
			side = model.SideAsk
		}
		price := opts.StartPrice - offset
		if side == model.SideAsk {
			price = opts.StartPrice + offset
		}
		qty := model.Quantity(rng.IntN(10) + 1)
		if rng.IntN(6) == 0 {
			qty = 0 // remove the level occasionally
		}
		changes = append(changes, model.LevelChange{Side: side, Price: price, QuantityAfter: qty})
	}
	return changes
}

// Factory builds a sim source from generic options.
// Recognized options: "seed", "interval" (Go duration), "gapEvery".
func Factory(cfg marketdata.SourceConfig) (marketdata.Source, error) {
	opts := Options{Seed: 42, Interval: 100 * time.Millisecond}
	if v, ok := cfg.Options["seed"]; ok {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		opts.Seed = n
	}
	if v, ok := cfg.Options["interval"]; ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, err
		}
		opts.Interval = d
	}
	if v, ok := cfg.Options["gapEvery"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		opts.GapEvery = n
	}
	return New(cfg.Markets, opts), nil
}

// Register adds the sim source to a registry under the name "sim".
func Register(r *marketdata.Registry) {
	r.Register("sim", Factory)
}
