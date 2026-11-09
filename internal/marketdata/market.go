package marketdata

import (
	"fmt"
	"strings"

	"sixletters/back-to-the-past/internal/model"
)

// ParseMarketKey parses specs like "sim:BTC-USD" or "coinbase:ETH-USD".
func ParseMarketKey(spec string) (model.MarketKey, error) {
	venue, instrument, ok := strings.Cut(spec, ":")
	if !ok || venue == "" || instrument == "" {
		return model.MarketKey{}, fmt.Errorf("marketdata: invalid market spec %q (want venue:instrument)", spec)
	}
	return model.MarketKey{Venue: venue, Instrument: instrument}, nil
}
