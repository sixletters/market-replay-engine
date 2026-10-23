// marketdata-daemon runs a long-lived market-data service: it consumes a
// pluggable feed source, maintains one order book per market, optionally
// records normalized events to JSONL, and serves an HTTP API.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"sixletters/back-to-the-past/internal/api"
	"sixletters/back-to-the-past/internal/daemon"
	"sixletters/back-to-the-past/internal/marketdata"
	"sixletters/back-to-the-past/internal/marketdata/sim"
	"sixletters/back-to-the-past/internal/model"
	"sixletters/back-to-the-past/internal/recorder"
)

func main() {
	os.Exit(run())
}

func run() int {
	fs := flag.NewFlagSet("marketdata-daemon", flag.ContinueOnError)
	addr := fs.String("addr", ":8080", "HTTP listen address")
	source := fs.String("source", "sim", "feed source name (registered in the venue registry)")
	marketsFlag := fs.String("markets", "sim:BTC-USD", "comma-separated venue:instrument pairs")
	simSeed := fs.Int64("sim-seed", 42, "sim source: RNG seed")
	simInterval := fs.Duration("sim-interval", 100*time.Millisecond, "sim source: time between update ticks")
	recordDir := fs.String("record-dir", "", "directory for JSONL recordings (empty = recording disabled)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return 1
	}

	// Generic venue framework: sources are resolved through a registry, so
	// adding a new exchange = registering a new factory here.
	registry := marketdata.NewRegistry()
	sim.Register(registry)

	markets, err := parseMarkets(*marketsFlag)
	if err != nil {
		log.Printf("error: %v", err)
		return 1
	}

	src, err := registry.New(*source, marketdata.SourceConfig{
		Markets: markets,
		Options: map[string]string{
			"seed":     fmt.Sprint(*simSeed),
			"interval": simInterval.String(),
		},
	})
	if err != nil {
		log.Printf("error: %v", err)
		return 1
	}

	var sinks []recorder.Sink
	if *recordDir != "" {
		sink, err := recorder.NewJSONL(*recordDir)
		if err != nil {
			log.Printf("error: %v", err)
			return 1
		}
		sinks = append(sinks, sink)
		log.Printf("recording to %s", *recordDir)
	}

	d := daemon.New(src, sinks...)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:              *addr,
		Handler:           api.NewServer(d),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 2)
	daemonDone := make(chan struct{})
	go func() {
		log.Printf("serving HTTP on %s", *addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	go func() {
		defer close(daemonDone)
		if err := d.Run(ctx); err != nil && err != context.Canceled {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Printf("shutting down")
	case err := <-errCh:
		log.Printf("fatal: %v", err)
		stop()
	}

	// Wait for the daemon to flush/close recorders before exiting.
	select {
	case <-daemonDone:
	case <-time.After(5 * time.Second):
		log.Printf("daemon did not stop in time")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	return 0
}

func parseMarkets(spec string) ([]model.MarketKey, error) {
	parts := strings.Split(spec, ",")
	out := make([]model.MarketKey, 0, len(parts))
	for _, p := range parts {
		m, err := marketdata.ParseMarketKey(strings.TrimSpace(p))
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}
