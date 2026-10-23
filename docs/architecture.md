# Architecture

## Design principles

1. **One code path for live, paper, and backtest.** Strategies and the OMS see the same interfaces regardless of where events come from. The only thing that changes is the event source (live feed vs. replay) and the clock (wall clock vs. virtual).
2. **Events are the source of truth.** Everything downstream of the feed adapters consumes normalized, immutable events. Replaying the same event stream must produce the same final state.
3. **Modular monolith first.** Start as a single Go process with typed in-process events. Split into services (and add gRPC/Kafka) only when a real constraint demands it.
4. **Correctness before breadth.** Two venues done correctly (gap recovery, invariants, benchmarks) beats ten venues done superficially.
5. **No binary floating point for money.** Prices, quantities, and balances are fixed-point integers with per-product scale metadata.

## Components

### Market-data adapters

One adapter per venue. Responsibilities:

- Connect to the venue WebSocket feed
- Fetch the initial order-book snapshot (REST)
- Buffer incremental updates until the snapshot is applied
- Translate venue messages into normalized events
- Detect sequence gaps and trigger re-sync
- Record both exchange timestamp and local receipt timestamp

Normalized event types:

```text
BookSnapshot { venue, product, sequence, bids[], asks[], exchange_ts, received_ts }
BookDelta    { venue, product, sequence, side, price, quantity, exchange_ts, received_ts }
Trade        { venue, product, sequence, side, price, quantity, exchange_ts, received_ts }
```

### Order-book engine

Maintains the local L2 book per (venue, product):

- Bid/ask price levels with aggregate quantities
- Best bid/offer, spread, midpoint
- Incremental application of deltas
- Enforced invariants (see [correctness.md](correctness.md))
- Deterministic snapshots (content-hashable)
- Derived metrics (e.g. book imbalance)

Benchmarked for updates/sec and p50/p95/p99 update latency.

### Event recorder & replay engine

- **Recorder:** appends normalized events to partitioned Parquet files:

  ```text
  data/
    venue=coinbase/
      product=BTC-USD/
        date=2026-09-21/
          events-0001.parquet
  ```

- **Replayer:** reads Parquet, emits events in order on a virtual clock. Supports real-time speed, accelerated replay, pause/resume, step, and seek-to-timestamp. Seeded clock and RNG for full determinism.

### Matching engine

Exchange-style simulator:

- Limit and market orders, cancels
- Price-time priority
- Partial fills
- Maker/taker fees
- Order validation (tick size, lot size)
- Self-trade prevention
- Rejections with reasons

See [execution-model.md](execution-model.md) for what is modeled exactly vs. approximated (queue position, latency).

### OMS & risk

Order lifecycle state machine:

```text
PendingNew → Open → PartiallyFilled → Filled
                 └→ PendingCancel → Cancelled
                 └→ Rejected
```

Pre-trade risk checks (configurable):

- Max order notional
- Max open position
- Max outstanding orders
- Price collar
- Sufficient balance
- Daily loss limit
- Global kill switch

### Strategy API

Strategies are deliberately simple — the infrastructure is the point.

```go
type Strategy interface {
    OnBook(ctx Context, book BookView) []OrderCommand
    OnTrade(ctx Context, trade Trade) []OrderCommand
    OnFill(ctx Context, fill Fill) []OrderCommand
}
```

Reference strategies: (1) spread/inventory-driven market making, (2) order-book imbalance.

### API & UI

- Go API server exposing WebSocket streams (book, trades, orders, fills, metrics) and REST for run control
- React UI: depth chart, ladder, recent trades, order/fill blotter, P&L, replay controls, run comparison

## Module layout

```text
/cmd                       # binaries (market-replay CLI, marketdata-daemon)
/internal/marketdata       # normalized events, Source contract, venue registry
/internal/marketdata/sim   # deterministic simulated feed source (implemented)
/internal/orderbook        # L2 book engine (implemented)
/internal/daemon           # book registry + dispatch loop (implemented)
/internal/recorder         # event sinks; JSONL implemented, Parquet pending
/internal/replayio         # JSONL event codec (decoder + encoder)
/internal/cli              # replay/validate/bench CLI (implemented)
/internal/api              # HTTP API server (REST implemented, WebSocket pending)
/internal/replay           # replay engine + virtual clock (pending)
/internal/matching         # matching engine (pending)
/internal/oms              # order state machine (pending)
/internal/risk             # pre-trade risk checks (pending)
/internal/portfolio        # positions, balances, P&L (pending)
/web                       # React UI (pending)
/python                    # notebooks, analysis, DuckDB queries (pending)
/docs                      # this documentation
```

## Data stores

| Store | Purpose |
|---|---|
| Parquet | Immutable historical market events (source of truth) |
| DuckDB | Ad-hoc analytics over Parquet, backtest run comparison |
| PostgreSQL | Control plane: run metadata, strategy configs, order/fill summaries |

Order-book updates are **not** written to PostgreSQL.

## Numeric representation

```go
type Price    int64  // in product's price atomic units
type Quantity int64  // in product's quantity atomic units

type ProductSpec struct {
    PriceScale    int64
    QuantityScale int64
    TickSize      Price
    LotSize       Quantity
}
```

All matching, risk, and P&L arithmetic happens in fixed-point. Decimals exist only at system boundaries (feed parsing, UI display).

## Observability

Metrics exposed via Prometheus:

- Feed messages/sec, sequence gaps, reconnect count per venue
- Book-update latency histogram
- Replay throughput (events/sec, speed factor)
- Order ack / fill latency
- Rejected order count by reason
- Strategy P&L, inventory, drawdown

Tracing via OpenTelemetry; profiling via Go pprof.
