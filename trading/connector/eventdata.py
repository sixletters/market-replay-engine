from pathlib import Path

import pandas as pd


def load_orderbook(path=None):
    if path is None:
        path = (
            Path(__file__).resolve().parents[2]
            / "data/session-001/venue=sim/instrument=BTC-USD/events.jsonl"
        )

    df = pd.read_json(path, lines=True)
    history = []
    bids, asks = {}, {}
    sequence = None

    for event in df.itertuples(index=False):
        next_sequence = event.cursor["sequence"]

        if event.type == "snapshot":
            bids = {
                level["price"]: level["quantity"]
                for level in event.bids if level["quantity"] > 0
            }
            asks = {
                level["price"]: level["quantity"]
                for level in event.asks if level["quantity"] > 0
            }

        elif event.type == "update":
            if sequence is None:
                raise ValueError("Need a snapshot before applying updates.")
            if next_sequence <= sequence:
                continue  # Skip duplicate or stale updates.
            if next_sequence != sequence + 1:
                raise ValueError(f"Missing events after sequence {sequence}.")

            for change in event.changes:
                book = bids if change["side"] == 0 else asks
                price = change["price"]
                quantity = change["quantityAfter"]

                if quantity == 0:
                    book.pop(price, None)
                else:
                    book[price] = quantity
        else:
            raise ValueError(f"Unknown event type: {event.type}")

        sequence = next_sequence


    # dict version
        history.append({
            "sequence": sequence,
            "venue" : event.market["venue"],
            "instrument" : event.market["instrument"],
            "bids": sorted(bids.items(), reverse=True),
            "asks": sorted(asks.items()),
        })

    return pd.DataFrame(
        history, columns=["sequence", "venue", "instrument", "bids", "asks"]
    )


if __name__ == "__main__":
    book_history = load_orderbook()
    print(book_history.head(10))
