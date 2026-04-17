package strategy

import (
	"fmt"
	"math"
	"time"

	"hft/internal/indicators"
	"hft/pkg/types"

	"github.com/rocketlaunchr/dataframe-go"
)

/*
   Regime-model-based entry / exit strategy.

   Ports the logic from regime-model/backtest_high_prob_v2.py and
   backtest_tranches.py into the Go backtest engine.

   Call order in engine.go:
     1. strategy.RunKalmanv2(df, logEvents)          // indicators
     2. ml_model.PredictRegimeFromDFStrided(df, 10)  // model predictions
     3. strategy.FindRegimeSignal(df, pos, positions, events)
*/

// ─── Config ──────────────────────────────────────────────────────────────────

// RegimeSignalConfig controls the regime-based trading strategy.
type RegimeSignalConfig struct {
	// Probability thresholds
	BullProbThresh float64
	BearProbThresh float64

	// Long risk params (% of entry price)
	LongSLPct    float64 // fixed SL %
	LongATRMult  float64 // ATR multiplier for SL
	LongBEPct    float64 // breakeven activation %
	LongTrailAct float64 // trailing SL activation %
	LongTrailOff float64 // trailing SL offset %

	// Short risk params
	ShortSLPct    float64
	ShortATRMult  float64
	ShortBEPct    float64
	ShortTrailAct float64
	ShortTrailOff float64

	MaxTradesPerDay int
	CooldownBars    int

	// Filters
	GapFollow       bool
	EarlyDirConfirm bool
	MaxVolProb      float64

	// Tranches — non-overlapping time windows (minutes from midnight IST)
	Tranches []Tranche
}

// Tranche defines a non-overlapping trading time window.
type Tranche struct {
	Name      string
	OpenMin   int // entry allowed from (inclusive)
	CutoffMin int // no new entries after (inclusive)
	CloseMin  int // EOD squareoff at (inclusive)
}

// DefaultRegimeSignalConfig returns the winning backtest config.
func DefaultRegimeSignalConfig() *RegimeSignalConfig {
	return &RegimeSignalConfig{
		BullProbThresh: 0.75,
		BearProbThresh: 0.75,

		LongSLPct:    0.35,
		LongATRMult:  2.5,
		LongBEPct:    0.35,
		LongTrailAct: 0.4,
		LongTrailOff: 0.4,

		ShortSLPct:    0.30,
		ShortATRMult:  2.5,
		ShortBEPct:    0.35,
		ShortTrailAct: 0.4,
		ShortTrailOff: 0.4,

		MaxTradesPerDay: 20,
		CooldownBars:    3,

		GapFollow:       true,
		EarlyDirConfirm: true,
		MaxVolProb:      0.25,

		Tranches: []Tranche{
			{"morning", 9*60 + 59, 11 * 60, 11*60 + 59},
			{"midday", 12 * 60, 13 * 60, 14*60 + 59},
			{"close", 15 * 60, 15*60 + 10, 15*60 + 20},
		},
	}
}

// ─── Day metadata ────────────────────────────────────────────────────────────

// trancheMeta holds per-tranche early signals (first 30 min from tranche open).
type trancheMeta struct {
	earlyDir   int     // +1 up, -1 down, 0 flat
	avgVolProb float64 // mean volatile prob in first 30 min of tranche
	orbHigh    float64 // 30-min opening range high
	orbLow     float64 // 30-min opening range low
}

type dayMeta struct {
	dayType  string  // "trend", "chop", "gap_reversal"
	gapPct   float64 // gap open vs prev close %
	tranches map[string]*trancheMeta
}

// ─── Public entry point ──────────────────────────────────────────────────────

// FindRegimeSignal uses ml model predictions on the DataFrame to generate
// entry/exit events. Requires pred_prob_bullish, pred_prob_bearish,
// pred_prob_volatile columns (added by PredictRegimeFromDFStrided).
func FindRegimeSignal(df *dataframe.DataFrame, currentPos *types.Position, positions []*types.Position, events chan *types.Event) {
	FindRegimeSignalWithConfig(df, currentPos, positions, events, DefaultRegimeSignalConfig())
}

// FindRegimeSignalWithConfig is the configurable version.
func FindRegimeSignalWithConfig(df *dataframe.DataFrame, currentPos *types.Position, positions []*types.Position, events chan *types.Event, cfg *RegimeSignalConfig) {
	if cfg == nil {
		cfg = DefaultRegimeSignalConfig()
	}

	n := df.NRows()
	if n < 2 {
		return
	}

	// ── Read series ──────────────────────────────────────────────
	_close := df.Series[indicators.FindIndexOf(df, "close")].(*dataframe.SeriesFloat64).Values
	_ts := df.Series[indicators.FindIndexOf(df, "timestamp")].(*dataframe.SeriesTime).Values

	probBullIdx := indicators.FindIndexOf(df, "pred_prob_bullish")
	probBearIdx := indicators.FindIndexOf(df, "pred_prob_bearish")
	probVolIdx := indicators.FindIndexOf(df, "pred_prob_volatile")
	atrIdx := indicators.FindIndexOf(df, "atr3")

	if probBullIdx < 0 || probBearIdx < 0 || probVolIdx < 0 {
		fmt.Println("FindRegimeSignal: prediction columns missing, skipping")
		return
	}

	probBull := df.Series[probBullIdx].(*dataframe.SeriesFloat64).Values
	probBear := df.Series[probBearIdx].(*dataframe.SeriesFloat64).Values
	probVol := df.Series[probVolIdx].(*dataframe.SeriesFloat64).Values
	var atrVals []float64
	if atrIdx >= 0 {
		atrVals = df.Series[atrIdx].(*dataframe.SeriesFloat64).Values
	}

	ist := time.FixedZone("IST", 19800)

	// ── Pre-compute day metadata ─────────────────────────────────
	// Group bar indices by date string.
	type barRef struct {
		idx  int
		mins int // minutes from midnight IST
	}
	dayBars := make(map[string][]barRef)
	dayOrder := make([]string, 0)
	seen := make(map[string]bool)
	for i := 0; i < n; i++ {
		if _ts[i] == nil {
			continue
		}
		t := _ts[i].In(ist)
		key := t.Format("2006-01-02")
		mins := t.Hour()*60 + t.Minute()
		dayBars[key] = append(dayBars[key], barRef{i, mins})
		if !seen[key] {
			seen[key] = true
			dayOrder = append(dayOrder, key)
		}
	}

	// Build per-day metadata.
	dayMetas := make(map[string]*dayMeta)
	var prevDayClose float64
	prevDayValid := false

	// Use the widest tranche open as the session start for OHLC computation.
	sessionOpen := 9*60 + 50
	if len(cfg.Tranches) > 0 {
		sessionOpen = cfg.Tranches[0].OpenMin
	}
	sessionClose := 15*60 + 20
	if len(cfg.Tranches) > 0 {
		sessionClose = cfg.Tranches[len(cfg.Tranches)-1].CloseMin
	}

	for _, dayKey := range dayOrder {
		bars := dayBars[dayKey]
		dm := &dayMeta{dayType: "chop"}

		// Collect in-session closes for OHLC.
		var dOpen, dHigh, dLow, dClose float64
		ohlcInit := false
		for _, b := range bars {
			if b.mins < sessionOpen || b.mins > sessionClose {
				continue
			}
			c := _close[b.idx]
			if !ohlcInit {
				dOpen = c
				dHigh = c
				dLow = c
				ohlcInit = true
			}
			if c > dHigh {
				dHigh = c
			}
			if c < dLow {
				dLow = c
			}
			dClose = c
		}

		if ohlcInit {
			dRange := dHigh - dLow
			if dRange > 0 && prevDayValid {
				gap := (dOpen - prevDayClose) / prevDayClose * 100
				dm.gapPct = gap
				gapUp := gap > 0.15
				gapDown := gap < -0.15
				bodyRatio := math.Abs(dClose-dOpen) / dRange
				if (gapUp && dClose < dOpen) || (gapDown && dClose > dOpen) {
					dm.dayType = "gap_reversal"
				} else if bodyRatio > 0.45 {
					dm.dayType = "trend"
				}
			}
			prevDayClose = dClose
			prevDayValid = true
		}

		// Per-tranche early signals: first 30 minutes from each tranche's OpenMin.
		dm.tranches = make(map[string]*trancheMeta)
		for _, tr := range cfg.Tranches {
			tm := &trancheMeta{}
			earlyBars := make([]barRef, 0)
			firstBarMin := -1
			for _, b := range bars {
				if b.mins >= tr.OpenMin && b.mins <= tr.CloseMin {
					if firstBarMin < 0 {
						firstBarMin = b.mins
					}
					if b.mins <= firstBarMin+30 {
						earlyBars = append(earlyBars, b)
					}
				}
			}
			if len(earlyBars) > 0 {
				eFirst := _close[earlyBars[0].idx]
				eLast := _close[earlyBars[len(earlyBars)-1].idx]
				orbH := eFirst
				orbL := eFirst
				var volSum float64
				for _, b := range earlyBars {
					c := _close[b.idx]
					if c > orbH {
						orbH = c
					}
					if c < orbL {
						orbL = c
					}
					volSum += probVol[b.idx]
				}
				tm.orbHigh = orbH
				tm.orbLow = orbL
				tm.avgVolProb = volSum / float64(len(earlyBars))
				move := eLast - eFirst
				if move > 0 {
					tm.earlyDir = 1
				} else if move < 0 {
					tm.earlyDir = -1
				}
			}
			dm.tranches[tr.Name] = tm
		}

		dayMetas[dayKey] = dm
	}

	// ── Trading loop ─────────────────────────────────────────────
	position := 0 // +1 long, -1 short, 0 flat
	var entryPrice float64
	var entryTime time.Time
	var entrySLPct float64
	var peakPrice float64
	var breakevenActive, trailActive bool
	var mfe, mae float64

	// Active risk config for open trade.
	var activeSL, activeBE, activeTrailAct, activeTrailOff float64

	currentDayKey := ""
	lastTrancheName := ""
	longTrancheCount := 0
	shortTrancheCount := 0
	longCooldown := 0
	shortCooldown := 0

	emitExit := func(i int, reason string) {
		price := _close[i]
		ts := *_ts[i]

		kind := "BUY"
		if position == -1 {
			kind = "SELL"
		}
		pnl := (price - entryPrice) * float64(position)

		currentPos.Exit(price, ts)
		events <- &types.Event{
			Kind:       kind,
			Type:       "EXIT",
			EntryPrice: price,
			Timestamp:  ts,
			Reason:     reason,
			PeakProfit: currentPos.PeakProfit,
			PeakLoss:   currentPos.PeakLoss,
		}
		currentPos.Reset()

		_ = pnl // logged by subscriber

		if position == 1 {
			longCooldown = cfg.CooldownBars
		} else {
			shortCooldown = cfg.CooldownBars
		}
		position = 0
		breakevenActive = false
		trailActive = false
	}

	for i := 0; i < n; i++ {
		if _ts[i] == nil {
			continue
		}
		t := _ts[i].In(ist)
		dayKey := t.Format("2006-01-02")
		mins := t.Hour()*60 + t.Minute()
		close := _close[i]

		// Day boundary reset.
		if dayKey != currentDayKey {
			currentDayKey = dayKey
			lastTrancheName = ""
			longTrancheCount = 0
			shortTrancheCount = 0
		}

		// Determine which tranche this bar belongs to (if any).
		inTranche := false
		var activeTranche Tranche
		for _, tr := range cfg.Tranches {
			if mins >= tr.OpenMin && mins <= tr.CloseMin {
				inTranche = true
				activeTranche = tr
				break
			}
		}

		// Outside all tranches — squareoff if in position.
		if !inTranche {
			if position != 0 {
				emitExit(i, "EOD_SQUAREOFF")
			}
			longCooldown = 0
			shortCooldown = 0
			continue
		}

		// EOD squareoff at tranche close boundary.
		if mins >= activeTranche.CloseMin {
			if position != 0 {
				emitExit(i, "EOD_SQUAREOFF")
			}
			longCooldown = 0
			shortCooldown = 0
			continue
		}

		// Per-tranche trade count reset: when tranche changes, reset counts.
		if activeTranche.Name != lastTrancheName {
			// Squareoff any position from previous tranche carried into gap.
			if position != 0 {
				emitExit(i, "EOD_SQUAREOFF")
			}
			lastTrancheName = activeTranche.Name
			longTrancheCount = 0
			shortTrancheCount = 0
			longCooldown = 0
			shortCooldown = 0
		}

		// Tick cooldowns.
		if longCooldown > 0 {
			longCooldown--
		}
		if shortCooldown > 0 {
			shortCooldown--
		}

		// ── Risk management (when in position) ───────────────────
		if position != 0 {
			// Update peak profit / loss on currentPos.
			if position == 1 {
				currentPos.Profit = close - entryPrice
			} else {
				currentPos.Profit = entryPrice - close
			}
			currentPos.PeakProfit = math.Max(currentPos.PeakProfit, currentPos.Profit)
			currentPos.PeakLoss = math.Min(currentPos.PeakLoss, currentPos.Profit)

			// MFE / MAE.
			fav := (close - entryPrice) * float64(position)
			adv := (entryPrice - close) * float64(position)
			if fav > mfe {
				mfe = fav
			}
			if adv > mae {
				mae = adv
			}

			unrealPct := (close - entryPrice) / entryPrice * 100 * float64(position)

			// 1) Stop loss (or breakeven stop).
			effectiveSL := -entrySLPct
			if breakevenActive {
				effectiveSL = 0.0
			}
			if unrealPct <= effectiveSL {
				reason := "STOP_LOSS"
				if breakevenActive {
					reason = "BREAKEVEN"
				}
				emitExit(i, reason)
				continue
			}

			// 2) Activate breakeven.
			if !breakevenActive && unrealPct >= activeBE {
				breakevenActive = true
			}

			// 3) Trailing SL.
			if position == 1 {
				if close > peakPrice {
					peakPrice = close
				}
			} else {
				if close < peakPrice {
					peakPrice = close
				}
			}
			if !trailActive && unrealPct >= activeTrailAct {
				trailActive = true
			}
			if trailActive {
				hitSL := false
				if position == 1 {
					trailSL := peakPrice * (1 - activeTrailOff/100)
					hitSL = close <= trailSL
				} else {
					trailSL := peakPrice * (1 + activeTrailOff/100)
					hitSL = close >= trailSL
				}
				if hitSL {
					emitExit(i, "TRAILING_SL")
					continue
				}
			}

			// Stay in position.
			continue
		}

		// ── Entry logic (flat) ───────────────────────────────────
		// No new entries after tranche cutoff.
		if mins >= activeTranche.CutoffMin {
			continue
		}

		dm := dayMetas[dayKey]
		if dm == nil {
			continue
		}

		// Look up per-tranche early signals.
		tm := dm.tranches[activeTranche.Name]

		// Max vol prob filter (per-tranche).
		if cfg.MaxVolProb > 0 && tm != nil && tm.avgVolProb > cfg.MaxVolProb {
			continue
		}

		// Candidate directions (per-tranche trade counts).
		wantLong := probBull[i] > cfg.BullProbThresh && longCooldown == 0 && longTrancheCount < cfg.MaxTradesPerDay
		wantShort := probBear[i] > cfg.BearProbThresh && shortCooldown == 0 && shortTrancheCount < cfg.MaxTradesPerDay

		// Gap follow filter.
		if cfg.GapFollow && math.Abs(dm.gapPct) > 0.15 {
			if dm.gapPct > 0 {
				wantShort = false // gap up → longs only
			} else {
				wantLong = false // gap down → shorts only
			}
		}

		// Early direction confirm (per-tranche).
		if cfg.EarlyDirConfirm && tm != nil {
			if tm.earlyDir <= 0 {
				wantLong = false
			}
			if tm.earlyDir >= 0 {
				wantShort = false
			}
		}

		// ── Execute entry ────────────────────────────────────────
		if wantLong {
			entryPrice = close
			entryTime = *_ts[i]
			peakPrice = close
			breakevenActive = false
			trailActive = false
			mfe = 0
			mae = 0
			position = 1

			// Compute SL: max(ATR-based, fixed)
			atr := 0.0
			if atrVals != nil {
				atr = atrVals[i]
			}
			if cfg.LongATRMult > 0 && atr > 0 {
				entrySLPct = math.Max(atr*cfg.LongATRMult/close*100, cfg.LongSLPct)
			} else {
				entrySLPct = cfg.LongSLPct
			}
			activeSL = cfg.LongSLPct
			activeBE = cfg.LongBEPct
			activeTrailAct = cfg.LongTrailAct
			activeTrailOff = cfg.LongTrailOff

			currentPos.Buy(close, entryTime)
			currentPos.PeakProfit = 0
			currentPos.PeakLoss = 0
			events <- &types.Event{
				Kind:       "BUY",
				Type:       "ENTRY",
				EntryPrice: close,
				Timestamp:  entryTime,
			}
			longTrancheCount++
		} else if wantShort {
			entryPrice = close
			entryTime = *_ts[i]
			peakPrice = close
			breakevenActive = false
			trailActive = false
			mfe = 0
			mae = 0
			position = -1

			atr := 0.0
			if atrVals != nil {
				atr = atrVals[i]
			}
			if cfg.ShortATRMult > 0 && atr > 0 {
				entrySLPct = math.Max(atr*cfg.ShortATRMult/close*100, cfg.ShortSLPct)
			} else {
				entrySLPct = cfg.ShortSLPct
			}
			activeSL = cfg.ShortSLPct
			activeBE = cfg.ShortBEPct
			activeTrailAct = cfg.ShortTrailAct
			activeTrailOff = cfg.ShortTrailOff

			currentPos.Sell(close, entryTime)
			currentPos.PeakProfit = 0
			currentPos.PeakLoss = 0
			events <- &types.Event{
				Kind:       "SELL",
				Type:       "ENTRY",
				EntryPrice: close,
				Timestamp:  entryTime,
			}
			shortTrancheCount++
		}
	}

	_, _ = activeSL, entryTime // suppress unused warnings
}
