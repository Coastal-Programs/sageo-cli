package recommendations

import (
	"log"
	"strings"

	"github.com/jakeschepis/sageo-cli/internal/common/urlnorm"
	"github.com/jakeschepis/sageo-cli/internal/forecast"
	"github.com/jakeschepis/sageo-cli/internal/state"
)

// AttachForecasts populates ForecastedLift on each recommendation in recs
// by combining the target position heuristic from the forecast package
// with search-volume and current-position signals pulled from st, then
// applying the calibration profile (if any) from .sageo/calibration.json.
//
// Signal lookup order:
//  1. GSC TopKeywords row matching TargetQuery (best — gives observed
//     CTR, position, and impressions for that query).
//  2. GSC TopPages row matching TargetURL (URL-normalised) — supplies
//     position and impression volume.
//  3. Labs keyword entry matching TargetQuery — supplies search volume.
//
// When none of these provide a usable search volume, ForecastedLift is
// left nil and a warning is logged; this is not an error condition.
func AttachForecasts(st *state.State, recs []Recommendation) {
	AttachForecastsWithBaseDir(st, recs, ".")
}

// AttachForecastsWithBaseDir is AttachForecasts with an explicit base
// directory for loading the calibration profile. Exposed for tests.
func AttachForecastsWithBaseDir(st *state.State, recs []Recommendation, baseDir string) {
	if st == nil {
		return
	}

	// Best-effort calibration profile. A missing or malformed file is
	// logged but not fatal — forecasts still ship, just without
	// historical calibration.
	profile, err := forecast.LoadCalibrationProfile(baseDir)
	if err != nil {
		log.Printf("forecast: calibration profile unavailable: %v", err)
		profile = nil
	}

	for i := range recs {
		rec := &recs[i]

		volume, currentPos, currentCTR := lookupSignals(st, rec)
		if volume <= 0 {
			log.Printf("forecast: no search-volume signal for %s (%s) — skipping", rec.TargetURL, rec.ChangeType)
			rec.ForecastedLift = nil
			continue
		}

		target := forecast.TargetPositionFor(*rec)
		if currentPos > 0 && (rec.ChangeType == ChangeTitle || rec.ChangeType == ChangeMeta) {
			target = currentPos
		}

		in := forecast.Input{
			CurrentPosition:     currentPos,
			TargetPosition:      target,
			MonthlySearchVolume: volume,
			CurrentCTR:          currentCTR,
		}
		raw := forecast.Estimate(in)

		// Pre-populate caveats that Adjust itself can't infer — the
		// "low search volume" signal comes from the input side, not the
		// calibration side.
		if volume < 100 {
			raw.Caveats = append(raw.Caveats, forecast.CaveatLowSearchVolume)
		}

		adjusted := forecast.Adjust(raw, profile, string(rec.ChangeType))
		rec.ForecastedLift = &adjusted
	}
}

// lookupSignals returns (monthlySearchVolume, currentPosition, currentCTR)
// for a recommendation, pulling from GSC first and Labs as a fallback.
func lookupSignals(st *state.State, rec *Recommendation) (int, float64, float64) {
	var volume int
	var pos, ctr float64

	// 1. GSC keyword row for the target query.
	if st.GSC != nil && rec.TargetQuery != "" {
		for _, row := range st.GSC.TopKeywords {
			if strings.EqualFold(row.Key, rec.TargetQuery) {
				volume = int(row.Impressions)
				pos = row.Position
				ctr = row.CTR
				break
			}
		}
	}

	// 2. GSC page row for the target URL — fills in missing position /
	//    volume if keyword lookup didn't resolve.
	if st.GSC != nil && (volume == 0 || pos == 0) {
		normTarget := urlnorm.Normalize(rec.TargetURL)
		for _, row := range st.GSC.TopPages {
			if urlnorm.Normalize(row.Key) == normTarget {
				if volume == 0 {
					volume = int(row.Impressions)
				}
				if pos == 0 {
					pos = row.Position
				}
				if ctr == 0 {
					ctr = row.CTR
				}
				break
			}
		}
	}

	// 3. Labs search volume as a last resort.
	if volume == 0 && st.Labs != nil && rec.TargetQuery != "" {
		for _, kw := range st.Labs.Keywords {
			if strings.EqualFold(kw.Keyword, rec.TargetQuery) {
				volume = kw.SearchVolume
				if pos == 0 && kw.Position > 0 {
					pos = float64(kw.Position)
				}
				break
			}
		}
	}

	return volume, pos, ctr
}
