// Package api exposes the daemon's market state over HTTP.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"sixletters/back-to-the-past/internal/daemon"
	"sixletters/back-to-the-past/internal/model"
)

// BookService is the read surface the API needs from the daemon.
type BookService interface {
	Markets() []daemon.MarketStatus
	View(m model.MarketKey, depth int) (model.BookView, error)
}

// Server routes HTTP requests to the book service.
type Server struct {
	svc BookService
	mux *http.ServeMux
}

func NewServer(svc BookService) *Server {
	s := &Server{svc: svc, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /readyz", s.handleReady)
	s.mux.HandleFunc("GET /v1/markets", s.handleMarkets)
	s.mux.HandleFunc("GET /v1/markets/{venue}/{instrument}/book", s.handleBook)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	markets := s.svc.Markets()
	notReady := make([]string, 0)
	for _, m := range markets {
		if !m.Ready {
			notReady = append(notReady, m.Market.Venue+"/"+m.Market.Instrument)
		}
	}
	if len(notReady) > 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status":   "not_ready",
			"notReady": notReady,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleMarkets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.svc.Markets())
}

func (s *Server) handleBook(w http.ResponseWriter, r *http.Request) {
	venue := r.PathValue("venue")
	instrument := r.PathValue("instrument")
	depth := 10
	if raw := r.URL.Query().Get("depth"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 1000 {
			writeError(w, http.StatusBadRequest, "depth must be 1..1000")
			return
		}
		depth = n
	}

	view, err := s.svc.View(model.MarketKey{Venue: venue, Instrument: instrument}, depth)
	if err != nil {
		switch {
		case errors.Is(err, daemon.ErrUnknownMarket):
			writeError(w, http.StatusNotFound, fmt.Sprintf("unknown market %s/%s", venue, instrument))
		case errors.Is(err, daemon.ErrNotReady):
			writeError(w, http.StatusConflict, "market not ready")
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
