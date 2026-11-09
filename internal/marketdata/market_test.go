package marketdata

import "testing"

func TestParseMarketKey(t *testing.T) {
	t.Parallel()

	got, err := ParseMarketKey("sim:BTC-USD")
	if err != nil {
		t.Fatalf("ParseMarketKey() error = %v", err)
	}
	if got.Venue != "sim" || got.Instrument != "BTC-USD" {
		t.Fatalf("ParseMarketKey() = %+v", got)
	}
}
