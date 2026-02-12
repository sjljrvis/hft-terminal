package indicators

import (
	"math"

	"github.com/rocketlaunchr/dataframe-go"
)

var (
	KalmanFFTWindow        = 64
	KalmanFFTCutoffDivisor = 128
)

func CalcSWAPKalman(df *dataframe.DataFrame, seriesname string, source string, factor float64) {
	_length := df.NRows()
	// _swap_base := make([]float64, _length)
	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64).Values
	_atr3 := df.Series[FindIndexOf(df, "atr3")].(*dataframe.SeriesFloat64).Values
	_time := df.Series[FindIndexOf(df, "timestamp")].(*dataframe.SeriesTime).Values

	swap := make([]float64, _length)
	swap[0] = 0
	swap[1] = 0
	expansion := false

	for i := 2; i < _length; i++ {
		swap[i] = swap[i-1]
		expansion = false || math.Abs(_source[i]-_source[i-1]) > factor*_atr3[i]

		if _source[i] > _source[i-1] && expansion {
			swap[i] = 1
		} else if _source[i] < _source[i-1] && expansion {
			swap[i] = -1
		}

		if !IsActiveSession(_time[i]) {
			swap[i] = 0
		}
	}

	_swapSeries := dataframe.NewSeriesFloat64(seriesname, nil, swap)
	df.AddSeries(_swapSeries, nil)
}

func lowPassFFTLast(window []float64, cutoffDivisor int) float64 {
	if len(window) == 0 {
		return 0
	}
	n := nextPow2(len(window))
	if n < 2 {
		return window[len(window)-1]
	}

	buf := make([]complex128, n)
	for i := 0; i < len(window); i++ {
		buf[i] = complex(window[i], 0)
	}

	fftInPlace(buf, false)

	cutoff := n / cutoffDivisor
	if cutoff < 1 {
		cutoff = 1
	}
	if cutoff > n/2 {
		cutoff = n / 2
	}
	for i := cutoff; i < n-cutoff; i++ {
		buf[i] = 0
	}

	fftInPlace(buf, true)
	return real(buf[len(window)-1])
}

func nextPow2(n int) int {
	if n <= 1 {
		return 1
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

func fftInPlace(a []complex128, invert bool) {
	n := len(a)
	if n <= 1 {
		return
	}

	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}

	for length := 2; length <= n; length <<= 1 {
		angle := 2 * math.Pi / float64(length)
		if invert {
			angle = -angle
		}
		wlen := complex(math.Cos(angle), math.Sin(angle))
		for i := 0; i < n; i += length {
			w := complex(1, 0)
			half := length / 2
			for j := 0; j < half; j++ {
				u := a[i+j]
				v := a[i+j+half] * w
				a[i+j] = u + v
				a[i+j+half] = u - v
				w *= wlen
			}
		}
	}

	if invert {
		denom := complex(float64(n), 0)
		for i := range a {
			a[i] /= denom
		}
	}
}

func residualVariance(source []float64, smoothed []float64) float64 {
	if len(source) == 0 || len(source) != len(smoothed) {
		return 0
	}
	sum := 0.0
	for i := range source {
		sum += source[i] - smoothed[i]
	}
	mean := sum / float64(len(source))
	variance := 0.0
	for i := range source {
		diff := (source[i] - smoothed[i]) - mean
		variance += diff * diff
	}
	return variance / float64(len(source))
}

func KalmanFilter(df *dataframe.DataFrame, seriesname string, source string, window int, cutoffDivisor int) {
	length := df.NRows()
	// _timestamp := df.Series[FindIndexOf(df, "timestamp")].(*dataframe.SeriesTime).Values

	if length == 0 {
		return
	}

	_source := df.Series[FindIndexOf(df, source)].(*dataframe.SeriesFloat64).Values
	fftSmoothed := make([]float64, length)

	for i := 0; i < length; i++ {

		KalmanFFTWindow = window
		KalmanFFTCutoffDivisor = cutoffDivisor

		start := 0
		if i+1 > KalmanFFTWindow {
			start = i + 1 - KalmanFFTWindow
		}
		window := _source[start : i+1]
		fftSmoothed[i] = lowPassFFTLast(window, KalmanFFTCutoffDivisor)
	}

	r := residualVariance(_source, fftSmoothed)
	if r < 1e-6 {
		r = 1e-6
	}
	q := r * 0.01
	if q < 1e-6 {
		q = 1e-6
	}

	kalman := make([]float64, length)
	x := fftSmoothed[0]
	p := r
	for i := 0; i < length; i++ {
		if i > 0 {
			p += q
		}
		k := p / (p + r)
		x = x + k*(fftSmoothed[i]-x)
		p = (1 - k) * p
		// kalman[i] = math.Round(x*1000) / 1000
		kalman[i] = math.Ceil(x)
	}

	_kalmanSeries := dataframe.NewSeriesFloat64(seriesname, nil, kalman)
	df.AddSeries(_kalmanSeries, nil)
}
