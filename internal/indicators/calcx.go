package indicators

import (
	"math"
	"time"

	ta "github.com/cinar/indicator"
	"github.com/rocketlaunchr/dataframe-go"
)

func CalcX(df *dataframe.DataFrame, seriesname string, source string, param float64, atrPeriod int, cci_source string, wma_source string) {
	_cci := df.Series[FindIndexOf(df, cci_source)].(*dataframe.SeriesFloat64).Values
	_wma := df.Series[FindIndexOf(df, wma_source)].(*dataframe.SeriesFloat64).Values
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64).Values

	_length := df.NRows()

	bufferDn := make([]float64, _length)
	bufferUp := make([]float64, _length)

	bufferDn[0] = _source[0]
	bufferUp[0] = _source[0]
	_x := make([]float64, _length)

	// Seed first value to avoid default zero and stay close to source.
	_x[0] = math.Round(_source[0]*1000) / 1000

	for i := 0; i < _length; i++ {
		bufferDn[i] = _source[i] + param*_wma[i]
		bufferUp[i] = _source[i] - param*_wma[i]
	}

	for i := 1; i < _length; i++ {
		if _cci[i] >= 0 && _cci[i-1] < 0 {
			bufferUp[i] = bufferDn[i-1]
		}

		if _cci[i] <= 0 && _cci[i-1] > 0 {
			bufferDn[i] = bufferUp[i-1]
		}

		if _cci[i] >= 0 {
			if bufferUp[i] < bufferUp[i-1] {
				bufferUp[i] = bufferUp[i-1]
			}
		} else {
			if _cci[i] <= 0 {
				if bufferDn[i] > bufferDn[i-1] {
					bufferDn[i] = bufferDn[i-1]
				}
			}
		}

		_x[i] = func() float64 {
			if _cci[i] >= 0 {
				return math.Round((bufferUp[i])*1000) / 1000
			} else {
				if _cci[i] <= 0 {
					return math.Round((bufferDn[i])*1000) / 1000
				} else {
					return math.Round((_x[i-1])*1000) / 1000
				}
			}
		}()
	}

	_xSeries := dataframe.NewSeriesFloat64(seriesname, nil, _x)
	df.AddSeries(_xSeries, nil)

}

// calcswap(param, deep) =>
//     swap = 0.0
//     direction = 'na'
//     _pr = 0.0
//     _pr_1 = 0.0

//     float avg_param = ta.sma(param, 30)

//     swap := param > avg_param and check_atr_expansion ? 1 : param < avg_param and check_atr_expansion ? -1 : swap[1]

//     _pr := math.round((param*1000))/1000
//     _pr_1 :=  math.round((param[1]*1000))/1000
//     _atr_ema = ta.sma(ta.atr(5), 10)
//     // _atr_ema = ta.atr(2)
//     if(math.abs(_pr - _pr_1[1]) < _atr_ema)
//         swap := swap[1]
//     if(not isActiveSession)
//         swap := 0
//     swap

func IsActiveSession(ts *time.Time) bool {
	// Trading hours in IST: 09:20 to 15:20 inclusive.
	if ts == nil || ts.IsZero() {
		return false
	}

	ist := time.FixedZone("IST", 19800) // UTC+5:30
	t := ts.In(ist)
	mins := t.Hour()*60 + t.Minute()
	start := 9*60 + 17 // 09:20
	end := 15*60 + 25  // 15:20
	return mins >= start && mins <= end
}

func CalcSwap(df *dataframe.DataFrame, seriesname string, source string, avg_param_source string) {
	_length := df.NRows()
	_time := df.Series[FindIndexOf(df, "timestamp")].(*dataframe.SeriesTime).Values
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64).Values
	_avg_param := df.Series[FindIndexOf(df, avg_param_source)].(*dataframe.SeriesFloat64).Values
	_high := df.Series[FindIndexOf(df, "high")].(*dataframe.SeriesFloat64).Values
	_low := df.Series[FindIndexOf(df, "low")].(*dataframe.SeriesFloat64).Values
	_close := df.Series[FindIndexOf(df, "close")].(*dataframe.SeriesFloat64).Values
	// _open := df.Series[FindIndexOf(df, "open")].(*dataframe.SeriesFloat64).Values

	// Pine parity: check ATR expansion on price bars and use ATR(5) smoothed by SMA(10) as the
	// "no change" threshold. ATR library returns a slice per price input; only the first slice is needed.
	atr3, _ := ta.Atr(3, _high, _low, _close)
	atr5, _ := ta.Atr(5, _high, _low, _close)
	atr10, _ := ta.Atr(10, _high, _low, _close)
	atrEma := ta.Sma(10, atr5)

	swap := make([]float64, _length)
	swap[0] = 0

	for i := 1; i < _length; i++ {
		swap[i] = swap[i-1]

		if i >= 2 {
			checkAtrExpansion := (atr3[i] + math.Abs(_close[i-2]-_close[i])) > atr10[i]

			if checkAtrExpansion {
				if _source[i] > _avg_param[i] {
					swap[i] = 1
				} else if _source[i] < _avg_param[i] {
					swap[i] = -1
				}
			}

			// // Pine compares current rounded value with the value two bars back.
			_pr := math.Round((_source[i] * 1000)) / 1000
			_pr_2 := math.Round((_source[i-2] * 1000)) / 1000

			if math.Abs(_pr-_pr_2) < atrEma[i] {
				swap[i] = swap[i-1]
			}

			// if math.Abs(_source[i]-_source[i-1]) < 0.3*atr3[i] {
			// 	swap[i] = swap[i-1]
			// }

		} else {
			// Early bars: fall back to simple comparison when insufficient history.
			if _source[i] > _avg_param[i] {
				swap[i] = 1
			} else if _source[i] < _avg_param[i] {
				swap[i] = -1
			}
		}

		if !IsActiveSession(_time[i]) {
			swap[i] = 0
		}
	}

	_swapSeries := dataframe.NewSeriesFloat64(seriesname, nil, swap)
	df.AddSeries(_swapSeries, nil)
}
