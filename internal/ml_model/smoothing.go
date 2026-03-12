package ml_model

import "math"

// ─── SmoothingConfig ─────────────────────────────────────────────────────────

// SmoothingConfig controls the hysteresis filter applied after raw inference.
//
// Matches the Python inference.smoothing section in model.yaml:
//
//	smoothing:
//	  enabled: true
//	  confirm_bars: 5
//	  min_confidence: 0.55
//	  forbid_volatile_after_nonvolatile: true
//	  start_state: "volatile"
type SmoothingConfig struct {
	// Enabled enables / disables smoothing entirely.
	// When false all other fields are ignored.
	Enabled bool

	// ConfirmBars is the number of consecutive bars a new regime must appear
	// before the switch is committed. Must be >= 1. (Python: confirm_bars)
	ConfirmBars int

	// MinConfidence gates confirmation: a bar only counts if softmax
	// probability of the candidate regime >= this value. 0 disables the gate.
	// (Python: min_confidence)
	MinConfidence float64

	// ForbidVolatileAfterNonVolatile blocks transitions back to Volatile once
	// the current regime is Bullish or Bearish. Implements the intraday rule:
	//   Volatile → Bullish / Bearish allowed
	//   Bullish / Bearish → Volatile blocked
	// (Python: forbid_volatile_after_nonvolatile)
	ForbidVolatileAfterNonVolatile bool

	// StartID is the class index used to pre-seed the hysteresis state before
	// any predictions are seen. Use -1 (or leave zero-value of int and
	// initialise explicitly) to start uninitialised.
	//
	// Mapping (matches REGIME_NAMES in Python):
	//   -1 = none (init with first prediction)
	//    0 = Bullish
	//    1 = Bearish
	//    2 = Volatile
	//
	// (Python: start_state: "volatile" → 2)
	StartID int
}

// DefaultSmoothingConfig returns the config from model.yaml (enabled, 5-bar,
// 0.55 confidence, forbid volatile revert, start volatile).
func DefaultSmoothingConfig() SmoothingConfig {
	return SmoothingConfig{
		Enabled:                        true,
		ConfirmBars:                    5,
		MinConfidence:                  0.55,
		ForbidVolatileAfterNonVolatile: true,
		StartID:                        2, // Volatile
	}
}

// ─── HysteresisState ─────────────────────────────────────────────────────────

// HysteresisState holds mutable per-predictor hysteresis state.
// It is NOT goroutine-safe — callers must hold the Predictor mutex.
//
// Maps to Python's HysteresisState dataclass in inference/smoothing.py.
type HysteresisState struct {
	currentID    int // accepted regime index; -1 = uninitialised
	pendingID    int // candidate regime index being counted; -1 = none
	pendingCount int // consecutive bars the candidate has been seen
}

// newHysteresisState creates a HysteresisState pre-seeded with startID.
// Pass -1 to start uninitialised (will be set from first valid prediction).
func newHysteresisState(startID int) HysteresisState {
	return HysteresisState{currentID: startID, pendingID: -1}
}

// Reset clears state and optionally re-seeds with startID (pass -1 to clear).
func (st *HysteresisState) Reset(startID int) {
	st.currentID = startID
	st.pendingID = -1
	st.pendingCount = 0
}

// Step processes one bar of raw inference output and returns the smoothed class
// index. Pass rawID = -1 for "not ready / buffering" bars.
//
// Mirrors Python's hysteresis_filter_step() in inference/smoothing.py.
func (st *HysteresisState) Step(rawID int, probs [3]float32, cfg *SmoothingConfig) int {
	if rawID < 0 {
		return -1
	}

	// Initialise with first ready prediction.
	if st.currentID < 0 {
		st.currentID = rawID
		st.pendingID = -1
		st.pendingCount = 0
		return st.currentID
	}

	cur := st.currentID

	// Stable — same as current.
	if rawID == cur {
		st.pendingID = -1
		st.pendingCount = 0
		return cur
	}

	// Intraday rule: block Bullish/Bearish → Volatile transition.
	// Regime indices: 0=Bullish, 1=Bearish, 2=Volatile
	if cfg.ForbidVolatileAfterNonVolatile && (cur == 0 || cur == 1) && rawID == 2 {
		st.pendingID = -1
		st.pendingCount = 0
		return cur
	}

	// Confidence gate: ignore low-confidence predictions.
	if cfg.MinConfidence > 0 {
		if float64(probs[rawID]) < cfg.MinConfidence {
			st.pendingID = -1
			st.pendingCount = 0
			return cur
		}
	}

	// Track pending transition.
	if st.pendingID != rawID {
		st.pendingID = rawID
		st.pendingCount = 1
	} else {
		st.pendingCount++
	}

	bars := cfg.ConfirmBars
	if bars < 1 {
		bars = 1
	}
	if st.pendingCount >= bars {
		st.currentID = rawID
		st.pendingID = -1
		st.pendingCount = 0
		return st.currentID
	}

	return cur
}

// ─── Batch helper ─────────────────────────────────────────────────────────────

// hysteresisFilterSeries applies hysteresis smoothing over a full series of raw
// predictions — direct equivalent of Python's hysteresis_filter_series().
//
// rawIDs: class indices per row, -1 = not ready.
// probs:  [3]float32 softmax probabilities per row (used for MinConfidence gate).
// Returns smoothed class indices (-1 where rawID was -1).
//
// Rows with rawID < 0 trigger a state reset (reset_mask behaviour in Python).
func hysteresisFilterSeries(
	rawIDs []int,
	probs [][3]float32,
	cfg *SmoothingConfig,
) []int {
	out := make([]int, len(rawIDs))
	st := newHysteresisState(cfg.StartID)

	for i, rid := range rawIDs {
		// Reset at segment boundaries (rows where model is not yet ready).
		if rid < 0 {
			st.Reset(cfg.StartID)
			out[i] = -1
			continue
		}
		var p [3]float32
		if i < len(probs) {
			p = probs[i]
		}
		out[i] = st.Step(rid, p, cfg)
	}
	return out
}

// ─── Math helpers ─────────────────────────────────────────────────────────────

// softmax3 computes numerically-stable softmax over 3 logits.
func softmax3(logits []float32) [3]float32 {
	maxV := logits[0]
	for _, v := range logits[1:3] {
		if v > maxV {
			maxV = v
		}
	}
	var sum float32
	var out [3]float32
	for i := 0; i < 3; i++ {
		out[i] = float32(math.Exp(float64(logits[i] - maxV)))
		sum += out[i]
	}
	if sum > 0 {
		for i := range out {
			out[i] /= sum
		}
	}
	return out
}

// argmaxIdx3 returns the index of the highest value among the first 3 elements.
func argmaxIdx3(logits []float32) int {
	idx := 0
	for i := 1; i < 3; i++ {
		if logits[i] > logits[idx] {
			idx = i
		}
	}
	return idx
}
