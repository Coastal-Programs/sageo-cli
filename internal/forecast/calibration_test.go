package forecast

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

// makePoints builds n synthetic data points for changeType where the
// observed lift equals predicted * ratio — ideal for checking that the
// median ratio round-trips through buildProfile.
func makePoints(changeType string, n int, predicted int, ratio float64) []CalibrationDataPoint {
	out := make([]CalibrationDataPoint, n)
	for i := 0; i < n; i++ {
		out[i] = CalibrationDataPoint{
			ChangeType:          changeType,
			PredictedLiftClicks: predicted,
			ObservedLiftClicks:  int(float64(predicted) * ratio),
		}
	}
	return out
}

func TestBuildProfile_BasicStats(t *testing.T) {
	// 60 points with observed = 0.7 * predicted → median ratio = 0.7.
	pts := makePoints("title", 60, 100, 0.7)
	p := BuildProfile(pts)

	if p.SampleSize != 60 {
		t.Errorf("SampleSize = %d, want 60", p.SampleSize)
	}
	if got := p.Overall.MedianRatio; got < 0.69 || got > 0.71 {
		t.Errorf("Overall MedianRatio = %v, want ~0.70", got)
	}
	if p.Overall.Bias != "overshoots" {
		t.Errorf("Overall Bias = %q, want overshoots (ratio 0.70 < 0.85)", p.Overall.Bias)
	}
	s := p.PerChangeType["title"]
	if s.SampleSize != 60 {
		t.Errorf("PerChangeType[title].SampleSize = %d, want 60", s.SampleSize)
	}
}

func TestAdjust_InsufficientOverall_ReturnsRawWithWideBounds(t *testing.T) {
	pts := makePoints("title", 3, 100, 0.7) // way under MinSampleOverall
	p := BuildProfile(pts)

	raw := state.Forecast{RawEstimate: 100, RawConfidenceLow: 70, RawConfidenceHigh: 130}
	out := Adjust(raw, p, "title")

	if out.CalibratedEstimate != nil {
		t.Error("expected no CalibratedEstimate when overall sample is below threshold")
	}
	if out.ConfidenceLabel != ConfidenceInsufficient {
		t.Errorf("ConfidenceLabel = %q, want %q", out.ConfidenceLabel, ConfidenceInsufficient)
	}
	if !contains(out.Caveats, CaveatInsufficientCalibration) {
		t.Errorf("expected %s caveat, got %v", CaveatInsufficientCalibration, out.Caveats)
	}
	// Bounds should widen — raw ±60%.
	if out.RawConfidenceLow > 50 || out.RawConfidenceHigh < 150 {
		t.Errorf("expected widened bounds (raw 100, ≥±60%%), got [%d..%d]",
			out.RawConfidenceLow, out.RawConfidenceHigh)
	}
}

func TestAdjust_NilProfile(t *testing.T) {
	raw := state.Forecast{RawEstimate: 500, RawConfidenceLow: 350, RawConfidenceHigh: 650}
	out := Adjust(raw, nil, "title")
	if out.ConfidenceLabel != ConfidenceInsufficient {
		t.Errorf("nil profile -> ConfidenceLabel = %q, want insufficient", out.ConfidenceLabel)
	}
	if out.PriorityTier != state.PriorityHigh {
		t.Errorf("tier on 500-click raw = %q, want high", out.PriorityTier)
	}
}

func TestAdjust_PerChangeTypeIsolation(t *testing.T) {
	// Mix: 30 title points with ratio 0.5, 30 body points with ratio 2.0.
	var pts []CalibrationDataPoint
	pts = append(pts, makePoints("title", 30, 100, 0.5)...)
	pts = append(pts, makePoints("body_expand", 30, 100, 2.0)...)
	p := BuildProfile(pts)

	// 60 total ≥ MinSampleOverall (50), so overall kicks in.
	// Per-ChangeType: 30 each ≥ MinSamplePerChangeType (20) so
	// per-type is used.
	rawTitle := state.Forecast{RawEstimate: 100}
	outTitle := Adjust(rawTitle, p, "title")
	if outTitle.CalibratedEstimate == nil {
		t.Fatal("expected title calibrated estimate")
	}
	// Median 0.5 * 100 = 50.
	if *outTitle.CalibratedEstimate < 45 || *outTitle.CalibratedEstimate > 55 {
		t.Errorf("title calibrated = %d, want ~50", *outTitle.CalibratedEstimate)
	}

	rawBody := state.Forecast{RawEstimate: 100}
	outBody := Adjust(rawBody, p, "body_expand")
	if outBody.CalibratedEstimate == nil {
		t.Fatal("expected body calibrated estimate")
	}
	// Median 2.0 * 100 = 200.
	if *outBody.CalibratedEstimate < 190 || *outBody.CalibratedEstimate > 210 {
		t.Errorf("body calibrated = %d, want ~200", *outBody.CalibratedEstimate)
	}
}

func TestAdjust_FallsBackToOverallWhenPerTypeTooFew(t *testing.T) {
	// 50 title points, zero body points. Title satisfies per-type;
	// body must fall back to overall.
	pts := makePoints("title", 50, 100, 0.5)
	p := BuildProfile(pts)

	out := Adjust(state.Forecast{RawEstimate: 100}, p, "ChangeBody_unseen")
	if out.CalibratedEstimate == nil {
		t.Fatal("expected overall fallback to adjust forecast")
	}
	// Overall ratio ≈ 0.5 since title is the only contributor.
	if *out.CalibratedEstimate < 45 || *out.CalibratedEstimate > 55 {
		t.Errorf("fallback calibrated = %d, want ~50", *out.CalibratedEstimate)
	}
	// Samples = overall size.
	if out.CalibrationSamples != 50 {
		t.Errorf("CalibrationSamples = %d, want 50", out.CalibrationSamples)
	}
}

func TestAdjust_WellCalibratedProfile(t *testing.T) {
	// 300 points, observed == predicted (ratio 1.0). Expect
	// "calibrated" bias and no overshoot/undershoot caveat.
	pts := makePoints("title", 300, 200, 1.0)
	p := BuildProfile(pts)

	out := Adjust(state.Forecast{RawEstimate: 200}, p, "title")
	if out.CalibratedEstimate == nil || *out.CalibratedEstimate < 195 || *out.CalibratedEstimate > 205 {
		t.Errorf("expected calibrated ≈ 200, got %v", out.CalibratedEstimate)
	}
	if contains(out.Caveats, CaveatForecasterOvershoots) || contains(out.Caveats, CaveatForecasterUndershoots) {
		t.Errorf("well-calibrated profile should not produce bias caveats, got %v", out.Caveats)
	}
}

func TestLoadCalibrationProfile_MissingFileReturnsNil(t *testing.T) {
	dir := t.TempDir()
	p, err := LoadCalibrationProfile(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Errorf("expected nil profile for missing file, got %+v", p)
	}
}

func TestLoadCalibrationProfile_ReadsFixture(t *testing.T) {
	dir := t.TempDir()
	sageoDir := filepath.Join(dir, state.DirName)
	if err := os.MkdirAll(sageoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	store := calibrationFileFormat{
		Version:    1,
		DataPoints: makePoints("title", 60, 100, 0.6),
	}
	body, _ := json.Marshal(store)
	if err := os.WriteFile(filepath.Join(sageoDir, "calibration.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := LoadCalibrationProfile(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if p == nil || p.SampleSize != 60 {
		t.Fatalf("expected 60-point profile, got %+v", p)
	}
	if p.Overall.Bias != "overshoots" {
		t.Errorf("Bias = %q, want overshoots", p.Overall.Bias)
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
