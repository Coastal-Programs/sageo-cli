package compare

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

// CalibrationFile is the filename under .sageo/ that accumulates
// (predicted, observed) pairs across every `sageo compare` invocation.
const CalibrationFile = "calibration.json"

// CalibrationStore is the on-disk append-only format. A new data point is
// added every time compare can pair a recommendation's forecasted lift
// with an observed outcome. Calibration (a separate task) reads this file
// to compute bias and shrink the forecaster's confidence intervals.
type CalibrationStore struct {
	Version    int                    `json:"version"`
	DataPoints []CalibrationDataPoint `json:"data_points"`
}

// CalibrationDataPoint is a single (predicted, observed) pair. All fields
// are populated from data the comparison already has — we never invent
// metrics here.
type CalibrationDataPoint struct {
	RecommendationID       string  `json:"recommendation_id"`
	ChangeType             string  `json:"change_type"`
	TargetURL              string  `json:"target_url"`
	TargetQuery            string  `json:"target_query,omitempty"`
	PredictedLiftClicks    int     `json:"predicted_lift_clicks"`
	PredictedLiftLow       int     `json:"predicted_lift_low,omitempty"`
	PredictedLiftHigh      int     `json:"predicted_lift_high,omitempty"`
	ObservedLiftClicks     int     `json:"observed_lift_clicks"`
	ObservedPositionDelta  float64 `json:"observed_position_delta,omitempty"`
	ObservedCitationsDelta int     `json:"observed_citations_delta,omitempty"`
	WindowDays             int     `json:"window_days"`
	SnapshotFrom           string  `json:"snapshot_from"`
	SnapshotTo             string  `json:"snapshot_to"`
	RecordedAt             string  `json:"recorded_at"`
	ConfidenceNote         string  `json:"confidence_note,omitempty"`
}

// AppendCalibration writes one data point per addressed-and-measurable
// recommendation to baseDir/.sageo/calibration.json. Existing points are
// preserved (append-only) and a version field is maintained so future
// schema changes can migrate forward.
//
// Returns the number of new data points appended.
func AppendCalibration(baseDir string, c *Comparison) (int, error) {
	if c == nil {
		return 0, nil
	}
	var points []CalibrationDataPoint
	for _, o := range c.Recs.Addressed {
		if o.ObservedLift == nil {
			continue
		}
		rec := o.Recommendation
		if rec.ForecastedLift == nil {
			// No prediction to calibrate against; still useful
			// to record the observation but skip to keep the
			// store focused on paired points.
			continue
		}
		pt := CalibrationDataPoint{
			RecommendationID:       rec.ID,
			ChangeType:             string(rec.ChangeType),
			TargetURL:              rec.TargetURL,
			TargetQuery:            rec.TargetQuery,
			PredictedLiftClicks:    rec.ForecastedLift.RawEstimate,
			PredictedLiftLow:       rec.ForecastedLift.RawConfidenceLow,
			PredictedLiftHigh:      rec.ForecastedLift.RawConfidenceHigh,
			ObservedLiftClicks:     o.ObservedLift.ClicksDelta,
			ObservedPositionDelta:  o.ObservedLift.PositionDelta,
			ObservedCitationsDelta: o.ObservedLift.CitationsDelta,
			WindowDays:             o.ObservedLift.WindowDays,
			SnapshotFrom:           formatSnapshotRef(c.From),
			SnapshotTo:             formatSnapshotRef(c.To),
			RecordedAt:             time.Now().UTC().Format(time.RFC3339),
			ConfidenceNote:         o.ObservedLift.ConfidenceNote,
		}
		points = append(points, pt)
	}
	if len(points) == 0 {
		return 0, nil
	}

	path := filepath.Join(baseDir, state.DirName, CalibrationFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, fmt.Errorf("compare: mkdir .sageo: %w", err)
	}

	store := CalibrationStore{Version: 1}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &store); err != nil {
			return 0, fmt.Errorf("compare: parse %s: %w", CalibrationFile, err)
		}
		if store.Version == 0 {
			store.Version = 1
		}
	} else if !os.IsNotExist(err) {
		return 0, fmt.Errorf("compare: read %s: %w", CalibrationFile, err)
	}

	store.DataPoints = append(store.DataPoints, points...)

	// Write-temp-then-rename so a crash mid-write never corrupts the
	// existing store.
	body, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return 0, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return 0, err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return 0, err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return 0, err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return 0, err
	}
	return len(points), nil
}

func formatSnapshotRef(s *state.Snapshot) string {
	if s == nil {
		return ""
	}
	return state.FormatSnapshotTimestamp(s.Timestamp)
}
