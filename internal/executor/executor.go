package executor

import (
	"time"

	"hft/internal/brokers"
	"hft/internal/dataframe"
	"hft/internal/strategy"
	"hft/pkg/types"

	_df_ "github.com/rocketlaunchr/dataframe-go"
)

// Executor submits and manages orders at the broker.
type Executor struct {
	mode      string
	DF        *_df_.DataFrame
	Position  *types.Position
	Positions []*types.Position
	Events    chan *types.Event
	TradeDF   *_df_.DataFrame
	LogEvents chan *types.LogEvent
}

// CurrentHFT holds the last connected HFT reference (global for quick access).
var CurrentHFT *types.HFT
var Instance *Executor
var stats *ExecutorStats

type ExecutorStats struct {
	TotalTrades       int
	WinningTrades     int
	LosingTrades      int
	BreakevenTrades   int
	NetProfit         float64
	GrossProfit       float64
	GrossLoss         float64
	MaxDrawdown       float64
	MaxProfit         float64
	PeakProfit        float64
	ProfitTargetExits int
	StopLossExits     int
	TrailingStopExits int
	SignalExits       int
	ExpectancyRatio   float64
}

var pendingPosition *types.Position

// NewExecutor constructs an executor configured for the provided mode.
func NewExecutor(mode string) *Executor {
	return &Executor{
		mode:      mode,
		DF:        dataframe.InitDataFrame(),
		Position:  &types.Position{},
		Positions: make([]*types.Position, 0),
		Events:    make(chan *types.Event),
		TradeDF:   dataframe.InitTradeDataFrame(),
		LogEvents: make(chan *types.LogEvent),
	}
}

func InitExecutor() {
	Instance = &Executor{
		DF: dataframe.InitDataFrame(),
		Position: &types.Position{
			ID:         "",
			Kind:       "",
			Type:       "",
			EntryPrice: 0,
			Quantity:   0,
			EntryTime:  time.Now(),
			ExitPrice:  0,
			ExitTime:   time.Now(),
			PeakProfit: 0,
			PeakLoss:   0,
		},
		Positions: make([]*types.Position, 0),
		Events:    make(chan *types.Event),
		TradeDF:   dataframe.InitTradeDataFrame(),
		LogEvents: make(chan *types.LogEvent),
	}
}

func SubscribeSignals() {
	stats = &ExecutorStats{}
	pendingPosition = nil

	for event := range Instance.Events {
		if event.Type == "ENTRY" {
			// Store pending position for matching with exit
			pendingPosition = &types.Position{
				Kind:       event.Kind,
				EntryPrice: event.EntryPrice,
				EntryTime:  event.Timestamp,
			}
		} else if event.Type == "EXIT" && pendingPosition != nil {
			pendingPosition.PeakProfit = event.PeakProfit
			pendingPosition.PeakLoss = event.PeakLoss

			// Calculate profit for the completed trade
			var profit float64
			if pendingPosition.Kind == "BUY" {
				profit = event.EntryPrice - pendingPosition.EntryPrice
			} else { // SELL/SHORT
				profit = pendingPosition.EntryPrice - event.EntryPrice
			}

			// Calculate profit percentage
			var profitPct float64
			if pendingPosition.EntryPrice != 0 {
				profitPct = (profit / pendingPosition.EntryPrice) * 100
			}

			// Append trade to TradeDF
			dataframe.AppendTrade(
				Instance.TradeDF,
				pendingPosition.EntryPrice,
				event.EntryPrice,
				pendingPosition.EntryTime,
				event.Timestamp,
				profit,
				profitPct,
				pendingPosition.Kind,
				event.Reason,
				pendingPosition.PeakProfit,
				pendingPosition.PeakLoss,
			)

			// Update statistics
			stats.TotalTrades++
			stats.NetProfit += profit

			if profit > 0 {
				stats.WinningTrades++
				stats.GrossProfit += profit
			} else if profit < 0 {
				stats.LosingTrades++
				stats.GrossLoss += profit // negative value
			} else {
				stats.BreakevenTrades++
			}

			// Track max profit and drawdown
			if stats.NetProfit > stats.PeakProfit {
				stats.PeakProfit = stats.NetProfit
			}
			drawdown := stats.PeakProfit - stats.NetProfit
			if drawdown > stats.MaxDrawdown {
				stats.MaxDrawdown = drawdown
			}
			if profit > stats.MaxProfit {
				stats.MaxProfit = profit
			}

			// Track exit reasons
			switch event.Reason {
			case "PROFIT_TARGET":
				stats.ProfitTargetExits++
			case "STOP_LOSS":
				stats.StopLossExits++
			case "TRAILING_STOP":
				stats.TrailingStopExits++
			case "SIGNAL":
				stats.SignalExits++
			}

			// Calculate Expectancy Ratio
			// Expectancy = (P_w × A_w) - (P_l × A_l)
			// Where P_l = 1 - P_w
			if stats.TotalTrades > 0 {
				winRate := float64(stats.WinningTrades) / float64(stats.TotalTrades)
				lossRate := 1 - winRate // P_l = 1 - P_w

				var avgWin, avgLoss float64
				if stats.WinningTrades > 0 {
					avgWin = stats.GrossProfit / float64(stats.WinningTrades)
				}
				if stats.LosingTrades > 0 {
					avgLoss = -stats.GrossLoss / float64(stats.LosingTrades) // GrossLoss is negative
				}

				stats.ExpectancyRatio = (winRate * avgWin) - (lossRate * avgLoss)
			}

			// Log trade summary
			// log.Printf("backtest: trade closed | %s | entry: %.2f | exit: %.2f | profit: %.2f pts | reason: %s",
			// 	pendingPosition.Kind, pendingPosition.EntryPrice, event.EntryPrice, profit, event.Reason)

			// Reset pending position
			pendingPosition = nil
		}
	}
}

func (e *Executor) Log(message string) {
	e.LogEvents <- &types.LogEvent{
		Message:   message,
		Timestamp: time.Now(),
	}
}

// Run starts the executor loop. Placeholder for real order logic.
func (e *Executor) Run() {
	go SubscribeSignals()
	Instance = e
	from := time.Now().AddDate(0, 0, -100)
	to := time.Now()
	symbol := "NSE:NIFTY50-INDEX"

	e.Log("executor routine started (mode=" + e.mode + ")")
	// connect with broker
	hftRef := &types.HFT{
		User:   types.User{Name: "Sejal"},
		Broker: brokers.GetBroker(),
		Time:   time.Now(),
	}
	CurrentHFT = hftRef

	e.Log("loading history from " + from.String() + " to " + to.String())
	ticks := brokers.LoadHistory(symbol, 1, from, to)
	dataframe.LoadHistoryLive(e.DF, ticks)
	strategy.RunKalmanv2(e.DF, e.LogEvents)
	strategy.FindKalmanSignalv2(e.DF, Instance.Position, Instance.Positions, Instance.Events)
	// strategy.RunKalmanv2(e.DF, e.LogEvents)
	// strategy.FindKalmanSignalv2(e.DF, Instance.Position, Instance.Positions, Instance.Events)

	close(Instance.Events)

	// Give subscriber time to process final events and print summary
	time.Sleep(100 * time.Millisecond)
	// TODO: implement live/backtest execution behavior.
	select {} // block to simulate a long-lived executor loop
}
