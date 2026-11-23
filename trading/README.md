# Trading

Python workspace for loading recorded market data, developing strategies, and presenting strategy research and backtest results.

## Folder contents and planned structure

```text
trading/
|-- connector/                       # Read recordings and prepare market data
|   |-- eventdata.py                 # Reconstruct order books from JSONL snapshots and updates
|   `-- best_bid_ask.py              # Extract best bid/ask prices and quantities
|-- backtest/                        # Strategy research notebooks and backtest results
|   |-- trading_config.py            # Fee, slippage, annualization, and strategy threshold settings
|   `-- <strategy>.ipynb             # Research: backtest runs, charts, and findings
|-- strategy/                        # Strategy logic and design
|   |-- cross_venue/
|   `-- cross_vwap/
|-- analysis/                        # Reusable performance calculations
|   `-- metrics.py                   # Return, CAGR, drawdown, volatility, Sharpe, and Sortino
`-- README.md
```

Keep data loading in `connector/`, reusable strategy logic in `strategy/`, and performance calculations in `analysis/`. Use `backtest/` to research each strategy and present its backtest results, including assumptions, trades, portfolio performance, charts, and conclusions. Strategy notebooks shown above are planned additions.

## Current workflow

```text
Recorded market data
        |
        v
Connector code reads the data
        |
        v
Backtest notebook prepares the data and calls a strategy
        |
        v
Strategy Python file runs within the backtest
        |
        v
Notebook displays backtest results and analysis
```

The cross-venue Python strategy currently estimates opportunities. Complete trade simulation and strategy research notebooks with backtest results are planned.
