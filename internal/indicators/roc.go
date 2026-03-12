package indicators

import (
	"github.com/rocketlaunchr/dataframe-go"
)

func ROC(df *dataframe.DataFrame, seriesname string, source string, period int) {
	// Python: series.pct_change(period) * 100  — non-NaN from index `period` (not period+1)
	length := df.NRows()
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64).Values
	rocValues := make([]float64, length)

	for i := period; i < length; i++ {
		rocValues[i] = ((_source[i] - _source[i-period]) / _source[i-period]) * 100
	}
	_roc := dataframe.NewSeriesFloat64(seriesname, nil, rocValues)
	df.AddSeries(_roc, nil)
}
