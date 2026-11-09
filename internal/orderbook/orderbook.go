package orderbook

import (
	"sixletters/back-to-the-past/internal/model"
)

type OrderBookLevel uint8

const (
	OrderBookLevelL1 OrderBookLevel = 1
	OrderBookLevelL2 OrderBookLevel = 2
	OrderBookLevelL3 OrderBookLevel = 3
)

type OrderBook interface {
	Market() model.MarketKey
	Cursor() model.SourceCursor
	IsReady() bool
	OrderBookLevel() OrderBookLevel

	// Mutations (L2)
	ApplySnapshot(model.BookState) error
	ApplyUpdate(model.BookStateUpdate) error

	// L2: market by price (aggregated quantity per price level)
	View(depth int) model.BookView
	Level(side model.Side, price model.Price) (model.PriceLevel, bool)

	// L1: top-of-book projections derived from L2 state
	BestBid() (model.PriceLevel, bool)
	BestAsk() (model.PriceLevel, bool)
	Spread() (model.Price, bool)
	MidPrice() (model.Price, bool)
}
