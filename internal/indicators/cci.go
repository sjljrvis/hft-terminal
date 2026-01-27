package indicators

import (
	"math"

	ta "github.com/cinar/indicator"
	"github.com/rocketlaunchr/dataframe-go"
)

func CCI(df *dataframe.DataFrame, seriesname string, source string, period int) {
	length := df.NRows()
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64)
	result := ta.CommunityChannelIndex(period, _source.Values, _source.Values, _source.Values)
	for i := 0; i < length; i++ {
		result[i] = math.Round(result[i]*100) / 100
	}
	result[0] = result[1]
	_cci := dataframe.NewSeriesFloat64(seriesname, nil, result)
	df.AddSeries(_cci, nil)
}

func calculateSMA(period int, data []float64) []float64 {
	sma := make([]float64, len(data))

	for i := period - 1; i < len(data); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += data[j]
		}
		sma[i] = sum / float64(period)
	}

	return sma
}

func calculateMeanDeviation(prices []float64, sma []float64, period int) []float64 {
	meanDeviation := make([]float64, len(prices))

	for i := period - 1; i < len(prices); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += math.Abs(prices[j] - sma[i])
		}
		meanDeviation[i] = sum / float64(period)
	}

	return meanDeviation
}

func CalculateCCI(df *dataframe.DataFrame, seriesname string, period int) {
	length := df.NRows()
	cciValues := make([]float64, length)

	_open := df.Series[FindIndexOf(df, "open")].(*dataframe.SeriesFloat64).Values
	_high := df.Series[FindIndexOf(df, "high")].(*dataframe.SeriesFloat64).Values
	_low := df.Series[FindIndexOf(df, "low")].(*dataframe.SeriesFloat64).Values
	_close := df.Series[FindIndexOf(df, "close")].(*dataframe.SeriesFloat64).Values

	typicalPrice := make([]float64, length)

	for i := 0; i < length; i++ {
		// typicalPrice[i] = (_high[i] + _low[i] + _close[i]) / float64(3)
		typicalPrice[i] = (_high[i] + 1*_low[i] + 1*_close[i] + _open[i]) / float64(4)

	}
	ma := calculateSMA(period, typicalPrice)
	meanDeviation := calculateMeanDeviation(typicalPrice, ma, period)
	for i := period - 1; i < length; i++ {
		if meanDeviation[i] == 0 {
			meanDeviation[i] = meanDeviation[i-1]
		}
		cci := (typicalPrice[i] - ma[i]) / (0.015 * meanDeviation[i])
		cciValues[i] = math.Round(cci*1000) / 1000
		if cciValues[i] == 0 {
			cciValues[i] = cciValues[i-1]
		}
	}

	_cci := dataframe.NewSeriesFloat64(seriesname, nil, cciValues)
	df.AddSeries(_cci, nil)

}
