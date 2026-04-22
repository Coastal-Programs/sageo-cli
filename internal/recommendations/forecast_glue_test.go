package recommendations

import (
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

func TestAttachForecasts_GSCHappyPath(t *testing.T) {
	st := &state.State{
		GSC: &state.GSCData{
			TopKeywords: []state.GSCRow{
				{Key: "buy widgets", Impressions: 2000, Clicks: 40, CTR: 0.02, Position: 8},
			},
		},
	}
	recs := []Recommendation{{
		ID:          "r1",
		TargetURL:   "https://example.com/widgets",
		TargetQuery: "buy widgets",
		ChangeType:  ChangeBody,
		Evidence:    []state.Evidence{{Metric: "position", Value: 8.0}},
	}}

	AttachForecasts(st, recs)

	if recs[0].ForecastedLift == nil {
		t.Fatal("expected ForecastedLift to be populated")
	}
	if recs[0].ForecastedLift.EstimatedMonthlyClicksDelta <= 0 {
		t.Errorf("expected positive click delta, got %d", recs[0].ForecastedLift.EstimatedMonthlyClicksDelta)
	}
	if recs[0].ForecastedLift.ConfidenceLow > recs[0].ForecastedLift.EstimatedMonthlyClicksDelta {
		t.Errorf("confidence low should be ≤ point estimate")
	}
	if recs[0].ForecastedLift.ConfidenceHigh < recs[0].ForecastedLift.EstimatedMonthlyClicksDelta {
		t.Errorf("confidence high should be ≥ point estimate")
	}
}

func TestAttachForecasts_FallbackToLabs(t *testing.T) {
	st := &state.State{
		Labs: &state.LabsData{
			Keywords: []state.LabsKeyword{
				{Keyword: "widgets guide", SearchVolume: 1500, Position: 9},
			},
		},
	}
	recs := []Recommendation{{
		TargetURL:   "https://example.com/guide",
		TargetQuery: "widgets guide",
		ChangeType:  ChangeSchema,
		Evidence:    []state.Evidence{{Metric: "position", Value: 9.0}},
	}}

	AttachForecasts(st, recs)

	if recs[0].ForecastedLift == nil {
		t.Fatal("expected ForecastedLift from Labs fallback")
	}
	if recs[0].ForecastedLift.EstimatedMonthlyClicksDelta <= 0 {
		t.Errorf("expected positive click delta, got %d", recs[0].ForecastedLift.EstimatedMonthlyClicksDelta)
	}
}

func TestAttachForecasts_NoDataLeavesNil(t *testing.T) {
	st := &state.State{}
	recs := []Recommendation{{
		TargetURL:   "https://example.com/x",
		TargetQuery: "no signal",
		ChangeType:  ChangeTitle,
	}}

	AttachForecasts(st, recs)

	if recs[0].ForecastedLift != nil {
		t.Errorf("expected ForecastedLift to stay nil, got %+v", recs[0].ForecastedLift)
	}
}

func TestAttachForecasts_GSCPageFallback(t *testing.T) {
	st := &state.State{
		GSC: &state.GSCData{
			TopPages: []state.GSCRow{
				{Key: "https://www.example.com/widgets/", Impressions: 3000, Clicks: 120, CTR: 0.04, Position: 6},
			},
		},
	}
	recs := []Recommendation{{
		TargetURL:  "https://example.com/widgets",
		ChangeType: ChangeBody,
		Evidence:   []state.Evidence{{Metric: "position", Value: 6.0}},
	}}

	AttachForecasts(st, recs)
	if recs[0].ForecastedLift == nil {
		t.Fatal("expected URL-normalised TopPages match to produce a forecast")
	}
}
