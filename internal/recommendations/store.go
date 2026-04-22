package recommendations

import "sort"

// UpsertRecommendations inserts or replaces recommendations on s by ID.
// This is a thin wrapper over state.State.UpsertRecommendations so callers
// can use the recommendations package as a single import.
func UpsertRecommendations(s *State, recs []Recommendation) {
	s.UpsertRecommendations(recs)
}

// Load returns all recommendations stored on s.
func Load(s *State) []Recommendation {
	if s == nil {
		return nil
	}
	out := make([]Recommendation, len(s.Recommendations))
	copy(out, s.Recommendations)
	return out
}

// LoadByURL returns all recommendations targeting the given URL.
func LoadByURL(s *State, url string) []Recommendation {
	if s == nil {
		return nil
	}
	var out []Recommendation
	for _, r := range s.Recommendations {
		if r.TargetURL == url {
			out = append(out, r)
		}
	}
	return out
}

// LoadTop returns the top n recommendations sorted by Priority descending.
// If n <= 0 or exceeds the number stored, all recommendations are returned.
func LoadTop(s *State, n int) []Recommendation {
	if s == nil {
		return nil
	}
	out := make([]Recommendation, len(s.Recommendations))
	copy(out, s.Recommendations)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Priority > out[j].Priority
	})
	if n > 0 && n < len(out) {
		out = out[:n]
	}
	return out
}
