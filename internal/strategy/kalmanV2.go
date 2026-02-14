package strategy

import (
	"hft/internal/indicators"
	"hft/pkg/types"
	"math"

	"github.com/rocketlaunchr/dataframe-go"
)

// KalmanExitConfig holds parameters for MFE-based exits.
type KalmanExitConfigv2 struct {
	ActivationMFEPts  float64
	MFECaptureRatio   float64
	SignalConfirmBars int
	EnableFixedSL     bool
	FixedSL           float64
}

// DefaultKalmanExitConfig returns the default exit parameters.
func DefaultKalmanExitConfigv2() *KalmanExitConfigv2 {
	return &KalmanExitConfigv2{
		ActivationMFEPts:  12,
		MFECaptureRatio:   0.4, // 0.4 stable // .35 most proft d
		SignalConfirmBars: 0,
		EnableFixedSL:     false,
		FixedSL:           -50,
	}
}

// KalmanTradeState captures per-trade state for exit logic.
type KalmanTradeStatev2 struct {
	EntryPrice       float64
	Side             string
	CurrentPrice     float64
	MFE              float64
	BarsInTrade      int
	ExitSignal       bool
	ExitSignalStreak int
}

func (ts *KalmanTradeStatev2) Reset() {
	ts.EntryPrice = 0
	ts.Side = ""
	ts.CurrentPrice = 0
	ts.MFE = 0
	ts.BarsInTrade = 0
	ts.ExitSignal = false
	ts.ExitSignalStreak = 0
}

// shouldExit applies MFE-based, signal-confirmed exit rules.
func shouldExitv2(state *KalmanTradeStatev2, cfg *KalmanExitConfigv2) bool {
	if state == nil || cfg == nil {
		return false
	}

	confirmBars := cfg.SignalConfirmBars
	if confirmBars < 1 {
		confirmBars = 1
	}

	// Compute current profit based on side.
	var currentProfit float64
	if state.Side == "BUY" {
		currentProfit = state.CurrentPrice - state.EntryPrice
	} else if state.Side == "SELL" {
		currentProfit = state.EntryPrice - state.CurrentPrice
	} else {
		return false
	}

	// Update MFE (max favorable excursion).
	if currentProfit > state.MFE {
		state.MFE = currentProfit
	}

	// Track consecutive exit signals for confirmation.
	if state.ExitSignal {
		state.ExitSignalStreak++
	} else {
		state.ExitSignalStreak = 0
	}

	// Regime A: pre-activation, exit immediately on signal.
	if state.MFE < cfg.ActivationMFEPts {
		return state.ExitSignal
	}

	// Regime B: protected mode requires confirmation or MFE degradation.
	confirmedSignal := state.ExitSignal && state.ExitSignalStreak >= confirmBars
	drawdown := state.MFE - currentProfit
	captureThreshold := state.MFE * (1 - cfg.MFECaptureRatio)
	degradationExit := drawdown >= captureThreshold

	return confirmedSignal || degradationExit
}

/*

RULES:
- _slow_tempx_kalman is base trend
- _fast_tempx_kalman is micro-trend
-  buy only when _slow_swap flips from (0 to 1 or -1 to 1) and deviation (slow_tempx_kalman - fast_tempx_kalman) is or or between (5,15)
-  sell only when _slow_swap flips from (0 to -1 or 1 to -1) and deviation (slow_tempx_kalman - fast_tempx_kalman) is or or between (5,15)

- check late entry if _fast_swap flips (-1 to 1) and _slow_swap = 1 with deviation (slow_tempx_kalman - fast_tempx_kalman) is or or between (5,15)
- check late entry if _fast_swap flips (1 to -1) and _slow_swap = -1 with deviation (slow_tempx_kalman - fast_tempx_kalman) is or or between (5,15)

- exit condtions
  if _slow_swap flips (1 to -1 or -1 to 1)

*/

func FindKalmanSignalv2(df *dataframe.DataFrame, current_position *types.Position, positions []*types.Position, events chan *types.Event) {
	FindKalmanSignalWithExitConfigv2(df, current_position, positions, events, DefaultKalmanExitConfigv2())
}

func FindKalmanSignalWithExitConfigv2(df *dataframe.DataFrame, current_position *types.Position, positions []*types.Position, events chan *types.Event, exitConfig *KalmanExitConfigv2) {
	if exitConfig == nil {
		exitConfig = DefaultKalmanExitConfigv2()
	}
	fixedSLHit := false

	_dataframe_length := df.NRows()
	_close := df.Series[indicators.FindIndexOf(df, "close")].(*dataframe.SeriesFloat64).Values
	_timestamp := df.Series[indicators.FindIndexOf(df, "timestamp")].(*dataframe.SeriesTime).Values
	_fast_tempx_kalman := df.Series[indicators.FindIndexOf(df, "fast_tempx_kalman")].(*dataframe.SeriesFloat64).Values
	_slow_tempx_kalman := df.Series[indicators.FindIndexOf(df, "slow_tempx_kalman")].(*dataframe.SeriesFloat64).Values
	_slow_swap := df.Series[indicators.FindIndexOf(df, "swap_base")].(*dataframe.SeriesFloat64).Values
	_fast_swap := df.Series[indicators.FindIndexOf(df, "swap")].(*dataframe.SeriesFloat64).Values
	// _tr := df.Series[indicators.FindIndexOf(df, "tr")].(*dataframe.SeriesFloat64).Values

	tradeState := &KalmanTradeStatev2{}

	for i := 0; i < _dataframe_length; i++ {
		deviation := math.Abs(_slow_tempx_kalman[i] - _fast_tempx_kalman[i])
		// normalizedDeviation := deviation / _tr[i]
		// print only for 23 december 2025

		deviationOK := deviation >= 7 && deviation <= 15
		// deviationOK := normalizedDeviation >= 0.5 && normalizedDeviation <= 2

		slowSwapFlipUp := i > 0 && _slow_swap[i] == 1 && (_slow_swap[i-1] == 0 || _slow_swap[i-1] == -1)
		slowSwapFlipDown := i > 0 && _slow_swap[i] == -1 && (_slow_swap[i-1] == 0 || _slow_swap[i-1] == 1)

		fastSwapFlipUp := i > 0 && _fast_swap[i] == 1 && _fast_swap[i-1] == -1
		fastSwapFlipDown := i > 0 && _fast_swap[i] == -1 && _fast_swap[i-1] == 1

		slowSwapFlipSignal := i > 0 && ((_slow_swap[i-1] == 1 && _slow_swap[i] == -1) || (_slow_swap[i-1] == -1 && _slow_swap[i] == 1))
		fastSwapFlipSignal := i > 0 && ((_fast_swap[i-1] == 1 && _fast_swap[i] == -1) || (_fast_swap[i-1] == -1 && _fast_swap[i] == 1))

		lateBuy := fastSwapFlipUp && _slow_swap[i] == 1
		lateSell := fastSwapFlipDown && _slow_swap[i] == -1
		// earlyBuy := i > 0 && _fast_swap[i] == 1 && _fast_swap[i-1] == 0
		// earlySell := i > 0 && _fast_swap[i] == -1 && _fast_swap[i-1] == 0

		_buy_condition := deviationOK && (slowSwapFlipUp || lateBuy)
		_sell_condition := deviationOK && (slowSwapFlipDown || lateSell)

		sessionClosed := !indicators.IsActiveSession(_timestamp[i])
		exitSignal := false

		// update peak profit and peak loss if trade is profitable or in loss (for both buy and sell)
		// peakprofit = max(peakprofit, current_position.Profit)
		// peakloss = min(peakloss, current_position.Profit)
		if current_position.Kind != "" {
			if current_position.Kind == "BUY" {
				current_position.Profit = _close[i] - current_position.EntryPrice
			} else if current_position.Kind == "SELL" {
				current_position.Profit = current_position.EntryPrice - _close[i]
			}

			current_position.PeakProfit = math.Max(current_position.PeakProfit, current_position.Profit)
			current_position.PeakLoss = math.Min(current_position.PeakLoss, current_position.Profit)
		}

		fixedSLHit = current_position.Profit <= exitConfig.FixedSL && exitConfig.EnableFixedSL

		if current_position.Kind == "BUY" {
			exitSignal = _sell_condition || slowSwapFlipSignal || fastSwapFlipSignal || fixedSLHit
		} else if current_position.Kind == "SELL" {
			exitSignal = _buy_condition || slowSwapFlipSignal || fastSwapFlipSignal || fixedSLHit
		}

		shouldExitTrade := false
		if current_position.Kind != "" {
			tradeState.EntryPrice = current_position.EntryPrice
			tradeState.Side = current_position.Kind
			tradeState.CurrentPrice = _close[i]
			tradeState.ExitSignal = exitSignal
			tradeState.BarsInTrade++
			shouldExitTrade = shouldExitv2(tradeState, exitConfig)
		}

		shouldCloseBuy := current_position.Kind == "BUY" && (shouldExitTrade || sessionClosed)
		shouldCloseSell := current_position.Kind == "SELL" && (shouldExitTrade || sessionClosed)

		if shouldCloseSell {
			current_position.Exit(_close[i], *_timestamp[i])
			exitEvent := &types.Event{
				Kind:       "SELL",
				Type:       "EXIT",
				EntryPrice: _close[i],
				Timestamp:  *_timestamp[i],
				Reason:     "SIGNAL",
				PeakProfit: current_position.PeakProfit,
				PeakLoss:   current_position.PeakLoss,
			}
			current_position.Reset()
			tradeState.Reset()
			events <- exitEvent
		}

		if shouldCloseBuy {
			current_position.Exit(_close[i], *_timestamp[i])
			exitEvent := &types.Event{
				Kind:       "BUY",
				Type:       "EXIT",
				EntryPrice: _close[i],
				Timestamp:  *_timestamp[i],
				Reason:     "SIGNAL",
				PeakProfit: current_position.PeakProfit,
				PeakLoss:   current_position.PeakLoss,
			}
			current_position.Reset()
			tradeState.Reset()
			events <- exitEvent
		}

		if current_position.Kind == "" && _buy_condition {
			current_position.Buy(_close[i], *_timestamp[i])
			current_position.PeakProfit = 0
			current_position.PeakLoss = 0
			tradeState.Reset()
			tradeState.EntryPrice = _close[i]
			tradeState.Side = "BUY"
			tradeState.CurrentPrice = _close[i]
			events <- &types.Event{
				Kind:       "BUY",
				Type:       "ENTRY",
				EntryPrice: _close[i],
				Timestamp:  *_timestamp[i],
			}
		}

		if current_position.Kind == "" && _sell_condition {
			current_position.Sell(_close[i], *_timestamp[i])
			current_position.PeakProfit = 0
			current_position.PeakLoss = 0
			tradeState.Reset()
			tradeState.EntryPrice = _close[i]
			tradeState.Side = "SELL"
			tradeState.CurrentPrice = _close[i]
			events <- &types.Event{
				Kind:       "SELL",
				Type:       "ENTRY",
				EntryPrice: _close[i],
				Timestamp:  *_timestamp[i],
			}
		}
	}
}

func RunKalmanv2(df *dataframe.DataFrame, logEvents chan *types.LogEvent) {
	indicators.CCI(df, "fast_cci", "close", 2)
	indicators.ATR(df, "tr", "close", 5)
	indicators.WMA(df, "wma_tr_2", "tr", 30)

	indicators.CalcX(df, "fast_tempx", "close", 0.1, 2, "fast_cci", "wma_tr_2")
	indicators.CalcX(df, "slow_tempx", "close", 0.1, 2, "fast_cci", "wma_tr_2")

	indicators.EMA(df, "ema_fast_tempx", "fast_tempx", 3) // 3
	indicators.EMA(df, "ema_slow_tempx", "slow_tempx", 3) // 21

	indicators.KalmanFilter(df, "fast_tempx_kalman", "ema_fast_tempx", 2, 2, true)
	indicators.KalmanFilter(df, "slow_tempx_kalman", "ema_slow_tempx", 64, 128, true)

	indicators.ATR(df, "atr3", "fast_tempx_kalman", 2)
	indicators.ATR(df, "atr3_base", "slow_tempx_kalman", 2)

	indicators.CalcSWAPKalman(df, "swap", "fast_tempx_kalman", 0.25)     // 0.349
	indicators.CalcSWAPKalman(df, "swap_base", "slow_tempx_kalman", 0.3) // 0.295
}
