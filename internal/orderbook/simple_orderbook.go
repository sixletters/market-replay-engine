package orderbook

import (
	"errors"
	"slices"

	"sixletters/back-to-the-past/internal/model"
)

var (
	ErrNotReady = errors.New("orderbook: not ready")
	ErrGap      = errors.New("orderbook: sequence gap")
)

// SimpleOrderBook is a straightforward L2 implementation of OrderBook using
// a sorted slice + map per side. It is deterministic and easy to reason about.
type SimpleOrderBook struct {
	market model.MarketKey
	cursor model.SourceCursor
	ready  bool

	bids sideBook
	asks sideBook
}

func NewSimpleOrderBook(market model.MarketKey) *SimpleOrderBook {
	return &SimpleOrderBook{
		market: market,
		bids:   newSideBook(false),
		asks:   newSideBook(true),
	}
}

func (b *SimpleOrderBook) Market() model.MarketKey {
	return b.market
}

func (b *SimpleOrderBook) Cursor() model.SourceCursor {
	return b.cursor
}

func (b *SimpleOrderBook) IsReady() bool {
	return b.ready
}

func (b *SimpleOrderBook) OrderBookLevel() OrderBookLevel {
	return OrderBookLevelL2
}

func (b *SimpleOrderBook) ApplySnapshot(state model.BookState) error {
	if state.Market != b.market {
		return errors.New("orderbook: snapshot market mismatch")
	}

	b.bids.reset()
	b.asks.reset()

	for _, lvl := range state.Bids {
		b.bids.set(lvl.Price, lvl.Quantity)
	}
	for _, lvl := range state.Asks {
		b.asks.set(lvl.Price, lvl.Quantity)
	}

	b.cursor = state.LastAppliedCursor
	b.ready = true
	return nil
}

func (b *SimpleOrderBook) ApplyUpdate(update model.BookStateUpdate) error {
	if !b.ready {
		return ErrNotReady
	}
	if update.Market != b.market {
		return errors.New("orderbook: update market mismatch")
	}

	// Sequencing rule:
	// - same or older sequence => stale/duplicate
	// - sequence gap => resync required
	// - exact next sequence => apply
	if update.Cursor.Sequence <= b.cursor.Sequence {
		return nil
	}
	if update.Cursor.Sequence > b.cursor.Sequence+1 {
		b.ready = false
		return ErrGap
	}

	for _, ch := range update.Changes {
		side := &b.bids
		if ch.Side == model.SideAsk {
			side = &b.asks
		}
		side.set(ch.Price, ch.QuantityAfter)
	}

	b.cursor = update.Cursor
	return nil
}

func (b *SimpleOrderBook) View(depth int) model.BookView {
	return model.BookView{
		Market: b.market,
		Cursor: b.cursor,
		Bids:   b.bids.view(depth),
		Asks:   b.asks.view(depth),
	}
}

func (b *SimpleOrderBook) Level(side model.Side, price model.Price) (model.PriceLevel, bool) {
	if side == model.SideBid {
		return b.bids.level(price)
	}
	return b.asks.level(price)
}

func (b *SimpleOrderBook) BestBid() (model.PriceLevel, bool) {
	return b.bids.best()
}

func (b *SimpleOrderBook) BestAsk() (model.PriceLevel, bool) {
	return b.asks.best()
}

func (b *SimpleOrderBook) Spread() (model.Price, bool) {
	bid, okBid := b.BestBid()
	ask, okAsk := b.BestAsk()
	if !okBid || !okAsk {
		return 0, false
	}
	return ask.Price - bid.Price, true
}

func (b *SimpleOrderBook) MidPrice() (model.Price, bool) {
	bid, okBid := b.BestBid()
	ask, okAsk := b.BestAsk()
	if !okBid || !okAsk {
		return 0, false
	}
	return (bid.Price + ask.Price) / 2, true
}

// sideBook is one side of the L2 book.
// prices is sorted with the best price first.
// levels keeps quantity by price.
type sideBook struct {
	ascending bool
	prices    []model.Price
	levels    map[model.Price]model.Quantity
}

func newSideBook(ascending bool) sideBook {
	return sideBook{
		ascending: ascending,
		prices:    make([]model.Price, 0),
		levels:    make(map[model.Price]model.Quantity),
	}
}

func (s *sideBook) reset() {
	s.prices = s.prices[:0]
	clear(s.levels)
}

func (s *sideBook) set(price model.Price, qty model.Quantity) {
	idx, found := s.find(price)
	if qty == 0 {
		if found {
			delete(s.levels, price)
			s.prices = slices.Delete(s.prices, idx, idx+1)
		}
		return
	}
	s.levels[price] = qty
	if !found {
		s.prices = slices.Insert(s.prices, idx, price)
	}
}

func (s *sideBook) best() (model.PriceLevel, bool) {
	if len(s.prices) == 0 {
		return model.PriceLevel{}, false
	}
	price := s.prices[0]
	qty, ok := s.levels[price]
	if !ok {
		return model.PriceLevel{}, false
	}
	return model.PriceLevel{Price: price, Quantity: qty}, true
}

func (s *sideBook) level(price model.Price) (model.PriceLevel, bool) {
	qty, ok := s.levels[price]
	if !ok {
		return model.PriceLevel{}, false
	}
	return model.PriceLevel{Price: price, Quantity: qty}, true
}

func (s *sideBook) view(depth int) []model.PriceLevel {
	if depth <= 0 || len(s.prices) == 0 {
		return nil
	}
	if depth > len(s.prices) {
		depth = len(s.prices)
	}
	out := make([]model.PriceLevel, 0, depth)
	for i := 0; i < depth; i++ {
		price := s.prices[i]
		out = append(out, model.PriceLevel{Price: price, Quantity: s.levels[price]})
	}
	return out
}

func (s *sideBook) find(price model.Price) (int, bool) {
	cmp := func(a, b model.Price) int {
		if a == b {
			return 0
		}
		if s.ascending {
			if a < b {
				return -1
			}
			return 1
		}
		if a > b {
			return -1
		}
		return 1
	}
	return slices.BinarySearchFunc(s.prices, price, cmp)
}
