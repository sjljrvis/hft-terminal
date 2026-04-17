package indicators

import (
	"math"

	"github.com/rocketlaunchr/dataframe-go"
)

func ROC(df *dataframe.DataFrame, seriesname string, source string, period int) {
	// Python: series.pct_change(period) * 100  — NaN for rows [0, period), valid from index `period`
	length := df.NRows()
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64).Values
	rocValues := make([]float64, length)

	for i := 0; i < period; i++ {
		rocValues[i] = math.NaN()
	}
	for i := period; i < length; i++ {
		rocValues[i] = ((_source[i] - _source[i-period]) / _source[i-period]) * 100
	}
	_roc := dataframe.NewSeriesFloat64(seriesname, nil, rocValues)
	df.AddSeries(_roc, nil)
}
