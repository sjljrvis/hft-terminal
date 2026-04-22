package executor

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"hft/internal/dataframe"
	"hft/internal/indicators"
	"hft/internal/ml_model"
	"hft/internal/storage/sqlite"
	"hft/internal/strategy"

	_df_ "github.com/rocketlaunchr/dataframe-go"
)

// SimConfig controls the live simulation.
type SimConfig struct {
	SimDate    string        // date to simulate, e.g. "2026-03-13"
	WarmupDays int           // days of history before SimDate for indicator warmup
	TickDelay  time.Duration // delay between bars (1s to mock live)

	// OnEvent is called for every simulation event.
	// Wire this to Hub.BroadcastMessage to forward to WebSocket clients.
	OnEvent func(eventType string, data map[string]interface{})
}

func (cfg *SimConfig) emit(eventType string, data map[string]interface{}) {
	if cfg.OnEvent != nil {
		cfg.OnEvent(eventType, data)
	}
}

// ── Per-tranche metadata (mirrors regime_signal.go) ─────────────────────────

type simTrancheM struct {
	earlyDir   int
	avgVolProb float64
	orbHigh    float64
	orbLow     float64
}

type simDayM struct {
	gapPct   float64
	dayType  string
	tranches map[string]*simTrancheM
}

// RunSimulation replays a past date tick-by-tick: for each bar it runs
// prediction and the full signal logic (identical to FindRegimeSignal)
// with delays between bars.
func RunSimulation(cfg SimConfig) error {
	if cfg.WarmupDays <= 0 {
		cfg.WarmupDays = 100
	}
	if cfg.TickDelay <= 0 {
		cfg.TickDelay = 1 * time.Second
	}

	ist := time.FixedZone("IST", 19800)

	// ── 1. Load data ────────────────────────────────────────────────────────
	simDay, err := time.ParseInLocation("2006-01-02", cfg.SimDate, ist)
	if err != nil {
		return fmt.Errorf("invalid sim date: %w", err)
	}
	warmupFrom := simDay.AddDate(0, 0, -cfg.WarmupDays).Format("2006-01-02")
	simEnd := simDay.Format("2006-01-02")

	cfg.emit("sim_start", map[string]interface{}{
		"simDate":    cfg.SimDate,
		"warmupFrom": warmupFrom,
		"tickDelay":  cfg.TickDelay.Seconds(),
		"status":     "loading",
	})
	log.Printf("simulate: loading ticks %s → %s (warmup %d days)", warmupFrom, simEnd, cfg.WarmupDays)

	ctx := context.Background()
	db := sqlite.DefaultDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	ticks, err := db.Ticks.ListTicksFiltered(ctx, "nifty", "", 0, warmupFrom, simEnd)
	if err != nil {
		return fmt.Errorf("load ticks: %w", err)
	}
	log.Printf("simulate: loaded %d ticks", len(ticks))

	// Find sim day boundary.
	simDayStr := simDay.Format("2006-01-02")
	simStartIdx := -1
	for i, t := range ticks {
		if t.Timestamp.In(ist).Format("2006-01-02") == simDayStr {
			simStartIdx = i
			break
		}
	}
	if simStartIdx < 0 {
		return fmt.Errorf("no ticks found for sim date %s", cfg.SimDate)
	}
	simBars := len(ticks) - simStartIdx
	log.Printf("simulate: %d warmup bars, %d sim bars", simStartIdx, simBars)

	// ── 2. Compute indicators (batch — deterministic from price data) ───────
	df := dataframe.InitDataFrame()
	dataframe.LoadHistoryBacktest(df, ticks)
	strategy.RunKalmanv2(df, nil)

	// ── 3. Extract feature columns for per-tick prediction ──────────────────
	pred := ml_model.GetPredictor()
	if pred == nil {
		return fmt.Errorf("predictor not initialized")
	}
	featureNames := pred.FeatureCols()
	featureCols := make([][]float64, len(featureNames))
	for fi, name := range featureNames {
		idx := indicators.FindIndexOf(df, name)
		if idx < 0 {
			// Missing column → zeros (model handles gracefully)
			featureCols[fi] = make([]float64, df.NRows())
		} else {
			featureCols[fi] = df.Series[idx].(*_df_.SeriesFloat64).Values
		}
	}

	// ── 4. Read other series ────────────────────────────────────────────────
	n := df.NRows()
	closeVals := df.Series[indicators.FindIndexOf(df, "close")].(*_df_.SeriesFloat64).Values
	tsVals := df.Series[indicators.FindIndexOf(df, "timestamp")].(*_df_.SeriesTime).Values
	var atrVals []float64
	if idx := indicators.FindIndexOf(df, "atr3"); idx >= 0 {
		atrVals = df.Series[idx].(*_df_.SeriesFloat64).Values
	}

	// Prediction results per-bar (filled during sim day loop).
	probBull := make([]float64, n)
	probBear := make([]float64, n)
	probVol := make([]float64, n)
	regimeStr := make([]string, n)

	rcfg := strategy.DefaultRegimeSignalConfig()

	// ── 5. Pre-compute day metadata (mirrors FindRegimeSignal exactly) ──────
	type barRef struct {
		idx  int
		mins int
	}
	dayBarsMap := make(map[string][]barRef)
	dayOrder := make([]string, 0)
	seen := make(map[string]bool)
	for i := 0; i < n; i++ {
		if tsVals[i] == nil {
			continue
		}
		t := tsVals[i].In(ist)
		key := t.Format("2006-01-02")
		mins := t.Hour()*60 + t.Minute()
		dayBarsMap[key] = append(dayBarsMap[key], barRef{i, mins})
		if !seen[key] {
			seen[key] = true
			dayOrder = append(dayOrder, key)
		}
	}

	sessionOpen := rcfg.Tranches[0].OpenMin
	sessionClose := rcfg.Tranches[len(rcfg.Tranches)-1].CloseMin

	dayMetas := make(map[string]*simDayM)
	var prevDayClose float64
	prevDayValid := false

	for _, dayKey := range dayOrder {
		bars := dayBarsMap[dayKey]
		dm := &simDayM{dayType: "chop"}

		var dOpen, dHigh, dLow, dClose float64
		ohlcInit := false
		for _, b := range bars {
			if b.mins < sessionOpen || b.mins > sessionClose {
				continue
			}
			c := closeVals[b.idx]
			if !ohlcInit {
				dOpen, dHigh, dLow = c, c, c
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

		// Per-tranche early signals.
		dm.tranches = make(map[string]*simTrancheM)
		for _, tr := range rcfg.Tranches {
			tm := &simTrancheM{}
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
				eFirst := closeVals[earlyBars[0].idx]
				eLast := closeVals[earlyBars[len(earlyBars)-1].idx]
				orbH, orbL := eFirst, eFirst
				var volSum float64
				for _, b := range earlyBars {
					c := closeVals[b.idx]
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

	// ── 6. Per-tick loop: predict → decide → emit ───────────────────────────
	cfg.emit("sim_start", map[string]interface{}{
		"simDate":    cfg.SimDate,
		"warmupBars": simStartIdx,
		"simBars":    simBars,
		"status":     "replaying",
	})

	position := 0 // +1 long, -1 short, 0 flat
	var entryPrice float64
	var entrySLPct float64
	var peakPrice float64
	var breakevenActive, trailActive bool
	var activeBE, activeTrailAct, activeTrailOff float64

	currentDayKey := ""
	lastTrancheName := ""
	longTrancheCount := 0
	shortTrancheCount := 0
	longCooldown := 0
	shortCooldown := 0

	netPnL := 0.0
	tradeCount := 0
	winCount := 0

	emitExit := func(i int, reason string) {
		close := closeVals[i]
		pnl := (close - entryPrice) * float64(position)
		netPnL += pnl
		tradeCount++
		if pnl > 0 {
			winCount++
		}

		t := tsVals[i].In(ist)
		side := sideStr(position)

		log.Printf("simulate: ◼ EXIT %s (%s) @ %.2f | entry=%.2f | PnL=%.2f | net=%.2f",
			side, reason, close, entryPrice, pnl, netPnL)
		cfg.emit("sim_exit", map[string]interface{}{
			"time":       t.Format("15:04:05"),
			"side":       side,
			"reason":     reason,
			"entryPrice": entryPrice,
			"exitPrice":  close,
			"pnl":        math.Round(pnl*100) / 100,
			"netPnl":     math.Round(netPnL*100) / 100,
			"tradeCount": tradeCount,
			"winCount":   winCount,
		})

		if position == 1 {
			longCooldown = rcfg.CooldownBars
		} else {
			shortCooldown = rcfg.CooldownBars
		}
		position = 0
		breakevenActive = false
		trailActive = false
	}

	for i := simStartIdx; i < n; i++ {
		if tsVals[i] == nil {
			continue
		}
		t := tsVals[i].In(ist)
		dayKey := t.Format("2006-01-02")
		mins := t.Hour()*60 + t.Minute()
		close := closeVals[i]

		// Day boundary reset.
		if dayKey != currentDayKey {
			currentDayKey = dayKey
			lastTrancheName = ""
			longTrancheCount = 0
			shortTrancheCount = 0
		}

		// Determine active tranche.
		inTranche := false
		var activeTranche strategy.Tranche
		for _, tr := range rcfg.Tranches {
			if mins >= tr.OpenMin && mins <= tr.CloseMin {
				inTranche = true
				activeTranche = tr
				break
			}
		}

		// ── Run prediction for this tick ─────────────────────────────
		predStart := time.Now()
		tp, predErr := pred.PredictSingleRow(featureCols, i)
		predDur := time.Since(predStart)

		if predErr == nil {
			probBull[i] = tp.ProbBull
			probBear[i] = tp.ProbBear
			probVol[i] = tp.ProbVol
			regimeStr[i] = string(tp.Regime)
		}
		// else: prediction unavailable (warmup / invalid features), probs stay 0

		bull := probBull[i]
		bear := probBear[i]
		vol := probVol[i]
		regime := regimeStr[i]

		trancheStr := "outside"
		if inTranche {
			trancheStr = activeTranche.Name
		}

		log.Printf("simulate: %s | close=%.2f | regime=%-9s | bull=%.3f bear=%.3f vol=%.3f | tranche=%s | pos=%s | predict=%v",
			t.Format("15:04"), close, regime, bull, bear, vol, trancheStr, sideStr(position), predDur)

		// ── Emit tick ────────────────────────────────────────────────
		tickData := map[string]interface{}{
			"time":       t.Format("15:04:05"),
			"close":      close,
			"regime":     regime,
			"probBull":   math.Round(bull*1000) / 1000,
			"probBear":   math.Round(bear*1000) / 1000,
			"probVol":    math.Round(vol*1000) / 1000,
			"tranche":    trancheStr,
			"position":   sideStr(position),
			"netPnl":     math.Round(netPnL*100) / 100,
			"barIndex":   i - simStartIdx,
			"totalBars":  simBars,
			"predictMs":  float64(predDur.Microseconds()) / 1000.0,
		}
		if position != 0 {
			unrealPnl := (close - entryPrice) * float64(position)
			unrealPct := unrealPnl / entryPrice * 100
			tickData["unrealPnl"] = math.Round(unrealPnl*100) / 100
			tickData["unrealPct"] = math.Round(unrealPct*1000) / 1000
			tickData["entryPrice"] = entryPrice
		}
		cfg.emit("sim_tick", tickData)

		// ── Outside all tranches → squareoff ─────────────────────────
		if !inTranche {
			if position != 0 {
				emitExit(i, "EOD_SQUAREOFF")
			}
			longCooldown = 0
			shortCooldown = 0
			time.Sleep(cfg.TickDelay)
			continue
		}

		// ── EOD squareoff at tranche close ───────────────────────────
		if mins >= activeTranche.CloseMin {
			if position != 0 {
				emitExit(i, "EOD_SQUAREOFF")
			}
			longCooldown = 0
			shortCooldown = 0
			time.Sleep(cfg.TickDelay)
			continue
		}

		// ── Tranche change → squareoff + reset ───────────────────────
		if activeTranche.Name != lastTrancheName {
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

		// ── Risk management (in position) ────────────────────────────
		if position != 0 {
			unrealPct := (close - entryPrice) / entryPrice * 100 * float64(position)

			// Stop loss / breakeven.
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
				time.Sleep(cfg.TickDelay)
				continue
			}

			// Activate breakeven.
			if !breakevenActive && unrealPct >= activeBE {
				breakevenActive = true
			}

			// Trailing SL.
			if position == 1 && close > peakPrice {
				peakPrice = close
			} else if position == -1 && close < peakPrice {
				peakPrice = close
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
					time.Sleep(cfg.TickDelay)
					continue
				}
			}

			time.Sleep(cfg.TickDelay)
			continue
		}

		// ── Entry logic (flat) ───────────────────────────────────────
		if mins >= activeTranche.CutoffMin {
			time.Sleep(cfg.TickDelay)
			continue
		}

		dm := dayMetas[dayKey]
		if dm == nil {
			time.Sleep(cfg.TickDelay)
			continue
		}

		// Per-tranche early signals.
		tm := dm.tranches[activeTranche.Name]

		// Max vol prob filter.
		if rcfg.MaxVolProb > 0 && tm != nil && tm.avgVolProb > rcfg.MaxVolProb {
			time.Sleep(cfg.TickDelay)
			continue
		}

		// Candidate directions.
		wantLong := bull > rcfg.BullProbThresh && longCooldown == 0 && longTrancheCount < rcfg.MaxTradesPerDay
		wantShort := bear > rcfg.BearProbThresh && shortCooldown == 0 && shortTrancheCount < rcfg.MaxTradesPerDay

		// Gap follow filter.
		if rcfg.GapFollow && math.Abs(dm.gapPct) > 0.15 {
			if dm.gapPct > 0 {
				wantShort = false
			} else {
				wantLong = false
			}
		}

		// Early direction confirm.
		if rcfg.EarlyDirConfirm && tm != nil {
			if tm.earlyDir <= 0 {
				wantLong = false
			}
			if tm.earlyDir >= 0 {
				wantShort = false
			}
		}

		// ── Execute entry ────────────────────────────────────────────
		if wantLong {
			entryPrice = close
			peakPrice = close
			breakevenActive = false
			trailActive = false
			position = 1

			atr := 0.0
			if atrVals != nil {
				atr = atrVals[i]
			}
			if rcfg.LongATRMult > 0 && atr > 0 {
				entrySLPct = math.Max(atr*rcfg.LongATRMult/close*100, rcfg.LongSLPct)
			} else {
				entrySLPct = rcfg.LongSLPct
			}
			activeBE = rcfg.LongBEPct
			activeTrailAct = rcfg.LongTrailAct
			activeTrailOff = rcfg.LongTrailOff
			longTrancheCount++

			log.Printf("simulate: ▶ ENTRY BUY @ %.2f | SL%%=%.3f | tranche=%s", close, entrySLPct, activeTranche.Name)
			cfg.emit("sim_entry", map[string]interface{}{
				"time":       t.Format("15:04:05"),
				"side":       "LONG",
				"price":      close,
				"slPct":      math.Round(entrySLPct*1000) / 1000,
				"tranche":    activeTranche.Name,
				"probBull":   math.Round(bull*1000) / 1000,
				"tradeCount": tradeCount,
			})

		} else if wantShort {
			entryPrice = close
			peakPrice = close
			breakevenActive = false
			trailActive = false
			position = -1

			atr := 0.0
			if atrVals != nil {
				atr = atrVals[i]
			}
			if rcfg.ShortATRMult > 0 && atr > 0 {
				entrySLPct = math.Max(atr*rcfg.ShortATRMult/close*100, rcfg.ShortSLPct)
			} else {
				entrySLPct = rcfg.ShortSLPct
			}
			activeBE = rcfg.ShortBEPct
			activeTrailAct = rcfg.ShortTrailAct
			activeTrailOff = rcfg.ShortTrailOff
			shortTrancheCount++

			log.Printf("simulate: ▶ ENTRY SELL @ %.2f | SL%%=%.3f | tranche=%s", close, entrySLPct, activeTranche.Name)
			cfg.emit("sim_entry", map[string]interface{}{
				"time":       t.Format("15:04:05"),
				"side":       "SHORT",
				"price":      close,
				"slPct":      math.Round(entrySLPct*1000) / 1000,
				"tranche":    activeTranche.Name,
				"probBear":   math.Round(bear*1000) / 1000,
				"tradeCount": tradeCount,
			})
		}

		time.Sleep(cfg.TickDelay)
	}

	// ── Summary ─────────────────────────────────────────────────────────────
	winRate := 0.0
	if tradeCount > 0 {
		winRate = float64(winCount) / float64(tradeCount) * 100
	}
	log.Printf("simulate: === COMPLETE === trades=%d wins=%d (%.1f%%) net=%.2f pts",
		tradeCount, winCount, winRate, netPnL)

	cfg.emit("sim_end", map[string]interface{}{
		"simDate":    cfg.SimDate,
		"tradeCount": tradeCount,
		"winCount":   winCount,
		"winRate":    math.Round(winRate*10) / 10,
		"netPnl":     math.Round(netPnL*100) / 100,
	})

	return nil
}

func sideStr(pos int) string {
	switch pos {
	case 1:
		return "LONG"
	case -1:
		return "SHORT"
	default:
		return "FLAT"
	}
}
