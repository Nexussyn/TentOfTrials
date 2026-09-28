# Crypto Direction Engine v2

A public, no-login crypto direction cockpit designed to produce one actionable decision from live market data.

## What changed

- Direct order plan: action, entry, stop-loss, take-profit and R:R.
- Self-judging sequential evaluation: predictions are stored locally, resolved only after their requested horizon, then scored.
- Online learning: horizon-specific model weights update only after the future outcome is revealed.
- PASR-C4 is a dedicated feature. A second ablation model runs without PASR so its incremental value is measured rather than assumed.
- Data disagreement and feed health control the abstention state.
- The UI separates observed data from model inference and exposes only the information needed for the decision.

## Data

Primary: Hyperliquid public Info API and public WebSocket for mids, L2 book, candles and trades.
Cross-venue: Binance Futures public market data for price, OI and funding context.

## Important validation rule

The browser cannot reconstruct a missed future observation after it has been closed for a long period. Therefore old pending forecasts are discarded rather than scored against a later price. This prevents artificial long-horizon grading.

Brier score, log loss and hit rate are calculated sequentially from resolved real observations stored in browser local storage.

## PASR

The PASR-C4 feature is implemented as a fourth-order cumulant-style statistic over signed micro-flow buckets. Its model weight is learned online. The dashboard also compares the full model against a base ablation without PASR.

This is a falsifiable implementation: PASR can help, do nothing, or hurt. No benefit is asserted before OOS evidence exists.

## Order plan

Entry uses the current executable side of the Hyperliquid book when available. SL distance is derived from the current 1-minute ATR and spread. TP uses a transparent prior R multiple until enough real observations exist to justify a learned target.

NO TRADE is emitted when data quality, conflict, or directional evidence fails the gate.

## Public entry

The static bundle can be served directly by a public static host. No API keys, private keys, exchange account or execution permission are required.

## Browser preview
Development preview: https://raw.githack.com/Nexussyn/TentOfTrials/crypto-direction-engine/crypto-direction-engine-v2/index.html

## Repository archive
https://github.com/Nexussyn/TentOfTrials/archive/refs/heads/crypto-direction-engine.zip
