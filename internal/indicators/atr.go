package indicators

import (
	ta "github.com/cinar/indicator"
	"github.com/rocketlaunchr/dataframe-go"
)

// ATR placeholder for average true range indicator.
func ATR(df *dataframe.DataFrame, seriesname string, source string, period int) {
	_high := df.Series[FindIndexOf(df, "high")].(*dataframe.SeriesFloat64)
	_low := df.Series[FindIndexOf(df, "low")].(*dataframe.SeriesFloat64)
	_close := df.Series[FindIndexOf(df, "close")].(*dataframe.SeriesFloat64)

	tr, _ := ta.Atr(period, _high.Values, _low.Values, _close.Values)

	_tr := dataframe.NewSeriesFloat64(seriesname, nil, tr)
	df.AddSeries(_tr, nil)
}
