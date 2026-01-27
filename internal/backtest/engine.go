package backtest

import (
	"context"
	"log"
	"time"

	"hft/internal/dataframe"
	"hft/internal/indicators"
	"hft/internal/storage/sqlite"
	"hft/internal/strategy"
	"hft/pkg/types"

	_df_ "github.com/rocketlaunchr/dataframe-go"
)

type Backtest struct {
	DF        *_df_.DataFrame
	Position  *types.Position
	Positions []*types.Position
	Events    chan *types.Event
}

var Instance *Backtest

func Reset() {
	Instance = &Backtest{
		DF: dataframe.InitDataFrame(),
		Position: &types.Position{
			ID:         "",
			Kind:       "",
			Type:       "",
			EntryPrice: 0,
			Quantity:   0,
			EntryTime:  time.Now(),
			ExitPrice:  0,
			ExitTime:   time.Now(),
		},
	}
}

func InitBacktest() {
	Instance = &Backtest{
		DF:        dataframe.InitDataFrame(),
		Events:    make(chan *types.Event),
		Positions: make([]*types.Position, 0),
		Position: &types.Position{
			ID:         "",
			Kind:       "",
			Type:       "",
			EntryPrice: 0,
			Quantity:   0,
			EntryTime:  time.Now(),
			ExitPrice:  0,
			ExitTime:   time.Now(),
		},
	}
}

func SubscribeSignals() {
	for {
		event := <-Instance.Events
		log.Printf("backtest: received event: %v", event)
	}
}

// Run executes a minimal backtest pass: load all ticks and report count.
func Run() {
	log.Println("backtest: starting")
	startDate := "2026-01-15"
	endDate := "2026-01-25"
	InitBacktest()

	go SubscribeSignals()
	ctx := context.Background()
	df := Instance.DF
	db := sqlite.DefaultDB()
	if db == nil {
		log.Printf("backtest: db not initialized")
		return
	}

	ticks, err := db.Ticks.ListTicksFiltered(ctx, "", "", 0, startDate, endDate) // all rows
	if err != nil {
		log.Printf("backtest: load ticks: %v", err)
		return
	}

	log.Printf("backtest: loaded %d ticks", len(ticks))
	dataframe.LoadHistoryBacktest(df, ticks)
	strategy.Run(df)
	strategy.FindSignals(df, Instance.Position, Instance.Positions, Instance.Events)
}

func ToJSON() []map[string]interface{} {
	_json := make([]map[string]interface{}, Instance.DF.NRows())
	_data_frame := Instance.DF
	for i := 0; i < _data_frame.NRows(); i++ {
		_json[i] = map[string]interface{}{
			"open":       _data_frame.Series[indicators.FindIndexOf(_data_frame, "open")].Value(i),
			"high":       _data_frame.Series[indicators.FindIndexOf(_data_frame, "high")].Value(i),
			"low":        _data_frame.Series[indicators.FindIndexOf(_data_frame, "low")].Value(i),
			"close":      _data_frame.Series[indicators.FindIndexOf(_data_frame, "close")].Value(i),
			"time":       _data_frame.Series[indicators.FindIndexOf(_data_frame, "time")].Value(i),
			"timestamp":  _data_frame.Series[indicators.FindIndexOf(_data_frame, "timestamp")].Value(i),
			"swap":       _data_frame.Series[indicators.FindIndexOf(_data_frame, "swap")].Value(i),
			"swap_base":  _data_frame.Series[indicators.FindIndexOf(_data_frame, "swap_base")].Value(i),
			"tempx":      _data_frame.Series[indicators.FindIndexOf(_data_frame, "ema_tempx")].Value(i),
			"tempx_base": _data_frame.Series[indicators.FindIndexOf(_data_frame, "ema_tempx_base")].Value(i),
		}
	}
	return _json
}
