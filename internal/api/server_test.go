package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sixletters/back-to-the-past/internal/daemon"
	"sixletters/back-to-the-past/internal/model"
)

type fakeService struct {
	markets []daemon.MarketStatus
	views   map[model.MarketKey]model.BookView
}

func (f *fakeService) Markets() []daemon.MarketStatus { return f.markets }

func (f *fakeService) View(m model.MarketKey, depth int) (model.BookView, error) {
	v, ok := f.views[m]
	if !ok {
		return model.BookView{}, daemon.ErrUnknownMarket
	}
	return v, nil
}

func newFakeService() *fakeService {
	market := model.MarketKey{Venue: "sim", Instrument: "BTC-USD"}
	return &fakeService{
		markets: []daemon.MarketStatus{{
			Market: market, Ready: true,
			Cursor: model.SourceCursor{Sequence: 42}, Applied: 42,
			LastEvent: time.Now(),
		}},
		views: map[model.MarketKey]model.BookView{
			market: {
				Market: market,
				Cursor: model.SourceCursor{Sequence: 42},
				Bids:   []model.PriceLevel{{Price: 100, Quantity: 5}},
				Asks:   []model.PriceLevel{{Price: 102, Quantity: 4}},
			},
		},
	}
}

func TestHealthz(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(NewServer(newFakeService()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestReadyz(t *testing.T) {
	t.Parallel()

	svc := newFakeService()
	srv := httptest.NewServer(NewServer(svc))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	svc.markets[0].Ready = false
	resp, err = http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
}

func TestMarkets(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(NewServer(newFakeService()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/markets")
	if err != nil {
		t.Fatalf("GET /v1/markets: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var markets []daemon.MarketStatus
	if err := json.NewDecoder(resp.Body).Decode(&markets); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(markets) != 1 || markets[0].Market.Instrument != "BTC-USD" {
		t.Fatalf("markets = %+v", markets)
	}
}

func TestBook(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(NewServer(newFakeService()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/markets/sim/BTC-USD/book?depth=5")
	if err != nil {
		t.Fatalf("GET book: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body struct {
		Market struct {
			Venue      string `json:"venue"`
			Instrument string `json:"instrument"`
		} `json:"market"`
		Cursor struct {
			Sequence int64 `json:"sequence"`
		} `json:"cursor"`
		Bids []struct {
			Price    int64 `json:"price"`
			Quantity int64 `json:"quantity"`
		} `json:"bids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Market.Venue != "sim" || body.Market.Instrument != "BTC-USD" {
		t.Fatalf("market = %+v", body.Market)
	}
	if body.Cursor.Sequence != 42 {
		t.Fatalf("cursor = %d, want 42", body.Cursor.Sequence)
	}
	if len(body.Bids) != 1 || body.Bids[0].Price != 100 {
		t.Fatalf("bids = %+v", body.Bids)
	}
}

func TestBookUnknownMarket(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(NewServer(newFakeService()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/markets/sim/ETH-USD/book")
	if err != nil {
		t.Fatalf("GET book: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestBookInvalidDepth(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(NewServer(newFakeService()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/markets/sim/BTC-USD/book?depth=abc")
	if err != nil {
		t.Fatalf("GET book: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
