package indicators

import (
	"fmt"
	"math"
	"time"

	"github.com/rocketlaunchr/dataframe-go"
)

/* Mirrors regime-model/features/indicators.py:add_multi_timeframe_features.

   For each higher timeframe `tfMin` (e.g. 5, 15) we:
     1. group 1m bars into HT buckets via t.Truncate(tfMin*Minute)
        (Truncate operates on absolute time; IST offset 05:30 is divisible
        by 5min and 15min, so the boundaries align with NSE 09:15+).
     2. build HT OHLC by walking the 1m series in order
     3. compute EMA(5), EMA(21), RSI(7), ATR(7) on the HT close
     4. derive 5 scale-invariant features per HT bar
     5. shift by 1 HT bar (no look-ahead — the current 1m row only sees
        features from the *previous* completed HT bar)
     6. forward-fill back to 1m: every 1m row in bucket b reads HT[b-1]
*/

// AddMTFFeatures appends 5 columns suffixed _{tfMin}m:
//
//	dist_ema_fast_{tf}m, dist_ema_slow_{tf}m, ema_crossover_{tf}m,
//	rsi_{tf}m, atr_pct_{tf}m
func AddMTFFeatures(df *dataframe.DataFrame, tfMin int) {
	suffix := fmt.Sprintf("%dm", tfMin)
	names := []string{
		"dist_ema_fast_" + suffix,
		"dist_ema_slow_" + suffix,
		"ema_crossover_" + suffix,
		"rsi_" + suffix,
		"atr_pct_" + suffix,
	}

	n := df.NRows()
	emitZero := func() {
		for _, name := range names {
			df.AddSeries(dataframe.NewSeriesFloat64(name, nil, make([]float64, n)), nil)
		}
	}

	tsIdx := FindIndexOf(df, "timestamp")
	if tsIdx < 0 {
		emitZero()
		return
	}
	_ts := df.Series[tsIdx].(*dataframe.SeriesTime).Values
	_open := df.Series[FindIndexOf(df, "open")].(*dataframe.SeriesFloat64).Values
	_high := df.Series[FindIndexOf(df, "high")].(*dataframe.SeriesFloat64).Values
	_low := df.Series[FindIndexOf(df, "low")].(*dataframe.SeriesFloat64).Values
	_close := df.Series[FindIndexOf(df, "close")].(*dataframe.SeriesFloat64).Values

	bucket := time.Duration(tfMin) * time.Minute

	// Build HT bars + per-1m bucket index. bucketIdx[i] = -1 until first valid bar.
	htOpen := make([]float64, 0, n/tfMin+1)
	htHigh := make([]float64, 0, n/tfMin+1)
	htLow := make([]float64, 0, n/tfMin+1)
	htClose := make([]float64, 0, n/tfMin+1)
	bucketIdx := make([]int, n)
	for i := range bucketIdx {
		bucketIdx[i] = -1
	}

	bIdx := -1
	var lastBucket time.Time
	for i := 0; i < n; i++ {
		if _ts[i] == nil {
			bucketIdx[i] = bIdx
			continue
		}
		bs := _ts[i].Truncate(bucket)
		if bIdx < 0 || !bs.Equal(lastBucket) {
			bIdx++
			lastBucket = bs
			htOpen = append(htOpen, _open[i])
			htHigh = append(htHigh, _high[i])
			htLow = append(htLow, _low[i])
			htClose = append(htClose, _close[i])
		} else {
			if _high[i] > htHigh[bIdx] {
				htHigh[bIdx] = _high[i]
			}
			if _low[i] < htLow[bIdx] {
				htLow[bIdx] = _low[i]
			}
			htClose[bIdx] = _close[i]
		}
		bucketIdx[i] = bIdx
	}

	nHT := len(htClose)
	if nHT < 30 {
		emitZero()
		return
	}

	// Indicators on HT close.
	emaFast := emaArray(htClose, 5)
	emaSlow := emaArray(htClose, 21)
	rsiHT := rsiArray(htClose, 7)
	atrHT := atrArray(htHigh, htLow, htClose, 7)

	// Derived features per HT bar.
	htDistFast := make([]float64, nHT)
	htDistSlow := make([]float64, nHT)
	htCross := make([]float64, nHT)
	htRSI := make([]float64, nHT)
	htATRPct := make([]float64, nHT)
	for k := 0; k < nHT; k++ {
		htDistFast[k] = (htClose[k] - emaFast[k]) / (emaFast[k] + 1e-10)
		htDistSlow[k] = (htClose[k] - emaSlow[k]) / (emaSlow[k] + 1e-10)
		htCross[k] = (emaFast[k] - emaSlow[k]) / (emaSlow[k] + 1e-10)
		htRSI[k] = rsiHT[k]
		if c := htClose[k]; c != 0 {
			htATRPct[k] = atrHT[k] / (c + 1e-10)
		}
	}

	// Forward-fill to 1m with shift(1) — bucket b reads HT[b-1].
	// Rows in the first HT bucket (b<=0) get NaN to match Python's warmup behavior.
	outDistFast := make([]float64, n)
	outDistSlow := make([]float64, n)
	outCross := make([]float64, n)
	outRSI := make([]float64, n)
	outATRPct := make([]float64, n)
	for i := 0; i < n; i++ {
		b := bucketIdx[i]
		if b <= 0 {
			outDistFast[i] = math.NaN()
			outDistSlow[i] = math.NaN()
			outCross[i] = math.NaN()
			outRSI[i] = math.NaN()
			outATRPct[i] = math.NaN()
			continue
		}
		prev := b - 1
		outDistFast[i] = htDistFast[prev]
		outDistSlow[i] = htDistSlow[prev]
		outCross[i] = htCross[prev]
		outRSI[i] = htRSI[prev]
		outATRPct[i] = htATRPct[prev]
	}

	df.AddSeries(dataframe.NewSeriesFloat64(names[0], nil, outDistFast), nil)
	df.AddSeries(dataframe.NewSeriesFloat64(names[1], nil, outDistSlow), nil)
	df.AddSeries(dataframe.NewSeriesFloat64(names[2], nil, outCross), nil)
	df.AddSeries(dataframe.NewSeriesFloat64(names[3], nil, outRSI), nil)
	df.AddSeries(dataframe.NewSeriesFloat64(names[4], nil, outATRPct), nil)
}

// ── Standalone array indicators (don't need a DataFrame) ─────────────────────
// Each mirrors the corresponding df-based helper exactly.

// emaArray: Python series.ewm(span=period, adjust=False).mean()
func emaArray(src []float64, period int) []float64 {
	n := len(src)
	out := make([]float64, n)
	if n == 0 {
		return out
	}
	alpha := 2.0 / float64(period+1)
	out[0] = src[0]
	for i := 1; i < n; i++ {
		out[i] = alpha*src[i] + (1-alpha)*out[i-1]
	}
	return out
}

// rsiArray: matches indicators/rsi.go RSI exactly (ewm span = 2/(period+1)).
func rsiArray(src []float64, period int) []float64 {
	n := len(src)
	out := make([]float64, n)
	if n < 2 {
		return out
	}
	alpha := 2.0 / float64(period+1)

	d := src[1] - src[0]
	avgGain := math.Max(d, 0)
	avgLoss := math.Max(-d, 0)
	for i := 1; i < n; i++ {
		if i >= 2 {
			d = src[i] - src[i-1]
			gain := math.Max(d, 0)
			loss := math.Max(-d, 0)
			avgGain = alpha*gain + (1-alpha)*avgGain
			avgLoss = alpha*loss + (1-alpha)*avgLoss
		}
		rs := avgGain / (avgLoss + 1e-10)
		out[i] = 100.0 - (100.0 / (1.0 + rs))
	}
	return out
}

// atrArray: rolling-mean ATR matching ATRSmoothed in atr.go.
// Returns 0 (not NaN) for warmup rows so derived features stay finite.
func atrArray(high, low, closeS []float64, period int) []float64 {
	n := len(closeS)
	tr := make([]float64, n)
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		hl := high[i] - low[i]
		if i == 0 {
			tr[i] = hl
			continue
		}
		hpc := math.Abs(high[i] - closeS[i-1])
		lpc := math.Abs(low[i] - closeS[i-1])
		tr[i] = math.Max(hl, math.Max(hpc, lpc))
	}
	for i := period - 1; i < n; i++ {
		var sum float64
		for j := i - period + 1; j <= i; j++ {
			sum += tr[j]
		}
		out[i] = sum / float64(period)
	}
	return out
}
