package marketdata

import (
	"time"

	"sixletters/back-to-the-past/internal/model"
)

// Kind identifies the type of a normalized market event.
type Kind uint8

const (
	KindSnapshot Kind = iota + 1
	KindUpdate
)

// Event is a venue-agnostic market event.
// Exactly one of Snapshot or Update is non-nil according to Kind.
type Event struct {
	Kind       Kind
	ReceivedAt time.Time
	Snapshot   *model.BookState
	Update     *model.BookStateUpdate
}
