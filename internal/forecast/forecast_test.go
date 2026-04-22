package forecast

import (
	"math"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

func TestEstimate_Position10To1(t *testing.T) {
	SetCurve([10]float64{}) // reset to default
	in := Input{
		CurrentPosition:     10,
		TargetPosition:      1,
		MonthlySearchVolume: 1000,
	}
	got := Estimate(in)

	// Default curve: pos1=39.8%, pos10=1.6%. Delta = (0.398 - 0.016)*1000 = 382.
	want := 382
	if diff := math.Abs(float64(got.EstimatedMonthlyClicksDelta - want)); diff > 2 {
		t.Fatalf("pos10→pos1 delta = %d, want ~%d", got.EstimatedMonthlyClicksDelta, want)
	}
	if got.Method != "awr_2024_curve" {
		t.Errorf("Method = %q, want awr_2024_curve", got.Method)
	}
}

func TestEstimate_ConfidenceBounds(t *testing.T) {
	SetCurve([10]float64{})
	in := Input{CurrentPosition: 8, TargetPosition: 3, MonthlySearchVolume: 5000}
	got := Estimate(in)

	wantLow := int(math.Round(float64(got.EstimatedMonthlyClicksDelta) * 0.70 / 1.0))
	wantHigh := int(math.Round(float64(got.EstimatedMonthlyClicksDelta) * 1.30 / 1.0))
	// Allow ±1 for rounding drift between direct-from-delta vs direct-from-raw.
	if abs(got.ConfidenceLow-wantLow) > 1 {
		t.Errorf("low bound %d not within ±1 of 0.7*point (%d)", got.ConfidenceLow, wantLow)
	}
	if abs(got.ConfidenceHigh-wantHigh) > 1 {
		t.Errorf("high bound %d not within ±1 of 1.3*point (%d)", got.ConfidenceHigh, wantHigh)
	}
}

func TestEstimate_NoVolumeReturnsZero(t *testing.T) {
	got := Estimate(Input{CurrentPosition: 5, TargetPosition: 2})
	if got.EstimatedMonthlyClicksDelta != 0 || got.ConfidenceLow != 0 || got.ConfidenceHigh != 0 {
		t.Errorf("expected zero forecast when volume is 0, got %+v", got)
	}
}

func TestEstimate_SamePositionAppliesCTRUplift(t *testing.T) {
	SetCurve([10]float64{})
	// Title rewrite case: target == current, observed CTR 5%, volume 1000.
	// Uplift = 30% => extra clicks = 0.05 * 0.30 * 1000 = 15.
	got := Estimate(Input{
		CurrentPosition:     5,
		TargetPosition:      5,
		MonthlySearchVolume: 1000,
		CurrentCTR:          0.05,
	})
	if got.EstimatedMonthlyClicksDelta < 13 || got.EstimatedMonthlyClicksDelta > 17 {
		t.Errorf("CTR uplift delta = %d, want ~15", got.EstimatedMonthlyClicksDelta)
	}
}

func TestSetCurve_Swappable(t *testing.T) {
	custom := [10]float64{50, 25, 10, 5, 2, 1, 0.5, 0.25, 0.1, 0.05}
	SetCurve(custom)
	defer SetCurve([10]float64{})

	if got := CurrentCurve(); got != custom {
		t.Fatalf("CurrentCurve = %v, want %v", got, custom)
	}
	// Volume 100, pos2→pos1: (0.50 - 0.25) * 100 = 25.
	got := Estimate(Input{CurrentPosition: 2, TargetPosition: 1, MonthlySearchVolume: 100})
	if got.EstimatedMonthlyClicksDelta != 25 {
		t.Errorf("custom curve delta = %d, want 25", got.EstimatedMonthlyClicksDelta)
	}
}

func TestTargetPositionFor(t *testing.T) {
	mk := func(ct state.ChangeType, pos float64) state.Recommendation {
		return state.Recommendation{
			ChangeType: ct,
			Evidence: []state.Evidence{
				{Metric: "position", Value: pos},
			},
		}
	}

	cases := []struct {
		name string
		rec  state.Recommendation
		want float64
	}{
		{"title keeps position", mk(state.ChangeTitle, 5), 5},
		{"meta keeps position", mk(state.ChangeMeta, 7), 7},
		{"body on page2 targets pos3", mk(state.ChangeBody, 12), 3},
		{"schema on pos6 targets pos3", mk(state.ChangeSchema, 6), 3},
		{"body on pos2 unchanged", mk(state.ChangeBody, 2), 2},
		{"speed improves by 1", mk(state.ChangeSpeed, 5), 4},
		{"speed at pos1 unchanged", mk(state.ChangeSpeed, 1), 1},
		{"internal link unchanged", mk(state.ChangeInternalLink, 8), 8},
		// AI-citation levers: modelled as CTR-only uplifts — target
		// position equals current (research doc "Add" + signals matrix).
		{"tldr keeps position", mk(state.ChangeTLDR, 4), 4},
		{"list format keeps position", mk(state.ChangeListFormat, 6), 6},
		{"author byline keeps position", mk(state.ChangeAuthorByline, 9), 9},
		{"freshness keeps position", mk(state.ChangeFreshness, 3), 3},
		{"entity consistency keeps position", mk(state.ChangeEntityConsistency, 12), 12},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := TargetPositionFor(tc.rec)
			if got != tc.want {
				t.Errorf("TargetPositionFor = %v, want %v", got, tc.want)
			}
		})
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
