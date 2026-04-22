package recommendations

import "testing"

func TestHashID_Stability(t *testing.T) {
	a := HashID("https://example.com/page", "blue widgets", ChangeTitle)
	b := HashID("https://example.com/page", "blue widgets", ChangeTitle)
	if a != b {
		t.Fatalf("HashID not stable: %q vs %q", a, b)
	}
	if len(a) != 16 {
		t.Fatalf("expected 16-char id, got %d (%q)", len(a), a)
	}
}

func TestHashID_DifferentInputs(t *testing.T) {
	base := HashID("https://example.com/a", "q", ChangeTitle)
	cases := map[string]string{
		"different url":    HashID("https://example.com/b", "q", ChangeTitle),
		"different query":  HashID("https://example.com/a", "q2", ChangeTitle),
		"different change": HashID("https://example.com/a", "q", ChangeMeta),
		"empty query":      HashID("https://example.com/a", "", ChangeTitle),
	}
	for name, id := range cases {
		if id == base {
			t.Errorf("%s: expected different id from base, both were %q", name, id)
		}
		if len(id) != 16 {
			t.Errorf("%s: expected 16-char id, got %d", name, len(id))
		}
	}
}

func TestHashID_NoFieldCollision(t *testing.T) {
	// Without separators, ("ab","c") and ("a","bc") would produce the same
	// concatenated input. Ensure the separator defends against that.
	a := HashID("ab", "c", ChangeTitle)
	b := HashID("a", "bc", ChangeTitle)
	if a == b {
		t.Fatalf("field boundary collision: %q", a)
	}
}
