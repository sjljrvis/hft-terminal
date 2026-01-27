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

	for i := period - 1; i < length; i++ {
		sum := 0.0
		weightSum := 0.0

		for j := 0; j < period; j++ {
			weight := float64(j + 1) // Weight for the data point
			sum += _source.Values[i-j] * weight
			weightSum += weight
		}
		wma[i] = math.Round((sum/weightSum)*1000) / 1000
	}

	wma[0] = 0
	_wma := dataframe.NewSeriesFloat64(seriesname, nil, wma)
	df.AddSeries(_wma, nil)
}
