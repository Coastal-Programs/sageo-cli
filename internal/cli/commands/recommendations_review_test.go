package commands

import (
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/recommendations"
	"github.com/jakeschepis/sageo-cli/internal/state"
)

// newTestState returns a *state.State seeded with three drafted recs at
// different priorities, plus a rec with no drafted value (which must never
// enter the review queue).
func newTestState(t *testing.T) (*state.State, string) {
	t.Helper()
	dir := t.TempDir()
	st, err := state.Init(dir, "https://example.com")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	st.Recommendations = []recommendations.Recommendation{
		{
			ID: "r-high", TargetURL: "https://example.com/a", ChangeType: state.ChangeTitle,
			Priority: 90, RecommendedValue: "High-priority title", OriginalDraft: "High-priority title",
			ReviewStatus: state.ReviewPending,
		},
		{
			ID: "r-med", TargetURL: "https://example.com/b", ChangeType: state.ChangeMeta,
			Priority: 50, RecommendedValue: "Medium meta", OriginalDraft: "Medium meta",
			ReviewStatus: state.ReviewPending,
		},
		{
			ID: "r-low", TargetURL: "https://example.com/c", ChangeType: state.ChangeH2,
			Priority: 20, RecommendedValue: "Low H2", OriginalDraft: "Low H2",
			ReviewStatus: state.ReviewPending,
		},
		{
			// Non-copy change type with no drafted value — should never appear.
			ID: "r-speed", TargetURL: "https://example.com/a", ChangeType: state.ChangeSpeed,
			Priority: 70,
		},
	}
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	return st, dir
}

func TestBuildReviewQueue_SortedByPriority(t *testing.T) {
	st, _ := newTestState(t)
	q := buildReviewQueue(st, reviewFilter{})
	if len(q) != 3 {
		t.Fatalf("queue len = %d, want 3 (speed rec must be excluded)", len(q))
	}
	if q[0].ID != "r-high" || q[1].ID != "r-med" || q[2].ID != "r-low" {
		t.Errorf("queue order = %v, want r-high r-med r-low", []string{q[0].ID, q[1].ID, q[2].ID})
	}
}

func TestBuildReviewQueue_FilterByURL(t *testing.T) {
	st, _ := newTestState(t)
	q := buildReviewQueue(st, reviewFilter{URL: "https://example.com/b"})
	if len(q) != 1 || q[0].ID != "r-med" {
		t.Errorf("URL filter got %d items, first=%q", len(q), firstID(q))
	}
}

func TestBuildReviewQueue_FilterByType(t *testing.T) {
	st, _ := newTestState(t)
	q := buildReviewQueue(st, reviewFilter{ChangeType: string(state.ChangeTitle)})
	if len(q) != 1 || q[0].ID != "r-high" {
		t.Errorf("type filter got %d items, first=%q", len(q), firstID(q))
	}
}

// TestBulkApproveUnder asserts the escape hatch flag approves low-priority
// drafts and persists OriginalDraft on each.
func TestBulkApproveUnder(t *testing.T) {
	st, _ := newTestState(t)
	n := bulkApproveUnder(st, 60, "cli")
	if n != 2 {
		t.Fatalf("approved %d, want 2 (priorities 50 and 20)", n)
	}
	// r-high (90) must remain pending; r-med and r-low must be approved.
	for _, r := range st.Recommendations {
		switch r.ID {
		case "r-high":
			if r.EffectiveReviewStatus() != state.ReviewPending {
				t.Errorf("r-high = %q, want pending", r.EffectiveReviewStatus())
			}
		case "r-med", "r-low":
			if r.ReviewStatus != state.ReviewApproved {
				t.Errorf("%s = %q, want approved", r.ID, r.ReviewStatus)
			}
			if r.ReviewedAt == nil {
				t.Errorf("%s ReviewedAt should be set", r.ID)
			}
		}
	}
}

// scriptedPrompter drives runReviewLoop without any TUI. Each action maps to
// the recommendation ID so tests can assert exact dispatch.
type scriptedPrompter struct {
	script map[string]reviewAction
}

func (s *scriptedPrompter) Ask(rec recommendations.Recommendation, index, total int) (reviewAction, error) {
	if a, ok := s.script[rec.ID]; ok {
		return a, nil
	}
	return reviewAction{Kind: "skip"}, nil
}

// TestRunReviewLoop_AppliesDecisions covers approve, edit, and reject, and
// asserts that OriginalDraft is preserved through edits.
func TestRunReviewLoop_AppliesDecisions(t *testing.T) {
	st, dir := newTestState(t)
	queue := buildReviewQueue(st, reviewFilter{})

	prompter := &scriptedPrompter{script: map[string]reviewAction{
		"r-high": {Kind: "approve"},
		"r-med":  {Kind: "edit", Edited: "Human-edited meta"},
		"r-low":  {Kind: "reject", Reason: "off-brand"},
	}}

	n, err := runReviewLoop(st, queue, prompter, "tester", dir)
	if err != nil {
		t.Fatalf("runReviewLoop: %v", err)
	}
	if n != 3 {
		t.Fatalf("processed = %d, want 3", n)
	}

	byID := func(id string) recommendations.Recommendation {
		for _, r := range st.Recommendations {
			if r.ID == id {
				return r
			}
		}
		t.Fatalf("missing rec %q", id)
		return recommendations.Recommendation{}
	}

	if r := byID("r-high"); r.ReviewStatus != state.ReviewApproved || r.ReviewedBy != "tester" {
		t.Errorf("r-high = %+v", r)
	}
	if r := byID("r-med"); r.ReviewStatus != state.ReviewEdited ||
		r.RecommendedValue != "Human-edited meta" ||
		r.OriginalDraft != "Medium meta" {
		t.Errorf("r-med edit failed: value=%q original=%q status=%q",
			r.RecommendedValue, r.OriginalDraft, r.ReviewStatus)
	}
	if r := byID("r-low"); r.ReviewStatus != state.ReviewRejected || r.ReviewNotes != "off-brand" {
		t.Errorf("r-low reject failed: status=%q notes=%q", r.ReviewStatus, r.ReviewNotes)
	}

	// Counts metadata should reflect the decisions.
	counts := ReviewStatusCounts(st)
	if counts["approved"] != 1 || counts["edited"] != 1 || counts["rejected"] != 1 || counts["pending"] != 0 {
		t.Errorf("counts = %v", counts)
	}
}

// TestRunReviewLoop_QuitStopsEarly asserts the quit action breaks out of the
// loop and returns the number of items processed so far.
func TestRunReviewLoop_QuitStopsEarly(t *testing.T) {
	st, dir := newTestState(t)
	queue := buildReviewQueue(st, reviewFilter{})
	prompter := &scriptedPrompter{script: map[string]reviewAction{
		"r-high": {Kind: "approve"},
		"r-med":  {Kind: "quit"},
	}}
	n, err := runReviewLoop(st, queue, prompter, "tester", dir)
	if err == nil || err.Error() != errReviewQuit.Error() {
		t.Fatalf("err = %v, want errReviewQuit", err)
	}
	if n != 1 {
		t.Errorf("processed = %d, want 1 (approve before quit)", n)
	}
}

func firstID(q []recommendations.Recommendation) string {
	if len(q) == 0 {
		return ""
	}
	return q[0].ID
}
