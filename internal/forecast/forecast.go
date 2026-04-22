// Package forecast estimates the additional monthly clicks a
// recommendation is likely to deliver if applied.
//
// The core model is a position→CTR curve applied to a monthly search
// volume: current clicks = volume * ctr(current_position), projected
// clicks = volume * ctr(target_position). The delta is the forecast,
// with ±30% confidence bounds to reflect the inherent uncertainty of
// search ranking projections.
package forecast

import (
	"math"
	"sync"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

// defaultCurve is the Advanced Web Ranking 2024 public click-through-rate
// curve for positions 1–10, expressed as percentages (e.g. 39.8 = 39.8%).
// Source: https://www.advancedwebranking.com/ctrstudy/ (2024 desktop+mobile
// blended baseline). Positions 11+ follow a linear decay handled in ctrAt().
var defaultCurve = [10]float64{
	39.8, 18.7, 10.2, 7.2, 5.1, 4.4, 3.0, 2.1, 1.9, 1.6,
}

var (
	curveMu sync.RWMutex
	curve   = defaultCurve
)

// SetCurve overrides the built-in position→CTR curve. Intended for tests
// and for swapping in a site-specific or freshly-measured curve. Passing
// the zero value restores the baseline.
func SetCurve(c [10]float64) {
	curveMu.Lock()
	defer curveMu.Unlock()
	if c == ([10]float64{}) {
		curve = defaultCurve
		return
	}
	curve = c
}

// CurrentCurve returns a copy of the active curve for inspection.
func CurrentCurve() [10]float64 {
	curveMu.RLock()
	defer curveMu.RUnlock()
	return curve
}

// Input holds the signals needed to forecast click delta for a single
// recommendation on a single URL+query pair.
type Input struct {
	CurrentPosition     float64
	TargetPosition      float64
	MonthlySearchVolume int
	CurrentCTR          float64 // observed CTR as a fraction (0–1); optional, used for CTR-only uplifts
	DeviceMix           string  // "desktop" | "mobile" | "" (blended)
}

// ctrAt returns the expected CTR (as a fraction 0–1) for a given search
// position using the active curve. Non-integer positions are linearly
// interpolated between adjacent integer positions. Positions beyond 10
// decay linearly toward zero at position 30, matching the tail behaviour
// observed in the AWR data.
func ctrAt(position float64) float64 {
	curveMu.RLock()
	c := curve
	curveMu.RUnlock()

	if position < 1 {
		position = 1
	}
	// Positions 1–10: linear interpolation between table entries.
	if position <= 10 {
		lo := int(math.Floor(position))
		hi := int(math.Ceil(position))
		if lo < 1 {
			lo = 1
		}
		if hi > 10 {
			hi = 10
		}
		loVal := c[lo-1] / 100.0
		hiVal := c[hi-1] / 100.0
		if lo == hi {
			return loVal
		}
		frac := position - float64(lo)
		return loVal + (hiVal-loVal)*frac
	}

	// Positions 11+: linear decay from CTR(10) to 0 at position 30.
	pos10 := c[9] / 100.0
	if position >= 30 {
		return 0
	}
	slope := pos10 / 20.0 // drops to 0 over 20 positions (10→30)
	return pos10 - slope*(position-10)
}

// Estimate computes the forecasted monthly click delta for a single
// recommendation given its Input signals. A negative or zero delta is
// clamped to zero — recommendations should not be presented as harmful.
// Confidence bounds are fixed at ±30% of the point estimate to reflect
// typical variance in search forecasting.
func Estimate(in Input) state.Forecast {
	if in.MonthlySearchVolume <= 0 {
		return state.Forecast{Method: "awr_2024_curve"}
	}

	vol := float64(in.MonthlySearchVolume)

	// Baseline clicks at the current position. If the caller supplied a
	// measured CTR we prefer it over the curve — it captures page-specific
	// factors like rich snippets or title quality.
	currentCTR := ctrAt(in.CurrentPosition)
	if in.CurrentCTR > 0 {
		currentCTR = in.CurrentCTR
	}
	currentClicks := vol * currentCTR

	// Projected clicks at the target position, using the curve.
	targetCTR := ctrAt(in.TargetPosition)

	// Special case: when target position equals current position the caller
	// is asking about a CTR uplift (e.g. title rewrite). Apply a 30% uplift
	// to the observed CTR rather than using the curve, since the curve
	// cannot distinguish "better snippet at position 5" from "average at 5".
	if math.Abs(in.TargetPosition-in.CurrentPosition) < 0.001 {
		targetCTR = currentCTR * 1.30
	}

	projectedClicks := vol * targetCTR
	delta := projectedClicks - currentClicks
	if delta < 0 {
		delta = 0
	}

	point := int(math.Round(delta))
	low := int(math.Round(delta * 0.70))
	high := int(math.Round(delta * 1.30))

	return state.Forecast{
		RawEstimate:       point,
		RawConfidenceLow:  low,
		RawConfidenceHigh: high,
		Method:            "awr_2024_curve",
	}
}

// TargetPositionFor returns the modelled target search position for a
// recommendation based on its ChangeType and current position.
//
// Heuristics (classic search lift):
//   - ChangeTitle / ChangeMeta: the SERP snippet changes but the ranking
//     position typically does not move in the short term. Target position
//     equals current position; Estimate interprets this as a CTR uplift.
//   - ChangeBody / ChangeSchema: content depth and structured data are
//     well-established ranking signals. For pages on page 1 bottom / page 2
//     (positions 4–20) the realistic ceiling is a move into the top 3.
//   - ChangeSpeed: Core Web Vitals are a minor but real ranking factor.
//     Model a one-position improvement as a conservative upper bound.
//
// Heuristics (AI-citation lift, per
// docs/research/ai-citation-signals-2026.md):
//   - ChangeTLDR: Direct-answer blocks recapture clicks from AI Overviews
//     rather than move SERP rank. Modelled as a CTR-only uplift (target
//     position = current), which Estimate applies as the default 30%
//     CTR bump. This is deliberately the same mechanic as ChangeTitle but
//     distinct in rationale — on an AI-Overview-eaten query, a TL;DR can
//     lift observed CTR meaningfully even without rank change.
//   - ChangeListFormat / ChangeFreshness / ChangeAuthorByline /
//     ChangeEntityConsistency: trust and extraction signals that affect
//     AI citation eligibility more than classic rank. Modelled as CTR-only
//     uplifts (target = current) rather than rank movements.
//
// All other change types return the current position unchanged, which
// causes Estimate to model them as CTR-only uplifts when paired with the
// default 30% uplift, or as no-ops if the caller chooses not to forecast
// them.
func TargetPositionFor(rec state.Recommendation) float64 {
	current := currentPosition(rec)

	switch rec.ChangeType {
	case state.ChangeTitle, state.ChangeMeta:
		return current
	case state.ChangeBody, state.ChangeSchema:
		if current >= 4 && current <= 20 {
			return 3
		}
		return current
	case state.ChangeSpeed:
		if current > 1 {
			return current - 1
		}
		return current
	case state.ChangeTLDR,
		state.ChangeListFormat,
		state.ChangeFreshness,
		state.ChangeAuthorByline,
		state.ChangeEntityConsistency:
		// AI-citation levers — modelled as CTR-only uplift.
		return current
	default:
		return current
	}
}

// currentPosition extracts a current position from the recommendation's
// evidence when available. It looks for an Evidence entry with
// Metric == "position" and a numeric Value. Returns 0 if no position
// evidence is present.
func currentPosition(rec state.Recommendation) float64 {
	for _, ev := range rec.Evidence {
		if ev.Metric != "position" {
			continue
		}
		switch v := ev.Value.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0
}
