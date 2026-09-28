# Crypto Direction Engine

Public no-login crypto market direction cockpit.

## Live public build

- Standalone browser build: https://rawcdn.githack.com/Nexussyn/TentOfTrials/592a4d0e9daedbab49fd91d8d28a91040b573158/crypto-direction-engine/public/live.html
- Source tree: https://github.com/Nexussyn/TentOfTrials/tree/crypto-direction-engine/crypto-direction-engine

The standalone page is static and calls the public Hyperliquid and Binance market-data endpoints directly from the browser. No exchange key or account is used.

## Decision model

The engine converges real observations from:
- Hyperliquid mids, perpetual contexts, L2 book and 1m candles.
- Binance Futures ticker, open interest, funding/premium and 1m klines.

It returns LONG / SHORT / NO TRADE for 5m, 15m, 1h and 4h horizons. It also exposes evidence blocks, disagreement, data-quality and explicit “what would change my mind” conditions.

## Validation

The probability readout is intentionally not marketed as a guaranteed predictor. Calibration begins in COLD START. Brier score and log-loss remain unreported until enough real walk-forward outcomes exist.

## Server build

The Node build in this folder uses npm start and provides a server-side API proxy for environments where direct browser API access is undesirable.