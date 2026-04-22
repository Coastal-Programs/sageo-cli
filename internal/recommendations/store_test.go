package recommendations

import (
	"testing"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

func newRec(url, query string, ct ChangeType, priority int) Recommendation {
	return Recommendation{
		ID:         HashID(url, query, ct),
		TargetURL:  url,
		ChangeType: ct,
		Priority:   priority,
	}
}

func TestUpsertIdempotent(t *testing.T) {
	s := &state.State{}
	r := newRec("https://example.com/a", "", ChangeTitle, 50)

	UpsertRecommendations(s, []Recommendation{r})
	UpsertRecommendations(s, []Recommendation{r})

	if got := len(s.Recommendations); got != 1 {
		t.Fatalf("expected 1 recommendation after double upsert, got %d", got)
	}
}

func TestUpsertReplacesByID(t *testing.T) {
	s := &state.State{}
	r := newRec("https://example.com/a", "", ChangeTitle, 50)
	UpsertRecommendations(s, []Recommendation{r})

	created := s.Recommendations[0].CreatedAt
	if created.IsZero() {
		t.Fatal("expected CreatedAt to be set on insert")
	}

	r2 := r
	r2.Priority = 80
	r2.Rationale = "updated"
	// Simulate a fresh pipeline run — zero CreatedAt should not wipe original.
	r2.CreatedAt = time.Time{}
	UpsertRecommendations(s, []Recommendation{r2})

	if len(s.Recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(s.Recommendations))
	}
	got := s.Recommendations[0]
	if got.Priority != 80 || got.Rationale != "updated" {
		t.Fatalf("upsert did not replace fields: %+v", got)
	}
	if !got.CreatedAt.Equal(created) {
		t.Fatalf("CreatedAt was overwritten: got %v want %v", got.CreatedAt, created)
	}
}

func TestLoadByURL(t *testing.T) {
	s := &state.State{}
	UpsertRecommendations(s, []Recommendation{
		newRec("https://example.com/a", "", ChangeTitle, 10),
		newRec("https://example.com/a", "", ChangeMeta, 20),
		newRec("https://example.com/b", "", ChangeTitle, 30),
	})

	got := LoadByURL(s, "https://example.com/a")
	if len(got) != 2 {
		t.Fatalf("expected 2 recs for /a, got %d", len(got))
	}
	for _, r := range got {
		if r.TargetURL != "https://example.com/a" {
			t.Errorf("unexpected url: %s", r.TargetURL)
		}
	}
}

func TestLoadTopOrdering(t *testing.T) {
	s := &state.State{}
	UpsertRecommendations(s, []Recommendation{
		newRec("https://example.com/a", "", ChangeTitle, 10),
		newRec("https://example.com/b", "", ChangeTitle, 90),
		newRec("https://example.com/c", "", ChangeTitle, 50),
		newRec("https://example.com/d", "", ChangeTitle, 70),
	})

	top := LoadTop(s, 2)
	if len(top) != 2 {
		t.Fatalf("expected 2, got %d", len(top))
	}
	if top[0].Priority != 90 || top[1].Priority != 70 {
		t.Fatalf("wrong ordering: %d, %d", top[0].Priority, top[1].Priority)
	}

	all := LoadTop(s, 0)
	if len(all) != 4 {
		t.Fatalf("expected 4 when n=0, got %d", len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].Priority < all[i].Priority {
			t.Fatalf("not sorted desc at %d: %d < %d", i, all[i-1].Priority, all[i].Priority)
		}
	}
}

func TestLoadAllReturnsCopy(t *testing.T) {
	s := &state.State{}
	UpsertRecommendations(s, []Recommendation{newRec("https://example.com/a", "", ChangeTitle, 10)})

	out := Load(s)
	out[0].Priority = 999
	if s.Recommendations[0].Priority == 999 {
		t.Fatal("Load returned a reference to the underlying slice, expected a copy")
	}
}
