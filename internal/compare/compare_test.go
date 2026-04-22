package compare

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/recommendations"
	"github.com/jakeschepis/sageo-cli/internal/state"
)

// seedProject creates a fresh .sageo project and writes two snapshots with
// the provided states. Returns project dir and the two snapshots.
func seedProject(t *testing.T, fromState, toState *state.State) (string, *state.Snapshot, *state.Snapshot) {
	t.Helper()
	dir := t.TempDir()
	// Initialize state.
	if _, err := state.Init(dir, "https://example.com"); err != nil {
		t.Fatalf("init: %v", err)
	}

	earlier := time.Now().UTC().Add(-30 * 24 * time.Hour)
	later := time.Now().UTC()

	from, err := state.CreateSnapshot(dir, fromState, state.SnapshotMeta{
		StartedAt:   earlier,
		CompletedAt: earlier,
		Outcome:     "success",
	}, nil)
	if err != nil {
		t.Fatalf("from snapshot: %v", err)
	}
	to, err := state.CreateSnapshot(dir, toState, state.SnapshotMeta{
		StartedAt:   later,
		CompletedAt: later,
		Outcome:     "success",
	}, nil)
	if err != nil {
		t.Fatalf("to snapshot: %v", err)
	}
	return dir, from, to
}

func baseState() *state.State {
	return &state.State{
		Site:        "https://example.com",
		Initialized: time.Now().UTC().Format(time.RFC3339),
		Score:       75,
	}
}

func TestCompute_GSCDeltas(t *testing.T) {
	from := baseState()
	from.GSC = &state.GSCData{
		TopKeywords: []state.GSCRow{
			{Key: "gains", Clicks: 10, Impressions: 100, Position: 8},
			{Key: "static", Clicks: 5, Impressions: 50, Position: 5},
			{Key: "losing", Clicks: 20, Impressions: 200, Position: 3},
		},
	}
	to := baseState()
	to.GSC = &state.GSCData{
		TopKeywords: []state.GSCRow{
			{Key: "gains", Clicks: 25, Impressions: 150, Position: 4}, // improved
			{Key: "static", Clicks: 5, Impressions: 50, Position: 5},  // unchanged
			{Key: "new", Clicks: 3, Impressions: 30, Position: 10},    // new query
			// "losing" removed entirely
		},
	}
	_, f, to2 := seedProject(t, from, to)
	c, err := Compute(f, to2)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if c.GSC.TotalQueriesFrom != 3 || c.GSC.TotalQueriesTo != 3 {
		t.Errorf("totals: got %d/%d", c.GSC.TotalQueriesFrom, c.GSC.TotalQueriesTo)
	}
	if len(c.GSC.QueriesGained) != 1 || c.GSC.QueriesGained[0].Query != "new" {
		t.Errorf("queries gained: %+v", c.GSC.QueriesGained)
	}
	if len(c.GSC.QueriesLost) != 1 || c.GSC.QueriesLost[0].Query != "losing" {
		t.Errorf("queries lost: %+v", c.GSC.QueriesLost)
	}
	if len(c.GSC.PositionImproved) != 1 || c.GSC.PositionImproved[0].Query != "gains" {
		t.Errorf("improved: %+v", c.GSC.PositionImproved)
	}
	// Click delta: +15 gains, +3 new, -20 lost = -2
	if c.GSC.ClicksDelta != -2 {
		t.Errorf("clicks delta: got %d want -2", c.GSC.ClicksDelta)
	}
}

func TestCompute_PSI_EntersGoodBand(t *testing.T) {
	from := baseState()
	from.PSI = &state.PSIData{Pages: []state.PSIResult{
		{URL: "https://example.com/slow", Strategy: "mobile", LCP: 4500, CLS: 0.3, PerformanceScore: 0.4},
	}}
	to := baseState()
	to.PSI = &state.PSIData{Pages: []state.PSIResult{
		{URL: "https://example.com/slow", Strategy: "mobile", LCP: 2200, CLS: 0.05, PerformanceScore: 0.85},
	}}
	_, f, to2 := seedProject(t, from, to)
	c, err := Compute(f, to2)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if len(c.PSI.Changed) != 1 {
		t.Fatalf("psi changed: got %d", len(c.PSI.Changed))
	}
	ch := c.PSI.Changed[0]
	if !ch.EnteredGoodBandLCP || !ch.EnteredGoodBandCLS {
		t.Errorf("expected both good bands: %+v", ch)
	}
}

func TestDetectors_TitleCleared(t *testing.T) {
	from := baseState()
	from.Findings = []state.Finding{{Rule: "title-missing", URL: "https://example.com/a"}}
	to := baseState()
	// No title finding on /a anymore.

	rec := recommendations.Recommendation{
		ID:         "r1",
		TargetURL:  "https://example.com/a",
		ChangeType: recommendations.ChangeTitle,
	}
	addressed, ev := detectAddressed(rec, from, to)
	if !addressed {
		t.Fatalf("expected addressed, got %q", ev)
	}
	if !strings.Contains(ev, "title") {
		t.Errorf("evidence should mention title: %q", ev)
	}
}

func TestDetectors_TitleNotCleared(t *testing.T) {
	from := baseState()
	from.Findings = []state.Finding{{Rule: "title-missing", URL: "https://example.com/a"}}
	to := baseState()
	to.Findings = []state.Finding{{Rule: "title-missing", URL: "https://example.com/a"}}

	rec := recommendations.Recommendation{
		ID:         "r1",
		TargetURL:  "https://example.com/a",
		ChangeType: recommendations.ChangeTitle,
	}
	addressed, _ := detectAddressed(rec, from, to)
	if addressed {
		t.Fatal("did not expect addressed when finding still fires")
	}
}

func TestDetectors_UndetectableChangeType(t *testing.T) {
	from, to := baseState(), baseState()
	rec := recommendations.Recommendation{
		ID: "r1", TargetURL: "https://example.com/a",
		ChangeType: recommendations.ChangeTLDR,
	}
	if ok, _ := detectAddressed(rec, from, to); ok {
		t.Fatal("TLDR should be undetectable from persisted state today")
	}
}

func TestCompute_RecommendationsAddressedAndLift(t *testing.T) {
	from := baseState()
	from.GSC = &state.GSCData{TopKeywords: []state.GSCRow{
		{Key: "widgets", Clicks: 50, Impressions: 500, Position: 8},
	}}
	from.Findings = []state.Finding{{Rule: "title-missing", URL: "https://example.com/widgets"}}
	from.Recommendations = []recommendations.Recommendation{{
		ID:          "rec-widgets-title",
		TargetURL:   "https://example.com/widgets",
		TargetQuery: "widgets",
		ChangeType:  recommendations.ChangeTitle,
		ForecastedLift: &recommendations.Forecast{
			RawEstimate:       100,
			RawConfidenceLow:  50,
			RawConfidenceHigh: 200,
		},
	}}

	to := baseState()
	to.GSC = &state.GSCData{TopKeywords: []state.GSCRow{
		{Key: "widgets", Clicks: 130, Impressions: 600, Position: 5},
	}}
	// title-missing gone.

	_, f, to2 := seedProject(t, from, to)
	c, err := Compute(f, to2)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if len(c.Recs.Addressed) != 1 {
		t.Fatalf("expected 1 addressed, got %d", len(c.Recs.Addressed))
	}
	o := c.Recs.Addressed[0]
	if o.ObservedLift == nil {
		t.Fatal("expected ObservedLift populated with GSC data present")
	}
	if o.ObservedLift.ClicksDelta != 80 {
		t.Errorf("clicks delta: got %d want 80", o.ObservedLift.ClicksDelta)
	}
	if o.ObservedLift.PositionDelta != -3 {
		t.Errorf("position delta: got %v want -3", o.ObservedLift.PositionDelta)
	}
}

func TestCompute_ObservedLiftOnlyWhenGSCPresent(t *testing.T) {
	from := baseState()
	from.Findings = []state.Finding{{Rule: "title-missing", URL: "https://example.com/a"}}
	from.Recommendations = []recommendations.Recommendation{{
		ID:          "r1",
		TargetURL:   "https://example.com/a",
		TargetQuery: "a",
		ChangeType:  recommendations.ChangeTitle,
	}}
	to := baseState() // no GSC

	_, f, to2 := seedProject(t, from, to)
	c, err := Compute(f, to2)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if len(c.Recs.Addressed) != 1 {
		t.Fatalf("addressed: %d", len(c.Recs.Addressed))
	}
	if c.Recs.Addressed[0].ObservedLift != nil {
		t.Error("should not populate ObservedLift without GSC data on both sides")
	}
}

func TestAppendCalibration_OnlyPairedPoints(t *testing.T) {
	from := baseState()
	from.GSC = &state.GSCData{TopKeywords: []state.GSCRow{{Key: "a", Clicks: 10, Position: 8}}}
	from.Findings = []state.Finding{{Rule: "title-missing", URL: "https://example.com/a"}}
	from.Recommendations = []recommendations.Recommendation{
		{
			ID: "with-forecast", TargetURL: "https://example.com/a", TargetQuery: "a",
			ChangeType:     recommendations.ChangeTitle,
			ForecastedLift: &recommendations.Forecast{RawEstimate: 20},
		},
		{
			ID: "no-forecast", TargetURL: "https://example.com/a", TargetQuery: "a",
			ChangeType: recommendations.ChangeTitle,
		},
	}
	to := baseState()
	to.GSC = &state.GSCData{TopKeywords: []state.GSCRow{{Key: "a", Clicks: 30, Position: 4}}}

	dir, f, to2 := seedProject(t, from, to)
	c, err := Compute(f, to2)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	n, err := AppendCalibration(dir, c)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	// Both recs collapse to one because detect dedupes, but there's only
	// one finding to clear — only one rec gets "addressed". Still, the
	// point: only recs with ForecastedLift are stored.
	if n < 1 {
		t.Fatalf("expected at least 1 point, got %d", n)
	}

	path := filepath.Join(dir, state.DirName, CalibrationFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var store CalibrationStore
	if err := json.Unmarshal(data, &store); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if store.Version != 1 {
		t.Errorf("version: %d", store.Version)
	}
	for _, p := range store.DataPoints {
		if p.RecommendationID == "no-forecast" {
			t.Error("should not persist points without a prediction")
		}
	}

	// Append-only: running again keeps previous points.
	n2, err := AppendCalibration(dir, c)
	if err != nil {
		t.Fatalf("append2: %v", err)
	}
	data2, _ := os.ReadFile(path)
	var store2 CalibrationStore
	_ = json.Unmarshal(data2, &store2)
	if len(store2.DataPoints) != len(store.DataPoints)+n2 {
		t.Errorf("append-only broken: was %d, now %d, n2=%d",
			len(store.DataPoints), len(store2.DataPoints), n2)
	}
}

func TestRenderText_Stable(t *testing.T) {
	from := baseState()
	from.GSC = &state.GSCData{TopKeywords: []state.GSCRow{{Key: "foo", Clicks: 5, Position: 10}}}
	to := baseState()
	to.GSC = &state.GSCData{TopKeywords: []state.GSCRow{{Key: "foo", Clicks: 15, Position: 4}}}
	_, f, to2 := seedProject(t, from, to)
	c, err := Compute(f, to2)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	var buf bytes.Buffer
	if err := RenderText(&buf, c); err != nil {
		t.Fatalf("text: %v", err)
	}
	s := buf.String()
	for _, want := range []string{"Sageo Compare", "Recommendations", "GSC", "Caveats"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in output:\n%s", want, s)
		}
	}
}

func TestRenderHTML_Stable(t *testing.T) {
	from, to := baseState(), baseState()
	_, f, to2 := seedProject(t, from, to)
	c, err := Compute(f, to2)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	var buf bytes.Buffer
	if err := RenderHTML(&buf, c); err != nil {
		t.Fatalf("html: %v", err)
	}
	s := buf.String()
	if !strings.HasPrefix(s, "<!doctype html>") {
		t.Error("html missing doctype")
	}
	if !strings.Contains(s, "Reading this report") {
		t.Error("html missing caveats section")
	}
}

func TestCompute_RequiresBothSnapshots(t *testing.T) {
	if _, err := Compute(nil, nil); err == nil {
		t.Error("expected error on nil snapshots")
	}
}
