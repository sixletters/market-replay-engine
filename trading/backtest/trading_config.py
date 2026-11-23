# trading_config.py

maker_rate = 0.0002  # 0.02% per filled order, for OKX, BINANCE & COINBASE
taker_rate = 0.0005  # 0.05% per filled order, for OKX< BINACE, 0.04% for COINBASE
slippage = 0.0002  # 0.02% per execution
periods_per_year = 365  # For daily crypto observations only

cross_venue_threshold = 2 # total profit of USD 2 per trade