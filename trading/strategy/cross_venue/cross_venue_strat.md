
# L2 Data

## Cross-Venue Strategy

The strategy compares the same product across multiple venues using top-of-book snapshots.

| Venue      | Product | Bid       | Bid quantity | Ask       | Ask quantity |
| ---------- | ------- | --------- | ------------ | --------- | ------------ |
| Exchange A | BTC-USD | 62,000.00 | 1.25 BTC     | 62,001.00 | 0.80 BTC     |
| Exchange B | BTC-USD | 62,050.00 | 0.60 BTC     | 62,051.00 | 1.10 BTC     |

For each venue, bid is the highest available buy price and ask is the lowest available sell price. The quantities are the available amounts at those prices.

This strategy looks for a temporary price difference between two exchanges where the same crypto trading pair can be bought on one exchange for less than it can be sold for on another, after costs. The entry rule uses the executable net spread plus a required execution buffer. It does not require the spread to be unusually large relative to recent observations.

#### Net spread and entry rule

For a proposed quantity **Q**, let **A** be the buying venue’s ask and **B** the selling venue’s bid. Both prices must use the same quote currency and refer to the same transferable base asset.

$$
\text{NetSpread}(Q) = B - A - \frac{C(Q)}{Q}, \qquad Q > 0
$$

$$
\text{Enter only if } \text{NetSpread}(Q) > b, \qquad b > 0
$$

Here, **C(Q)** is the estimated total cost of both legs in quote currency, and **b** is a configurable minimum net spread in quote currency per unit of the base asset. The buffer is a required margin for execution uncertainty, not an additional fee. Choose its value before the backtest; no numerical default is assumed here.

Costs include both venues’ taker fees, fixed order fees, expected slippage on both legs, and any applicable conversion, borrowing, funding, and inventory-rebalancing costs allocated to the trade. For proportional quote-currency fees, the trading-fee component is `Q * (A * buy_fee_rate + B * sell_fee_rate)`. Convert fees charged in other assets into quote currency and account for any reduction in available base quantity. Do not subtract the bid-ask spread again: using B minus A already accounts for the quoted execution prices. Expected slippage belongs in C(Q); the buffer covers uncertainty beyond that estimate.

| Strategy rule       | Description                                                                                                                                                                                                                    |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Compare             | Evaluate each buy/sell venue direction for the same trading pair using fresh, sufficiently aligned snapshots                                                                                                                   |
| Gross spread        | Selling venue’s bid B minus buying venue’s ask A                                                                                                                                                                             |
| Net spread          | Gross spread minus estimated total costs C(Q) divided by Q                                                                                                                                                                     |
| Entry condition     | NetSpread(Q) > b, with a configured positive buffer b; no rolling mean or standard-deviation threshold                                                                                                                         |
| Execution           | Submit a market buy and an equal-quantity market sell as close together as possible, using prefunded spot balances                                                                                                             |
| Quantity            | Cap Q by both top-of-book quantities, available quote balance including buy costs, available sell inventory, and position limits; round down to a quantity valid on both venues and recheck net spread and minimum order sizes |
| Costs and execution | Recalculate costs for the final Q; skip stale quotes, insufficient balances, or invalid orders; track actual fills and any unmatched exposure                                                                                  |

#### Expected and realized profit

If both spot legs fill for Q at the quoted prices and costs equal C(Q), expected net profit is:

$$
\Pi_{\text{expected}} = Q(B-A) - C(Q) = Q\,\text{NetSpread}(Q) > Qb
$$

Using the snapshot above, the gross spread is USD 49 per BTC and the displayed quantity cap is 0.60 BTC, before balance and order constraints. With illustrative taker fees of 0.02% on each leg and no other costs, trading fees are about USD 24.81 per BTC, leaving a net spread of USD 24.19 per BTC. A buffer of USD 10 per BTC would allow entry, with expected net profit of about USD 14.51 for 0.60 BTC. These are hypothetical costs and buffer values, not venue fee quotes or selected backtest settings.

The buffer establishes a minimum expected margin, not a fixed payout on every trade. Profit varies with spread, quantity, and actual costs. For equal filled quantities, realized profit is `Q * (actual sell VWAP - actual buy VWAP) - actual total costs`; the buffer is not subtracted from realized profit. Adverse price moves, stale data, rejected orders, partial fills, or a failed leg can reduce profit or create a loss. Cross-venue orders are not atomic; record and resolve unmatched exposure under a predefined hedge/unwind rule before enabling execution.

Prefunding avoids waiting for transfers during execution, but repeated trades consume quote balance on the buying venue and base inventory on the selling venue. Rebalancing costs and capital availability therefore matter even when each matched pair fills as expected.

#### Execution Type 1: Buy, Then Sell

Buy the product on the venue with the lower ask. Once the purchase fills, sell the **actual filled quantity** on the venue with the higher bid as soon as possible.

Apply the net-spread rule and quantity constraints above before buying. Recheck the sell quote after the buy fills and handle any residual exposure if the sell cannot fill. Profit depends on the selling price remaining favourable during the delay between executions; the initial buffer does not lock in that price.

Selling immediately requires existing inventory on the second venue. Otherwise, the purchased asset must first be transferred, adding transfer time and costs.

#### Execution Type 2: Long and Short

Open a long position on the lower-priced venue and a short position of equal quantity on the higher-priced venue, as close together as possible. Hold both positions while waiting for the price difference to narrow.

This is a convergence variant with a later exit, rather than immediate capture through two prefunded spot trades. The opening net-spread check alone does not establish its profitability; estimate closing costs and funding or borrowing over the holding period separately.

| Risk or exit rule  | Description                                                                                                                              |
| ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| Profit target      | Close both positions when their combined net profit reaches the initial cross-venue price difference multiplied by the executed quantity |
| Loss limit         | Close both positions when their combined loss reaches a predefined threshold                                                             |
| Profit calculation | Include both positions’ gains and losses, trading fees, and applicable funding or borrowing costs                                       |

The initial price difference is a **profit target, not profit already earned**. Actual profit depends on the prices at which both positions open and close. The loss threshold and mismatch-handling rule remain to be defined.
