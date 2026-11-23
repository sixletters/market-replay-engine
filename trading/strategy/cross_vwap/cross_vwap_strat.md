# L3 Data

## Cross-Venue VWAP Strategy

The strategy compares the same trading pair across exchanges using order-book snapshots containing multiple bid and ask levels.

For a proposed quantity **Q**, it calculates the volume-weighted average execution price (VWAP) by consuming enough order-book depth to fill that quantity. The buy VWAP uses asks from lowest to highest; the sell VWAP uses bids from highest to lowest. Only the required quantity from the final price level is included.

#### Core formula

For quantity Q, use a **VWAP across the order book**, consuming enough depth to fill Q:

$$
VWAP_Q = \frac{\sum_k P_k q_k}{\sum_k q_k}, \qquad \sum_k q_k = Q
$$

Here, P_k is the price at level k, and q_k is the quantity consumed at that level.

The executable trading signal becomes:

**Edge(Q) = Sell VWAP(Q) − Buy VWAP(Q) − Cost(Q)**

Here, cost is expressed per unit of the product so it is comparable with the VWAP prices. If the first venue is the buying venue, the formula is the second venue’s sell VWAP minus the first venue’s buy VWAP, minus costs.

| Strategy rule        | Description                                                                                                                    |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| Compare              | The same trading pair across different venues, using available order-book depth                                                |
| Buy price            | VWAP of the asks needed to buy Q on the buying venue                                                                           |
| Sell price           | VWAP of the bids needed to sell Q on the selling venue                                                                         |
| Trading signal       | Edge(Q): the difference between sell and buy VWAP, after costs                                                                 |
| Entry condition      | Edge(Q) is positive and at least two standard deviations above its historical mean for the same venue direction and quantity Q |
| Historical window    | Initially 50 preceding observations; the time covered depends on the sampling interval                                         |
| Quantity             | Evaluate a proposed Q only when both books contain sufficient depth to fill it                                                 |
| Execution            | Use either buy-then-sell or paired long-and-short execution, with the corresponding exit and risk rules                        |
| Execution constraint | Snapshot-based VWAP is an estimate; prices and available depth may change before orders fill                                   |

The method for choosing Q and the costs included remain to be defined.
