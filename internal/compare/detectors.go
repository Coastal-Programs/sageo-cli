package compare

import (
	"fmt"
	"strings"

	"github.com/jakeschepis/sageo-cli/internal/recommendations"
	"github.com/jakeschepis/sageo-cli/internal/state"
)

// detectAddressed returns (true, evidence) if on-site evidence in the
// later snapshot suggests the recommended change was made.
//
// Pragmatic note on detection surface: sageo's snapshots persist audit
// Findings, PSI metrics per URL+strategy, and backlink summaries. Raw
// per-URL page content (title, meta, word count, schema list) is NOT
// persisted today — it's re-crawled on demand. That means our strongest
// signal for content-type changes is "the audit finding that flagged the
// gap has disappeared" rather than direct before/after string comparison.
// Each detector documents its own proxy and the false-positive shape it
// is vulnerable to. When we can't detect a change at all we return
// (false, "") — the recommendation falls into StillOpen or Resolved.
func detectAddressed(rec recommendations.Recommendation, from, to *state.State) (bool, string) {
	switch rec.ChangeType {
	case recommendations.ChangeTitle:
		return detectFindingCleared(rec, from, to, []string{"title-missing", "title-too-long"}, "title")
	case recommendations.ChangeMeta:
		return detectFindingCleared(rec, from, to, []string{"meta-description-missing", "meta-description-too-long"}, "meta description")
	case recommendations.ChangeH1:
		return detectFindingCleared(rec, from, to, []string{"h1-missing", "h1-multiple"}, "H1")
	case recommendations.ChangeH2:
		// No audit rule specifically for H2 absence — treat as
		// undetectable from persisted state.
		return false, ""
	case recommendations.ChangeSchema:
		return detectFindingCleared(rec, from, to, []string{"schema-missing"}, "schema")
	case recommendations.ChangeBody:
		return detectFindingCleared(rec, from, to, []string{"thin-content"}, "word count")
	case recommendations.ChangeInternalLink:
		return detectFindingCleared(rec, from, to, []string{"orphan-page"}, "internal linking")
	case recommendations.ChangeSpeed:
		return detectSpeedImproved(rec, from, to)
	case recommendations.ChangeIndexability:
		return detectFindingCleared(rec, from, to, []string{"broken-page", "viewport-missing"}, "indexability")
	case recommendations.ChangeBacklink:
		return detectBacklinkAcquired(rec, from, to)
	case recommendations.ChangeAuthorByline:
		// Requires a Person schema inspection we don't persist; mark as
		// undetectable until we add author-visibility to snapshots.
		return false, ""
	case recommendations.ChangeFreshness:
		// Requires dateModified / visible published date, not
		// persisted. Undetectable today.
		return false, ""
	case recommendations.ChangeTLDR, recommendations.ChangeListFormat, recommendations.ChangeEntityConsistency:
		// No corresponding audit rule and no snapshot of the page
		// content; treat as undetectable.
		return false, ""
	}
	return false, ""
}

// detectFindingCleared is the workhorse detector: if any of the named
// audit rules fired against rec.TargetURL in the earlier snapshot and none
// of them fire in the later snapshot, we treat the recommendation as
// addressed. The "evidence" string names the cleared rules for the
// human-readable report.
//
// False-positive shape: a rule can clear because (a) the page was fixed,
// (b) the page stopped being crawled, or (c) the audit ruleset changed
// between versions. We can't distinguish from findings alone. Calibration
// should be taken with this caveat.
func detectFindingCleared(rec recommendations.Recommendation, from, to *state.State, rules []string, label string) (bool, string) {
	ruleSet := map[string]bool{}
	for _, r := range rules {
		ruleSet[r] = true
	}

	var firedBefore []string
	for _, f := range from.Findings {
		if f.URL != rec.TargetURL {
			continue
		}
		if ruleSet[f.Rule] {
			firedBefore = append(firedBefore, f.Rule)
		}
	}
	if len(firedBefore) == 0 {
		return false, ""
	}
	for _, f := range to.Findings {
		if f.URL != rec.TargetURL {
			continue
		}
		if ruleSet[f.Rule] {
			// At least one flagged rule still fires.
			return false, ""
		}
	}
	return true, fmt.Sprintf("%s issue cleared on %s (rules: %s)",
		label, rec.TargetURL, strings.Join(firedBefore, ", "))
}

// detectSpeedImproved says "addressed" if LCP or CLS for rec.TargetURL
// moved from the red band into the "good" band. We pick either strategy
// (mobile preferred) where both snapshots have data.
func detectSpeedImproved(rec recommendations.Recommendation, from, to *state.State) (bool, string) {
	if from.PSI == nil || to.PSI == nil {
		return false, ""
	}
	pick := func(psi *state.PSIData, url string) *state.PSIResult {
		// Prefer mobile strategy if present; fall back to first match.
		var desktop *state.PSIResult
		for i := range psi.Pages {
			if psi.Pages[i].URL != url {
				continue
			}
			if psi.Pages[i].Strategy == "mobile" {
				r := psi.Pages[i]
				return &r
			}
			d := psi.Pages[i]
			desktop = &d
		}
		return desktop
	}
	fp := pick(from.PSI, rec.TargetURL)
	tp := pick(to.PSI, rec.TargetURL)
	if fp == nil || tp == nil {
		return false, ""
	}
	lcpImproved := fp.LCP > goodLCPMs && tp.LCP <= goodLCPMs
	clsImproved := fp.CLS > goodCLS && tp.CLS <= goodCLS
	if !lcpImproved && !clsImproved {
		return false, ""
	}
	var parts []string
	if lcpImproved {
		parts = append(parts, fmt.Sprintf("LCP %.0fms → %.0fms (entered good band)", fp.LCP, tp.LCP))
	}
	if clsImproved {
		parts = append(parts, fmt.Sprintf("CLS %.2f → %.2f (entered good band)", fp.CLS, tp.CLS))
	}
	return true, strings.Join(parts, "; ")
}

// detectBacklinkAcquired says "addressed" if the referring-domain count
// grew between snapshots. This is a weak signal — any new link counts,
// not only the specific target domains called out by the recommendation.
// We record the delta so calibration can account for the coarseness.
func detectBacklinkAcquired(rec recommendations.Recommendation, from, to *state.State) (bool, string) {
	if from.Backlinks == nil || to.Backlinks == nil {
		return false, ""
	}
	delta := to.Backlinks.TotalReferringDomains - from.Backlinks.TotalReferringDomains
	if delta <= 0 {
		return false, ""
	}
	return true, fmt.Sprintf("referring domains +%d (%d → %d)",
		delta, from.Backlinks.TotalReferringDomains, to.Backlinks.TotalReferringDomains)
}
