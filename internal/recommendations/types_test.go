package recommendations

import (
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

// TestEffectiveReviewStatus_BackwardsCompat documents the backfill rule for
// state files that predate the review gate:
//
//   - No RecommendedValue → empty status (nothing drafted, nothing to review).
//   - RecommendedValue set but no ReviewStatus → ReviewPending (safe default).
//   - ReviewStatus explicitly set → returned verbatim.
func TestEffectiveReviewStatus_BackwardsCompat(t *testing.T) {
	tests := []struct {
		name string
		rec  state.Recommendation
		want state.ReviewStatus
	}{
		{
			name: "no drafted value",
			rec:  state.Recommendation{},
			want: "",
		},
		{
			name: "legacy state with value but no status",
			rec:  state.Recommendation{RecommendedValue: "A new title"},
			want: state.ReviewPending,
		},
		{
			name: "explicit approved",
			rec: state.Recommendation{
				RecommendedValue: "A new title",
				ReviewStatus:     state.ReviewApproved,
			},
			want: state.ReviewApproved,
		},
		{
			name: "explicit rejected",
			rec: state.Recommendation{
				RecommendedValue: "A new title",
				ReviewStatus:     state.ReviewRejected,
			},
			want: state.ReviewRejected,
		},
		{
			name: "explicit edited",
			rec: state.Recommendation{
				RecommendedValue: "Edited title",
				OriginalDraft:    "Draft title",
				ReviewStatus:     state.ReviewEdited,
			},
			want: state.ReviewEdited,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.rec.EffectiveReviewStatus(); got != tc.want {
				t.Errorf("EffectiveReviewStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestDraft_SetsPendingReview ensures the drafter never auto-approves its
// own output. Every successful draft must land in the review queue with
// OriginalDraft preserved for audit.
func TestDraft_SetsPendingReview(t *testing.T) {
	p := &stubProvider{responses: []string{"Best Running Shoes 2026"}}
	rec := &state.Recommendation{ChangeType: state.ChangeTitle}
	if err := Draft(t.Context(), p, rec, PageContext{}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if rec.RecommendedValue == "" {
		t.Fatal("expected RecommendedValue to be set")
	}
	if rec.ReviewStatus != state.ReviewPending {
		t.Errorf("ReviewStatus = %q, want %q", rec.ReviewStatus, state.ReviewPending)
	}
	if rec.OriginalDraft != rec.RecommendedValue {
		t.Errorf("OriginalDraft = %q, want %q", rec.OriginalDraft, rec.RecommendedValue)
	}
	if rec.ReviewedAt != nil {
		t.Error("ReviewedAt should remain nil until a human reviews")
	}
}
