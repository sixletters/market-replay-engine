package model

// Price is a fixed-point price in the smallest supported unit.
type Price int64

// Quantity is a fixed-point quantity in the smallest supported unit.
type Quantity int64

// Side identifies the book side.
type Side uint8

const (
	// SideBid is the buy side of the book.
	SideBid Side = iota
	// SideAsk is the sell side of the book.
	SideAsk
)

// PriceLevel stores aggregated quantity at a price.
type PriceLevel struct {
	Price    Price    `json:"price"`
	Quantity Quantity `json:"quantity"` // L2: aggregated quantity at this price level
}

// MarketKey identifies a venue/product pair.
type MarketKey struct {
	Venue      string `json:"venue"`      // exchange/venue identifier
	Instrument string `json:"instrument"` // product/instrument identifier (e.g. BTC-USD)
}

// SourceCursor tracks the last applied feed sequence.
type SourceCursor struct {
	Sequence int64 `json:"sequence"`
}

// BookState is a full L2 snapshot for one market.
type BookState struct {
	Market            MarketKey    `json:"market"`
	LastAppliedCursor SourceCursor `json:"cursor"`
	Bids              []PriceLevel `json:"bids"` // best -> worst (descending price)
	Asks              []PriceLevel `json:"asks"` // best -> worst (ascending price)
}

// LevelChange is a single L2 price-level update.
type LevelChange struct {
	Side          Side     `json:"side"`
	Price         Price    `json:"price"`
	QuantityAfter Quantity `json:"quantityAfter"` // absolute quantity after applying; 0 means remove level
}

// BookStateUpdate is a sequenced L2 delta for one market.
type BookStateUpdate struct {
	Market  MarketKey     `json:"market"`
	Cursor  SourceCursor  `json:"cursor"`
	Changes []LevelChange `json:"changes"`
}

// BookView is a depth-limited copy of the book state.
type BookView struct {
	Market MarketKey    `json:"market"`
	Cursor SourceCursor `json:"cursor"`
	Bids   []PriceLevel `json:"bids"` // best -> worst (descending price)
	Asks   []PriceLevel `json:"asks"` // best -> worst (ascending price)
}
