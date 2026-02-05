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

func InitTradeDataFrame() *dataframe.DataFrame {
	// entry price, exit price, entry time, exit time, profit
	_entryPrice := dataframe.NewSeriesFloat64("entryPrice", nil)
	_exitPrice := dataframe.NewSeriesFloat64("exitPrice", nil)
	_entryTime := dataframe.NewSeriesTime("entryTime", nil)
	_exitTime := dataframe.NewSeriesTime("exitTime", nil)
	_profit := dataframe.NewSeriesFloat64("profit", nil)
	_profitPct := dataframe.NewSeriesFloat64("profitPct", nil)
	_type := dataframe.NewSeriesString("type", nil)
	_reason := dataframe.NewSeriesString("reason", nil)
	_peakProfit := dataframe.NewSeriesFloat64("peakProfit", nil)
	_peakLoss := dataframe.NewSeriesFloat64("peakLoss", nil)
	_dataFrame := dataframe.NewDataFrame(_entryPrice, _exitPrice, _entryTime, _exitTime, _profit, _profitPct, _type, _reason, _peakProfit, _peakLoss)
	return _dataFrame
}

// AppendTrade adds a completed trade to the trade dataframe
func AppendTrade(df *dataframe.DataFrame, entryPrice, exitPrice float64, entryTime, exitTime time.Time, profit, profitPct float64, tradeType, reason string, peakProfit, peakLoss float64) {
	df.Append(nil, map[string]interface{}{
		"entryPrice": entryPrice,
		"exitPrice":  exitPrice,
		"entryTime":  entryTime,
		"exitTime":   exitTime,
		"profit":     profit,
		"profitPct":  profitPct,
		"type":       tradeType,
		"reason":     reason,
		"peakProfit": peakProfit,
		"peakLoss":   peakLoss,
	})
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
