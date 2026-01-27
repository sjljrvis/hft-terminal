hft-algo/
│
├── server/                     # HTTP + WebSocket API (entrypoint)
│   └── main.go
│
├── internal/
│   ├── config/                 # Env + config loaders
│   │   └── config.go
│   │
│   ├── logger/                 # Structured + async logging
│   │   ├── logger.go
│   │   └── zap.go
│   │
│   ├── clock/                  # Exchange / monotonic clock
│   │   └── clock.go
│   │
│   ├── dataframe/              # Lightweight dataframe engine
│   │   ├── dataframe.go
│   │   ├── column.go
│   │   └── ops.go
│   │
│   ├── marketdata/
│   │   ├── feed.go             # Tick / candle feed abstraction
│   │   ├── websocket.go
│   │   └── snapshot.go
│   │
│   ├── indicators/             # Ultra-fast indicators
│   │   ├── ema.go
│   │   ├── rsi.go
│   │   ├── atr.go
│   │   ├── adx.go
│   │   └── registry.go
│   │
│   ├── signals/
│   │   ├── signal.go            # Signal interface
│   │   ├── directional.go
│   │   ├── breakout.go
│   │   └── pyramiding.go
│   │
│   ├── risk/
│   │   ├── manager.go
│   │   ├── sl_tp.go
│   │   └── position_sizing.go
│   │
│   ├── strategy/
│   │   ├── engine.go            # Orchestrates indicators + signals
│   │   ├── context.go
│   │   └── strategy.go
│   │
│   ├── execution/
│   │   ├── executor.go
│   │   ├── order.go
│   │   ├── broker.go            # Broker abstraction
│   │   └── zerodha.go           # Example broker adapter
│   │
│   ├── oms/
│   │   ├── order_manager.go
│   │   ├── state.go
│   │   └── persistence.go
│   │
│   ├── metrics/
│   │   ├── metrics.go
│   │   └── prometheus.go
│   │
│   ├── backtest/
│   │   ├── engine.go
│   │   ├── fill_model.go
│   │   └── pnl.go
│   │
│   └── web/
│       ├── handlers/
│       │   ├── health.go
│       │   ├── signals.go
│       │   └── orders.go
│       │
│       ├── websocket/
│       │   └── stream.go
│       │
│       └── router.go
│
├── pkg/
│   ├── ringbuffer/              # Lock-free queues
│   │   └── ring.go
│   │
│   ├── mathx/                   # Fast math utils
│   │   └── math.go
│   │
│   └── types/                   # Shared structs
│       ├── tick.go
│       ├── candle.go
│       └── signal.go
│
├── web-ui/
│   ├── public/
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   └── api.ts
│   └── vite.config.ts
│
├── scripts/
│   ├── replay_ticks/
│   │   └── main.go
│   └── migrate/
│       └── main.go
│
├── configs/
│   ├── dev.yaml
│   ├── prod.yaml
│   └── backtest.yaml
│
├── go.mod
├── go.sum
└── README.md



