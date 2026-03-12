package backtest

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"hft/internal/dataframe"
	"hft/internal/indicators"
	"hft/internal/ml_model"
	"hft/internal/storage/sqlite"
	"hft/internal/strategy"
	"hft/pkg/types"

	_df_ "github.com/rocketlaunchr/dataframe-go"
	"github.com/rocketlaunchr/dataframe-go/exports"
)

type Backtest struct {
	DF        *_df_.DataFrame
	Position  *types.Position
	Positions []*types.Position
	Events    chan *types.Event
	TradeDF   *_df_.DataFrame
	LogEvents chan *types.LogEvent
}

var Instance *Backtest

func Reset() {
	Instance = &Backtest{
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
		TradeDF: dataframe.InitTradeDataFrame(),
	}
}

func InitBacktest() {
	Instance = &Backtest{
		DF:        dataframe.InitDataFrame(),
		TradeDF:   dataframe.InitTradeDataFrame(),
		Events:    make(chan *types.Event),
		LogEvents: make(chan *types.LogEvent),
		Positions: make([]*types.Position, 0),
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
	}
}

// BacktestStats holds running statistics for the backtest
type BacktestStats struct {
	TotalTrades     int
	WinningTrades   int
	LosingTrades    int
	BreakevenTrades int
	NetProfit       float64
	GrossProfit     float64
	GrossLoss       float64
	MaxDrawdown     float64
	MaxProfit       float64
	PeakProfit      float64

	// Exit reason breakdown
	ProfitTargetExits int
	StopLossExits     int
	TrailingStopExits int
	SignalExits       int
	ExpectancyRatio   float64
}

// Pending position to match entries with exits
var pendingPosition *types.Position
var stats *BacktestStats

func SubscribeSignals() {
	stats = &BacktestStats{}
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

	// Print final statistics when channel closes
	printBacktestSummary()
}

func printBacktestSummary() {
	if stats == nil || stats.TotalTrades == 0 {
		log.Println("backtest: no trades executed")
		return
	}

	winRate := float64(stats.WinningTrades) / float64(stats.TotalTrades) * 100
	avgProfit := stats.NetProfit / float64(stats.TotalTrades)

	var profitFactor float64
	if stats.GrossLoss != 0 {
		profitFactor = stats.GrossProfit / -stats.GrossLoss
	}

	log.Println("========== BACKTEST SUMMARY ==========")
	log.Printf("Total Trades:      %d", stats.TotalTrades)
	log.Printf("Winning Trades:    %d (%.2f%%)", stats.WinningTrades, winRate)
	log.Printf("Losing Trades:     %d", stats.LosingTrades)
	log.Printf("Breakeven Trades:  %d", stats.BreakevenTrades)
	log.Println("---------------------------------------")
	log.Printf("Net Profit:        %.2f pts", stats.NetProfit)
	log.Printf("Gross Profit:      %.2f pts", stats.GrossProfit)
	log.Printf("Gross Loss:        %.2f pts", stats.GrossLoss)
	log.Printf("Avg Profit/Trade:  %.2f pts", avgProfit)
	log.Printf("Profit Factor:     %.2f", profitFactor)
	log.Printf("Expectancy Ratio:  %.2f pts/trade", stats.ExpectancyRatio)
	log.Printf("Max Drawdown:      %.2f pts", stats.MaxDrawdown)
	log.Printf("Max Single Profit: %.2f pts", stats.MaxProfit)
	log.Println("---------------------------------------")
	log.Printf("Exit Reasons:")
	log.Printf("  Profit Target:   %d", stats.ProfitTargetExits)
	log.Printf("  Stop Loss:       %d", stats.StopLossExits)
	log.Printf("  Trailing Stop:   %d", stats.TrailingStopExits)
	log.Printf("  Signal:          %d", stats.SignalExits)
	log.Println("=======================================")
}

// GetBacktestStats returns the current backtest statistics
func GetBacktestStats() *BacktestStats {
	return stats
}

// Run executes a minimal backtest pass: load all ticks and report count.
func Run() {
	log.Println("backtest: starting")
	startDate := "2022-01-01"
	endDate := "2026-03-12"
	RunWithDatesWarmup(startDate, endDate, "2025-12-01")
}

// RunWithDates executes a backtest pass with custom start and end dates.
// Indicators are computed cold from startDate — use RunWithDatesWarmup to
// avoid NaN warmup rows affecting early predictions.
func RunWithDates(startDate, endDate string) error {
	return RunWithDatesWarmup(startDate, endDate, "")
}

// RunWithDatesWarmup executes a backtest with a warmup period before startDate.
// warmupFromDate should be ~2 months before startDate to fully warm rolling
// indicators (rolling_std_60 + vol_expansion need ~120 bars; seqLen=120 more).
// Warmup rows are loaded and processed but predictions before startDate are
// excluded from the exported CSV and trade signals.
func RunWithDatesWarmup(startDate, endDate, warmupFromDate string) error {
	loadFrom := startDate
	if warmupFromDate != "" {
		loadFrom = warmupFromDate
	}
	log.Printf("backtest: loading %s→%s (warmup from %s, analysis from %s)", loadFrom, endDate, loadFrom, startDate)
	InitBacktest()

	go SubscribeSignals()
	ctx := context.Background()
	df := Instance.DF
	db := sqlite.DefaultDB()
	if db == nil {
		log.Printf("backtest: db not initialized")
		return fmt.Errorf("database not initialized")
	}

	ticks, err := db.Ticks.ListTicksFiltered(ctx, "nifty", "", 0, loadFrom, endDate)
	if err != nil {
		log.Printf("backtest: load ticks: %v", err)
		return fmt.Errorf("failed to load ticks: %v", err)
	}

	dataframe.LoadHistoryBacktest(df, ticks)

	// strategy.RunKalman(df, Instance.LogEvents)
	// strategy.FindKalmanSignal(df, Instance.Position, Instance.Positions, Instance.Events)

	strategy.RunKalmanv2(df, Instance.LogEvents)
	strategy.FindKalmanSignalv2(df, Instance.Position, Instance.Positions, Instance.Events)

	err = ml_model.PredictRegimeFromDF(df)
	if err != nil {
		log.Printf("backtest: predict regime: %v", err)
		return fmt.Errorf("failed to predict regime: %v", err)
	}

	// Close events channel to trigger summary printout
	close(Instance.Events)

	// Give subscriber time to process final events and print summary
	time.Sleep(100 * time.Millisecond)
	return nil
}

func ToJSON() []map[string]interface{} {
	if Instance == nil || Instance.DF == nil {
		return []map[string]interface{}{}
	}
	_data_frame := Instance.DF

	// colVal safely retrieves series value at row i.
	// Returns nil when the column is absent (FindIndexOf returns -1),
	// preventing a "index out of range [-1]" panic for optional columns
	// such as "regime" which is only present when the ML predictor is active.
	colVal := func(name string, i int) interface{} {
		idx := indicators.FindIndexOf(_data_frame, name)
		if idx < 0 {
			return nil
		}
		return _data_frame.Series[idx].Value(i)
	}

	_json := make([]map[string]interface{}, _data_frame.NRows())
	for i := 0; i < _data_frame.NRows(); i++ {
		_json[i] = map[string]interface{}{
			"open":       colVal("open", i),
			"high":       colVal("high", i),
			"low":        colVal("low", i),
			"close":      colVal("close", i),
			"time":       colVal("time", i),
			"timestamp":  colVal("timestamp", i),
			"swap":       colVal("swap", i),
			"swap_base":  colVal("swap_base", i),
			"fast_tempx": colVal("fast_tempx_kalman", i),
			"slow_tempx": colVal("slow_tempx_kalman", i),
			"regime":     colVal("regime", i), // nil when predictor not active
		}
	}
	return _json
}

// TradesToJSON exports the TradeDF as a JSON-compatible slice
func TradesToJSON() []map[string]interface{} {
	if Instance == nil || Instance.TradeDF == nil {
		return []map[string]interface{}{}
	}
	tradeDF := Instance.TradeDF
	nRows := tradeDF.NRows()
	_json := make([]map[string]interface{}, nRows)

	for i := 0; i < nRows; i++ {
		_json[i] = map[string]interface{}{
			"entryPrice": tradeDF.Series[indicators.FindIndexOf(tradeDF, "entryPrice")].Value(i),
			"exitPrice":  tradeDF.Series[indicators.FindIndexOf(tradeDF, "exitPrice")].Value(i),
			"entryTime":  tradeDF.Series[indicators.FindIndexOf(tradeDF, "entryTime")].Value(i),
			"exitTime":   tradeDF.Series[indicators.FindIndexOf(tradeDF, "exitTime")].Value(i),
			"profit":     tradeDF.Series[indicators.FindIndexOf(tradeDF, "profit")].Value(i),
			"profitPct":  tradeDF.Series[indicators.FindIndexOf(tradeDF, "profitPct")].Value(i),
			"type":       tradeDF.Series[indicators.FindIndexOf(tradeDF, "type")].Value(i),
			"peakProfit": tradeDF.Series[indicators.FindIndexOf(tradeDF, "peakProfit")].Value(i),
			"peakLoss":   tradeDF.Series[indicators.FindIndexOf(tradeDF, "peakLoss")].Value(i),
			"reason":     tradeDF.Series[indicators.FindIndexOf(tradeDF, "reason")].Value(i),
		}
	}
	return _json
}

// GetTradeCount returns the number of trades in TradeDF
func GetTradeCount() int {
	if Instance == nil || Instance.TradeDF == nil {
		return 0
	}
	return Instance.TradeDF.NRows()
}

func DownloadData() {
	// Download data frame to csv
	ctx := context.Background()
	file, _ := os.Create("export/predicted.csv")
	exports.ExportToCSV(ctx, file, Instance.DF)

}
