.PHONY: build build-daemon run run-daemon test test-verbose test-race bench tidy

GO ?= go

build:
	$(GO) build -o bin/market-replay ./cmd/market-replay

build-daemon:
	$(GO) build -o bin/marketdata-daemon ./cmd/marketdata-daemon

run: build
	./bin/market-replay replay --file fixtures/sample.jsonl

run-daemon: build-daemon
	./bin/marketdata-daemon --markets sim:BTC-USD --record-dir data

test:
	$(GO) test ./...

test-verbose:
	$(GO) test -v ./...

test-race:
	$(GO) test -race ./...

bench:
	$(GO) test -bench=. ./...

tidy:
	$(GO) mod tidy
