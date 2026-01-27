package dataframe

import (
	"hft/pkg/types"
	"time"

	"github.com/rocketlaunchr/dataframe-go"
)

func InitDataFrame() *dataframe.DataFrame {
	_o := dataframe.NewSeriesFloat64("open", nil)
	_h := dataframe.NewSeriesFloat64("high", nil)
	_l := dataframe.NewSeriesFloat64("low", nil)
	_c := dataframe.NewSeriesFloat64("close", nil)
	_timestamp := dataframe.NewSeriesTime("timestamp", nil)
	_time := dataframe.NewSeriesTime("time", nil)
	_volume := dataframe.NewSeriesFloat64("volume", nil)
	_dataFrame := dataframe.NewDataFrame(_o, _h, _l, _c, _volume, _timestamp, _time)
	return _dataFrame
}

func loadTicksToDataFrame(df *dataframe.DataFrame, ticks []types.Tick) {
	for _, tick := range ticks {
		df.Append(nil, map[string]interface{}{
			"open":      tick.Open,
			"high":      tick.High,
			"low":       tick.Low,
			"close":     tick.Close,
			"volume":    tick.Volume,
			"timestamp": tick.Timestamp,
		})
	}
}

func LoadHistoryBacktest(df *dataframe.DataFrame, data []types.Tick) {
	for _, tick := range data {
		_epoch := time.Unix(tick.Time, 0)

		df.Append(nil, map[string]interface{}{
			"open":      tick.Open,
			"high":      tick.High,
			"low":       tick.Low,
			"close":     tick.Close,
			"timestamp": tick.Timestamp,
			"volume":    tick.Volume,
			"time":      _epoch,
		})
	}
}
