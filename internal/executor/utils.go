package executor

import (
	"fmt"
	"hft/internal/indicators"
)

func ToJSON() []map[string]interface{} {
	if Instance == nil || Instance.DF == nil {
		return []map[string]interface{}{}
	}
	_json := make([]map[string]interface{}, Instance.DF.NRows())
	_data_frame := Instance.DF

	fmt.Println(_data_frame.NRows())
	for i := 0; i < _data_frame.NRows(); i++ {
		_json[i] = map[string]interface{}{
			"open":       _data_frame.Series[indicators.FindIndexOf(_data_frame, "open")].Value(i),
			"high":       _data_frame.Series[indicators.FindIndexOf(_data_frame, "high")].Value(i),
			"low":        _data_frame.Series[indicators.FindIndexOf(_data_frame, "low")].Value(i),
			"close":      _data_frame.Series[indicators.FindIndexOf(_data_frame, "close")].Value(i),
			"time":       _data_frame.Series[indicators.FindIndexOf(_data_frame, "time")].Value(i),
			"timestamp":  _data_frame.Series[indicators.FindIndexOf(_data_frame, "timestamp")].Value(i),
			"fast_tempx": _data_frame.Series[indicators.FindIndexOf(_data_frame, "fast_tempx_kalman")].Value(i),
			"slow_tempx": _data_frame.Series[indicators.FindIndexOf(_data_frame, "slow_tempx_kalman")].Value(i),
			"swap":       _data_frame.Series[indicators.FindIndexOf(_data_frame, "swap")].Value(i),
			"swap_base":  _data_frame.Series[indicators.FindIndexOf(_data_frame, "swap_base")].Value(i),
		}
	}
	return _json
}

// PositionToJSON returns the current open position (or nil if flat).
func PositionToJSON() map[string]interface{} {
	if Instance == nil || pendingPosition == nil {
		return nil
	}
	return map[string]interface{}{
		"kind":       pendingPosition.Kind,
		"entryPrice": pendingPosition.EntryPrice,
		"entryTime":  pendingPosition.EntryTime,
		"peakProfit": pendingPosition.PeakProfit,
		"peakLoss":   pendingPosition.PeakLoss,
	}
}

// StatsToJSON returns the current executor statistics.
func StatsToJSON() map[string]interface{} {
	if stats == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"totalTrades":       stats.TotalTrades,
		"winningTrades":     stats.WinningTrades,
		"losingTrades":      stats.LosingTrades,
		"breakevenTrades":   stats.BreakevenTrades,
		"netProfit":         stats.NetProfit,
		"grossProfit":       stats.GrossProfit,
		"grossLoss":         stats.GrossLoss,
		"maxDrawdown":       stats.MaxDrawdown,
		"maxProfit":         stats.MaxProfit,
		"peakProfit":        stats.PeakProfit,
		"profitTargetExits": stats.ProfitTargetExits,
		"stopLossExits":     stats.StopLossExits,
		"trailingStopExits": stats.TrailingStopExits,
		"signalExits":       stats.SignalExits,
		"expectancyRatio":   stats.ExpectancyRatio,
	}
}

func TradesToJSON() []map[string]interface{} {
	if Instance == nil || Instance.TradeDF == nil {
		return []map[string]interface{}{}
	}
	tradeDF := Instance.TradeDF
	nRows := tradeDF.NRows()
	_json := make([]map[string]interface{}, nRows)
	for i := 0; i < nRows; i++ {
		_json[i] = map[string]interface{}{
			"entryPrice": tradeDF.Series[indicators.FindIndexOf(tradeDF, "entryPrice")].Value(i),
			"exitPrice":  tradeDF.Series[indicators.FindIndexOf(tradeDF, "exitPrice")].Value(i),
			"entryTime":  tradeDF.Series[indicators.FindIndexOf(tradeDF, "entryTime")].Value(i),
			"exitTime":   tradeDF.Series[indicators.FindIndexOf(tradeDF, "exitTime")].Value(i),
			"profit":     tradeDF.Series[indicators.FindIndexOf(tradeDF, "profit")].Value(i),
			"profitPct":  tradeDF.Series[indicators.FindIndexOf(tradeDF, "profitPct")].Value(i),
			"type":       tradeDF.Series[indicators.FindIndexOf(tradeDF, "type")].Value(i),
			"peakProfit": tradeDF.Series[indicators.FindIndexOf(tradeDF, "peakProfit")].Value(i),
			"peakLoss":   tradeDF.Series[indicators.FindIndexOf(tradeDF, "peakLoss")].Value(i),
			"reason":     tradeDF.Series[indicators.FindIndexOf(tradeDF, "reason")].Value(i),
		}
	}
	return _json
}
