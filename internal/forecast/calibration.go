// Package-level calibration utilities.
//
// This file implements the "have we been right before?" feedback loop:
// it reads the (predicted, observed) pairs accumulated by `sageo compare`
// in .sageo/calibration.json and turns them into a CalibrationProfile
// that Adjust applies as a simple multiplicative correction on raw
// forecasts, with uncertainty bounds derived from the historical spread.
//
// The approach is deliberately unsophisticated:
//
//   - ratio = observed / predicted for every paired data point;
//   - median = bias correction factor (less sensitive to outliers than
//     the mean);
//   - P25 / P75 = spread of that correction, used to bound the
//     calibrated range;
//   - MAE (mean absolute error) = a single-number "how wrong are we
//     typically" summary, useful for reporting but not used to Adjust.
//
// We do NOT fit a model, do not learn per-URL factors, do not reach for
// regression. We just honestly report "the last N times we predicted
// this kind of change, we were off by roughly this much." That's it.
//
// See https://en.wikipedia.org/wiki/Calibration_(statistics) for the
// underlying idea; this is closer to bias correction than a full
// calibration curve, but the intent is the same: make future forecasts
// consistent with past reality.
package forecast

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

// Minimum sample sizes before a calibration is trusted. These thresholds
// are deliberately conservative — it's far better to say "we don't have
// enough data to adjust" than to overfit to a handful of early outcomes.
const (
	// MinSamplePerChangeType is the floor for per-ChangeType calibration
	// to kick in. Below this, we fall back to the overall profile.
	MinSamplePerChangeType = 20
	// MinSampleOverall is the floor for the overall profile to be used
	// at all. Below this, forecasts are not adjusted and the
	// ConfidenceLabel is set to "insufficient_data".
	MinSampleOverall = 50
)

// Confidence labels surfaced in the Forecast for UI consumers.
const (
	ConfidenceHigh         = "high_confidence"
	ConfidenceModerate     = "moderate_confidence"
	ConfidenceLow          = "low_confidence"
	ConfidenceInsufficient = "insufficient_data"
)

// Priority-tier thresholds (monthly click delta). Tiers are the primary
// output — they're meant to be robust to the noise in the point
// estimate.
const (
	TierHighThreshold   = 200
	TierMediumThreshold = 50
)

// Caveat strings shipped alongside a Forecast. Exported so tests and
// presenters can pattern-match rather than stringly-typed-comparing.
const (
	CaveatInsufficientCalibration = "insufficient_calibration_data"
	CaveatLowConfidence           = "low_confidence"
	CaveatLowSearchVolume         = "low_search_volume"
	CaveatForecasterOvershoots    = "forecaster_tends_to_overshoot"
	CaveatForecasterUndershoots   = "forecaster_tends_to_undershoot"
)

// CalibrationDataPoint is a single (predicted, observed) pair read from
// .sageo/calibration.json. This mirrors the on-disk schema written by
// internal/compare.AppendCalibration; we don't import that package to
// avoid an import cycle (compare -> recommendations -> forecast).
type CalibrationDataPoint struct {
	RecommendationID    string `json:"recommendation_id"`
	ChangeType          string `json:"change_type"`
	TargetURL           string `json:"target_url"`
	TargetQuery         string `json:"target_query,omitempty"`
	PredictedLiftClicks int    `json:"predicted_lift_clicks"`
	ObservedLiftClicks  int    `json:"observed_lift_clicks"`
	WindowDays          int    `json:"window_days"`
	SnapshotFrom        string `json:"snapshot_from"`
	SnapshotTo          string `json:"snapshot_to"`
}

// calibrationFileFormat is the on-disk envelope. Matches
// internal/compare.CalibrationStore.
type calibrationFileFormat struct {
	Version    int                    `json:"version"`
	DataPoints []CalibrationDataPoint `json:"data_points"`
}

// CalibrationStats summarises the observed/predicted ratio over a set of
// historical data points.
type CalibrationStats struct {
	SampleSize        int     `json:"sample_size"`
	MedianRatio       float64 `json:"median_ratio"`
	P25Ratio          float64 `json:"p25_ratio"`
	P75Ratio          float64 `json:"p75_ratio"`
	MeanAbsoluteError float64 `json:"mean_absolute_error"`
	// Bias is a human-readable summary: "overshoots" (median < 0.85),
	// "undershoots" (median > 1.15), "calibrated" (in between), or
	// "insufficient_data" when SampleSize is below the relevant floor.
	Bias string `json:"bias"`
}

// CalibrationProfile is the full picture: per-ChangeType stats plus an
// overall fallback.
type CalibrationProfile struct {
	PerChangeType map[string]CalibrationStats `json:"per_change_type"`
	Overall       CalibrationStats            `json:"overall"`
	SampleSize    int                         `json:"sample_size"`
	LastUpdated   time.Time                   `json:"last_updated"`
}

// LoadCalibrationProfile reads baseDir/.sageo/calibration.json and
// computes a CalibrationProfile. Returns (nil, nil) when the file does
// not exist — that's the expected state for a fresh project and should
// not be treated as an error.
func LoadCalibrationProfile(baseDir string) (*CalibrationProfile, error) {
	path := filepath.Join(baseDir, state.DirName, "calibration.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("forecast: read %s: %w", path, err)
	}
	var store calibrationFileFormat
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("forecast: parse %s: %w", path, err)
	}
	return buildProfile(store.DataPoints), nil
}

// BuildProfile constructs a CalibrationProfile from an in-memory slice
// of data points. Exported for tests and for callers that want to work
// from calibration data sourced elsewhere (e.g. a test fixture).
func BuildProfile(points []CalibrationDataPoint) *CalibrationProfile {
	return buildProfile(points)
}

func buildProfile(points []CalibrationDataPoint) *CalibrationProfile {
	profile := &CalibrationProfile{
		PerChangeType: map[string]CalibrationStats{},
		SampleSize:    0,
		LastUpdated:   time.Now().UTC(),
	}

	// Partition by ChangeType; also keep all usable points for the
	// overall stats.
	byType := map[string][]CalibrationDataPoint{}
	var all []CalibrationDataPoint
	for _, p := range points {
		// Skip points where the predicted value is zero — dividing by
		// zero gives us no signal. A predicted-zero/observed-N pair is
		// interesting in aggregate ("we said nothing would happen and
		// something did") but not for multiplicative bias correction.
		if p.PredictedLiftClicks <= 0 {
			continue
		}
		byType[p.ChangeType] = append(byType[p.ChangeType], p)
		all = append(all, p)
	}

	profile.SampleSize = len(all)
	profile.Overall = statsFrom(all, MinSampleOverall)
	for ct, pts := range byType {
		profile.PerChangeType[ct] = statsFrom(pts, MinSamplePerChangeType)
	}
	return profile
}

// statsFrom computes CalibrationStats for a set of data points. When the
// sample size is below minSample, Bias is set to "insufficient_data"
// and the numeric fields stay at their neutral "no adjustment" values
// (median ratio = 1, spread = ±0).
func statsFrom(points []CalibrationDataPoint, minSample int) CalibrationStats {
	n := len(points)
	if n == 0 {
		return CalibrationStats{Bias: "insufficient_data"}
	}

	ratios := make([]float64, 0, n)
	var absErrSum float64
	for _, p := range points {
		ratio := float64(p.ObservedLiftClicks) / float64(p.PredictedLiftClicks)
		ratios = append(ratios, ratio)
		absErrSum += math.Abs(float64(p.ObservedLiftClicks - p.PredictedLiftClicks))
	}
	sort.Float64s(ratios)

	stats := CalibrationStats{
		SampleSize:        n,
		MedianRatio:       percentile(ratios, 0.50),
		P25Ratio:          percentile(ratios, 0.25),
		P75Ratio:          percentile(ratios, 0.75),
		MeanAbsoluteError: absErrSum / float64(n),
	}

	switch {
	case n < minSample:
		stats.Bias = "insufficient_data"
	case stats.MedianRatio < 0.85:
		stats.Bias = "overshoots"
	case stats.MedianRatio > 1.15:
		stats.Bias = "undershoots"
	default:
		stats.Bias = "calibrated"
	}
	return stats
}

// percentile returns the value at the given fractional rank (0..1) of a
// pre-sorted slice using linear interpolation. This is the "type 7"
// definition used by R/NumPy's default.
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[n-1]
	}
	pos := p * float64(n-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	frac := pos - float64(lo)
	return sorted[lo] + frac*(sorted[hi]-sorted[lo])
}

// Adjust applies a CalibrationProfile to a raw Forecast, populating
// CalibratedEstimate / CalibratedConfidenceLow / CalibratedConfidenceHigh
// and the human-facing ConfidenceLabel / Caveats / CalibrationSamples /
// PriorityTier fields.
//
// Behaviour:
//
//  1. If the profile is nil or the overall sample size is below
//     MinSampleOverall, the raw estimate is returned with wide bounds
//     (raw ±60%), ConfidenceLabel = "insufficient_data", and the
//     "insufficient_calibration_data" caveat.
//  2. If the per-ChangeType sample size is at least
//     MinSamplePerChangeType, per-ChangeType stats are used.
//  3. Otherwise the overall profile is used and ConfidenceLabel is
//     "moderate_confidence".
//
// Tier is always populated — even when calibration is insufficient —
// based on the raw estimate, so users still get a high/medium/low
// signal without a calibrated number.
func Adjust(raw state.Forecast, profile *CalibrationProfile, changeType string) state.Forecast {
	out := raw // copy
	out.Caveats = append([]string(nil), raw.Caveats...)

	// No calibration data at all — ship raw with wide bounds and an
	// explicit caveat. The priority tier is still computed off the raw
	// estimate so users get *some* signal.
	if profile == nil || profile.SampleSize < MinSampleOverall {
		out.ConfidenceLabel = ConfidenceInsufficient
		out.CalibrationSamples = 0
		if profile != nil {
			out.CalibrationSamples = profile.SampleSize
		}
		out.Caveats = appendUnique(out.Caveats, CaveatInsufficientCalibration)
		// Widen the raw bounds to reflect true uncertainty — the raw
		// ±30% is the forecaster's self-reported noise, not the
		// real-world noise. Without historical evidence we assume the
		// real-world noise is at least ±60%.
		out.RawConfidenceLow = widenLow(raw.RawEstimate, 0.40)
		out.RawConfidenceHigh = widenHigh(raw.RawEstimate, 1.60)
		out.PriorityTier = Tier(raw.RawEstimate, ConfidenceInsufficient)
		return out
	}

	stats, label := selectStats(profile, changeType)
	out.CalibrationSamples = stats.SampleSize
	out.ConfidenceLabel = label

	// Apply the multiplicative correction.
	point := int(math.Round(float64(raw.RawEstimate) * stats.MedianRatio))
	low := int(math.Round(float64(raw.RawEstimate) * stats.P25Ratio))
	high := int(math.Round(float64(raw.RawEstimate) * stats.P75Ratio))
	// Guard against degenerate inversions when the ratios carry noise.
	if low > point {
		low = point
	}
	if high < point {
		high = point
	}
	if low < 0 {
		low = 0
	}

	out.CalibratedEstimate = &point
	out.CalibratedConfidenceLow = &low
	out.CalibratedConfidenceHigh = &high

	switch stats.Bias {
	case "overshoots":
		out.Caveats = appendUnique(out.Caveats, CaveatForecasterOvershoots)
	case "undershoots":
		out.Caveats = appendUnique(out.Caveats, CaveatForecasterUndershoots)
	}
	if label == ConfidenceLow {
		out.Caveats = appendUnique(out.Caveats, CaveatLowConfidence)
	}

	out.PriorityTier = Tier(point, label)
	return out
}

// selectStats picks the best calibration stats for a ChangeType and
// reports the confidence label that should accompany the adjustment.
func selectStats(profile *CalibrationProfile, changeType string) (CalibrationStats, string) {
	if changeType != "" {
		if s, ok := profile.PerChangeType[changeType]; ok && s.SampleSize >= MinSamplePerChangeType {
			label := ConfidenceHigh
			if s.SampleSize < MinSamplePerChangeType*2 {
				label = ConfidenceModerate
			}
			return s, label
		}
	}
	// Fall back to the overall profile.
	if profile.Overall.SampleSize >= MinSampleOverall*2 {
		return profile.Overall, ConfidenceModerate
	}
	return profile.Overall, ConfidenceLow
}

// Tier buckets a (point estimate, confidence label) into a PriorityTier.
// When the label signals we don't have enough data, we return
// PriorityUnknown for tiny estimates but still bucket non-trivial ones —
// even an uncalibrated forecast of 400 clicks/mo is probably "high"
// regardless of calibration confidence, we just won't stand behind the
// specific number.
func Tier(point int, label string) state.PriorityTier {
	if label == ConfidenceInsufficient && point < TierHighThreshold {
		return state.PriorityUnknown
	}
	switch {
	case point > TierHighThreshold:
		return state.PriorityHigh
	case point >= TierMediumThreshold:
		return state.PriorityMedium
	default:
		return state.PriorityLow
	}
}

// widenLow returns the lower bound of a widened range as an int, floored
// at zero.
func widenLow(point int, factor float64) int {
	v := int(math.Round(float64(point) * factor))
	if v < 0 {
		return 0
	}
	return v
}

func widenHigh(point int, factor float64) int {
	return int(math.Round(float64(point) * factor))
}

func appendUnique(xs []string, s string) []string {
	for _, x := range xs {
		if x == s {
			return xs
		}
	}
	return append(xs, s)
}
