package pipeline

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

// setupProject creates a temp project dir with an initialised state.json and
// returns its path. Cleanup is handled by t.Cleanup.
func setupProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Init writes .sageo/state.json.
	if _, err := state.Init(dir, "https://example.com"); err != nil {
		t.Fatalf("state.Init: %v", err)
	}
	return dir
}

// fakeStage is a Stage whose Run records its invocation order.
func fakeStage(name string, paid bool, costUSD float64, order *[]string, err error) Stage {
	return Stage{
		Name: name,
		Paid: paid,
		EstimateUSD: func(*state.State) float64 {
			return costUSD
		},
		Run: func(ctx context.Context, s *state.State) error {
			*order = append(*order, name)
			s.AddHistory("test."+name, "")
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func TestRunExecutesStagesInOrder(t *testing.T) {
	dir := setupProject(t)
	var order []string
	stages := []Stage{
		fakeStage("a", false, 0, &order, nil),
		fakeStage("b", false, 0, &order, nil),
		fakeStage("c", false, 0, &order, nil),
	}
	res, err := Run(context.Background(), Config{WorkDir: dir, Out: &bytes.Buffer{}}, stages)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got, want := order, []string{"a", "b", "c"}; !equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if res.Outcome != "success" {
		t.Fatalf("outcome = %q, want success", res.Outcome)
	}
	if len(res.StagesRun) != 3 {
		t.Fatalf("StagesRun = %v", res.StagesRun)
	}
}

func TestSkipAndOnlyFilters(t *testing.T) {
	dir := setupProject(t)
	var order []string
	mkStages := func() []Stage {
		order = order[:0]
		return []Stage{
			fakeStage("a", false, 0, &order, nil),
			fakeStage("b", false, 0, &order, nil),
			fakeStage("c", false, 0, &order, nil),
		}
	}

	// Skip
	if _, err := Run(context.Background(), Config{WorkDir: dir, Skip: map[string]bool{"b": true}, Out: &bytes.Buffer{}}, mkStages()); err != nil {
		t.Fatal(err)
	}
	if got, want := order, []string{"a", "c"}; !equal(got, want) {
		t.Fatalf("skip: order = %v, want %v", got, want)
	}

	// Only
	if _, err := Run(context.Background(), Config{WorkDir: dir, Only: map[string]bool{"b": true, "c": true}, Out: &bytes.Buffer{}}, mkStages()); err != nil {
		t.Fatal(err)
	}
	if got, want := order, []string{"b", "c"}; !equal(got, want) {
		t.Fatalf("only: order = %v, want %v", got, want)
	}
}

func TestBudgetEnforcement(t *testing.T) {
	dir := setupProject(t)
	var order []string
	stages := []Stage{
		fakeStage("cheap", true, 0.10, &order, nil),
		fakeStage("expensive", true, 5.00, &order, nil), // will be skipped
		fakeStage("free", false, 0, &order, nil),        // still runs
	}
	res, err := Run(context.Background(), Config{WorkDir: dir, BudgetUSD: 1.00, Out: &bytes.Buffer{}}, stages)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got, want := order, []string{"cheap", "free"}; !equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	if len(res.Skipped) != 1 || res.Skipped[0] != "expensive" {
		t.Fatalf("Skipped = %v", res.Skipped)
	}
	if res.Outcome != "budget_exceeded" {
		t.Fatalf("outcome = %q", res.Outcome)
	}
	if res.TotalCostUSD != 0.10 {
		t.Fatalf("TotalCostUSD = %v", res.TotalCostUSD)
	}
}

func TestErrorPropagationAndResume(t *testing.T) {
	dir := setupProject(t)
	var order []string
	boom := errors.New("boom")
	stages := []Stage{
		fakeStage("a", false, 0, &order, nil),
		fakeStage("b", false, 0, &order, boom),
		fakeStage("c", false, 0, &order, nil),
	}
	res, err := Run(context.Background(), Config{WorkDir: dir, Out: &bytes.Buffer{}}, stages)
	if err == nil {
		t.Fatal("expected error")
	}
	if res.FailedStage != "b" {
		t.Fatalf("FailedStage = %q", res.FailedStage)
	}
	if got, want := order, []string{"a", "b"}; !equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}

	// Verify cursor is persisted at the failing stage.
	st, err := state.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.PipelineCursor != "b" {
		t.Fatalf("PipelineCursor = %q, want b", st.PipelineCursor)
	}

	// Resume: rebuild stages with 'b' now succeeding. Resume picks up AFTER b.
	order = order[:0]
	stagesResume := []Stage{
		fakeStage("a", false, 0, &order, nil),
		fakeStage("b", false, 0, &order, nil),
		fakeStage("c", false, 0, &order, nil),
	}
	if _, err := Run(context.Background(), Config{WorkDir: dir, Resume: true, Out: &bytes.Buffer{}}, stagesResume); err != nil {
		t.Fatalf("resume Run: %v", err)
	}
	if got, want := order, []string{"c"}; !equal(got, want) {
		t.Fatalf("resume order = %v, want %v", got, want)
	}
}

func TestDryRunDoesNotExecute(t *testing.T) {
	dir := setupProject(t)
	var order []string
	stages := []Stage{
		fakeStage("a", true, 0.5, &order, nil),
		fakeStage("b", true, 0.25, &order, nil),
	}
	buf := &bytes.Buffer{}
	res, err := Run(context.Background(), Config{WorkDir: dir, DryRun: true, Out: buf}, stages)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 0 {
		t.Fatalf("dry run executed stages: %v", order)
	}
	if res.Outcome != "dry_run" {
		t.Fatalf("outcome = %q", res.Outcome)
	}
	if res.TotalCostUSD != 0.75 {
		t.Fatalf("TotalCostUSD = %v", res.TotalCostUSD)
	}
}

func TestContextCancellation(t *testing.T) {
	dir := setupProject(t)
	var order []string
	ctx, cancel := context.WithCancel(context.Background())
	stages := []Stage{
		Stage{
			Name: "a",
			Run: func(c context.Context, s *state.State) error {
				order = append(order, "a")
				cancel()
				return nil
			},
		},
		fakeStage("b", false, 0, &order, nil),
	}
	res, err := Run(ctx, Config{WorkDir: dir, Out: &bytes.Buffer{}}, stages)
	if err == nil {
		t.Fatal("expected context error")
	}
	if res.Outcome != "failed" {
		t.Fatalf("outcome = %q", res.Outcome)
	}
	if len(order) != 1 {
		t.Fatalf("expected only stage a to run, got %v", order)
	}
}

func TestPipelineRunRecordedInState(t *testing.T) {
	dir := setupProject(t)
	var order []string
	stages := []Stage{fakeStage("a", false, 0, &order, nil)}
	if _, err := Run(context.Background(), Config{WorkDir: dir, Out: &bytes.Buffer{}}, stages); err != nil {
		t.Fatal(err)
	}
	st, err := state.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.PipelineRuns) != 1 {
		t.Fatalf("PipelineRuns = %v", st.PipelineRuns)
	}
	if st.PipelineRuns[0].Outcome != "success" {
		t.Fatalf("Outcome = %q", st.PipelineRuns[0].Outcome)
	}
	if time.Since(st.PipelineRuns[0].StartedAt) > 5*time.Second {
		t.Fatalf("StartedAt looks wrong: %v", st.PipelineRuns[0].StartedAt)
	}
	// Cursor cleared on full success.
	if st.PipelineCursor != "" {
		t.Fatalf("PipelineCursor = %q, want empty after success", st.PipelineCursor)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

var _ = os.Stderr
