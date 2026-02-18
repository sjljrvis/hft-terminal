package strategy

import (
	"hft/internal/indicators"
	"hft/pkg/types"
	"math"
	"time"

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
		ActivationMFEPts:  500,
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

// isAfter915 checks if the timestamp is at or after 9:15 IST
func isAfter915(ts *time.Time) bool {
	if ts == nil || ts.IsZero() {
		return false
	}
	ist := time.FixedZone("IST", 19800) // UTC+5:30
	t := ts.In(ist)
	mins := t.Hour()*60 + t.Minute()
	return mins >= 9*60+15
}

// hasIntersected checks if fast_tempx_kalman has intersected slow_tempx_kalman between i-1 and i
func hasIntersected(fastPrev, fastCurr, slowPrev, slowCurr float64) bool {
	// Check if lines have crossed: sign change in (fast - slow)
	diffPrev := fastPrev - slowPrev
	diffCurr := fastCurr - slowCurr
	return (diffPrev > 0 && diffCurr <= 0) || (diffPrev < 0 && diffCurr >= 0)
}

// isParallel checks if fast and slow are parallel (same direction of movement)
func isParallel(fastPrev, fastCurr, slowPrev, slowCurr float64) bool {
	fastDir := fastCurr - fastPrev
	slowDir := slowCurr - slowPrev
	// Parallel means both moving in same direction (both up or both down)
	return (fastDir > 0 && slowDir > 0) || (fastDir < 0 && slowDir < 0)
}

// isSwapYellow checks if swap value represents "yellow" (neutral/zero)
func isSwapYellow(swap float64) bool {
	return swap == 0
}

// isSwapGreen checks if swap value represents "green" (bullish/positive)
func isSwapGreen(swap float64) bool {
	return swap == 1
}

// isSwapPink checks if swap value represents "pink" (bearish/negative)
func isSwapPink(swap float64) bool {
	return swap == -1
}

/*
	RULES:
	- first trade might be volatile most of thee time, so we need to be careful with the entry conditions
	- fast entry - fast exit is required for first trade of the day
	- params to track:
	   (_fast_tempx_kalman , _slow_tempx_kalman) : check if they are parallel to each other
	   count to maintain total number of intersections between fast and slow tempx kalman

	- First trade:
	  - Check for intersection from 9:15
	    - If _fast_tempx_kalman interesects _slow_tempx_kalman and _slow_tempx_kalman is yellow and _fast_tempx_kalman is not yellow then wait till _slow_tempx_kalman is flipped (ignore the signal)
	    - if _fast_tempx_kalman is parallel to _slow_tempx_kalman from market open till _fast_tempx_kalman is flipped then take the trade based on _fast_tempx_kalman color (for buy _fast_tempx_kalman should be above _slow_tempx_kalman and for sell _fast_tempx_kalman should be below _slow_tempx_kalman)

-  Once first trade is over, purely  trade when _fast_tempx_kalman aligns with _slow_tempx_kalman ( color for both should be same and _fast_tempx_kalman should be above _slow_tempx_kalman for buy and _fast_tempx_kalman should be below _slow_tempx_kalman for sell)
*/
func FindKalmanSignalWithExitConfigv2(df *dataframe.DataFrame, current_position *types.Position, positions []*types.Position, events chan *types.Event, exitConfig *KalmanExitConfigv2) {
	if exitConfig == nil {
		exitConfig = DefaultKalmanExitConfigv2()
	}

	_dataframe_length := df.NRows()
	_close := df.Series[indicators.FindIndexOf(df, "close")].(*dataframe.SeriesFloat64).Values
	_timestamp := df.Series[indicators.FindIndexOf(df, "timestamp")].(*dataframe.SeriesTime).Values
	_fast_tempx_kalman := df.Series[indicators.FindIndexOf(df, "fast_tempx_kalman")].(*dataframe.SeriesFloat64).Values
	_slow_tempx_kalman := df.Series[indicators.FindIndexOf(df, "slow_tempx_kalman")].(*dataframe.SeriesFloat64).Values
	_slow_swap := df.Series[indicators.FindIndexOf(df, "swap_base")].(*dataframe.SeriesFloat64).Values
	_fast_swap := df.Series[indicators.FindIndexOf(df, "swap")].(*dataframe.SeriesFloat64).Values

	tradeState := &KalmanTradeStatev2{}

	// Track first trade state
	firstTradeCompleted := false
	firstTradeDay := -1             // Track which day we're on
	waitingForSlowSwapFlip := false // Track if we're waiting for slow swap flip after intersection
	parallelFromOpen := false       // Track if lines have been parallel from market open

	for i := 0; i < _dataframe_length; i++ {
		// Check if we're in a new trading day
		currentDay := _timestamp[i].YearDay()
		if currentDay != firstTradeDay {
			firstTradeCompleted = false
			firstTradeDay = currentDay
			waitingForSlowSwapFlip = false
			parallelFromOpen = false
		}

		// Swap flip conditions
		slowSwapFlipUp := i > 0 && _slow_swap[i] == 1 && (_slow_swap[i-1] == 0 || _slow_swap[i-1] == -1)
		slowSwapFlipDown := i > 0 && _slow_swap[i] == -1 && (_slow_swap[i-1] == 0 || _slow_swap[i-1] == 1)
		fastSwapFlipUp := i > 0 && _fast_swap[i] == 1 && _fast_swap[i-1] == -1
		fastSwapFlipDown := i > 0 && _fast_swap[i] == -1 && _fast_swap[i-1] == 1
		slowSwapFlipSignal := i > 0 && ((_slow_swap[i-1] == 1 && _slow_swap[i] == -1) || (_slow_swap[i-1] == -1 && _slow_swap[i] == 1))
		fastSwapFlipSignal := i > 0 && ((_fast_swap[i-1] == 1 && _fast_swap[i] == -1) || (_fast_swap[i-1] == -1 && _fast_swap[i] == 1))

		// Late entry conditions
		lateBuy := fastSwapFlipUp && _slow_swap[i] == 1
		lateSell := fastSwapFlipDown && _slow_swap[i] == -1

		// Check for intersection and parallel conditions (only if we have previous data)
		hasIntersection := false
		isParallelNow := false
		if i > 0 {
			hasIntersection = hasIntersected(_fast_tempx_kalman[i-1], _fast_tempx_kalman[i],
				_slow_tempx_kalman[i-1], _slow_tempx_kalman[i])
			isParallelNow = isParallel(_fast_tempx_kalman[i-1], _fast_tempx_kalman[i],
				_slow_tempx_kalman[i-1], _slow_tempx_kalman[i])
		}

		// First trade logic: check for intersection from 9:15
		if !firstTradeCompleted && isAfter915(_timestamp[i]) {
			if hasIntersection {
				// If fast intersects slow and slow is yellow (0) and fast is not yellow, wait for slow flip
				if isSwapYellow(_slow_swap[i]) && !isSwapYellow(_fast_swap[i]) {
					waitingForSlowSwapFlip = true
				} else {
					waitingForSlowSwapFlip = false
				}
				parallelFromOpen = false
			} else if isParallelNow {
				// Track if parallel from market open
				if !parallelFromOpen {
					parallelFromOpen = true
				}
			} else {
				parallelFromOpen = false
			}

			// Check if slow swap has flipped (clearing the wait condition)
			if waitingForSlowSwapFlip && slowSwapFlipSignal {
				waitingForSlowSwapFlip = false
			}
		}

		// Determine buy/sell conditions based on first trade vs subsequent trades
		var _buy_condition, _sell_condition bool

		if !firstTradeCompleted {
			// First trade logic
			if waitingForSlowSwapFlip {
				// Ignore signals while waiting for slow swap flip
				_buy_condition = false
				_sell_condition = false
			} else if parallelFromOpen && (fastSwapFlipUp || fastSwapFlipDown) {
				// If parallel from open and fast swap flips, take trade based on fast color
				// For buy: fast should be above slow and fast swap should be green (1)
				// For sell: fast should be below slow and fast swap should be pink (-1)
				if fastSwapFlipUp && _fast_tempx_kalman[i] > _slow_tempx_kalman[i] {
					_buy_condition = true
					_sell_condition = false
				} else if fastSwapFlipDown && _fast_tempx_kalman[i] < _slow_tempx_kalman[i] {
					_sell_condition = true
					_buy_condition = false
				} else {
					_buy_condition = false
					_sell_condition = false
				}
			} else {
				// Fallback to original logic for first trade
				_buy_condition = slowSwapFlipUp || lateBuy
				_sell_condition = slowSwapFlipDown || lateSell
			}
		} else {
			// Subsequent trades: align when both swaps have same color
			// Buy: both green (1), fast above slow
			// Sell: both pink (-1), fast below slow
			bothGreen := isSwapGreen(_slow_swap[i]) && isSwapGreen(_fast_swap[i])
			bothPink := isSwapPink(_slow_swap[i]) && isSwapPink(_fast_swap[i])
			fastAboveSlow := _fast_tempx_kalman[i] > _slow_tempx_kalman[i]
			fastBelowSlow := _fast_tempx_kalman[i] < _slow_tempx_kalman[i]

			_buy_condition = bothGreen && fastAboveSlow && (slowSwapFlipUp || lateBuy)
			_sell_condition = bothPink && fastBelowSlow && (slowSwapFlipDown || lateSell)
		}

		sessionClosed := !indicators.IsActiveSession(_timestamp[i])
		exitSignal := false

		// Update peak profit and peak loss
		if current_position.Kind != "" {
			if current_position.Kind == "BUY" {
				current_position.Profit = _close[i] - current_position.EntryPrice
			} else if current_position.Kind == "SELL" {
				current_position.Profit = current_position.EntryPrice - _close[i]
			}

			current_position.PeakProfit = math.Max(current_position.PeakProfit, current_position.Profit)
			current_position.PeakLoss = math.Min(current_position.PeakLoss, current_position.Profit)
		}

		fixedSLHit := current_position.Profit <= exitConfig.FixedSL && exitConfig.EnableFixedSL

		// Determine exit signal
		if current_position.Kind == "BUY" {
			exitSignal = _sell_condition || slowSwapFlipSignal || fastSwapFlipSignal || fixedSLHit
		} else if current_position.Kind == "SELL" {
			exitSignal = _buy_condition || slowSwapFlipSignal || fastSwapFlipSignal || fixedSLHit
		}

		// Check if should exit trade
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

		// Handle exits
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
			if !firstTradeCompleted {
				firstTradeCompleted = true
			}
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
			if !firstTradeCompleted {
				firstTradeCompleted = true
			}
		}

		// Handle entries
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

	indicators.KalmanFilter(df, "fast_tempx_kalman", "ema_fast_tempx", 32, 32, true)
	indicators.KalmanFilter(df, "slow_tempx_kalman", "ema_slow_tempx", 64, 64, true)

	indicators.ATR(df, "atr3", "fast_tempx_kalman", 2)
	indicators.ATR(df, "atr3_base", "slow_tempx_kalman", 2)

	indicators.CalcSWAPKalman(df, "swap", "fast_tempx_kalman", 0.25)     // 0.349
	indicators.CalcSWAPKalman(df, "swap_base", "slow_tempx_kalman", 0.3) // 0.295
}
