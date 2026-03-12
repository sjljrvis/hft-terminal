# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Backend (Go)

```bash
# Run server (dev mode with live reload via fresh)
fresh

# Run server directly
go run . server -config configs/dev.yaml

# Run server in specific mode
go run . server -config configs/dev.yaml -mode live
go run . server -config configs/dev.yaml -mode dryrun

# Run scripts
go run . migrate
go run . replay

# Build
go build ./...

# Test
go test ./...

# Test a single package
go test ./internal/indicators/...
```

### Frontend (React/Vite)

```bash
cd web-ui
npm install
npm run dev      # dev server (Vite)
npm run build    # build to web-ui/dist/
npm run preview
```

In dev mode, Vite handles the frontend separately. The Go server skips launching its web server if `web-ui/dist/` doesn't exist.

## Architecture

This is a Go-based HFT (High-Frequency Trading) engine targeting Indian markets (NSE/Fyers broker), with a React web UI.

### Entry Points

- `main.go` — root dispatcher; delegates to `go run ./server` or scripts
- `server/main.go` — starts all services: API server, web server, executor, backtest, market clock, WebSocket hub
- `scripts/` — standalone utilities: `data_scrape`, `migrate`, `replay_ticks`

### Core Subsystems

**Executor** (`internal/executor/`)
The live trading loop. `Executor.Run()` loads historical ticks from Fyers, runs indicator pipelines on a DataFrame, then calls `strategy.FindKalmanSignalv2()` which emits `types.Event` (ENTRY/EXIT) on a channel. `SubscribeSignals()` consumes events, tracks stats, and appends trades to `TradeDF`.

**Backtest** (`internal/backtest/engine.go`)
Same pipeline as executor but reads ticks from SQLite instead of Fyers. Runs on server startup. Results exported to `export/example.csv`.

**Strategy** (`internal/strategy/`)
- `kalmanV2.go` — active strategy: Kalman-filtered price series → signal generation with MFE-based exits (`KalmanExitConfigv2`)
- `kalman.go` — older version (kept for reference)
- `strategy.go` — original CCI/WMA-based strategy with `TrailingStopConfig`; still usable

**Indicators** (`internal/indicators/`)
All indicators operate on `*dataframe.DataFrame` from `rocketlaunchr/dataframe-go`. Each indicator appends a named `SeriesFloat64` to the DataFrame. Use `indicators.FindIndexOf(df, "series_name")` to look up series by name.

Key indicators: EMA, SMA, WMA, ATR, CCI, ADX, RSI, ROC, Kalman filter, CalcX (custom signal transform), CalcSwap (crossover signal), microstructure features.

**DataFrame** (`internal/dataframe/`)
Initializes the OHLCV DataFrame schema and provides `LoadHistoryLive` / `LoadHistoryBacktest` loaders. Series names: `open`, `high`, `low`, `close`, `volume`, `timestamp`, `time`.

**Broker** (`internal/brokers/fyers.go`)
Fyers API v3 integration. Handles OAuth2 auth-code flow, JWT token storage/refresh, margin fetching, and OHLC history download. Tokens are persisted in SQLite (`tokens` table). Login URL: `/broker/fyers/callback`.

**Storage** (`internal/storage/sqlite/`)
SQLite facade with three stores: `TickStore`, `OrderStore`, `TokenStore`. Single shared `*DB` instance initialized via `sqlite.MustInitDefault(dbPath)`.

**Server Routes** (`server/routes/`)
- API server (port 5000 default): `/hft/status`, `/backtest/*`, `/live/*`, `/broker/fyers/*`, `/ws/events`
- Web server (port 5001 default): serves `web-ui/dist/` static files
- WebSocket hub broadcasts executor `LogEvent`s to connected clients

**Market Clock** (`internal/clock/`)
Configurable market session timer (default IST 09:15–15:30). Emits `openC` and `minuteC` channels.

**ML Model** (`internal/ml_model/`)
ONNX runtime inference via `yalue/onnxruntime_go`. Currently separate from the main signal pipeline.

**Risk** (`internal/risk/`)
Position sizing and stop-loss/take-profit utilities (not yet wired into the live executor loop).

### Configuration

YAML config loaded from `configs/dev.yaml` (not committed; copy from `configs/prod.yaml` as template). Key fields: `mode` (live|dryrun), `api_port`, `web_port`, `db_path`, `broker[].fyers.*`, `clock.*`.

### Data Flow

```
Fyers API / SQLite ticks
        |
   dataframe.LoadHistory*()
        |
   strategy.RunKalmanv2()     ← appends indicator series to DataFrame
        |
   strategy.FindKalmanSignalv2()  ← iterates rows, emits Events
        |
   executor.SubscribeSignals()    ← accumulates trade stats
        |
   WebSocket hub → web UI
```

### Module Name

Go module is `hft` (see `go.mod`). All internal imports use `hft/internal/...` and `hft/pkg/...`.
