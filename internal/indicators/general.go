package indicators

import (
	"math"

	"github.com/rocketlaunchr/dataframe-go"
	"github.com/samber/lo"
)

func FindIndexOf(df *dataframe.DataFrame, key string) int {
	_, index, _ := lo.FindIndexOf(df.Names(), func(i string) bool {
		return i == key
	})
	return index
}

func OHLC4(df *dataframe.DataFrame, seriesname string) {
	length := df.NRows()
	ohlc4Values := make([]float64, length)

	_open := df.Series[FindIndexOf(df, "open")].(*dataframe.SeriesFloat64).Values
	_high := df.Series[FindIndexOf(df, "high")].(*dataframe.SeriesFloat64).Values
	_low := df.Series[FindIndexOf(df, "low")].(*dataframe.SeriesFloat64).Values
	_close := df.Series[FindIndexOf(df, "close")].(*dataframe.SeriesFloat64).Values

	for i := 0; i < length; i++ {
		ohlc4Values[i] = math.Round(((_open[i]+_high[i]+_low[i]+_close[i])/4)*100) / 100
	}

	_ohlc4 := dataframe.NewSeriesFloat64(seriesname, nil, ohlc4Values)
	df.AddSeries(_ohlc4, nil)
}

func WMA(df *dataframe.DataFrame, seriesname string, source string, period int) {
	length := df.NRows()
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64)

	wma := make([]float64, length)

	// Warmup rows stay 0.0 (not NaN) because WMA feeds into CalcX → EMA chain,
	// and Go's EMA propagates NaN forever (unlike Python's ewm which skips NaN).
	for i := period - 1; i < length; i++ {
		sum := 0.0
		weightSum := 0.0

		for j := 0; j < period; j++ {
			weight := float64(j + 1) // Weight for the data point
			sum += _source.Values[i-j] * weight
			weightSum += weight
		}
		wma[i] = sum / weightSum
	}
	_wma := dataframe.NewSeriesFloat64(seriesname, nil, wma)
	df.AddSeries(_wma, nil)
}

// bodySize    = math.abs(close - open)
// avgBody     = ta.sma(bodySize, lookback)
// isWideRange = bodySize > (avgBody * 1.2)

func BodySize(df *dataframe.DataFrame, seriesname string) {
	length := df.NRows()
	_close := df.Series[FindIndexOf(df, "close")].(*dataframe.SeriesFloat64).Values
	_open := df.Series[FindIndexOf(df, "open")].(*dataframe.SeriesFloat64).Values
	bodySize := make([]float64, length)

	for i := 0; i < length; i++ {
		bodySize[i] = math.Abs(_close[i] - _open[i])
	}

	_bodySize := dataframe.NewSeriesFloat64(seriesname, nil, bodySize)
	df.AddSeries(_bodySize, nil)
}

// Mean Reversion Check
// stdDev      = ta.stdev(close, ema_fast_len)
// isOverextended = math.abs(close - ta.ema(close, ema_fast_len)) > (stdDev * 2)
func StandardDeviation(df *dataframe.DataFrame, seriesname string, source string, period int) {

	// Standard deviation of source series from close price
	length := df.NRows()
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64).Values

	standardDeviation := make([]float64, length)

	for i := period - 1; i < length; i++ {
		sum := 0.0
		for j := 0; j < period; j++ {
			sum += _source[i-j]
		}
		mean := sum / float64(period)
		for j := 0; j < period; j++ {
			standardDeviation[i] += math.Pow(_source[i-j]-mean, 2)
		}
		standardDeviation[i] = math.Sqrt(standardDeviation[i] / float64(period))
	}

	_standardDeviation := dataframe.NewSeriesFloat64(seriesname, nil, standardDeviation)
	df.AddSeries(_standardDeviation, nil)
}

func Slope(df *dataframe.DataFrame, seriesname string, source string, period int) {
	// Python: series.diff(period) / period  — NaN for rows [0, period), valid from index `period`
	length := df.NRows()
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64).Values
	slope := make([]float64, length)

	for i := 0; i < period; i++ {
		slope[i] = math.NaN()
	}
	for i := period; i < length; i++ {
		slope[i] = (_source[i] - _source[i-period]) / float64(period)
	}
	_slope := dataframe.NewSeriesFloat64(seriesname, nil, slope)
	df.AddSeries(_slope, nil)
}

func PriceDistance(df *dataframe.DataFrame, seriesname string, source string, source2 string) {
	// python: (source - source2) / (source2 + 1e-10)  — no rounding, values are tiny (~1e-4)
	length := df.NRows()
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64).Values
	_source2 := df.Series[FindIndexOf(df, source2)].(*dataframe.SeriesFloat64).Values
	priceDistance := make([]float64, length)

	for i := 0; i < length; i++ {
		priceDistance[i] = (_source[i] - _source2[i]) / (_source2[i] + 1e-10)
	}

	_priceDistance := dataframe.NewSeriesFloat64(seriesname, nil, priceDistance)
	df.AddSeries(_priceDistance, nil)
}

// python:   df["log_ret"] = np.log(df["close"] / df["close"].shift(1))
func LogReturn(df *dataframe.DataFrame, seriesname string, source string, shift int) {
	length := df.NRows()
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64).Values
	logReturn := make([]float64, length)

	for i := 0; i < shift; i++ {
		logReturn[i] = math.NaN()
	}
	for i := shift; i < length; i++ {
		logReturn[i] = math.Log(_source[i] / _source[i-shift])
	}
	_logReturn := dataframe.NewSeriesFloat64(seriesname, nil, logReturn)
	df.AddSeries(_logReturn, nil)
}

func CalculateRollingStd(logRet []float64, window int) []float64 {
	n := len(logRet)
	result := make([]float64, n)

	for i := 0; i < window-1; i++ {
		result[i] = math.NaN()
	}
	for i := window - 1; i < n; i++ {
		start := i - window + 1
		sum := 0.0

		for j := start; j <= i; j++ {
			sum += logRet[j]
		}

		mean := sum / float64(window)

		variance := 0.0
		for j := start; j <= i; j++ {
			diff := logRet[j] - mean
			variance += diff * diff
		}

		// Python rolling().std() uses ddof=1 (sample std, divide by n-1)
		result[i] = math.Sqrt(variance / float64(window-1))
	}

	return result
}

func RollingStd(df *dataframe.DataFrame, seriesname string, source string, period int) {
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64).Values
	rollingStd := CalculateRollingStd(_source, period)
	_rollingStd := dataframe.NewSeriesFloat64(seriesname, nil, rollingStd)
	df.AddSeries(_rollingStd, nil)
}

// python: df["hl_range_pct"] = (df["high"] - df["low"]) / (df["close"] + 1e-10)  — no rounding
func HLRangePct(df *dataframe.DataFrame, seriesname string) {
	length := df.NRows()
	_high := df.Series[FindIndexOf(df, "high")].(*dataframe.SeriesFloat64).Values
	_low := df.Series[FindIndexOf(df, "low")].(*dataframe.SeriesFloat64).Values
	_close := df.Series[FindIndexOf(df, "close")].(*dataframe.SeriesFloat64).Values
	hlRangePct := make([]float64, length)

	for i := 0; i < length; i++ {
		hlRangePct[i] = (_high[i] - _low[i]) / (_close[i] + 1e-10)
	}

	_hlRangePct := dataframe.NewSeriesFloat64(seriesname, nil, hlRangePct)
	df.AddSeries(_hlRangePct, nil)
}

func CalculateVolExpansion(rollingStd []float64, window int) []float64 {

	n := len(rollingStd)
	result := make([]float64, n)

	for i := 0; i < window-1; i++ {
		result[i] = math.NaN()
	}
	for i := window - 1; i < n; i++ {
		start := i - window + 1
		sum := 0.0

		for j := start; j <= i; j++ {
			sum += rollingStd[j]
		}

		mean := sum / float64(window)

		result[i] = rollingStd[i] / (mean + 1e-10)
	}

	return result
}

// python: df["vol_expansion"] = df["rolling_std"] / (df["rolling_std"].rolling(60).mean() + 1e-10)
func VolExpansion(df *dataframe.DataFrame, seriesname string, source string, period int) {
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64).Values
	volExpansion := CalculateVolExpansion(_source, period)
	_volExpansion := dataframe.NewSeriesFloat64(seriesname, nil, volExpansion)
	df.AddSeries(_volExpansion, nil)
}
