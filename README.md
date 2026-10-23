# Back to the Past

A deterministic, multi-venue **market replay and exchange simulator**.

It consumes live order-book feeds (starting with Coinbase), reconstructs a correct local L2 order book, records normalized market events, and replays them deterministically through an exchange-style matching engine — so strategies can be backtested with realistic fees, latency, slippage, and partial fills.

The core design principle:

> Live trading, paper trading, and historical backtesting all use the **same** strategy and order-management interfaces.

## Why this project

This is an engineering project, not a trading bot. It demonstrates the systems topics that markets/exchange engineering teams care about:

- **Market data** — feed handlers, snapshot + incremental updates, sequence-gap recovery
- **Matching** — price-time priority, partial fills, order lifecycle
- **OMS & risk** — order state machine, pre-trade risk checks, kill switch
- **Determinism** — event sourcing, reproducible replay, seeded clocks
- **Correctness & performance** — invariants, benchmarks, latency histograms

## Architecture at a glance

```text
 Coinbase WebSocket       Second Venue WebSocket
          │                       │
          ▼                       ▼
  Coinbase Adapter          Venue Adapter
          └────────┬──────────────┘
                   ▼
           Normalized Events
      BookSnapshot / BookDelta / Trade
                   │
         ┌─────────┴─────────┐
         ▼                   ▼
  Order Book Engine     Event Recorder (Parquet)
         │                   │
         └─────────┬─────────┘
                   ▼
             Replay Engine
                   │
            Strategy Interface
                   │
                   ▼
       OMS → Risk → Execution Simulator
                   │
                   ▼
          Fills / Positions / P&L
                   │
                   ▼
            WebSocket API → React UI
```

See [docs/architecture.md](docs/architecture.md) for the full design.

## Tech stack

| Area | Choice |
|---|---|
| Core systems | **Go** |
| Research / analysis | **Python** (Polars, DuckDB, Jupyter) |
| UI | **TypeScript + React + Vite** |
| Historical events | **Parquet** |
| Analytical queries | **DuckDB** |
| Metadata (runs, configs) | **PostgreSQL** |
| Live UI transport | **WebSocket** |
| Numeric representation | **Fixed-point integers** (never float64 for money) |
| Observability | **OpenTelemetry + Prometheus + Grafana** |
| Packaging | **Docker Compose** |
| CI | **GitHub Actions** |

## Documentation

- [docs/architecture.md](docs/architecture.md) — components, data flow, module layout
- [docs/correctness.md](docs/correctness.md) — feed sequence handling, book invariants, determinism guarantees
- [docs/execution-model.md](docs/execution-model.md) — what the simulator models exactly vs. approximates
- [docs/roadmap.md](docs/roadmap.md) — phased implementation plan

## Status

🚧 Scaffolding only — see the [roadmap](docs/roadmap.md). Phase 1 (gap-safe order-book reconstruction) is next.

## Non-goals

- Not a candle-based indicator backtester
- Not an "AI trading bot"
- No real-money execution — simulated and paper trading only
