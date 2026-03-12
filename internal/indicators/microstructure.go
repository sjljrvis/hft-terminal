package indicators

import (
	"fmt"
	"math"

	"github.com/rocketlaunchr/dataframe-go"
)

/* python:
def add_microstructure_features(df: pd.DataFrame) -> pd.DataFrame:
    """
    Compute candle-level microstructure features:
      - candle body ratio
      - upper / lower wick ratio
      - consecutive bullish/bearish candle count
      - distance from Kalman-filtered prices (pre-computed in data)
    """
    body = df["close"] - df["open"]
    full_range = df["high"] - df["low"] + 1e-10

    df["candle_body_ratio"] = body.abs() / full_range
    df["candle_direction"] = np.sign(body).astype(np.float32)

    df["upper_wick_ratio"] = (
        (df["high"] - df[["open", "close"]].max(axis=1)) / full_range
    )
    df["lower_wick_ratio"] = (
        (df[["open", "close"]].min(axis=1) - df["low"]) / full_range
    )

    # Consecutive candle count (positive = bullish streak, negative = bearish)
    direction = np.sign(body)
    groups = (direction != direction.shift(1)).cumsum()
    df["consec_candle_count"] = direction.groupby(groups).cumcount() + 1
    df["consec_candle_count"] = df["consec_candle_count"] * direction

    # Kalman distance features (data already has these columns)
    if "fast_tempx_kalman" in df.columns:
        df["kalman_fast_dist"] = (
            (df["close"] - df["fast_tempx_kalman"]) / (df["close"] + 1e-10)
        )
    if "slow_tempx_kalman" in df.columns:
        df["kalman_slow_dist"] = (
            (df["close"] - df["slow_tempx_kalman"]) / (df["close"] + 1e-10)
        )
    if "fast_tempx_kalman" in df.columns and "slow_tempx_kalman" in df.columns:
        df["kalman_crossover"] = (
            (df["fast_tempx_kalman"] - df["slow_tempx_kalman"])
            / (df["close"] + 1e-10)
        )

    return df
*/

func Sign(x float64) float64 {
	if x > 0 {
		return 1
	} else if x < 0 {
		return -1
	}
	return 0
}

func SignedConsecutiveCount(direction []float64) []float64 {

	n := len(direction)
	result := make([]float64, n)

	count := 1
	result[0] = float64(direction[0])

	for i := 1; i < n; i++ {

		if direction[i] == direction[i-1] {
			count++
		} else {
			count = 1
		}

		result[i] = float64(count) * float64(direction[i])
	}

	return result
}

func AddMicrostructureFeatures(df *dataframe.DataFrame) {
	fmt.Println("reached here 0")
	length := df.NRows()
	_close := df.Series[FindIndexOf(df, "close")].(*dataframe.SeriesFloat64).Values
	_open := df.Series[FindIndexOf(df, "open")].(*dataframe.SeriesFloat64).Values
	_high := df.Series[FindIndexOf(df, "high")].(*dataframe.SeriesFloat64).Values
	_low := df.Series[FindIndexOf(df, "low")].(*dataframe.SeriesFloat64).Values

	_fast_tempx_kalman := df.Series[FindIndexOf(df, "fast_tempx_kalman")].(*dataframe.SeriesFloat64).Values
	_slow_tempx_kalman := df.Series[FindIndexOf(df, "slow_tempx_kalman")].(*dataframe.SeriesFloat64).Values

	body := make([]float64, length)
	full_range := make([]float64, length)

	for i := 0; i < length; i++ {
		body[i] = math.Abs(_close[i] - _open[i])
		full_range[i] = _high[i] - _low[i] + 1e-10
	}

	body_ratio := make([]float64, length)
	direction := make([]float64, length)
	upper_wick_ratio := make([]float64, length)
	lower_wick_ratio := make([]float64, length)
	consec_candle_count := make([]float64, length)
	kalman_fast_dist := make([]float64, length)
	kalman_slow_dist := make([]float64, length)
	kalman_crossover := make([]float64, length)

	for i := 0; i < length; i++ {
		body_ratio[i] = body[i] / full_range[i]
		direction[i] = Sign(_close[i] - _open[i])
		upper_wick_ratio[i] = (_high[i] - math.Max(_open[i], _close[i])) / full_range[i]
		lower_wick_ratio[i] = (math.Min(_open[i], _close[i]) - _low[i]) / full_range[i]

		// consec_candle_count[i] = float64(ConsecutiveCount(direction)[i]) * direction[i]
		kalman_fast_dist[i] = (_close[i] - _fast_tempx_kalman[i]) / (_close[i] + 1e-10)
		kalman_slow_dist[i] = (_close[i] - _slow_tempx_kalman[i]) / (_close[i] + 1e-10)
		kalman_crossover[i] = (_fast_tempx_kalman[i] - _slow_tempx_kalman[i]) / (_close[i] + 1e-10)
	}

	consec_candle_count = SignedConsecutiveCount(direction)

	body_ratio_series := dataframe.NewSeriesFloat64("candle_body_ratio", nil, body_ratio)
	df.AddSeries(body_ratio_series, nil)
	direction_series := dataframe.NewSeriesFloat64("candle_direction", nil, direction)
	df.AddSeries(direction_series, nil)
	upper_wick_ratio_series := dataframe.NewSeriesFloat64("upper_wick_ratio", nil, upper_wick_ratio)
	df.AddSeries(upper_wick_ratio_series, nil)
	lower_wick_ratio_series := dataframe.NewSeriesFloat64("lower_wick_ratio", nil, lower_wick_ratio)
	df.AddSeries(lower_wick_ratio_series, nil)
	consec_candle_count_series := dataframe.NewSeriesFloat64("consec_candle_count", nil, consec_candle_count)
	df.AddSeries(consec_candle_count_series, nil)
	kalman_fast_dist_series := dataframe.NewSeriesFloat64("kalman_fast_dist", nil, kalman_fast_dist)
	df.AddSeries(kalman_fast_dist_series, nil)
	kalman_slow_dist_series := dataframe.NewSeriesFloat64("kalman_slow_dist", nil, kalman_slow_dist)
	df.AddSeries(kalman_slow_dist_series, nil)
	kalman_crossover_series := dataframe.NewSeriesFloat64("kalman_crossover", nil, kalman_crossover)
	df.AddSeries(kalman_crossover_series, nil)
}
