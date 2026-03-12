package indicators

import (
	"math"

	"github.com/rocketlaunchr/dataframe-go"
)

// RSI matches Python's formula exactly:
//
//	delta = series.diff()
//	gain  = delta.where(delta > 0, 0.0)
//	loss  = -delta.where(delta < 0, 0.0)
//	avg_gain = gain.ewm(span=period, adjust=False).mean()   alpha = 2/(period+1)
//	avg_loss = loss.ewm(span=period, adjust=False).mean()
//	rsi = 100 - 100/(1 + avg_gain/(avg_loss+1e-10))
//
// Note: this is NOT Wilder's RSI (alpha=1/period). Python uses ewm span, so alpha=2/(period+1).
func RSI(df *dataframe.DataFrame, seriesname string, source string, period int) {
	length := df.NRows()
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64).Values

	alpha := 2.0 / float64(period+1)
	rsiValues := make([]float64, length)

	if length < 2 {
		_rsi := dataframe.NewSeriesFloat64(seriesname, nil, rsiValues)
		df.AddSeries(_rsi, nil)
		return
	}

	// Compute delta[i] = source[i] - source[i-1] (delta[0] = NaN → seed EWM with 0)
	avgGain := 0.0
	avgLoss := 0.0

	// Seed: first delta is source[1]-source[0]
	d := _source[1] - _source[0]
	gain := math.Max(d, 0)
	loss := math.Max(-d, 0)
	avgGain = gain
	avgLoss = loss

	for i := 1; i < length; i++ {
		if i >= 2 {
			d = _source[i] - _source[i-1]
			gain = math.Max(d, 0)
			loss = math.Max(-d, 0)
			avgGain = alpha*gain + (1-alpha)*avgGain
			avgLoss = alpha*loss + (1-alpha)*avgLoss
		}
		rs := avgGain / (avgLoss + 1e-10)
		rsiValues[i] = 100.0 - (100.0 / (1.0 + rs))
	}

	_rsi := dataframe.NewSeriesFloat64(seriesname, nil, rsiValues)
	df.AddSeries(_rsi, nil)
}
