package recorder

import "sixletters/back-to-the-past/internal/marketdata"

// Sink consumes normalized market events.
type Sink interface {
	Record(ev marketdata.Event) error
	Close() error
}
