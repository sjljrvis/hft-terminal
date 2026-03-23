# hft-algo

Skeleton project structure for a high-frequency trading engine.

Project : HFT algorithmic trade singal identification and order execution












  1. Walk-Forward Validation (biggest gap right now)

  Everything so far is in-sample. The +36k PnL means nothing if it doesn't hold out-of-sample. Split by time:
  - Train: 2015–2022
  - Val: 2023–2024
  - Test: 2025–2026 (never seen during training)

  2. Forward-Looking Labels

  Your current labels use current bar's indicators (slope, kalman at time T). Better: label based on next N bars'
  actual price movement. E.g., if price goes up 0.2% over the next 30 bars → bullish. This gives the model a real
  target to predict rather than a proxy.

  3. Confidence-Gated Trading

  The model outputs probabilities but the backtest ignores them. Only trade when pred_confidence > threshold (e.g.,
  0.65). Skip low-conviction signals — this alone can cut stop-losses in half.

  4. Attention on top of TCN

  Add a lightweight self-attention layer between the TCN and classifier. The TCN treats all 120 timesteps equally after
   pooling — attention lets it focus on the most informative bars (e.g., a sharp reversal 5 bars ago matters more than
  a flat bar 100 bars ago).

  5. Multi-timeframe Features

  Add 5-min and 15-min aggregated features (resampled OHLC, slopes, RSI) alongside the 1-min features. Trends look
  different at different scales — a 1-min bearish dip inside a 15-min bullish trend is a buying opportunity, not a
  regime change.