package indicators

import (
	"math"

	ta "github.com/cinar/indicator"
	"github.com/rocketlaunchr/dataframe-go"
)

// EMA placeholder for exponential moving average indicator.
func EMA(df *dataframe.DataFrame, seriesname string, source string, period int) {
	length := df.NRows()
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64)

	result := ta.Ema(period, _source.Values)
	for i := 0; i < length; i++ {
		// result[i] = math.Round(result[i]*100) / 100
		result[i] = math.Ceil(result[i])
	}
	_ema := dataframe.NewSeriesFloat64(seriesname, nil, result)
	df.AddSeries(_ema, nil)
}

func SMA(df *dataframe.DataFrame, seriesname string, source string, period int) {
	length := df.NRows()
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64)

	result := ta.Sma(period, _source.Values)
	for i := 0; i < length; i++ {
		result[i] = math.Round(result[i]*100) / 100
	}
	_sma := dataframe.NewSeriesFloat64(seriesname, nil, result)
	df.AddSeries(_sma, nil)
}
