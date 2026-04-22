package ml_model

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"hft/pkg/types"

	"github.com/rocketlaunchr/dataframe-go"
	"github.com/samber/lo"
	ort "github.com/yalue/onnxruntime_go"
)

// seriesIdx returns the column index for name in df, or -1 if not found.
func seriesIdx(df *dataframe.DataFrame, name string) int {
	_, idx, found := lo.FindIndexOf(df.Names(), func(s string) bool { return s == name })
	if !found {
		return -1
	}
	return idx
}

// Regime represents the predicted market regime.
type Regime string

const (
	RegimeVolatile Regime = "volatile"
	RegimeBullish  Regime = "bullish"
	RegimeBearish  Regime = "bearish"

	// numFeatures is the number of model input features per timestep.
	// Must match the Features struct below and meta.n_features.
	numFeatures = 47
)

// ErrNotWarmedUp is returned when the circular buffer is not yet full.
var ErrNotWarmedUp = fmt.Errorf("model not ready: sequence window not yet filled")

// regimes maps argmax class index → Regime label.
// Order matches the PyTorch training label encoding:
//
//	0 = bullish, 1 = bearish, 2 = volatile
var regimes = [3]Regime{RegimeBullish, RegimeBearish, RegimeVolatile}

// ─── JSON loading structs ─────────────────────────────────────────────────────

type modelMeta struct {
	SeqLen      int      `json:"seq_len"`
	NFeatures   int      `json:"n_features"`
	FeatureCols []string `json:"feature_cols"`
	// OutputName is the ONNX output tensor name (e.g. "logits").
	// The legacy "output" field in model.meta.json is a human-readable description,
	// NOT the tensor name — do not use it as an ONNX node name.
	OutputName string         `json:"output_name"`
	Smoothing  *metaSmoothing `json:"smoothing,omitempty"`
}

type metaSmoothing struct {
	Enabled                        bool    `json:"enabled"`
	ConfirmBars                    int     `json:"confirm_bars"`
	MinConfidence                  float64 `json:"min_confidence"`
	ForbidVolatileAfterNonVolatile bool    `json:"forbid_volatile_after_nonvolatile"`
	StartState                     string  `json:"start_state"`
}

type scalerJSONFile struct {
	Center []float64 `json:"center"`
	Scale  []float64 `json:"scale"`
}

func startStateToID(s string) int {
	switch s {
	case "bullish":
		return 0
	case "bearish":
		return 1
	default:
		return 2 // volatile
	}
}

// ─── Features ───────────────────────────────────────────────────────────────

// Features holds the 31 numeric inputs the ONNX model expects per timestep.
// Field order matches the training feature list exactly.
type Features struct {
	// ── Raw indicators (1-3) ────────────────────────────────────────────────
	FastCCI float64 `json:"fast_cci"` // 1.  fast_cci
	TR      float64 `json:"tr"`       // 2.  tr
	WmaTR2  float64 `json:"wma_tr_2"` // 3.  wma_tr_2

	// ── ATR of Kalman signals (4-5) ─────────────────────────────────────────
	ATR3     float64 `json:"atr3"`      // 4.  atr3
	ATR3Base float64 `json:"atr3_base"` // 5.  atr3_base

	// ── Swap signals (6-7) ──────────────────────────────────────────────────
	Swap     float64 `json:"swap"`      // 6.  swap
	SwapBase float64 `json:"swap_base"` // 7.  swap_base

	// ── EMA slopes & price distances (8-12) ─────────────────────────────────
	EmaSlopeFast  float64 `json:"ema_slope_fast"`      // 8.  ema_slope_fast
	EmaSlopeSlow  float64 `json:"ema_slope_slow"`      // 9.  ema_slope_slow
	PriceDistFast float64 `json:"price_dist_ema_fast"` // 10. price_dist_ema_fast
	PriceDistSlow float64 `json:"price_dist_ema_slow"` // 11. price_dist_ema_slow
	EmaCrossover  float64 `json:"ema_crossover"`       // 12. ema_crossover  (+1 / -1 / 0)

	// ── Momentum (13-18) ────────────────────────────────────────────────────
	RSI      float64 `json:"rsi"`        // 13. rsi
	ROC      float64 `json:"roc"`        // 14. roc
	LogRet   float64 `json:"log_ret"`    // 15. log_ret
	LogRet5  float64 `json:"log_ret_5"`  // 16. log_ret_5
	LogRet15 float64 `json:"log_ret_15"` // 17. log_ret_15
	LogRet30 float64 `json:"log_ret_30"` // 18. log_ret_30

	// ── Volatility (19-21) ──────────────────────────────────────────────────
	ATRComputed  float64 `json:"atr_computed"`   // 19. atr_computed
	RollingStd   float64 `json:"rolling_std"`    // 20. rolling_std
	RollingStd60 float64 `json:"rolling_std_60"` // 21. rolling_std_60

	// ── Range / candle shape (22-28) ────────────────────────────────────────
	HLRangePct      float64 `json:"hl_range_pct"`        // 22. hl_range_pct
	VolExpansion    float64 `json:"vol_expansion"`       // 23. vol_expansion
	CandleBodyRatio float64 `json:"candle_body_ratio"`   // 24. candle_body_ratio
	CandleDirection float64 `json:"candle_direction"`    // 25. candle_direction  (+1 / -1)
	UpperWickRatio  float64 `json:"upper_wick_ratio"`    // 26. upper_wick_ratio
	LowerWickRatio  float64 `json:"lower_wick_ratio"`    // 27. lower_wick_ratio
	ConsecCandles   float64 `json:"consec_candle_count"` // 28. consec_candle_count

	// ── Kalman distances (29-31) ─────────────────────────────────────────────
	KalmanFastDist  float64 `json:"kalman_fast_dist"` // 29. kalman_fast_dist
	KalmanSlowDist  float64 `json:"kalman_slow_dist"` // 30. kalman_slow_dist
	KalmanCrossover float64 `json:"kalman_crossover"` // 31. kalman_crossover  (+1 / -1 / 0)

	// ── Time context (32) ────────────────────────────────────────────────────
	MinuteOfDay float64 `json:"minute_of_day"` // 32. minute_of_day  (0–1 over NSE session)

	// ── Volume features (33-37) ──────────────────────────────────────────────
	// All zero for index instruments (e.g. Nifty) where volume == 0.
	VolSMARatio  float64 `json:"vol_sma_ratio"`  // 33. vol_sma_ratio
	VolROC       float64 `json:"vol_roc"`        // 34. vol_roc
	VolPriceCorr float64 `json:"vol_price_corr"` // 35. vol_price_corr
	OBVSlope     float64 `json:"obv_slope"`      // 36. obv_slope
	VWAPDist     float64 `json:"vwap_dist"`      // 37. vwap_dist

	// ── Multi-timeframe 5m (38-42) ───────────────────────────────────────────
	DistEMAFast5m  float64 `json:"dist_ema_fast_5m"` // 38. dist_ema_fast_5m
	DistEMASlow5m  float64 `json:"dist_ema_slow_5m"` // 39. dist_ema_slow_5m
	EMACrossover5m float64 `json:"ema_crossover_5m"` // 40. ema_crossover_5m
	RSI5m          float64 `json:"rsi_5m"`           // 41. rsi_5m
	ATRPct5m       float64 `json:"atr_pct_5m"`       // 42. atr_pct_5m

	// ── Multi-timeframe 15m (43-47) ──────────────────────────────────────────
	DistEMAFast15m  float64 `json:"dist_ema_fast_15m"` // 43. dist_ema_fast_15m
	DistEMASlow15m  float64 `json:"dist_ema_slow_15m"` // 44. dist_ema_slow_15m
	EMACrossover15m float64 `json:"ema_crossover_15m"` // 45. ema_crossover_15m
	RSI15m          float64 `json:"rsi_15m"`           // 46. rsi_15m
	ATRPct15m       float64 `json:"atr_pct_15m"`       // 47. atr_pct_15m

	// ── Metadata — not fed to the model ─────────────────────────────────────
	Timestamp time.Time
}

// toFloat32Row packs the 47 features into a fixed-size array in model order.
// NaN / Inf values are sanitised to 0.
func (f Features) toFloat32Row() [numFeatures]float32 {
	raw := [numFeatures]float64{
		f.FastCCI, f.TR, f.WmaTR2, // 1-3
		f.ATR3, f.ATR3Base, // 4-5
		f.Swap, f.SwapBase, // 6-7
		f.EmaSlopeFast, f.EmaSlopeSlow, f.PriceDistFast, f.PriceDistSlow, // 8-11
		f.EmaCrossover,                                            // 12
		f.RSI, f.ROC, f.LogRet, f.LogRet5, f.LogRet15, f.LogRet30, // 13-18
		f.ATRComputed, f.RollingStd, f.RollingStd60, // 19-21
		f.HLRangePct, f.VolExpansion, // 22-23
		f.CandleBodyRatio, f.CandleDirection, // 24-25
		f.UpperWickRatio, f.LowerWickRatio, f.ConsecCandles, // 26-28
		f.KalmanFastDist, f.KalmanSlowDist, f.KalmanCrossover, // 29-31
		f.MinuteOfDay, // 32
		f.VolSMARatio, f.VolROC, f.VolPriceCorr, f.OBVSlope, f.VWAPDist, // 33-37
		f.DistEMAFast5m, f.DistEMASlow5m, f.EMACrossover5m, f.RSI5m, f.ATRPct5m, // 38-42
		f.DistEMAFast15m, f.DistEMASlow15m, f.EMACrossover15m, f.RSI15m, f.ATRPct15m, // 43-47
	}
	var out [numFeatures]float32
	for i, v := range raw {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			v = 0
		}
		out[i] = float32(v)
	}
	return out
}

// sanitize32 converts a float64 to float32, replacing NaN/Inf with 0.
func sanitize32(v float64) float32 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return float32(v)
}

// robustScale applies RobustScaler in-place: out[i] = (in[i] - center[i]) / scale[i].
// Invalid (NaN/Inf) results are zeroed.
func robustScale(row []float32, center, scale []float32) {
	for i := range row {
		v := (row[i] - center[i]) / scale[i]
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			v = 0
		}
		row[i] = v
	}
}

// ─── Predictor ──────────────────────────────────────────────────────────────

// Predictor holds a single AdvancedSession and all configuration loaded from
// model.meta.json and scaler.json at startup.
type Predictor struct {
	session      *ort.AdvancedSession
	inputTensor  *ort.Tensor[float32]
	outputTensor *ort.Tensor[float32]
	mu           sync.Mutex
	modelDir     string

	// Loaded from model.meta.json
	seqLen      int
	nFeatures   int
	featureCols []string
	outputName  string

	// Loaded from scaler.json
	scalerCenter []float32
	scalerScale  []float32

	// Circular buffer for live tick-by-tick inference.
	// Flat slice of length seqLen * nFeatures; ring indexed by bufHead.
	seqBuf      []float32
	bufHead     int
	candleCount int

	// Hysteresis smoothing (auto-applied from meta, override via SetSmoothing).
	smoothing   SmoothingConfig
	smoothState HysteresisState
}

// SetSmoothing configures the hysteresis filter and resets its state.
func (p *Predictor) SetSmoothing(cfg SmoothingConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.smoothing = cfg
	p.smoothState = newHysteresisState(cfg.StartID)
}

// IsWarmedUp reports whether seqLen candles have been buffered and inference is valid.
func (p *Predictor) IsWarmedUp() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.candleCount >= p.seqLen
}

// WarmupCandles returns the number of candles required before inference is valid.
func (p *Predictor) WarmupCandles() int { return p.seqLen }

// CandleCount returns the total number of candles seen so far.
func (p *Predictor) CandleCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.candleCount
}

// ─── Construction ────────────────────────────────────────────────────────────

var (
	globalPredictor *Predictor
	predictorOnce   sync.Once
)

// InitPredictor initialises the package-level singleton.
// modelDir is the directory containing model.onnx, model.meta.json, scaler.json.
func InitPredictor(modelDir, ortLibPath string) error {
	var err error
	predictorOnce.Do(func() {
		globalPredictor, err = NewPredictor(modelDir, ortLibPath)
	})
	return err
}

// GetPredictor returns the package-level singleton (nil if not initialised).
func GetPredictor() *Predictor { return globalPredictor }

// NewPredictor loads model.meta.json, scaler.json, and model.onnx from modelDir,
// then allocates the ONNX session and I/O tensors.
func NewPredictor(modelDir, ortLibPath string) (*Predictor, error) {
	if !filepath.IsAbs(modelDir) {
		cwd, _ := os.Getwd()
		modelDir = filepath.Join(cwd, modelDir)
	}

	// ── Load model.meta.json ──────────────────────────────────────────────
	metaPath := filepath.Join(modelDir, "model.meta.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("read model.meta.json: %w", err)
	}
	var meta modelMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return nil, fmt.Errorf("parse model.meta.json: %w", err)
	}
	if meta.SeqLen <= 0 {
		return nil, fmt.Errorf("model.meta.json: invalid seq_len %d", meta.SeqLen)
	}
	if meta.NFeatures != numFeatures {
		return nil, fmt.Errorf("model.meta.json: n_features=%d but compiled numFeatures=%d", meta.NFeatures, numFeatures)
	}
	if len(meta.FeatureCols) != numFeatures {
		return nil, fmt.Errorf("model.meta.json: feature_cols length %d != %d", len(meta.FeatureCols), numFeatures)
	}
	// meta.OutputName maps to "output_name" in model.meta.json (the actual ONNX
	// tensor name). Fall back to "logits" when absent, because the legacy
	// "output" field in older meta files is a description string, not a name.
	outputName := meta.OutputName
	if outputName == "" {
		outputName = "logits"
	}

	// ── Load scaler.json ──────────────────────────────────────────────────
	scalerPath := filepath.Join(modelDir, "scaler.json")
	scalerData, err := os.ReadFile(scalerPath)
	if err != nil {
		return nil, fmt.Errorf("read scaler.json: %w", err)
	}
	var sj scalerJSONFile
	if err := json.Unmarshal(scalerData, &sj); err != nil {
		return nil, fmt.Errorf("parse scaler.json: %w", err)
	}
	if len(sj.Center) != numFeatures || len(sj.Scale) != numFeatures {
		return nil, fmt.Errorf("scaler.json: expected %d entries, got center=%d scale=%d", numFeatures, len(sj.Center), len(sj.Scale))
	}
	center := make([]float32, numFeatures)
	scale := make([]float32, numFeatures)
	for i := 0; i < numFeatures; i++ {
		center[i] = float32(sj.Center[i])
		scale[i] = float32(sj.Scale[i])
	}

	// ── Set up ORT ────────────────────────────────────────────────────────
	if !filepath.IsAbs(ortLibPath) {
		cwd, _ := os.Getwd()
		ortLibPath = filepath.Join(cwd, ortLibPath)
	}
	ort.SetSharedLibraryPath(ortLibPath)
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("onnxruntime init: %w", err)
	}

	modelPath := filepath.Join(modelDir, "model.onnx")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model not found: %s", modelPath)
	}

	opts, err := ort.NewSessionOptions()
	if err != nil {
		return nil, fmt.Errorf("session options: %w", err)
	}
	defer opts.Destroy()

	inputTensor, err := ort.NewEmptyTensor[float32](ort.Shape{1, int64(meta.SeqLen), numFeatures})
	if err != nil {
		return nil, fmt.Errorf("input tensor: %w", err)
	}

	outputTensor, err := ort.NewEmptyTensor[float32](ort.Shape{1, 3})
	if err != nil {
		inputTensor.Destroy()
		return nil, fmt.Errorf("output tensor: %w", err)
	}

	session, err := ort.NewAdvancedSession(
		modelPath,
		[]string{"x"},
		[]string{outputName},
		[]ort.Value{inputTensor},
		[]ort.Value{outputTensor},
		opts,
	)
	if err != nil {
		inputTensor.Destroy()
		outputTensor.Destroy()
		return nil, fmt.Errorf("create session: %w", err)
	}

	// ── Build smoothing config from meta (fallback to defaults) ───────────
	smoothCfg := DefaultSmoothingConfig()
	if ms := meta.Smoothing; ms != nil {
		smoothCfg = SmoothingConfig{
			Enabled:                        ms.Enabled,
			ConfirmBars:                    ms.ConfirmBars,
			MinConfidence:                  ms.MinConfidence,
			ForbidVolatileAfterNonVolatile: ms.ForbidVolatileAfterNonVolatile,
			StartID:                        startStateToID(ms.StartState),
		}
		if smoothCfg.ConfirmBars < 1 {
			smoothCfg.ConfirmBars = 1
		}
	}

	return &Predictor{
		session:      session,
		inputTensor:  inputTensor,
		outputTensor: outputTensor,
		modelDir:     modelDir,
		seqLen:       meta.SeqLen,
		nFeatures:    meta.NFeatures,
		featureCols:  meta.FeatureCols,
		outputName:   outputName,
		scalerCenter: center,
		scalerScale:  scale,
		seqBuf:       make([]float32, meta.SeqLen*numFeatures),
		smoothing:    smoothCfg,
		smoothState:  newHysteresisState(smoothCfg.StartID),
	}, nil
}

// ─── Single-tick inference ───────────────────────────────────────────────────

// PredictRegime buffers the current candle's features into the circular buffer
// and, once seqLen candles have been seen, runs one inference pass.
//
// Returns ErrNotWarmedUp until seqLen candles have been buffered.
func (p *Predictor) PredictRegime(f Features) (Regime, error) {
	if p == nil || p.session == nil {
		return "", fmt.Errorf("predictor not initialised")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// 1. Push current features into the circular buffer (scaled).
	row := f.toFloat32Row()
	robustScale(row[:], p.scalerCenter, p.scalerScale)
	base := p.bufHead * numFeatures
	copy(p.seqBuf[base:base+numFeatures], row[:])
	p.bufHead = (p.bufHead + 1) % p.seqLen
	p.candleCount++

	// 2. Guard: buffer must be full before we can form a complete sequence.
	if p.candleCount < p.seqLen {
		return "", ErrNotWarmedUp
	}

	// 3. Pack circular buffer → flat [1, seqLen, numFeatures] tensor (oldest first).
	tensorData := p.inputTensor.GetData()
	for s := 0; s < p.seqLen; s++ {
		slot := (p.bufHead + s) % p.seqLen
		srcBase := slot * numFeatures
		dstBase := s * numFeatures
		copy(tensorData[dstBase:dstBase+numFeatures], p.seqBuf[srcBase:srcBase+numFeatures])
	}

	// 4. Run inference.
	if err := p.session.Run(); err != nil {
		return "", fmt.Errorf("inference: %w", err)
	}

	// 5. Argmax over logits; apply hysteresis smoothing if enabled.
	logits := p.outputTensor.GetData()
	rawIdx := argmaxIdx3(logits)

	if !p.smoothing.Enabled {
		return regimes[rawIdx], nil
	}

	probs := softmax3(logits)
	smoothedIdx := p.smoothState.Step(rawIdx, probs, &p.smoothing)
	if smoothedIdx < 0 {
		return regimes[rawIdx], nil
	}
	return regimes[smoothedIdx], nil
}

// PredictRegimeFromTick is a convenience wrapper for live ticks.
func (p *Predictor) PredictRegimeFromTick(tick *types.Tick, extra Features) (Regime, error) {
	extra.Timestamp = tick.Timestamp
	return p.PredictRegime(extra)
}

// ─── Batch DF inference ──────────────────────────────────────────────────────

// PredictRegimeFromDF runs inference for every row in df that has a full
// seqLen-row history window of finite features, and appends prediction columns.
//
// Feature columns are read dynamically from featureCols loaded at startup.
//
// Columns added:
//
//	pred_regime_id_raw  float64  – raw argmax class index (NaN = no prediction)
//	pred_regime_raw     string   – raw regime label       ("" = no prediction)
//	pred_regime_id      float64  – smoothed class index   (NaN = no prediction)
//	pred_regime         string   – smoothed regime label  ("" = no prediction)
//	pred_prob_bullish   float64  – softmax probability for Bullish
//	pred_prob_bearish   float64  – softmax probability for Bearish
//	pred_prob_volatile  float64  – softmax probability for Volatile
//	pred_confidence     float64  – max softmax probability
//	pred_status         string   – "invalid_features" | "buffering" | "ready"
//	regime              string   – alias for pred_regime (backward-compat)
func (p *Predictor) PredictRegimeFromDF(df *dataframe.DataFrame) error {
	return p.predictRegimeFromDF(df, 1)
}

// PredictRegimeFromDFStrided is like PredictRegimeFromDF but runs inference
// only every stride rows, carrying the last prediction forward for in-between
// rows. stride=1 is equivalent to PredictRegimeFromDF. For backtest use
// stride=5 gives ~5x speedup with negligible loss of regime resolution.
func (p *Predictor) PredictRegimeFromDFStrided(df *dataframe.DataFrame, stride int) error {
	if stride < 1 {
		stride = 1
	}
	return p.predictRegimeFromDF(df, stride)
}

func (p *Predictor) predictRegimeFromDF(df *dataframe.DataFrame, stride int) error {
	if p == nil || p.session == nil {
		return fmt.Errorf("predictor not initialised")
	}

	n := df.NRows()

	p.mu.Lock()
	p.candleCount += n
	p.mu.Unlock()

	// ── 1. Pull all feature columns ───────────────────────────────────────
	col := func(name string) []float64 {
		idx := seriesIdx(df, name)
		if idx < 0 {
			return make([]float64, n) // missing → zeros (NaN-flagged below)
		}
		return df.Series[idx].(*dataframe.SeriesFloat64).Values
	}

	cols := make([][]float64, p.nFeatures)
	for fi, name := range p.featureCols {
		cols[fi] = col(name)
	}

	// ── 2. Per-row validity: all features must be finite ─────────────────
	rowValid := make([]bool, n)
	for i := 0; i < n; i++ {
		ok := true
		for fi := 0; fi < p.nFeatures; fi++ {
			if math.IsNaN(cols[fi][i]) || math.IsInf(cols[fi][i], 0) {
				ok = false
				break
			}
		}
		rowValid[i] = ok
	}

	// Prefix sum for O(1) window validity check.
	invalidPfx := make([]int, n+1)
	for i := 0; i < n; i++ {
		invalidPfx[i+1] = invalidPfx[i]
		if !rowValid[i] {
			invalidPfx[i+1]++
		}
	}
	windowValid := func(start, end int) bool {
		return invalidPfx[end+1]-invalidPfx[start] == 0
	}

	// ── 3. Prepare output slices ─────────────────────────────────────────
	rawIDs := make([]int, n)
	rawProbs := make([][3]float32, n)
	for i := range rawIDs {
		rawIDs[i] = -1
	}

	// ── 4. Inference loop — reuse pre-allocated tensor memory ─────────────
	p.mu.Lock()
	defer p.mu.Unlock()

	inData := p.inputTensor.GetData()

	lastRawID := -1
	var lastRawProbs [3]float32

	for rowIdx := p.seqLen - 1; rowIdx < n; rowIdx++ {
		startRow := rowIdx - p.seqLen + 1

		// On strided rows, carry the last prediction forward without re-running.
		if stride > 1 && (rowIdx-(p.seqLen-1))%stride != 0 {
			rawIDs[rowIdx] = lastRawID
			rawProbs[rowIdx] = lastRawProbs
			continue
		}

		if !windowValid(startRow, rowIdx) {
			continue
		}

		// Pack seqLen rows into inData, oldest first, with RobustScaler applied.
		for s := 0; s < p.seqLen; s++ {
			src := startRow + s
			base := s * p.nFeatures
			for fi := 0; fi < p.nFeatures; fi++ {
				v := float32(cols[fi][src])
				sv := (v - p.scalerCenter[fi]) / p.scalerScale[fi]
				if math.IsNaN(float64(sv)) || math.IsInf(float64(sv), 0) {
					sv = 0
				}
				inData[base+fi] = sv
			}
		}

		if err := p.session.Run(); err != nil {
			return fmt.Errorf("row %d inference: %w", rowIdx, err)
		}

		logits := p.outputTensor.GetData()
		rawIDs[rowIdx] = argmaxIdx3(logits)
		rawProbs[rowIdx] = softmax3(logits)
		lastRawID = rawIDs[rowIdx]
		lastRawProbs = rawProbs[rowIdx]
	}

	// ── 5. Smoothing ──────────────────────────────────────────────────────
	smoothedIDs := make([]int, n)
	copy(smoothedIDs, rawIDs)
	if p.smoothing.Enabled {
		smoothedIDs = hysteresisFilterSeries(rawIDs, rawProbs, &p.smoothing)
	}

	// ── 6. Build output slices ────────────────────────────────────────────
	nan := math.NaN()
	outRawID := make([]interface{}, n)
	outRawName := make([]interface{}, n)
	outSmID := make([]interface{}, n)
	outSmName := make([]interface{}, n)
	outProbBull := make([]interface{}, n)
	outProbBear := make([]interface{}, n)
	outProbVol := make([]interface{}, n)
	outConf := make([]interface{}, n)
	outStatus := make([]interface{}, n)
	outRegime := make([]interface{}, n)

	for i := 0; i < n; i++ {
		rid := rawIDs[i]
		sid := smoothedIDs[i]

		if !rowValid[i] {
			outStatus[i] = "invalid_features"
		} else if rid < 0 {
			outStatus[i] = "buffering"
		} else {
			outStatus[i] = "ready"
		}

		if rid >= 0 {
			outRawID[i] = float64(rid)
			outRawName[i] = string(regimes[rid])
		} else {
			outRawID[i] = nan
			outRawName[i] = ""
		}

		if sid >= 0 {
			outSmID[i] = float64(sid)
			outSmName[i] = string(regimes[sid])
			outRegime[i] = string(regimes[sid])
		} else {
			outSmID[i] = nan
			outSmName[i] = ""
			outRegime[i] = ""
		}

		if rid >= 0 {
			pr := rawProbs[i]
			outProbBull[i] = float64(pr[0])
			outProbBear[i] = float64(pr[1])
			outProbVol[i] = float64(pr[2])
			maxP := pr[0]
			if pr[1] > maxP {
				maxP = pr[1]
			}
			if pr[2] > maxP {
				maxP = pr[2]
			}
			outConf[i] = float64(maxP)
		} else {
			outProbBull[i] = nan
			outProbBear[i] = nan
			outProbVol[i] = nan
			outConf[i] = nan
		}
	}

	// ── 7. Append series to df ────────────────────────────────────────────
	df.AddSeries(dataframe.NewSeriesFloat64("pred_regime_id_raw", nil, outRawID...), nil)
	df.AddSeries(dataframe.NewSeriesString("pred_regime_raw", nil, outRawName...), nil)
	df.AddSeries(dataframe.NewSeriesFloat64("pred_regime_id", nil, outSmID...), nil)
	df.AddSeries(dataframe.NewSeriesString("pred_regime", nil, outSmName...), nil)
	df.AddSeries(dataframe.NewSeriesFloat64("pred_prob_bullish", nil, outProbBull...), nil)
	df.AddSeries(dataframe.NewSeriesFloat64("pred_prob_bearish", nil, outProbBear...), nil)
	df.AddSeries(dataframe.NewSeriesFloat64("pred_prob_volatile", nil, outProbVol...), nil)
	df.AddSeries(dataframe.NewSeriesFloat64("pred_confidence", nil, outConf...), nil)
	df.AddSeries(dataframe.NewSeriesString("pred_status", nil, outStatus...), nil)
	df.AddSeries(dataframe.NewSeriesString("regime", nil, outRegime...), nil)
	return nil
}

// ─── Single-row DF inference (for simulation) ────────────────────────────────

// TickPrediction holds the result of a single-row prediction.
type TickPrediction struct {
	Regime   Regime
	ProbBull float64
	ProbBear float64
	ProbVol  float64
}

// PredictSingleRow runs inference for a single row in the DF, using the
// seqLen-row window ending at rowIdx. Feature columns are read from cols
// (pre-extracted, one slice per feature in featureCols order).
// Returns an empty prediction if the window contains invalid features.
func (p *Predictor) PredictSingleRow(cols [][]float64, rowIdx int) (TickPrediction, error) {
	if p == nil || p.session == nil {
		return TickPrediction{}, fmt.Errorf("predictor not initialised")
	}
	if rowIdx < p.seqLen-1 {
		return TickPrediction{}, ErrNotWarmedUp
	}

	startRow := rowIdx - p.seqLen + 1

	// Validate all features in the window are finite.
	for s := startRow; s <= rowIdx; s++ {
		for fi := 0; fi < p.nFeatures; fi++ {
			v := cols[fi][s]
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return TickPrediction{}, fmt.Errorf("row %d feature %d invalid", s, fi)
			}
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Pack seqLen rows into input tensor (oldest first, scaled).
	inData := p.inputTensor.GetData()
	for s := 0; s < p.seqLen; s++ {
		src := startRow + s
		base := s * p.nFeatures
		for fi := 0; fi < p.nFeatures; fi++ {
			v := float32(cols[fi][src])
			sv := (v - p.scalerCenter[fi]) / p.scalerScale[fi]
			if math.IsNaN(float64(sv)) || math.IsInf(float64(sv), 0) {
				sv = 0
			}
			inData[base+fi] = sv
		}
	}

	if err := p.session.Run(); err != nil {
		return TickPrediction{}, fmt.Errorf("inference: %w", err)
	}

	logits := p.outputTensor.GetData()
	probs := softmax3(logits)
	idx := argmaxIdx3(logits)

	return TickPrediction{
		Regime:   regimes[idx],
		ProbBull: float64(probs[0]),
		ProbBear: float64(probs[1]),
		ProbVol:  float64(probs[2]),
	}, nil
}

// FeatureCols returns the ordered feature column names the model expects.
func (p *Predictor) FeatureCols() []string {
	if p == nil {
		return nil
	}
	return p.featureCols
}

// ─── Close ───────────────────────────────────────────────────────────────────

// Close destroys the session and its tensors, then shuts down the ORT env.
func (p *Predictor) Close() error {
	if p == nil {
		return nil
	}
	if p.session != nil {
		p.session.Destroy()
	}
	if p.inputTensor != nil {
		p.inputTensor.Destroy()
	}
	if p.outputTensor != nil {
		p.outputTensor.Destroy()
	}
	ort.DestroyEnvironment()
	return nil
}

// ─── Package-level helpers ───────────────────────────────────────────────────

// PredictRegime calls PredictRegime on the global singleton.
func PredictRegime(f Features) (Regime, error) {
	p := GetPredictor()
	if p == nil {
		return "", fmt.Errorf("predictor not initialised; call InitPredictor first")
	}
	return p.PredictRegime(f)
}

// PredictRegimeFromTick calls PredictRegimeFromTick on the global singleton.
func PredictRegimeFromTick(tick *types.Tick, extra Features) (Regime, error) {
	p := GetPredictor()
	if p == nil {
		return "", fmt.Errorf("predictor not initialised; call InitPredictor first")
	}
	return p.PredictRegimeFromTick(tick, extra)
}

// PredictRegimeFromDF calls PredictRegimeFromDF on the global singleton.
func PredictRegimeFromDF(df *dataframe.DataFrame) error {
	p := GetPredictor()
	if p == nil {
		return fmt.Errorf("predictor not initialised; call InitPredictor first")
	}
	return p.PredictRegimeFromDF(df)
}

// PredictRegimeFromDFStrided calls PredictRegimeFromDFStrided on the global singleton.
func PredictRegimeFromDFStrided(df *dataframe.DataFrame, stride int) error {
	p := GetPredictor()
	if p == nil {
		return fmt.Errorf("predictor not initialised; call InitPredictor first")
	}
	return p.PredictRegimeFromDFStrided(df, stride)
}
