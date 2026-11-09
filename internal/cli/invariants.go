package cli

import (
	"errors"
	"fmt"

	"sixletters/back-to-the-past/internal/model"
)

var errInvariant = errors.New("invariant violation")

// checkInvariants enforces the book invariants from docs/correctness.md:
// sorted sides, positive quantities, and an uncrossed book.
func checkInvariants(v model.BookView) error {
	for i, lvl := range v.Bids {
		if lvl.Quantity <= 0 {
			return fmt.Errorf("%w: bid level %d (price %d) has non-positive quantity", errInvariant, i, lvl.Price)
		}
		if i > 0 && v.Bids[i-1].Price <= lvl.Price {
			return fmt.Errorf("%w: bids not strictly descending at level %d", errInvariant, i)
		}
	}
	for i, lvl := range v.Asks {
		if lvl.Quantity <= 0 {
			return fmt.Errorf("%w: ask level %d (price %d) has non-positive quantity", errInvariant, i, lvl.Price)
		}
		if i > 0 && v.Asks[i-1].Price >= lvl.Price {
			return fmt.Errorf("%w: asks not strictly ascending at level %d", errInvariant, i)
		}
	}
	if len(v.Bids) > 0 && len(v.Asks) > 0 && v.Bids[0].Price >= v.Asks[0].Price {
		return fmt.Errorf("%w: crossed book (best bid %d >= best ask %d)", errInvariant, v.Bids[0].Price, v.Asks[0].Price)
	}
	return nil
}
