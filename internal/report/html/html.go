// Package html renders a self-contained HTML report of the Sageo audit,
// recommendations, and forecast from a populated state.State.
//
// The output is a single HTML file with inlined CSS and minimal vanilla JS.
// No external resources are referenced — the file works offline. Users who
// want a PDF can use their browser's print-to-PDF (Cmd/Ctrl+P). A dedicated
// @media print stylesheet is included for clean A4 output.
package html

import (
	"bytes"
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

//go:embed templates/*.html templates/*.css
var templatesFS embed.FS

// Options controls HTML report rendering.
type Options struct {
	IncludeAppendix bool
	BrandColorHex   string
	LogoDataURI     string // optional inline base64 logo (data:image/png;base64,...)
	Title           string // default "Sageo Audit Report"
}

const defaultBrandHex = "#1E40AF"

// Render writes a single self-contained HTML document for st into out.
func Render(st *state.State, out io.Writer, opts Options) error {
	if st == nil {
		return fmt.Errorf("html: state is nil")
	}
	if out == nil {
		return fmt.Errorf("html: writer is nil")
	}
	if opts.Title == "" {
		opts.Title = "Sageo Audit Report"
	}
	if opts.BrandColorHex == "" {
		opts.BrandColorHex = defaultBrandHex
	}

	styles, err := loadCSS("templates/styles.css", "templates/print.css")
	if err != nil {
		return err
	}

	tpl, err := template.New("layout.html").Funcs(funcMap()).ParseFS(templatesFS,
		"templates/layout.html",
		"templates/components.html",
		"templates/cover.html",
		"templates/summary.html",
		"templates/quickwins.html",
		"templates/diagnosis.html",
		"templates/recommendations.html",
		"templates/forecast.html",
		"templates/appendix.html",
	)
	if err != nil {
		return fmt.Errorf("html: parse templates: %w", err)
	}

	data := buildViewModel(st, opts, styles)
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, "layout.html", data); err != nil {
		return fmt.Errorf("html: execute: %w", err)
	}
	if _, err := out.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("html: write: %w", err)
	}
	return nil
}

// RenderWithStats renders and returns the number of bytes written.
func RenderWithStats(st *state.State, out io.Writer, opts Options) (int64, error) {
	cw := &countingWriter{w: out}
	if err := Render(st, cw, opts); err != nil {
		return cw.n, err
	}
	return cw.n, nil
}

// LoadLogoAsDataURI reads a logo file from disk and returns a base64 data URI
// suitable for inlining into Options.LogoDataURI. Returns the empty string if
// path is empty.
func LoadLogoAsDataURI(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("html: read logo: %w", err)
	}
	ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if ct == "" {
		ct = "image/png"
	}
	return "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func loadCSS(paths ...string) (template.CSS, error) {
	var b strings.Builder
	for _, p := range paths {
		data, err := templatesFS.ReadFile(p)
		if err != nil {
			return "", fmt.Errorf("html: read %s: %w", p, err)
		}
		b.Write(data)
		b.WriteByte('\n')
	}
	return template.CSS(b.String()), nil
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

// -----------------------------------------------------------------------------
// View model
// -----------------------------------------------------------------------------

type viewModel struct {
	Title              string
	BrandColor         string
	LogoDataURI        template.URL
	Styles             template.CSS
	GeneratedAt        string
	Site               string
	Score              float64
	PagesCrawled       int
	IssueCount         int
	RecCount           int
	TotalLift          int
	ForecastLow        int
	ForecastHigh       int
	TierHighCount      int
	TierMediumCount    int
	TierLowCount       int
	TierUnknownCount   int
	HasCalibration     bool
	CalibrationSamples int
	AEOCoverage        float64
	HasAEOCoverage     bool
	TopRecs            []recView
	AllRecs            []recView
	ForecastRows       []forecastRow
	IncludeAppendix    bool

	AuditFindings []state.Finding
	AuditTotal    int

	PSIFailing []state.PSIResult
	PSIAll     []state.PSIResult
	PSITotal   int

	GSCUnder []state.GSCRow
	GSCTotal int

	SERPQueries []state.SERPQueryResult
	SERPAI      int
	SERPNotP1   int
	SERPTotal   int

	Backlinks *state.BacklinksData

	AEOData      []aeoRow
	AEOPrompts   int
	AEOResponses int

	// QuickWins is a filtered subset of recommendations that are
	// high-impact-per-effort (priority >= 70 AND effort_minutes <= 30).
	QuickWins []recView

	// PSIView decorates PSI rows with CWV band classes so the template
	// can render colour-coded cells without inlining the band logic.
	PSIView []psiRowView

	// GSCView decorates GSC rows with position/CTR band classes.
	GSCView []gscRowView
}

type psiRowView struct {
	URL       string
	Strategy  string
	Score     float64 // 0-100 integer for display
	ScoreBand string  // "good" | "needs" | "poor"
	LCP       float64
	LCPBand   string
	CLS       float64
	CLSBand   string
}

type gscRowView struct {
	Query        string
	Impressions  float64
	Clicks       float64
	CTR          float64
	CTRBand      string
	Position     float64
	PositionBand string
}

type recView struct {
	R                state.Recommendation
	PriorityClass    string
	TierLabel        string // HIGH | MEDIUM | LOW | UNKNOWN
	TierClass        string // css class, e.g. tier-high
	TierHeadline     string // full plain-English headline line
	IsProvisional    bool   // tier was derived from rule priority because calibration is insufficient
	LiftRange        string
	EffortLabel      string
	CurrentValue     string
	RecommendedValue string
	CurrentLen       int // character count of CurrentValue (for diff-block display)
	RecommendedLen   int
	CaveatLines      []string
	ResearchCitation string // docs/research/ai-citation-signals-2026.md section reference, if any
	// ReviewBadge is a short string shown next to the title when the
	// draft is still awaiting human sign-off. Empty for approved/edited
	// recommendations so the report stays clean once reviewed.
	ReviewBadge string
}

type forecastRow struct {
	Index      int
	URL        string
	Query      string
	ChangeType string
	Priority   int
	TierLabel  string
	TierClass  string
	LiftEst    int
	LiftLow    int
	LiftHigh   int
	LiftRange  string
	Caveats    string
	Effort     string
}

type aeoRow struct {
	Prompt         string
	EngineCount    int
	BrandMentioned bool
}

func buildViewModel(st *state.State, opts Options, styles template.CSS) viewModel {
	// Filter out rejected recommendations entirely — they must never appear
	// in a client-facing report. Approved, edited, and pending-review recs
	// all render; pending-review recs get a visible badge so the reader
	// knows the draft has not been signed off on yet.
	reportable := filterReportable(st.Recommendations)
	recs := sortedRecs(reportable)
	totalLift, lo, hi := forecastTotalsOf(reportable)
	topRaw := recs
	if len(topRaw) > 5 {
		topRaw = topRaw[:5]
	}

	// Audit
	var auditFindings []state.Finding
	if n := len(st.Findings); n > 0 {
		if n > 20 {
			n = 20
		}
		auditFindings = st.Findings[:n]
	}

	// PSI
	var psiFailing, psiShow []state.PSIResult
	var psiTotal int
	if st.PSI != nil {
		psiTotal = len(st.PSI.Pages)
		for _, p := range st.PSI.Pages {
			if p.PerformanceScore < 0.9 || p.LCP > 2500 || p.CLS > 0.1 {
				psiFailing = append(psiFailing, p)
			}
		}
		psiShow = psiFailing
		if len(psiShow) == 0 {
			psiShow = st.PSI.Pages
		}
		if len(psiShow) > 20 {
			psiShow = psiShow[:20]
		}
	}

	// GSC
	var gscUnder []state.GSCRow
	var gscTotal int
	if st.GSC != nil {
		gscTotal = len(st.GSC.TopKeywords)
		for _, r := range st.GSC.TopKeywords {
			if r.Impressions >= 100 && (r.Position > 10 || r.CTR < 0.02) {
				gscUnder = append(gscUnder, r)
			}
		}
		sort.Slice(gscUnder, func(i, j int) bool { return gscUnder[i].Impressions > gscUnder[j].Impressions })
		if len(gscUnder) > 20 {
			gscUnder = gscUnder[:20]
		}
	}

	// SERP
	var serpList []state.SERPQueryResult
	serpAI, serpNotP1, serpTotal := 0, 0, 0
	if st.SERP != nil {
		serpTotal = len(st.SERP.Queries)
		for _, q := range st.SERP.Queries {
			if q.HasAIOverview {
				serpAI++
			}
			if q.OurPosition <= 0 || q.OurPosition > 10 {
				serpNotP1++
			}
		}
		serpList = st.SERP.Queries
		if len(serpList) > 20 {
			serpList = serpList[:20]
		}
	}

	// AEO
	var aeoRows []aeoRow
	aeoPrompts, aeoResponses := 0, 0
	if st.AEO != nil {
		aeoPrompts = len(st.AEO.Responses)
		for _, p := range st.AEO.Responses {
			aeoResponses += len(p.Results)
			aeoRows = append(aeoRows, aeoRow{
				Prompt:         p.Prompt,
				EngineCount:    len(p.Results),
				BrandMentioned: promptMentionsBrand(p, st.BrandTerms),
			})
		}
		if len(aeoRows) > 20 {
			aeoRows = aeoRows[:20]
		}
	}

	// AEO coverage
	coverage := aeoBrandCoverage(st)
	hasCoverage := coverage >= 0

	// Forecast rows
	fRecs := sortedByForecast(reportable)
	if len(fRecs) > 20 {
		fRecs = fRecs[:20]
	}
	var fRows []forecastRow
	for i, r := range fRecs {
		tier := tierOf(r)
		fRows = append(fRows, forecastRow{
			Index:      i + 1,
			URL:        r.TargetURL,
			Query:      r.TargetQuery,
			ChangeType: string(r.ChangeType),
			Priority:   r.Priority,
			TierLabel:  tierLabel(tier),
			TierClass:  tierClass(tier),
			LiftEst:    liftEst(r),
			LiftLow:    liftLow(r),
			LiftHigh:   liftHigh(r),
			LiftRange:  liftRangeHuman(r),
			Caveats:    strings.Join(humanCaveats(r), "; "),
			Effort:     effortLabel(r),
		})
	}

	highN, medN, lowN, unkN := tierCounts(reportable)
	quickWins := toRecViews(filterQuickWins(recs))
	psiView := toPSIView(psiShow)
	gscView := toGSCView(gscUnder)
	hasCalibration, calSamples := calibrationSummary(reportable)

	return viewModel{
		Title:              opts.Title,
		BrandColor:         opts.BrandColorHex,
		LogoDataURI:        template.URL(opts.LogoDataURI),
		Styles:             styles,
		GeneratedAt:        time.Now().UTC().Format("2 Jan 2006"),
		Site:               safeSite(st.Site),
		Score:              st.Score,
		PagesCrawled:       st.PagesCrawled,
		IssueCount:         len(st.Findings),
		RecCount:           recShippedCount(reportable),
		TotalLift:          totalLift,
		ForecastLow:        lo,
		ForecastHigh:       hi,
		TierHighCount:      highN,
		TierMediumCount:    medN,
		TierLowCount:       lowN,
		TierUnknownCount:   unkN,
		HasCalibration:     hasCalibration,
		CalibrationSamples: calSamples,
		AEOCoverage:        coverage,
		HasAEOCoverage:     hasCoverage,
		TopRecs:            toRecViews(topRaw),
		AllRecs:            toRecViews(recs),
		ForecastRows:       fRows,
		IncludeAppendix:    opts.IncludeAppendix,
		AuditFindings:      auditFindings,
		AuditTotal:         len(st.Findings),
		PSIFailing:         psiFailing,
		PSIAll:             psiShow,
		PSITotal:           psiTotal,
		GSCUnder:           gscUnder,
		GSCTotal:           gscTotal,
		SERPQueries:        serpList,
		SERPAI:             serpAI,
		SERPNotP1:          serpNotP1,
		SERPTotal:          serpTotal,
		Backlinks:          st.Backlinks,
		AEOData:            aeoRows,
		AEOPrompts:         aeoPrompts,
		AEOResponses:       aeoResponses,
		QuickWins:          quickWins,
		PSIView:            psiView,
		GSCView:            gscView,
	}
}

// filterQuickWins picks low-effort high-priority recommendations. Used
// by the Quick Wins callout at the top of the report.
func filterQuickWins(in []state.Recommendation) []state.Recommendation {
	out := make([]state.Recommendation, 0, len(in))
	for _, r := range in {
		if r.Priority >= 70 && r.EffortMinutes > 0 && r.EffortMinutes <= 30 {
			out = append(out, r)
		}
		if len(out) >= 5 {
			break
		}
	}
	return out
}

func toPSIView(in []state.PSIResult) []psiRowView {
	out := make([]psiRowView, 0, len(in))
	for _, p := range in {
		out = append(out, psiRowView{
			URL:       p.URL,
			Strategy:  p.Strategy,
			Score:     p.PerformanceScore * 100,
			ScoreBand: scoreBand(p.PerformanceScore * 100),
			LCP:       p.LCP,
			LCPBand:   lcpBand(p.LCP),
			CLS:       p.CLS,
			CLSBand:   clsBand(p.CLS),
		})
	}
	return out
}

func toGSCView(in []state.GSCRow) []gscRowView {
	out := make([]gscRowView, 0, len(in))
	for _, r := range in {
		out = append(out, gscRowView{
			Query:        r.Key,
			Impressions:  r.Impressions,
			Clicks:       r.Clicks,
			CTR:          r.CTR,
			CTRBand:      ctrBand(r.CTR),
			Position:     r.Position,
			PositionBand: positionBand(r.Position),
		})
	}
	return out
}

// scoreBand maps a 0-100 score to a colour band class.
func scoreBand(score float64) string {
	switch {
	case score >= 90:
		return "good"
	case score >= 50:
		return "needs"
	default:
		return "poor"
	}
}

// lcpBand classifies a Largest Contentful Paint (ms) per the Core Web
// Vitals thresholds.
func lcpBand(ms float64) string {
	switch {
	case ms <= 2500:
		return "good"
	case ms <= 4000:
		return "needs"
	default:
		return "poor"
	}
}

// clsBand classifies a Cumulative Layout Shift per the Core Web Vitals
// thresholds.
func clsBand(cls float64) string {
	switch {
	case cls <= 0.1:
		return "good"
	case cls <= 0.25:
		return "needs"
	default:
		return "poor"
	}
}

// ctrBand classifies a GSC CTR (0-1 float) into a colour band.
func ctrBand(ctr float64) string {
	switch {
	case ctr >= 0.05:
		return "good"
	case ctr >= 0.02:
		return "needs"
	default:
		return "poor"
	}
}

// positionBand classifies a GSC average position into a colour band.
// Position 0 means "not ranking" and is treated as poor.
func positionBand(pos float64) string {
	switch {
	case pos > 0 && pos <= 3:
		return "good"
	case pos > 0 && pos <= 10:
		return "needs"
	default:
		return "poor"
	}
}

func toRecViews(rs []state.Recommendation) []recView {
	out := make([]recView, 0, len(rs))
	for _, r := range rs {
		tier := tierOf(r)
		cv := r.CurrentValue
		rv := r.RecommendedValue
		out = append(out, recView{
			R:                r,
			PriorityClass:    priorityClass(r.Priority),
			TierLabel:        tierLabel(tier),
			TierClass:        tierClass(tier),
			TierHeadline:     tierHeadline(r),
			IsProvisional:    isProvisionalTier(r),
			LiftRange:        liftRangeHuman(r),
			EffortLabel:      effortLabel(r),
			CurrentValue:     emptyDash(cv),
			RecommendedValue: emptyDash(rv),
			CurrentLen:       len(cv),
			RecommendedLen:   len(rv),
			CaveatLines:      humanCaveats(r),
			ResearchCitation: researchCitation(r.ChangeType),
			ReviewBadge:      reviewBadge(r),
		})
	}
	return out
}

// isProvisionalTier reports true when the recommendation's tier was
// derived from the rule-engine priority score as a fallback because
// calibration data is insufficient. UI surfaces this as a "(provisional)"
// hint next to the tier badge so readers don't mistake it for a
// calibrated signal.
func isProvisionalTier(r state.Recommendation) bool {
	if r.ForecastedLift == nil {
		return false
	}
	if r.ForecastedLift.ConfidenceLabel != "insufficient_data" {
		return false
	}
	return r.ForecastedLift.PriorityTier != state.PriorityUnknown && r.ForecastedLift.PriorityTier != ""
}

// researchCitation returns a short footnote pointing at the section of
// docs/research/ai-citation-signals-2026.md that underwrites this change
// type, so rendered evidence can credit the source of each lever.
// Returns empty string when no citation is defined.
func researchCitation(ct state.ChangeType) string {
	switch ct {
	case state.ChangeTitle:
		return "Keep (well-supported), §ChangeTitle"
	case state.ChangeMeta:
		return "Demote (weak evidence), §ChangeMeta"
	case state.ChangeH1, state.ChangeH2:
		return "Keep (well-supported), §ChangeH1/H2"
	case state.ChangeSchema:
		return "Keep (well-supported), §ChangeSchema"
	case state.ChangeBody:
		return "Keep (well-supported), §ChangeBody"
	case state.ChangeSpeed:
		return "Demote (weak evidence), §ChangeSpeed"
	case state.ChangeBacklink:
		return "Demote (weak evidence), §ChangeBacklink"
	case state.ChangeIndexability:
		return "Keep (well-supported), §ChangeIndexability"
	case state.ChangeTLDR:
		return "Add, §ChangeDirectAnswerIntro (Growth Memo, 18,012 citations)"
	case state.ChangeListFormat:
		return "Add, §ChangeListFormat (Averi / Growth Memo passage extraction)"
	case state.ChangeAuthorByline:
		return "Add, §ChangeAuthor (E-E-A-T, Person schema)"
	case state.ChangeFreshness:
		return "Add, §ChangeFreshness (Ahrefs 17M citations, Discovered Labs)"
	case state.ChangeEntityConsistency:
		return "Add, §ChangeEntityConsistency (brand mentions > backlinks as citation predictor)"
	}
	return ""
}

// filterReportable drops rejected recommendations. Pending, approved, and
// edited drafts all ship; pending drafts are badged in the template.
func filterReportable(in []state.Recommendation) []state.Recommendation {
	out := make([]state.Recommendation, 0, len(in))
	for _, r := range in {
		if r.EffectiveReviewStatus() == state.ReviewRejected {
			continue
		}
		out = append(out, r)
	}
	return out
}

// recShippedCount returns the number of recommendations that count towards
// the executive-summary headline — approved + edited only. Pending drafts
// are visible in the report but don't inflate the "X recommendations" count.
func recShippedCount(in []state.Recommendation) int {
	n := 0
	for _, r := range in {
		switch r.EffectiveReviewStatus() {
		case state.ReviewApproved, state.ReviewEdited, "":
			n++
		}
	}
	return n
}

// forecastTotalsOf is forecastTotals but operates on a supplied slice so we
// can exclude rejected recs and (for the headline) pending ones.
func forecastTotalsOf(in []state.Recommendation) (total, low, high int) {
	for _, r := range in {
		// Pending drafts are shown in the report but shouldn't inflate the
		// forecasted-uplift headline until a human signs off.
		if r.EffectiveReviewStatus() == state.ReviewPending {
			continue
		}
		if r.ForecastedLift != nil {
			total += r.ForecastedLift.Point()
			low += r.ForecastedLift.Low()
			high += r.ForecastedLift.High()
		}
	}
	return
}

// tierCounts returns (high, medium, low, unknown) counts across the
// supplied recommendations. Pending drafts are excluded so the exec
// summary stays aligned with forecastTotalsOf.
func tierCounts(in []state.Recommendation) (hi, med, lo, unk int) {
	for _, r := range in {
		if r.EffectiveReviewStatus() == state.ReviewPending {
			continue
		}
		switch tierOf(r) {
		case state.PriorityHigh:
			hi++
		case state.PriorityMedium:
			med++
		case state.PriorityLow:
			lo++
		default:
			unk++
		}
	}
	return
}

// calibrationSummary reports whether any recommendation in the report
// was calibrated against historical data, and if so how many historical
// samples backed the most-calibrated entry.
func calibrationSummary(in []state.Recommendation) (bool, int) {
	has := false
	maxSamples := 0
	for _, r := range in {
		if r.ForecastedLift == nil {
			continue
		}
		if r.ForecastedLift.IsCalibrated() {
			has = true
		}
		if r.ForecastedLift.CalibrationSamples > maxSamples {
			maxSamples = r.ForecastedLift.CalibrationSamples
		}
	}
	return has, maxSamples
}

// reviewBadge returns a short badge string for pending-review drafts and
// the empty string otherwise.
func reviewBadge(r state.Recommendation) string {
	if r.EffectiveReviewStatus() == state.ReviewPending {
		return "🔍 Pending review"
	}
	return ""
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func sortedRecs(in []state.Recommendation) []state.Recommendation {
	out := make([]state.Recommendation, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Priority > out[j].Priority })
	return out
}

func sortedByForecast(in []state.Recommendation) []state.Recommendation {
	out := make([]state.Recommendation, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		return liftEst(out[i]) > liftEst(out[j])
	})
	return out
}

func liftEst(r state.Recommendation) int  { return r.ForecastedLift.Point() }
func liftLow(r state.Recommendation) int  { return r.ForecastedLift.Low() }
func liftHigh(r state.Recommendation) int { return r.ForecastedLift.High() }

// liftRangeHuman renders the estimated range without implying a
// specific point number when calibration data is insufficient.
func liftRangeHuman(r state.Recommendation) string {
	if r.ForecastedLift == nil {
		return "."
	}
	f := r.ForecastedLift
	if f.ConfidenceLabel == "insufficient_data" {
		return fmt.Sprintf("~%s to %s clicks/mo (unverified)",
			formatInt(f.Low()), formatInt(f.High()))
	}
	return fmt.Sprintf("%s to %s clicks/mo",
		formatInt(f.Low()), formatInt(f.High()))
}

// tierOf returns the tier stored on the ForecastedLift, or
// PriorityUnknown when no forecast is attached.
func tierOf(r state.Recommendation) state.PriorityTier {
	if r.ForecastedLift == nil || r.ForecastedLift.PriorityTier == "" {
		return state.PriorityUnknown
	}
	return r.ForecastedLift.PriorityTier
}

func tierLabel(t state.PriorityTier) string {
	switch t {
	case state.PriorityHigh:
		return "HIGH"
	case state.PriorityMedium:
		return "MEDIUM"
	case state.PriorityLow:
		return "LOW"
	default:
		return "UNKNOWN"
	}
}

func tierClass(t state.PriorityTier) string {
	switch t {
	case state.PriorityHigh:
		return "tier-high"
	case state.PriorityMedium:
		return "tier-medium"
	case state.PriorityLow:
		return "tier-low"
	default:
		return "tier-unknown"
	}
}

// tierHeadline renders the one-line plain-English summary that leads
// every recommendation card: tier first, range second, caveat hint
// third.
func tierHeadline(r state.Recommendation) string {
	tier := tierLabel(tierOf(r))
	if r.ForecastedLift == nil {
		return fmt.Sprintf("Priority: %s. No traffic forecast available.", tier)
	}
	f := r.ForecastedLift
	switch f.ConfidenceLabel {
	case "insufficient_data":
		return fmt.Sprintf("Priority: %s. Likely meaningful traffic impact. Specific click numbers are not reliable yet (need at least %d past %s outcomes to calibrate).",
			tier, 20, r.ChangeType)
	case "low_confidence":
		return fmt.Sprintf("Priority: %s. Estimated %s to %s more clicks/month, but similar-type history is thin so take the specific numbers with a grain of salt.",
			tier, formatInt(f.Low()), formatInt(f.High()))
	default:
		return fmt.Sprintf("Priority: %s. Estimated %s to %s more clicks/month if implemented (calibrated against %d past outcomes).",
			tier, formatInt(f.Low()), formatInt(f.High()), f.CalibrationSamples)
	}
}

// humanCaveats renders a Forecast's caveat list into plain-English
// phrases ready to show in the UI.
func humanCaveats(r state.Recommendation) []string {
	if r.ForecastedLift == nil {
		return nil
	}
	var out []string
	for _, c := range r.ForecastedLift.Caveats {
		switch c {
		case "insufficient_calibration_data":
			out = append(out, "Historical calibration data is thin: the click range here is the raw model output, not verified against past outcomes.")
		case "low_confidence":
			out = append(out, "Similar-type history is thin; specific numbers may be off.")
		case "low_search_volume":
			out = append(out, "Low search volume for this query makes predictions noisy.")
		case "forecaster_tends_to_overshoot":
			out = append(out, "For this change type, past forecasts have overshot reality; the range has been adjusted down.")
		case "forecaster_tends_to_undershoot":
			out = append(out, "For this change type, past forecasts have undershot reality; the range has been adjusted up.")
		default:
			out = append(out, c)
		}
	}
	return out
}

func effortLabel(r state.Recommendation) string {
	if r.EffortMinutes <= 0 {
		return "."
	}
	return fmt.Sprintf("%d min", r.EffortMinutes)
}

func priorityClass(p int) string {
	switch {
	case p >= 80:
		return "priority-high"
	case p >= 60:
		return "priority-med"
	default:
		return "priority-low"
	}
}

func aeoBrandCoverage(st *state.State) float64 {
	if st.AEO == nil || len(st.AEO.Responses) == 0 {
		return -1
	}
	if len(st.BrandTerms) == 0 {
		return -1
	}
	hits := 0
	for _, p := range st.AEO.Responses {
		if promptMentionsBrand(p, st.BrandTerms) {
			hits++
		}
	}
	return float64(hits) / float64(len(st.AEO.Responses)) * 100
}

func promptMentionsBrand(p state.AEOPromptResult, brandTerms []string) bool {
	for _, res := range p.Results {
		low := strings.ToLower(res.Response)
		for _, t := range brandTerms {
			if t == "" {
				continue
			}
			if strings.Contains(low, strings.ToLower(t)) {
				return true
			}
		}
	}
	return false
}

func safeSite(s string) string {
	if s == "" {
		return "(no site)"
	}
	return s
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(empty)"
	}
	return s
}

func formatInt(n int) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "…"
}

func evidenceLine(e state.Evidence) string {
	if e.Description != "" {
		return e.Description
	}
	if e.Metric != "" && e.Value != nil {
		return fmt.Sprintf("%s = %v", e.Metric, e.Value)
	}
	if e.Metric != "" {
		return e.Metric
	}
	return fmt.Sprintf("%v", e.Value)
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"formatInt":    formatInt,
		"formatInt64":  func(n int64) string { return formatInt(int(n)) },
		"truncate":     truncate,
		"evidenceLine": evidenceLine,
		"pct":          func(f float64) string { return fmt.Sprintf("%.0f%%", f) },
		"pctCTR":       func(f float64) string { return fmt.Sprintf("%.1f%%", f*100) },
		"mul100":       func(f float64) float64 { return f * 100 },
		"pctPerf":      func(f float64) string { return fmt.Sprintf("%.0f", f*100) },
		"fmtFloat":     func(f float64, d int) string { return fmt.Sprintf("%.*f", d, f) },
		"yesNo": func(b bool) string {
			if b {
				return "yes"
			}
			return "no"
		},
		"hasPrefix": strings.HasPrefix,
		"lower":     strings.ToLower,
		"nonZero":   func(n int) bool { return n != 0 },
		// Visual helpers for the redesigned report.
		"gaugeColor":     gaugeColor,
		"gaugeDashArray": gaugeDashArray,
		"tierIcon":       tierIcon,
		"bandIcon":       bandIcon,
		"priorityTotal": func(v viewModel) int {
			return v.TierHighCount + v.TierMediumCount + v.TierLowCount + v.TierUnknownCount
		},
		"dict": func(pairs ...interface{}) map[string]interface{} {
			m := map[string]interface{}{}
			for i := 0; i+1 < len(pairs); i += 2 {
				if k, ok := pairs[i].(string); ok {
					m[k] = pairs[i+1]
				}
			}
			return m
		},
	}
}

// gaugeColor returns the stroke colour for a circular score gauge.
// Palette matches the Core Web Vitals bands used throughout the report.
func gaugeColor(score float64) string {
	switch {
	case score >= 90:
		return "#047857"
	case score >= 50:
		return "#CA8A04"
	default:
		return "#B91C1C"
	}
}

// gaugeDashArray returns the SVG stroke-dasharray for a score so the
// ring fills proportionally. Circumference for r=52 is ~326.7.
func gaugeDashArray(score float64) string {
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	const circumference = 326.73
	filled := circumference * score / 100
	return fmt.Sprintf("%.2f %.2f", filled, circumference-filled)
}

// tierIcon returns a short glyph prefix for a priority tier badge so
// colour is never the sole signal.
func tierIcon(label string) string {
	switch label {
	case "HIGH":
		return "▲"
	case "MEDIUM":
		return "◆"
	case "LOW":
		return "▾"
	default:
		return "?"
	}
}

// bandIcon returns a glyph for a CWV / position / CTR band so colour
// is never the sole signal.
func bandIcon(band string) string {
	switch band {
	case "good":
		return "✓ "
	case "needs":
		return "! "
	case "poor":
		return "✕ "
	}
	return ""
}
