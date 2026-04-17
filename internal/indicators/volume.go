package indicators

import (
	"math"

	"github.com/rocketlaunchr/dataframe-go"
)

/* Mirrors regime-model/features/indicators.py:add_volume_features.

   For instruments without volume (e.g. Nifty index, sum=0), all five
   columns are emitted as zeros so the feature dimensionality stays
   consistent with the model contract.
*/

// AddVolumeFeatures appends:
//   vol_sma_ratio, vol_roc, vol_price_corr, obv_slope, vwap_dist
func AddVolumeFeatures(df *dataframe.DataFrame) {
	n := df.NRows()
	zero := func(name string) {
		df.AddSeries(dataframe.NewSeriesFloat64(name, nil, make([]float64, n)), nil)
	}

	volIdx := FindIndexOf(df, "volume")
	if volIdx < 0 {
		zero("vol_sma_ratio")
		zero("vol_roc")
		zero("vol_price_corr")
		zero("obv_slope")
		zero("vwap_dist")
		return
	}
	vol := df.Series[volIdx].(*dataframe.SeriesFloat64).Values

	// Index data path: sum(vol)==0 → emit zero columns and return.
	sumV := 0.0
	for _, v := range vol {
		if !math.IsNaN(v) {
			sumV += v
		}
	}
	if sumV == 0 {
		zero("vol_sma_ratio")
		zero("vol_roc")
		zero("vol_price_corr")
		zero("obv_slope")
		zero("vwap_dist")
		return
	}

	closeS := df.Series[FindIndexOf(df, "close")].(*dataframe.SeriesFloat64).Values

	// 1) vol_sma_ratio = vol / SMA20(vol) - 1   (min_periods=1, expanding warmup)
	vSMARatio := make([]float64, n)
	{
		const w = 20
		var sum float64
		for i := 0; i < n; i++ {
			sum += vol[i]
			if i >= w {
				sum -= vol[i-w]
			}
			cnt := w
			if i+1 < w {
				cnt = i + 1
			}
			sma := sum / float64(cnt)
			vSMARatio[i] = vol[i]/(sma+1e-10) - 1.0
		}
	}

	// 2) vol_roc = (vol - vol[t-5]) / vol[t-5]  — NaN for rows [0,5)
	vROC := make([]float64, n)
	for i := 0; i < 5; i++ {
		vROC[i] = math.NaN()
	}
	for i := 5; i < n; i++ {
		vROC[i] = (vol[i] - vol[i-5]) / (vol[i-5] + 1e-10)
	}

	// 3) vol_price_corr: rolling20 corr(vol, |close.pct_change()|), min_periods=10
	absRet := make([]float64, n)
	absRet[0] = math.NaN()
	for i := 1; i < n; i++ {
		if closeS[i-1] != 0 {
			absRet[i] = math.Abs((closeS[i] - closeS[i-1]) / closeS[i-1])
		}
	}
	vCorr := rollingCorr(vol, absRet, 20, 10)

	// 4) obv_slope: (SMA10(OBV) - SMA10(OBV)[t-10]) / |SMA10(OBV)[t-10]|
	dir := make([]float64, n)
	for i := 1; i < n; i++ {
		d := closeS[i] - closeS[i-1]
		switch {
		case d > 0:
			dir[i] = 1
		case d < 0:
			dir[i] = -1
		}
	}
	obv := make([]float64, n)
	for i := 0; i < n; i++ {
		if i == 0 {
			obv[i] = vol[i] * dir[i]
		} else {
			obv[i] = obv[i-1] + vol[i]*dir[i]
		}
	}
	obvSMA := make([]float64, n)
	{
		const w = 10
		var sum float64
		for i := 0; i < n; i++ {
			sum += obv[i]
			if i >= w {
				sum -= obv[i-w]
			}
			cnt := w
			if i+1 < w {
				cnt = i + 1
			}
			obvSMA[i] = sum / float64(cnt)
		}
	}
	obvSlope := make([]float64, n)
	for i := 0; i < 10; i++ {
		obvSlope[i] = math.NaN()
	}
	for i := 10; i < n; i++ {
		prev := obvSMA[i-10]
		obvSlope[i] = (obvSMA[i] - prev) / (math.Abs(prev) + 1e-10)
	}

	// 5) vwap_dist: rolling30 VWAP, min_periods=1
	vwapDist := make([]float64, n)
	{
		const w = 30
		var numSum, denSum float64
		for i := 0; i < n; i++ {
			numSum += closeS[i] * vol[i]
			denSum += vol[i]
			if i >= w {
				numSum -= closeS[i-w] * vol[i-w]
				denSum -= vol[i-w]
			}
			vwap := numSum / (denSum + 1e-10)
			vwapDist[i] = (closeS[i] - vwap) / (vwap + 1e-10)
		}
	}

	df.AddSeries(dataframe.NewSeriesFloat64("vol_sma_ratio", nil, vSMARatio), nil)
	df.AddSeries(dataframe.NewSeriesFloat64("vol_roc", nil, vROC), nil)
	df.AddSeries(dataframe.NewSeriesFloat64("vol_price_corr", nil, vCorr), nil)
	df.AddSeries(dataframe.NewSeriesFloat64("obv_slope", nil, obvSlope), nil)
	df.AddSeries(dataframe.NewSeriesFloat64("vwap_dist", nil, vwapDist), nil)
}

// rollingCorr computes Pearson correlation over a sliding window of `window`
// bars. min is the minimum number of valid pairs required (matches pandas
// min_periods); for indices below that the result is NaN.
func rollingCorr(x, y []float64, window, minPeriods int) []float64 {
	n := len(x)
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		start := i - window + 1
		if start < 0 {
			start = 0
		}
		count := i - start + 1
		if count < minPeriods {
			out[i] = math.NaN()
			continue
		}
		var sumX, sumY float64
		for j := start; j <= i; j++ {
			sumX += x[j]
			sumY += y[j]
		}
		mx := sumX / float64(count)
		my := sumY / float64(count)
		var num, varX, varY float64
		for j := start; j <= i; j++ {
			dx := x[j] - mx
			dy := y[j] - my
			num += dx * dy
			varX += dx * dx
			varY += dy * dy
		}
		den := math.Sqrt(varX * varY)
		if den == 0 {
			continue
		}
		c := num / den
		if math.IsNaN(c) || math.IsInf(c, 0) {
			continue
		}
		out[i] = c
	}
	return out
}
