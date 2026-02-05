# Task: Implement Signal-Based Exit Strategy Targeting ~40% MFE Capture

You are implementing exit logic for an algorithmic trading system.
This is a **signal-based strategy** — do NOT use stop-losses, trailing stops, or fixed targets.

## Context
- Entry logic already exists.
- Exit signals already exist (boolean `exit_signal`).
- Trades currently exit immediately on `exit_signal`, which causes premature exits.
- We want to improve exits by capturing approximately **40% of Maximum Favorable Excursion (MFE)**.
- This must remain **path-dependent and signal-driven**, not price-level-based.

## Trade State Variables (Available Per Trade)
- `entry_price: float`
- `side: "BUY" | "SELL"`
- `current_price: float`
- `mfe: float` (max favorable excursion since entry, always positive)
- `bars_in_trade: int`
- `exit_signal: bool` (from existing strategy logic)

## New Parameters (Configurable)
- `activation_mfe_pts: float` (default: 25)
- `mfe_capture_ratio: float` (default: 0.40)
- `signal_confirm_bars: int` (default: 2)

## Requirements

### 1. MFE Update
- Continuously update `mfe`:
  - BUY: `mfe = max(mfe, current_price - entry_price)`
  - SELL: `mfe = max(mfe, entry_price - current_price)`

### 2. Two Exit Regimes

#### Regime A: Pre-Activation (Before MFE Threshold)
- If `mfe < activation_mfe_pts`:
  - Exit immediately when `exit_signal == true`
  - Behavior must match existing system

#### Regime B: Protected Mode (After MFE Threshold)
- Activate protected mode once `mfe >= activation_mfe_pts`
- In protected mode:
  - Do NOT exit on a single exit signal
  - Exit only if ONE of the following conditions is met:

##### Condition 1: Confirmed Exit Signal
- Exit if `exit_signal == true` for `signal_confirm_bars` consecutive bars

##### Condition 2: Profit Degradation (Not a Stop-Loss)
- Compute current profit:
  - BUY: `current_profit = current_price - entry_price`
  - SELL: `current_profit = entry_price - current_price`
- Compute drawdown from MFE:
  - `drawdown = mfe - current_profit`
- Exit only if:
  - `drawdown >= mfe * (1 - mfe_capture_ratio)`
- This guarantees capturing at least `mfe_capture_ratio` of MFE

### 3. Constraints (Strict)
- ❌ No stop-loss
- ❌ No trailing stop
- ❌ No fixed profit targets
- ❌ No new indicators
- ✅ Path-dependent logic only
- ✅ Deterministic and testable

## Deliverables
- Implement a function `should_exit(trade_state) -> bool`
- Code must be clean, readable, and production-ready
- Add inline comments explaining each decision branch
- Do NOT modify entry logic

## Goal
- Preserve fast exits for losers
- Hold strong winners longer
- Achieve ~40% MFE capture on high-quality trades
