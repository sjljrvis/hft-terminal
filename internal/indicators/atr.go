package indicators

import (
	"math"

	ta "github.com/cinar/indicator"
	"github.com/rocketlaunchr/dataframe-go"
)

// ATR computes two series from ta.Atr:
//   - seriesname        → TR  (instantaneous true range, first return)
//   - seriesname+"_atr" → ATR (rolling-smoothed average, second return)
//
// Use seriesname for raw TR columns (e.g. "tr", "atr3", "atr3_base").
// For the Python-equivalent smoothed ATR (rolling mean of TR), call ATRSmoothed.
func ATR(df *dataframe.DataFrame, seriesname string, source string, period int) {
	_high := df.Series[FindIndexOf(df, "high")].(*dataframe.SeriesFloat64)
	_low := df.Series[FindIndexOf(df, "low")].(*dataframe.SeriesFloat64)
	_close := df.Series[FindIndexOf(df, "close")].(*dataframe.SeriesFloat64)

	tr, _ := ta.Atr(period, _high.Values, _low.Values, _close.Values)

	_tr := dataframe.NewSeriesFloat64(seriesname, nil, tr)
	df.AddSeries(_tr, nil)
}

// ATRSmoothed computes the rolling-mean ATR matching Python's compute_atr:
//
//	tr = max(high-low, |high-prevClose|, |low-prevClose|)
//	atr = tr.rolling(period).mean()
//
// Use this for model input features (e.g. "atr_computed").
func ATRSmoothed(df *dataframe.DataFrame, seriesname string, period int) {
	_high := df.Series[FindIndexOf(df, "high")].(*dataframe.SeriesFloat64).Values
	_low := df.Series[FindIndexOf(df, "low")].(*dataframe.SeriesFloat64).Values
	_close := df.Series[FindIndexOf(df, "close")].(*dataframe.SeriesFloat64).Values
	n := len(_close)
	tr := make([]float64, n)
	atr := make([]float64, n)

	for i := 0; i < n; i++ {
		hl := _high[i] - _low[i]
		if i == 0 {
			tr[i] = hl
		} else {
			hpc := math.Abs(_high[i] - _close[i-1])
			lpc := math.Abs(_low[i] - _close[i-1])
			tr[i] = math.Max(hl, math.Max(hpc, lpc))
		}
	}

	// rolling mean of TR over `period` bars (matches pandas tr.rolling(period).mean())
	for i := 0; i < n; i++ {
		if i < period-1 {
			atr[i] = math.NaN()
			continue
		}
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += tr[j]
		}
		atr[i] = sum / float64(period)
	}

	df.AddSeries(dataframe.NewSeriesFloat64(seriesname, nil, atr), nil)
}
