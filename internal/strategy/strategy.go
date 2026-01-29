package strategy

import (
	"hft/internal/indicators"
	"hft/pkg/types"
	"log"
	"math"
	"time"

	"github.com/rocketlaunchr/dataframe-go"
)

// TrailingStopConfig holds configuration for trailing stop-loss
type TrailingStopConfig struct {
	UseTrailingStop           bool
	TrailActivationPoints     float64 // Start trailing after this profit (in points)
	TrailDistancePoints       float64 // Distance to maintain from highest profit
	UseATRTrailing            bool
	ATRTrailingMultiplier     float64
	UseBreakeven              bool
	BreakevenActivationPoints float64 // Move stop to breakeven after this profit
	CapturePoints             float64 // Target profit points
	StopLossPoints            float64 // Fixed stop loss points
}

// DefaultTrailingStopConfig returns default trailing stop configuration matching Pine Script
func DefaultTrailingStopConfig() *TrailingStopConfig {
	return &TrailingStopConfig{
		UseTrailingStop:           true,
		TrailActivationPoints:     10.0,
		TrailDistancePoints:       10.0,
		UseATRTrailing:            true,
		ATRTrailingMultiplier:     0.55,
		UseBreakeven:              true,
		BreakevenActivationPoints: 10.0,
		CapturePoints:             100.0,
		StopLossPoints:            100.0,
	}
}

// TrailingStopState holds state for trailing stop calculation
type TrailingStopState struct {
	HighestProfitLong       float64
	HighestProfitShort      float64
	TrailingStopLong        float64
	TrailingStopShort       float64
	BreakevenActivatedLong  bool
	BreakevenActivatedShort bool
}

// NewTrailingStopState creates a new trailing stop state
func NewTrailingStopState() *TrailingStopState {
	return &TrailingStopState{}
}

// ResetLong resets trailing stop state for long position
func (ts *TrailingStopState) ResetLong() {
	ts.HighestProfitLong = 0.0
	ts.TrailingStopLong = 0.0
	ts.BreakevenActivatedLong = false
}

// ResetShort resets trailing stop state for short position
func (ts *TrailingStopState) ResetShort() {
	ts.HighestProfitShort = 0.0
	ts.TrailingStopShort = 0.0
	ts.BreakevenActivatedShort = false
}

func Run(df *dataframe.DataFrame) {
	log.Println("running strategy")
	start := time.Now()
	indicators.OHLC4(df, "ohlc4")
	indicators.EMA(df, "ema_ohlc4", "ohlc4", 2)
	indicators.CCI(df, "fast_cci", "ema_ohlc4", 2)
	indicators.CCI(df, "slow_cci", "ema_ohlc4", 14)
	indicators.ATR(df, "tr", "close", 14)
	indicators.ATR(df, "general_atr", "close", 3)
	indicators.WMA(df, "wma_tr_2", "tr", 900)
	indicators.WMA(df, "wma_tr_400", "tr", 1200)

	indicators.CalcX(df, "tempx", "close", 1.95, 2, "fast_cci", "wma_tr_2")
	indicators.CalcX(df, "tempx_base", "close", 0.25, 2, "slow_cci", "wma_tr_400")

	indicators.EMA(df, "ema_tempx", "tempx", 6)
	indicators.EMA(df, "ema_tempx_base", "tempx_base", 6)

	indicators.SMA(df, "sma_tempx", "tempx", 10)
	indicators.SMA(df, "sma_tempx_base", "tempx_base", 10)

	indicators.ATR(df, "atr_ema_tempx", "ema_tempx", 5)
	indicators.SMA(df, "sma_atr_ema_tempx", "atr_ema_tempx", 10)
	indicators.CalcSwap(df, "swap", "ema_tempx", "sma_tempx")
	indicators.CalcSwap(df, "swap_base", "ema_tempx_base", "sma_tempx_base")
	log.Println("time taken to calculate indicators", time.Since(start))
	// log.Println(df.Table())
}

func FindSignals(df *dataframe.DataFrame, current_position *types.Position, positions []*types.Position, events chan *types.Event) {
	FindSignalsWithTrailingStop(df, current_position, positions, events, DefaultTrailingStopConfig(), NewTrailingStopState())
}

// FindSignalsWithTrailingStop implements signal finding with configurable trailing stop-loss
func FindSignalsWithTrailingStop(df *dataframe.DataFrame, current_position *types.Position, positions []*types.Position, events chan *types.Event, config *TrailingStopConfig, tsState *TrailingStopState) {
	_dataframe_length := df.NRows()
	_close := df.Series[indicators.FindIndexOf(df, "close")].(*dataframe.SeriesFloat64).Values
	_timestamp := df.Series[indicators.FindIndexOf(df, "timestamp")].(*dataframe.SeriesTime).Values
	_swap := df.Series[indicators.FindIndexOf(df, "swap")].(*dataframe.SeriesFloat64).Values
	_swap_base := df.Series[indicators.FindIndexOf(df, "swap_base")].(*dataframe.SeriesFloat64).Values
	_general_atr := df.Series[indicators.FindIndexOf(df, "general_atr")].(*dataframe.SeriesFloat64).Values

	// Get ATR values for ATR-based trailing (uses existing tr/atr series)
	var _atr []float64
	atrIdx := indicators.FindIndexOf(df, "tr")
	if atrIdx >= 0 {
		_atr = df.Series[atrIdx].(*dataframe.SeriesFloat64).Values
	}

	for i := 0; i < _dataframe_length; i++ {
		stop_loss_points := math.Min(config.StopLossPoints, _general_atr[i]*4.5)
		_buy_condition := _swap[i] == 1 && _swap_base[i] == 1
		_sell_condition := _swap[i] == -1 && _swap_base[i] == -1

		// ========== TRAILING STOP-LOSS LOGIC ==========
		var trailingStopHitLong, trailingStopHitShort bool
		var profitHit, stopHit bool

		// Calculate current profit for LONG position
		if current_position.Kind == "BUY" {
			currentProfitLong := _close[i] - current_position.EntryPrice

			// Update highest profit
			if currentProfitLong > tsState.HighestProfitLong {
				tsState.HighestProfitLong = currentProfitLong
			}

			// Calculate trailing stop distance
			trailDist := config.TrailDistancePoints
			if config.UseATRTrailing && _atr != nil && i < len(_atr) {
				trailDist = _atr[i] * config.ATRTrailingMultiplier
			}

			// Activate trailing stop after reaching activation threshold
			if config.UseTrailingStop && tsState.HighestProfitLong >= config.TrailActivationPoints {
				newTrailingStop := _close[i] - trailDist
				if newTrailingStop > tsState.TrailingStopLong || tsState.TrailingStopLong == 0 {
					tsState.TrailingStopLong = newTrailingStop
				}
			}

			// Move to breakeven
			if config.UseBreakeven && !tsState.BreakevenActivatedLong && tsState.HighestProfitLong >= config.BreakevenActivationPoints {
				tsState.TrailingStopLong = current_position.EntryPrice
				tsState.BreakevenActivatedLong = true
			}

			// Check trailing stop hit
			trailingStopHitLong = config.UseTrailingStop && tsState.TrailingStopLong > 0 && _close[i] <= tsState.TrailingStopLong

			// Check profit target and stop loss
			profitHit = currentProfitLong >= config.CapturePoints
			stopHit = currentProfitLong <= -stop_loss_points
		}

		// Calculate current profit for SHORT position
		if current_position.Kind == "SELL" {
			currentProfitShort := current_position.EntryPrice - _close[i]

			// Update highest profit
			if currentProfitShort > tsState.HighestProfitShort {
				tsState.HighestProfitShort = currentProfitShort
			}

			// Calculate trailing stop distance
			trailDist := config.TrailDistancePoints
			if config.UseATRTrailing && _atr != nil && i < len(_atr) {
				trailDist = _atr[i] * config.ATRTrailingMultiplier
			}

			// Activate trailing stop after reaching activation threshold
			if config.UseTrailingStop && tsState.HighestProfitShort >= config.TrailActivationPoints {
				newTrailingStop := _close[i] + trailDist
				if newTrailingStop < tsState.TrailingStopShort || tsState.TrailingStopShort == 0 {
					tsState.TrailingStopShort = newTrailingStop
				}
			}

			// Move to breakeven
			if config.UseBreakeven && !tsState.BreakevenActivatedShort && tsState.HighestProfitShort >= config.BreakevenActivationPoints {
				tsState.TrailingStopShort = current_position.EntryPrice
				tsState.BreakevenActivatedShort = true
			}

			// Check trailing stop hit
			trailingStopHitShort = config.UseTrailingStop && tsState.TrailingStopShort > 0 && _close[i] >= tsState.TrailingStopShort

			// Check profit target and stop loss
			profitHit = currentProfitShort >= config.CapturePoints
			stopHit = currentProfitShort <= -stop_loss_points
		}

		// ========== EXIT CONDITIONS ==========
		shouldCloseBuy := current_position.Kind == "BUY" && (profitHit || stopHit || trailingStopHitLong || _sell_condition)
		shouldCloseSell := current_position.Kind == "SELL" && (profitHit || stopHit || trailingStopHitShort || _buy_condition)

		// Determine exit reason for event
		getExitReason := func(profitHit, stopHit, trailingHit, signalExit bool) string {
			if profitHit {
				return "PROFIT_TARGET"
			}
			if stopHit {
				return "STOP_LOSS"
			}
			if trailingHit {
				return "TRAILING_STOP"
			}
			return "SIGNAL"
		}

		if shouldCloseSell || !indicators.IsActiveSession(_timestamp[i]) {
			exitReason := getExitReason(profitHit, stopHit, trailingStopHitShort, _buy_condition)
			current_position.Exit(_close[i], *_timestamp[i])
			current_position.Reset()
			tsState.ResetShort()
			events <- &types.Event{
				Kind:       "SELL",
				Type:       "EXIT",
				EntryPrice: _close[i],
				Timestamp:  *_timestamp[i],
				Reason:     exitReason,
			}
		}

		if shouldCloseBuy || !indicators.IsActiveSession(_timestamp[i]) {
			exitReason := getExitReason(profitHit, stopHit, trailingStopHitLong, _sell_condition)
			current_position.Exit(_close[i], *_timestamp[i])
			current_position.Reset()
			tsState.ResetLong()
			events <- &types.Event{
				Kind:       "BUY",
				Type:       "EXIT",
				EntryPrice: _close[i],
				Timestamp:  *_timestamp[i],
				Reason:     exitReason,
			}
		}

		// ========== ENTRY CONDITIONS ==========
		if current_position.Kind == "" && _buy_condition {
			current_position.Buy(_close[i], *_timestamp[i])
			tsState.ResetLong() // Reset trailing stop state for new position
			events <- &types.Event{
				Kind:       "BUY",
				Type:       "ENTRY",
				EntryPrice: _close[i],
				Timestamp:  *_timestamp[i],
			}
		}

		if current_position.Kind == "" && _sell_condition {
			current_position.Sell(_close[i], *_timestamp[i])
			tsState.ResetShort() // Reset trailing stop state for new position
			events <- &types.Event{
				Kind:       "SELL",
				Type:       "ENTRY",
				EntryPrice: _close[i],
				Timestamp:  *_timestamp[i],
			}
		}
	}
}

// CalculateTrailingStop is a utility function to calculate trailing stop for a given position
func CalculateTrailingStop(positionKind string, entryPrice, currentPrice, highestProfit float64, config *TrailingStopConfig, atr float64) (trailingStop float64, shouldExit bool) {
	trailDist := config.TrailDistancePoints
	if config.UseATRTrailing && atr > 0 {
		trailDist = atr * config.ATRTrailingMultiplier
	}

	if positionKind == "BUY" {
		currentProfit := currentPrice - entryPrice
		if currentProfit > highestProfit {
			highestProfit = currentProfit
		}

		if config.UseTrailingStop && highestProfit >= config.TrailActivationPoints {
			trailingStop = currentPrice - trailDist
		}

		// Breakeven logic
		if config.UseBreakeven && highestProfit >= config.BreakevenActivationPoints {
			trailingStop = math.Max(trailingStop, entryPrice)
		}

		shouldExit = trailingStop > 0 && currentPrice <= trailingStop
	} else if positionKind == "SELL" {
		currentProfit := entryPrice - currentPrice
		if currentProfit > highestProfit {
			highestProfit = currentProfit
		}

		if config.UseTrailingStop && highestProfit >= config.TrailActivationPoints {
			trailingStop = currentPrice + trailDist
		}

		// Breakeven logic
		if config.UseBreakeven && highestProfit >= config.BreakevenActivationPoints {
			if trailingStop == 0 {
				trailingStop = entryPrice
			} else {
				trailingStop = math.Min(trailingStop, entryPrice)
			}
		}

		shouldExit = trailingStop > 0 && currentPrice >= trailingStop
	}

	return trailingStop, shouldExit
}
