// Package compare diffs two sageo snapshots and emits a structured report
// describing what actually changed between the two runs: GSC ranking
// movement, PSI performance shifts, SERP / AEO citations gained or lost,
// recommendations that look addressed, and new issues that appeared.
//
// This is the feedback signal that unlocks forecast calibration. For every
// recommendation we detect as "addressed" and can pair with a measurable
// outcome (GSC click delta, AEO citation delta, etc.), we append a
// calibration data point to .sageo/calibration.json so a later calibration
// pass can compare predicted vs observed impact.
//
// A note on causation: none of the deltas here are causal. Search ranking
// moves with algorithm updates, seasonality, PR, competitor activity, and
// concurrent on-site changes. compare surfaces observations, never claims.
// All human-readable output includes an explicit caveat to that effect.
package compare

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/recommendations"
	"github.com/jakeschepis/sageo-cli/internal/state"
)

// Comparison is the full structured diff between two snapshots.
type Comparison struct {
	From     *state.Snapshot `json:"-"`
	To       *state.Snapshot `json:"-"`
	FromRef  string          `json:"from_ref"`
	ToRef    string          `json:"to_ref"`
	FromTime time.Time       `json:"from_time"`
	ToTime   time.Time       `json:"to_time"`
	Duration time.Duration   `json:"duration_ns"`

	GSC       GSCDelta             `json:"gsc"`
	PSI       PSIDelta             `json:"psi"`
	SERP      SERPDelta            `json:"serp"`
	AEO       AEODelta             `json:"aeo"`
	Backlinks BacklinksDelta       `json:"backlinks"`
	Audit     AuditDelta           `json:"audit"`
	Recs      RecommendationsDelta `json:"recommendations"`

	// Caveats is the honest framing copy that every human-readable output
	// must include. Machine consumers can ignore it but it is surfaced in
	// the JSON envelope so downstream tooling can propagate the warning.
	Caveats []string `json:"caveats"`
}

// GSCDelta summarises GSC movement between snapshots.
type GSCDelta struct {
	QueriesGained    []GSCQueryChange `json:"queries_gained,omitempty"`    // appeared between snapshots
	QueriesLost      []GSCQueryChange `json:"queries_lost,omitempty"`      // disappeared
	PositionImproved []GSCQueryChange `json:"position_improved,omitempty"` // sorted by magnitude
	PositionDeclined []GSCQueryChange `json:"position_declined,omitempty"`
	ClicksDelta      int              `json:"clicks_delta"`
	ImpressionsDelta int              `json:"impressions_delta"`
	CTRDelta         float64          `json:"ctr_delta"`
	TotalQueriesFrom int              `json:"total_queries_from"`
	TotalQueriesTo   int              `json:"total_queries_to"`
}

// GSCQueryChange is a single keyword's before/after in GSC.
//
// Position uses GSC's convention: 1 is best, higher numbers are worse. We
// expose DeltaPosition as signed-with-negative-being-good (To − From) so
// "-3.1" reads as "moved from position 8 to position 5" on inspection.
type GSCQueryChange struct {
	Query         string  `json:"query"`
	URL           string  `json:"url,omitempty"`
	FromPosition  float64 `json:"from_position"` // 0 if new
	ToPosition    float64 `json:"to_position"`   // 0 if lost
	FromClicks    int     `json:"from_clicks"`
	ToClicks      int     `json:"to_clicks"`
	DeltaClicks   int     `json:"delta_clicks"`
	DeltaPosition float64 `json:"delta_position"` // negative = improvement
}

// PSIDelta summarises PSI metric changes keyed by URL+strategy.
type PSIDelta struct {
	Changed []PSIChange `json:"changed,omitempty"`
}

type PSIChange struct {
	URL                string  `json:"url"`
	Strategy           string  `json:"strategy"`
	FromPerformance    float64 `json:"from_performance"`
	ToPerformance      float64 `json:"to_performance"`
	DeltaPerformance   float64 `json:"delta_performance"`
	FromLCP            float64 `json:"from_lcp_ms"`
	ToLCP              float64 `json:"to_lcp_ms"`
	DeltaLCP           float64 `json:"delta_lcp_ms"` // negative = improvement
	FromCLS            float64 `json:"from_cls"`
	ToCLS              float64 `json:"to_cls"`
	DeltaCLS           float64 `json:"delta_cls"`
	EnteredGoodBandLCP bool    `json:"entered_good_band_lcp,omitempty"`
	EnteredGoodBandCLS bool    `json:"entered_good_band_cls,omitempty"`
}

// SERPDelta summarises SERP feature / position movement per query.
type SERPDelta struct {
	Changed []SERPChange `json:"changed,omitempty"`
}

type SERPChange struct {
	Query            string   `json:"query"`
	FromPosition     int      `json:"from_position"`
	ToPosition       int      `json:"to_position"`
	DeltaPosition    int      `json:"delta_position"` // negative = improvement
	FeaturesGained   []string `json:"features_gained,omitempty"`
	FeaturesLost     []string `json:"features_lost,omitempty"`
	AIOverviewGained bool     `json:"ai_overview_gained,omitempty"`
	AIOverviewLost   bool     `json:"ai_overview_lost,omitempty"`
}

// AEODelta summarises LLM brand-citation movement per term.
type AEODelta struct {
	Terms []AEOTermDelta `json:"terms,omitempty"`
}

type AEOTermDelta struct {
	Term           string   `json:"term"`
	CitationsFrom  int      `json:"citations_from"`
	CitationsTo    int      `json:"citations_to"`
	DeltaCitations int      `json:"delta_citations"`
	PagesGained    []string `json:"pages_gained,omitempty"` // URLs newly cited
	PagesLost      []string `json:"pages_lost,omitempty"`
}

// BacklinksDelta summarises backlink profile changes.
type BacklinksDelta struct {
	Available             bool  `json:"available"`
	DeltaBacklinks        int64 `json:"delta_backlinks"`
	DeltaReferringDomains int64 `json:"delta_referring_domains"`
	DeltaBroken           int64 `json:"delta_broken"`
	DomainsGained         int   `json:"domains_gained"` // top-referrers gained
	DomainsLost           int   `json:"domains_lost"`
}

// AuditDelta summarises audit findings change.
type AuditDelta struct {
	FromScore         float64 `json:"from_score"`
	ToScore           float64 `json:"to_score"`
	DeltaScore        float64 `json:"delta_score"`
	FindingsResolved  int     `json:"findings_resolved"`
	FindingsAppeared  int     `json:"findings_appeared"`
	FromFindingsCount int     `json:"from_findings_count"`
	ToFindingsCount   int     `json:"to_findings_count"`
}

// RecommendationsDelta is the headline feedback loop: which earlier
// recommendations look addressed, which remain open, which are newly
// surfaced, and which were resolved without having been flagged originally.
type RecommendationsDelta struct {
	Addressed  []RecommendationOutcome          `json:"addressed,omitempty"`
	StillOpen  []recommendations.Recommendation `json:"still_open,omitempty"`
	NewlyAdded []recommendations.Recommendation `json:"newly_added,omitempty"`
	Resolved   []recommendations.Recommendation `json:"resolved,omitempty"`
}

// RecommendationOutcome ties a specific earlier recommendation to the
// detector evidence that says it was addressed, plus the observational lift
// (if any) we were able to measure against the later snapshot.
type RecommendationOutcome struct {
	Recommendation recommendations.Recommendation `json:"recommendation"`
	Addressed      bool                           `json:"addressed"`
	Evidence       string                         `json:"evidence,omitempty"`
	ObservedLift   *ObservedLift                  `json:"observed_lift,omitempty"`
}

// ObservedLift is the raw observational outcome — not a causal estimate.
// Calibration consumes this alongside the ForecastedLift to train the
// forecaster's optimism/pessimism bias over time.
type ObservedLift struct {
	ClicksDelta      int     `json:"clicks_delta"`
	ImpressionsDelta int     `json:"impressions_delta"`
	PositionDelta    float64 `json:"position_delta"` // negative = improvement
	CitationsDelta   int     `json:"citations_delta,omitempty"`
	WindowDays       int     `json:"window_days"`
	ConfidenceNote   string  `json:"confidence_note,omitempty"`
}

// defaultCaveats is attached to every Comparison so downstream consumers see
// the same honest framing the CLI surfaces.
var defaultCaveats = []string{
	"Changes observed between two snapshots are correlational. Search engine algorithm updates, seasonality, and concurrent changes (PR, other SEO work) are not controlled for.",
	"Low-volume queries (<10 impressions/day) are noisy — observed deltas may not reflect the change's impact.",
	"AEO citations are sampled from a small number of LLM responses per term; day-to-day variance can be large.",
}

// Compute builds a Comparison between two snapshots. Either argument may be
// nil to signal "no prior snapshot available", in which case the function
// returns an error — compare is only meaningful on two real snapshots.
func Compute(from, to *state.Snapshot) (*Comparison, error) {
	if from == nil || to == nil {
		return nil, errors.New("compare: both snapshots required")
	}
	fromState, err := from.State()
	if err != nil {
		return nil, fmt.Errorf("compare: load from state: %w", err)
	}
	toState, err := to.State()
	if err != nil {
		return nil, fmt.Errorf("compare: load to state: %w", err)
	}

	c := &Comparison{
		From:     from,
		To:       to,
		FromTime: from.Timestamp,
		ToTime:   to.Timestamp,
		Duration: to.Timestamp.Sub(from.Timestamp),
		Caveats:  append([]string{}, defaultCaveats...),
	}

	c.GSC = diffGSC(fromState.GSC, toState.GSC)
	c.PSI = diffPSI(fromState.PSI, toState.PSI)
	c.SERP = diffSERP(fromState.SERP, toState.SERP)
	c.AEO = diffAEO(fromState.Mentions, toState.Mentions)
	c.Backlinks = diffBacklinks(fromState.Backlinks, toState.Backlinks)
	c.Audit = diffAudit(fromState, toState)
	c.Recs = diffRecommendations(fromState, toState, c)

	return c, nil
}

// ---- GSC ---------------------------------------------------------------

func diffGSC(from, to *state.GSCData) GSCDelta {
	d := GSCDelta{}
	fromMap := gscKeyMap(from)
	toMap := gscKeyMap(to)
	d.TotalQueriesFrom = len(fromMap)
	d.TotalQueriesTo = len(toMap)

	seen := map[string]bool{}
	for k, fr := range fromMap {
		seen[k] = true
		tr, ok := toMap[k]
		if !ok {
			d.QueriesLost = append(d.QueriesLost, GSCQueryChange{
				Query:        k,
				FromPosition: fr.Position,
				FromClicks:   int(fr.Clicks),
				DeltaClicks:  -int(fr.Clicks),
			})
			d.ClicksDelta -= int(fr.Clicks)
			d.ImpressionsDelta -= int(fr.Impressions)
			continue
		}
		change := GSCQueryChange{
			Query:         k,
			FromPosition:  fr.Position,
			ToPosition:    tr.Position,
			FromClicks:    int(fr.Clicks),
			ToClicks:      int(tr.Clicks),
			DeltaClicks:   int(tr.Clicks) - int(fr.Clicks),
			DeltaPosition: tr.Position - fr.Position,
		}
		d.ClicksDelta += change.DeltaClicks
		d.ImpressionsDelta += int(tr.Impressions) - int(fr.Impressions)
		if change.DeltaPosition < 0 {
			d.PositionImproved = append(d.PositionImproved, change)
		} else if change.DeltaPosition > 0 {
			d.PositionDeclined = append(d.PositionDeclined, change)
		}
	}
	for k, tr := range toMap {
		if seen[k] {
			continue
		}
		d.QueriesGained = append(d.QueriesGained, GSCQueryChange{
			Query:       k,
			ToPosition:  tr.Position,
			ToClicks:    int(tr.Clicks),
			DeltaClicks: int(tr.Clicks),
		})
		d.ClicksDelta += int(tr.Clicks)
		d.ImpressionsDelta += int(tr.Impressions)
	}

	// Sort improved ascending by DeltaPosition (most negative = biggest
	// improvement first); declined descending.
	sort.Slice(d.PositionImproved, func(i, j int) bool {
		return d.PositionImproved[i].DeltaPosition < d.PositionImproved[j].DeltaPosition
	})
	sort.Slice(d.PositionDeclined, func(i, j int) bool {
		return d.PositionDeclined[i].DeltaPosition > d.PositionDeclined[j].DeltaPosition
	})
	sort.Slice(d.QueriesGained, func(i, j int) bool {
		return d.QueriesGained[i].ToClicks > d.QueriesGained[j].ToClicks
	})
	sort.Slice(d.QueriesLost, func(i, j int) bool {
		return d.QueriesLost[i].FromClicks > d.QueriesLost[j].FromClicks
	})

	if from != nil && to != nil {
		d.CTRDelta = totalCTR(toMap) - totalCTR(fromMap)
	}
	return d
}

func gscKeyMap(d *state.GSCData) map[string]state.GSCRow {
	out := map[string]state.GSCRow{}
	if d == nil {
		return out
	}
	for _, r := range d.TopKeywords {
		out[r.Key] = r
	}
	return out
}

func totalCTR(m map[string]state.GSCRow) float64 {
	if len(m) == 0 {
		return 0
	}
	var sum float64
	for _, r := range m {
		sum += r.CTR
	}
	return sum / float64(len(m))
}

// ---- PSI ---------------------------------------------------------------

// LCP / CLS "good" band thresholds per Google's Core Web Vitals guidance.
const (
	goodLCPMs = 2500.0
	goodCLS   = 0.1
)

func diffPSI(from, to *state.PSIData) PSIDelta {
	d := PSIDelta{}
	if from == nil || to == nil {
		return d
	}
	fromMap := map[string]state.PSIResult{}
	for _, p := range from.Pages {
		fromMap[p.URL+"|"+p.Strategy] = p
	}
	for _, tp := range to.Pages {
		key := tp.URL + "|" + tp.Strategy
		fp, ok := fromMap[key]
		if !ok {
			continue
		}
		change := PSIChange{
			URL:              tp.URL,
			Strategy:         tp.Strategy,
			FromPerformance:  fp.PerformanceScore,
			ToPerformance:    tp.PerformanceScore,
			DeltaPerformance: tp.PerformanceScore - fp.PerformanceScore,
			FromLCP:          fp.LCP,
			ToLCP:            tp.LCP,
			DeltaLCP:         tp.LCP - fp.LCP,
			FromCLS:          fp.CLS,
			ToCLS:            tp.CLS,
			DeltaCLS:         tp.CLS - fp.CLS,
		}
		if fp.LCP > goodLCPMs && tp.LCP <= goodLCPMs {
			change.EnteredGoodBandLCP = true
		}
		if fp.CLS > goodCLS && tp.CLS <= goodCLS {
			change.EnteredGoodBandCLS = true
		}
		// Only record rows that actually moved (avoid flooding output
		// with unchanged metrics).
		if change.DeltaPerformance != 0 || change.DeltaLCP != 0 || change.DeltaCLS != 0 {
			d.Changed = append(d.Changed, change)
		}
	}
	sort.Slice(d.Changed, func(i, j int) bool {
		return math.Abs(d.Changed[i].DeltaLCP) > math.Abs(d.Changed[j].DeltaLCP)
	})
	return d
}

// ---- SERP --------------------------------------------------------------

func diffSERP(from, to *state.SERPData) SERPDelta {
	d := SERPDelta{}
	if from == nil || to == nil {
		return d
	}
	fromMap := map[string]state.SERPQueryResult{}
	for _, q := range from.Queries {
		fromMap[strings.ToLower(q.Query)] = q
	}
	for _, tq := range to.Queries {
		fq, ok := fromMap[strings.ToLower(tq.Query)]
		if !ok {
			continue
		}
		change := SERPChange{
			Query:         tq.Query,
			FromPosition:  fq.OurPosition,
			ToPosition:    tq.OurPosition,
			DeltaPosition: tq.OurPosition - fq.OurPosition,
		}
		fromFeats := featureSet(fq)
		toFeats := featureSet(tq)
		for f := range toFeats {
			if !fromFeats[f] {
				change.FeaturesGained = append(change.FeaturesGained, f)
			}
		}
		for f := range fromFeats {
			if !toFeats[f] {
				change.FeaturesLost = append(change.FeaturesLost, f)
			}
		}
		if !fq.HasAIOverview && tq.HasAIOverview {
			change.AIOverviewGained = true
		}
		if fq.HasAIOverview && !tq.HasAIOverview {
			change.AIOverviewLost = true
		}
		if change.DeltaPosition != 0 ||
			len(change.FeaturesGained) > 0 ||
			len(change.FeaturesLost) > 0 ||
			change.AIOverviewGained || change.AIOverviewLost {
			sort.Strings(change.FeaturesGained)
			sort.Strings(change.FeaturesLost)
			d.Changed = append(d.Changed, change)
		}
	}
	return d
}

func featureSet(q state.SERPQueryResult) map[string]bool {
	s := map[string]bool{}
	for _, f := range q.Features {
		s[f.Type] = true
	}
	return s
}

// ---- AEO ---------------------------------------------------------------

func diffAEO(from, to []state.MentionsData) AEODelta {
	d := AEODelta{}
	fromMap := map[string]state.MentionsData{}
	for _, m := range from {
		fromMap[m.Term] = m
	}
	toMap := map[string]state.MentionsData{}
	for _, m := range to {
		toMap[m.Term] = m
	}
	for term, tm := range toMap {
		fm, ok := fromMap[term]
		fromCount := 0
		fromPages := map[string]bool{}
		if ok {
			for _, p := range fm.TopPages {
				fromCount += p.Mentions
				fromPages[p.URL] = true
			}
		}
		toCount := 0
		toPages := map[string]bool{}
		for _, p := range tm.TopPages {
			toCount += p.Mentions
			toPages[p.URL] = true
		}
		td := AEOTermDelta{
			Term:           term,
			CitationsFrom:  fromCount,
			CitationsTo:    toCount,
			DeltaCitations: toCount - fromCount,
		}
		for u := range toPages {
			if !fromPages[u] {
				td.PagesGained = append(td.PagesGained, u)
			}
		}
		for u := range fromPages {
			if !toPages[u] {
				td.PagesLost = append(td.PagesLost, u)
			}
		}
		sort.Strings(td.PagesGained)
		sort.Strings(td.PagesLost)
		d.Terms = append(d.Terms, td)
	}
	// Surface terms lost entirely too.
	for term, fm := range fromMap {
		if _, still := toMap[term]; still {
			continue
		}
		fromCount := 0
		var pages []string
		for _, p := range fm.TopPages {
			fromCount += p.Mentions
			pages = append(pages, p.URL)
		}
		sort.Strings(pages)
		d.Terms = append(d.Terms, AEOTermDelta{
			Term:           term,
			CitationsFrom:  fromCount,
			DeltaCitations: -fromCount,
			PagesLost:      pages,
		})
	}
	sort.Slice(d.Terms, func(i, j int) bool {
		return absInt(d.Terms[i].DeltaCitations) > absInt(d.Terms[j].DeltaCitations)
	})
	return d
}

// ---- Backlinks ---------------------------------------------------------

func diffBacklinks(from, to *state.BacklinksData) BacklinksDelta {
	d := BacklinksDelta{}
	if from == nil || to == nil {
		return d
	}
	d.Available = true
	d.DeltaBacklinks = to.TotalBacklinks - from.TotalBacklinks
	d.DeltaReferringDomains = to.TotalReferringDomains - from.TotalReferringDomains
	d.DeltaBroken = to.BrokenBacklinks - from.BrokenBacklinks

	fromRefs := toSet(from.TopReferrers)
	toRefs := toSet(to.TopReferrers)
	for r := range toRefs {
		if !fromRefs[r] {
			d.DomainsGained++
		}
	}
	for r := range fromRefs {
		if !toRefs[r] {
			d.DomainsLost++
		}
	}
	return d
}

// ---- Audit -------------------------------------------------------------

func diffAudit(from, to *state.State) AuditDelta {
	d := AuditDelta{
		FromScore:         from.Score,
		ToScore:           to.Score,
		DeltaScore:        to.Score - from.Score,
		FromFindingsCount: len(from.Findings),
		ToFindingsCount:   len(to.Findings),
	}
	fromSet := findingSet(from.Findings)
	toSet := findingSet(to.Findings)
	for k := range fromSet {
		if !toSet[k] {
			d.FindingsResolved++
		}
	}
	for k := range toSet {
		if !fromSet[k] {
			d.FindingsAppeared++
		}
	}
	return d
}

func findingSet(fs []state.Finding) map[string]bool {
	s := map[string]bool{}
	for _, f := range fs {
		s[f.Rule+"|"+f.URL] = true
	}
	return s
}

// ---- Recommendations ---------------------------------------------------

func diffRecommendations(from, to *state.State, parent *Comparison) RecommendationsDelta {
	d := RecommendationsDelta{}

	fromByID := map[string]recommendations.Recommendation{}
	for _, r := range from.Recommendations {
		fromByID[r.ID] = r
	}
	toByID := map[string]recommendations.Recommendation{}
	for _, r := range to.Recommendations {
		toByID[r.ID] = r
	}

	// For each recommendation from the earlier snapshot, decide what
	// happened to it. "Addressed" means a detector confirmed the on-site
	// change. "Resolved" means the recommendation no longer surfaces in
	// the later snapshot but no detector fired — treat as a lucky fix
	// (could be crawl coverage, could be side-effect of other work).
	for id, rec := range fromByID {
		addressed, evidence := detectAddressed(rec, from, to)
		if addressed {
			outcome := RecommendationOutcome{
				Recommendation: rec,
				Addressed:      true,
				Evidence:       evidence,
			}
			if lift := observeLift(rec, from, to, parent); lift != nil {
				outcome.ObservedLift = lift
			}
			d.Addressed = append(d.Addressed, outcome)
			continue
		}
		if _, stillThere := toByID[id]; stillThere {
			d.StillOpen = append(d.StillOpen, rec)
		} else {
			d.Resolved = append(d.Resolved, rec)
		}
	}
	for id, rec := range toByID {
		if _, existed := fromByID[id]; !existed {
			d.NewlyAdded = append(d.NewlyAdded, rec)
		}
	}

	sort.Slice(d.Addressed, func(i, j int) bool {
		return d.Addressed[i].Recommendation.Priority > d.Addressed[j].Recommendation.Priority
	})
	sort.Slice(d.StillOpen, func(i, j int) bool {
		return d.StillOpen[i].Priority > d.StillOpen[j].Priority
	})
	sort.Slice(d.NewlyAdded, func(i, j int) bool {
		return d.NewlyAdded[i].Priority > d.NewlyAdded[j].Priority
	})
	sort.Slice(d.Resolved, func(i, j int) bool {
		return d.Resolved[i].Priority > d.Resolved[j].Priority
	})
	return d
}

// observeLift pulls the observational outcome for a recommendation out of
// the already-computed GSC and AEO deltas. It only populates fields we can
// actually measure; callers must treat missing fields as "unknown", not
// "zero".
func observeLift(rec recommendations.Recommendation, from, to *state.State, parent *Comparison) *ObservedLift {
	if from == nil || to == nil {
		return nil
	}
	var lift ObservedLift
	found := false

	// GSC: find the row for the targeted query (case-insensitive).
	if rec.TargetQuery != "" && from.GSC != nil && to.GSC != nil {
		q := strings.ToLower(rec.TargetQuery)
		var fr, tr *state.GSCRow
		for i := range from.GSC.TopKeywords {
			if strings.ToLower(from.GSC.TopKeywords[i].Key) == q {
				fr = &from.GSC.TopKeywords[i]
				break
			}
		}
		for i := range to.GSC.TopKeywords {
			if strings.ToLower(to.GSC.TopKeywords[i].Key) == q {
				tr = &to.GSC.TopKeywords[i]
				break
			}
		}
		if fr != nil && tr != nil {
			lift.ClicksDelta = int(tr.Clicks) - int(fr.Clicks)
			lift.ImpressionsDelta = int(tr.Impressions) - int(fr.Impressions)
			lift.PositionDelta = tr.Position - fr.Position
			found = true
		}
	}

	// AEO: citations delta for the target query or any brand term mention.
	if rec.TargetQuery != "" {
		for _, t := range parent.AEO.Terms {
			if strings.EqualFold(t.Term, rec.TargetQuery) {
				lift.CitationsDelta = t.DeltaCitations
				found = true
				break
			}
		}
	}

	if !found {
		return nil
	}

	windowDays := int(parent.Duration.Hours() / 24)
	lift.WindowDays = windowDays
	lift.ConfidenceNote = buildConfidenceNote(windowDays, rec)
	return &lift
}

func buildConfidenceNote(windowDays int, rec recommendations.Recommendation) string {
	var notes []string
	if windowDays < 14 {
		notes = append(notes, "short observation window — rank movement may be noise")
	}
	if windowDays > 90 {
		notes = append(notes, "long observation window — concurrent changes likely confound the delta")
	}
	// Low-volume warning: <10 imps/day equivalent over the window.
	if rec.ForecastedLift != nil && rec.ForecastedLift.RawEstimate < 5 {
		notes = append(notes, "low-volume query — delta dominated by daily variance")
	}
	notes = append(notes, "observational only, not causal")
	return strings.Join(notes, "; ")
}

// ---- helpers -----------------------------------------------------------

func toSet(s []string) map[string]bool {
	out := map[string]bool{}
	for _, v := range s {
		out[v] = true
	}
	return out
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
