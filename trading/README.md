# Python Strategy Backtesting

This workspace will support Python backtesting in Jupyter Notebooks. The workflow will load recorded market data through reusable connector code, prepare it in a notebook, and run strategies defined in separate Python files.

## Planned structure

- **`connector/`** will contain code for reading the market data output by the recording system and making it available to the backtest notebooks. The existing `booksnapshot_call.py` and `booksnapshot_requirements.md` will be rewritten for this purpose; their names may also change.
- **`backtest/`** will contain Jupyter Notebooks that import connector code to load and transform data, call strategies, run backtests, and inspect results.
- **`strategy/`** will contain multiple strategies as Python files. The backtest notebooks will import and run the selected strategy.

## Workflow

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

## Future implementation

Later work will connect the Python strategies to a trading engine that processes current order-book data, runs the strategy logic to generate signals, and manages order execution based on those signals.
