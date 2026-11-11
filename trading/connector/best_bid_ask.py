import pandas as pd


def to_best_bid_ask(book_history):
    rows = []

    for event in book_history.itertuples(index=False):
        bid_price, bid_quantity = event.bids[0] if event.bids else (None, None)
        ask_price, ask_quantity = event.asks[0] if event.asks else (None, None)
        rows.append((
            event.venue,
            event.instrument,
            event.sequence,
            bid_price,
            bid_quantity,
            ask_price,
            ask_quantity,
        ))

    return pd.DataFrame(rows, columns=[
        "venue", "instrument", "sequence",
        "bid_price", "bid_quantity", "ask_price", "ask_quantity",
    ])


if __name__ == "__main__":
    from eventdata import load_orderbook

    book_history = load_orderbook()
    best_bid_ask = to_best_bid_ask(book_history)
    print(best_bid_ask.head(10).to_string(index=False))
