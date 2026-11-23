import pandas as pd
from trading_config import taker_rate, slippage, cross_venue_threshold

# # best_bid_ask contains top of book for different venues over time (sequence)
# def cross_venue(best_bid_ask):
#         # finds when there is a cross spread in different venues
#     for best_bid_ask where venue != venue
#         best_bid_ask[bid] > best_bid_ask[ask]

#         # S = spread(bid - ask)
#         # Q = quantity
#         entry_fee = abs(quantity * entry_price) * taker_rate
#         exit_fee = abs(quantity * exit_price) * taker_rate
    
#     threshold = 2
#     if S - entry_fee - exit_fee - 2*Q*slippage is > threshold
#         signal = 1
#     else
#         signal = 0

#     signal.append(signal)   

#     trades = signal.shift(1)
# -------------------------------------------------------------------------------


def cross_venue(order_books):
    latest = {}
    results = []

    # Assumes sequence gives the global chronological event order.
    for row in order_books.sort_values("sequence").itertuples(index=False):
        # Replace this venue's previous book, including empty books.
        latest[(row.instrument, row.venue)] = (row.bids, row.asks)

        # Compare the updated venue against other venues for this instrument.
        for (instrument, venue), (bids, asks) in latest.items():
            if instrument != row.instrument or venue == row.venue:
                continue

            for buy_venue, buy_asks, sell_venue, sell_bids in (
                (row.venue, row.asks, venue, bids),
                (venue, asks, row.venue, row.bids),
            ):
                if not buy_asks or not sell_bids:
                    continue

                # Works even if book levels are not sorted.
                ask, ask_size = min(buy_asks, key=lambda level: level[0])
                bid, bid_size = max(sell_bids, key=lambda level: level[0])
                # If books are already sorted with the best price first, use to avoid scanning every entry.
                # ask, ask_size = buy_asks[0]
                # bid, bid_size = sell_bids[0]

                # Limit execution to the available top-of-book quantity.
                qty = min(ask_size, bid_size)
                if qty <= 0:
                    continue

                entry_fee = qty * abs(ask) * taker_rate
                exit_fee = qty * abs(bid) * taker_rate

                slippage_cost = qty * (ask + bid) * slippage

                net_profit = (
                    qty * (bid - ask)
                    - entry_fee
                    - exit_fee
                    - slippage_cost
                )

                results.append({
                    "sequence": row.sequence,
                    "instrument": instrument,
                    "buy_venue": buy_venue,
                    "sell_venue": sell_venue,
                    "buy_price": ask,
                    "sell_price": bid,
                    "quantity": qty,
                    "net_profit": net_profit,
                    "signal": int(bid > ask and net_profit > cross_venue_threshold),
                })

    return pd.DataFrame(results, columns=[
        "sequence", "instrument", "buy_venue", "sell_venue",
        "buy_price", "sell_price", "quantity", "net_profit", "signal",
    ])