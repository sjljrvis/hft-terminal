package strategy

import (
	"hft/internal/indicators"
	"hft/pkg/types"
	"log"
	"time"

	"github.com/rocketlaunchr/dataframe-go"
)

func Run(df *dataframe.DataFrame) {
	log.Println("running strategy")
	start := time.Now()
	indicators.OHLC4(df, "ohlc4")
	indicators.EMA(df, "ema_ohlc4", "ohlc4", 2)
	indicators.CCI(df, "fast_cci", "ema_ohlc4", 2)
	indicators.CCI(df, "slow_cci", "ema_ohlc4", 14)
	indicators.ATR(df, "tr", "close", 14)
	indicators.WMA(df, "wma_tr_2", "tr", 2)
	indicators.WMA(df, "wma_tr_400", "tr", 400)

	indicators.CalcX(df, "tempx", "close", 2, 2, "fast_cci", "wma_tr_2")
	indicators.CalcX(df, "tempx_base", "close", 0.1, 400, "slow_cci", "wma_tr_400")

	indicators.EMA(df, "ema_tempx", "tempx", 5)
	indicators.EMA(df, "ema_tempx_base", "tempx_base", 2)
	indicators.SMA(df, "sma_tempx", "tempx", 30)
	indicators.SMA(df, "sma_tempx_base", "tempx_base", 30)
	indicators.ATR(df, "atr_ema_tempx", "ema_tempx", 5)
	indicators.SMA(df, "sma_atr_ema_tempx", "atr_ema_tempx", 10)
	indicators.CalcSwap(df, "swap", "ema_tempx", "sma_tempx")
	indicators.CalcSwap(df, "swap_base", "ema_tempx_base", "sma_tempx_base")
	log.Println("time taken to calculate indicators", time.Since(start))
	// log.Println(df.Table())
}

func FindSignals(df *dataframe.DataFrame, current_position *types.Position, positions []*types.Position, events chan *types.Event) {
	_dataframe_length := df.NRows()
	_close := df.Series[indicators.FindIndexOf(df, "close")].(*dataframe.SeriesFloat64).Values
	_timestamp := df.Series[indicators.FindIndexOf(df, "timestamp")].(*dataframe.SeriesTime).Values
	_swap := df.Series[indicators.FindIndexOf(df, "swap")].(*dataframe.SeriesFloat64).Values
	_swap_base := df.Series[indicators.FindIndexOf(df, "swap_base")].(*dataframe.SeriesFloat64).Values

	for i := 0; i < _dataframe_length; i++ {
		_buy_condition := _swap[i] == 1 && _swap_base[i] == 1
		_sell_condition := _swap[i] == -1 && _swap_base[i] == -1

		if current_position.Kind == "SELL" && _buy_condition {
			current_position.Exit(_close[i], *_timestamp[i])
			current_position.Reset()
			events <- &types.Event{
				Kind:       "BUY",
				Type:       "EXIT",
				EntryPrice: _close[i],
				Timestamp:  *_timestamp[i],
			}
		}
		if current_position.Kind == "BUY" && _sell_condition {
			current_position.Exit(_close[i], *_timestamp[i])
			current_position.Reset()
			events <- &types.Event{
				Kind:       "SELL",
				Type:       "EXIT",
				EntryPrice: _close[i],
				Timestamp:  *_timestamp[i],
			}
		}

		if current_position.Kind == "" && _buy_condition {
			current_position.Buy(_close[i], *_timestamp[i])
			events <- &types.Event{
				Kind:       "BUY",
				Type:       "ENTRY",
				EntryPrice: _close[i],
				Timestamp:  *_timestamp[i],
			}
		}
		if current_position.Kind == "" && _sell_condition {
			current_position.Sell(_close[i], *_timestamp[i])
			events <- &types.Event{
				Kind:       "SELL",
				Type:       "ENTRY",
				EntryPrice: _close[i],
				Timestamp:  *_timestamp[i],
			}
		}
	}
}
