package orderbook

import (
	"errors"
	"slices"
	"testing"

	"sixletters/back-to-the-past/internal/model"
)

var _ OrderBook = (*SimpleOrderBook)(nil)

func TestSimpleOrderBookSnapshotAndDerivedValues(t *testing.T) {
	t.Parallel()

	market := model.MarketKey{Venue: "coinbase", Instrument: "BTC-USD"}
	book := NewSimpleOrderBook(market)

	snapshot := model.BookState{
		Market:            market,
		LastAppliedCursor: model.SourceCursor{Sequence: 100},
		Bids: []model.PriceLevel{
			{Price: 100, Quantity: 5},
			{Price: 99, Quantity: 10},
		},
		Asks: []model.PriceLevel{
			{Price: 102, Quantity: 4},
			{Price: 103, Quantity: 6},
		},
	}

	if err := book.ApplySnapshot(snapshot); err != nil {
		t.Fatalf("ApplySnapshot() error = %v", err)
	}

	if got := book.OrderBookLevel(); got != OrderBookLevelL2 {
		t.Fatalf("OrderBookLevel() = %v, want %v", got, OrderBookLevelL2)
	}
	if !book.IsReady() {
		t.Fatal("IsReady() = false, want true")
	}
	if got := book.Cursor(); got != snapshot.LastAppliedCursor {
		t.Fatalf("Cursor() = %+v, want %+v", got, snapshot.LastAppliedCursor)
	}

	bestBid, ok := book.BestBid()
	assertPriceLevel(t, "best bid", bestBid, ok, model.PriceLevel{Price: 100, Quantity: 5})

	bestAsk, ok := book.BestAsk()
	assertPriceLevel(t, "best ask", bestAsk, ok, model.PriceLevel{Price: 102, Quantity: 4})

	if got, ok := book.Spread(); !ok || got != 2 {
		t.Fatalf("Spread() = (%v, %v), want (2, true)", got, ok)
	}
	if got, ok := book.MidPrice(); !ok || got != 101 {
		t.Fatalf("MidPrice() = (%v, %v), want (101, true)", got, ok)
	}

	bidLevel, ok := book.Level(model.SideBid, 99)
	assertPriceLevel(t, "bid level", bidLevel, ok, model.PriceLevel{Price: 99, Quantity: 10})

	askLevel, ok := book.Level(model.SideAsk, 103)
	assertPriceLevel(t, "ask level", askLevel, ok, model.PriceLevel{Price: 103, Quantity: 6})

	missingLevel, ok := book.Level(model.SideBid, 101)
	assertMissingLevel(t, "missing level", missingLevel, ok)

	gotView := book.View(2)
	wantView := model.BookView{
		Market: market,
		Cursor: snapshot.LastAppliedCursor,
		Bids: []model.PriceLevel{
			{Price: 100, Quantity: 5},
			{Price: 99, Quantity: 10},
		},
		Asks: []model.PriceLevel{
			{Price: 102, Quantity: 4},
			{Price: 103, Quantity: 6},
		},
	}
	assertBookViewEqual(t, gotView, wantView)
}

func TestSimpleOrderBookApplyUpdateAndSequenceHandling(t *testing.T) {
	t.Parallel()

	market := model.MarketKey{Venue: "coinbase", Instrument: "BTC-USD"}
	book := NewSimpleOrderBook(market)

	if err := book.ApplySnapshot(model.BookState{
		Market:            market,
		LastAppliedCursor: model.SourceCursor{Sequence: 100},
		Bids:              []model.PriceLevel{{Price: 100, Quantity: 5}},
		Asks:              []model.PriceLevel{{Price: 102, Quantity: 4}, {Price: 103, Quantity: 6}},
	}); err != nil {
		t.Fatalf("ApplySnapshot() error = %v", err)
	}

	update := model.BookStateUpdate{
		Market: market,
		Cursor: model.SourceCursor{Sequence: 101},
		Changes: []model.LevelChange{
			{Side: model.SideBid, Price: 100, QuantityAfter: 3}, // reduce best bid
			{Side: model.SideAsk, Price: 102, QuantityAfter: 0},  // remove best ask
			{Side: model.SideAsk, Price: 101, QuantityAfter: 8},  // new best ask
		},
	}

	if err := book.ApplyUpdate(update); err != nil {
		t.Fatalf("ApplyUpdate() error = %v", err)
	}

	bestBid, ok := book.BestBid()
	assertPriceLevel(t, "updated best bid", bestBid, ok, model.PriceLevel{Price: 100, Quantity: 3})

	bestAsk, ok := book.BestAsk()
	assertPriceLevel(t, "updated best ask", bestAsk, ok, model.PriceLevel{Price: 101, Quantity: 8})

	removedAsk, ok := book.Level(model.SideAsk, 102)
	assertMissingLevel(t, "removed ask level", removedAsk, ok)

	newAsk, ok := book.Level(model.SideAsk, 101)
	assertPriceLevel(t, "new ask level", newAsk, ok, model.PriceLevel{Price: 101, Quantity: 8})

	if got := book.Cursor(); got != update.Cursor {
		t.Fatalf("Cursor() = %+v, want %+v", got, update.Cursor)
	}

	beforeDuplicate := book.View(10)
	if err := book.ApplyUpdate(update); err != nil {
		t.Fatalf("ApplyUpdate() duplicate error = %v, want nil", err)
	}
	assertBookViewEqual(t, book.View(10), beforeDuplicate)

	gap := model.BookStateUpdate{
		Market: market,
		Cursor: model.SourceCursor{Sequence: 103},
		Changes: []model.LevelChange{
			{Side: model.SideBid, Price: 101, QuantityAfter: 9},
		},
	}
	if err := book.ApplyUpdate(gap); !errors.Is(err, ErrGap) {
		t.Fatalf("ApplyUpdate() gap error = %v, want ErrGap", err)
	}
	if book.IsReady() {
		t.Fatal("IsReady() = true after gap, want false")
	}
	assertBookViewEqual(t, book.View(10), beforeDuplicate)
}

func TestSimpleOrderBookRequiresSnapshot(t *testing.T) {
	t.Parallel()

	market := model.MarketKey{Venue: "coinbase", Instrument: "BTC-USD"}
	book := NewSimpleOrderBook(market)

	if err := book.ApplyUpdate(model.BookStateUpdate{
		Market: market,
		Cursor: model.SourceCursor{Sequence: 1},
	}); !errors.Is(err, ErrNotReady) {
		t.Fatalf("ApplyUpdate() error = %v, want ErrNotReady", err)
	}
}

func assertPriceLevel(t *testing.T, name string, got model.PriceLevel, ok bool, want model.PriceLevel) {
	t.Helper()

	if !ok {
		t.Fatalf("%s: missing level, want %+v", name, want)
	}
	if got != want {
		t.Fatalf("%s: got %+v, want %+v", name, got, want)
	}
}

func assertMissingLevel(t *testing.T, name string, got model.PriceLevel, ok bool) {
	t.Helper()

	if ok {
		t.Fatalf("%s: got %+v, want missing level", name, got)
	}
}

func assertBookViewEqual(t *testing.T, got, want model.BookView) {
	t.Helper()

	if got.Market != want.Market {
		t.Fatalf("Market = %+v, want %+v", got.Market, want.Market)
	}
	if got.Cursor != want.Cursor {
		t.Fatalf("Cursor = %+v, want %+v", got.Cursor, want.Cursor)
	}
	if !slices.Equal(got.Bids, want.Bids) {
		t.Fatalf("Bids = %+v, want %+v", got.Bids, want.Bids)
	}
	if !slices.Equal(got.Asks, want.Asks) {
		t.Fatalf("Asks = %+v, want %+v", got.Asks, want.Asks)
	}
}
