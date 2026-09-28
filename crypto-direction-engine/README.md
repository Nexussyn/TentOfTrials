# Crypto Direction Engine

Public no-login trading intelligence cockpit. Hyperliquid is the primary venue and Binance Futures is the independent cross-venue reference.

## Status

This release is a transparent evidence ensemble. It does not claim that the probabilities are calibrated or that the dashboard is profitable. It starts in COLD START and intentionally does not fabricate Brier/log-loss values.

## Live data

Hyperliquid public Info API: metaAndAssetCtxs, allMids, l2Book, candleSnapshot.
Binance Futures public market data: 24h ticker, open interest, premium/funding, 1-minute klines.

## Design

The output combines structure, order-flow proxy, L2 depth imbalance, derivatives and cross-venue divergence. Each horizon (5m, 15m, 1h, 4h) has a separate scaling factor and an abstention rule that can return NO TRADE.

## Run

npm start
