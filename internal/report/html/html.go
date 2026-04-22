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
		"templates/cover.html",
		"templates/summary.html",
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
	Title           string
	BrandColor      string
	LogoDataURI     template.URL
	Styles          template.CSS
	GeneratedAt     string
	Site            string
	Score           float64
	PagesCrawled    int
	IssueCount      int
	RecCount        int
	TotalLift       int
	ForecastLow     int
	ForecastHigh    int
	AEOCoverage     float64
	HasAEOCoverage  bool
	TopRecs         []recView
	AllRecs         []recView
	ForecastRows    []forecastRow
	IncludeAppendix bool

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
}

type recView struct {
	R                state.Recommendation
	PriorityClass    string
	LiftRange        string
	EffortLabel      string
	CurrentValue     string
	RecommendedValue string
}

type forecastRow struct {
	Index      int
	URL        string
	Query      string
	ChangeType string
	Priority   int
	LiftEst    int
	LiftLow    int
	LiftHigh   int
	LiftRange  string
	Effort     string
}

type aeoRow struct {
	Prompt         string
	EngineCount    int
	BrandMentioned bool
}

func buildViewModel(st *state.State, opts Options, styles template.CSS) viewModel {
	recs := sortedRecs(st.Recommendations)
	totalLift, lo, hi := forecastTotals(st)
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
	fRecs := sortedByForecast(st.Recommendations)
	if len(fRecs) > 20 {
		fRecs = fRecs[:20]
	}
	var fRows []forecastRow
	for i, r := range fRecs {
		fRows = append(fRows, forecastRow{
			Index:      i + 1,
			URL:        r.TargetURL,
			Query:      r.TargetQuery,
			ChangeType: string(r.ChangeType),
			Priority:   r.Priority,
			LiftEst:    liftEst(r),
			LiftLow:    liftLow(r),
			LiftHigh:   liftHigh(r),
			LiftRange:  liftRange(r),
			Effort:     effortLabel(r),
		})
	}

	return viewModel{
		Title:           opts.Title,
		BrandColor:      opts.BrandColorHex,
		LogoDataURI:     template.URL(opts.LogoDataURI),
		Styles:          styles,
		GeneratedAt:     time.Now().UTC().Format("2 Jan 2006"),
		Site:            safeSite(st.Site),
		Score:           st.Score,
		PagesCrawled:    st.PagesCrawled,
		IssueCount:      len(st.Findings),
		RecCount:        len(st.Recommendations),
		TotalLift:       totalLift,
		ForecastLow:     lo,
		ForecastHigh:    hi,
		AEOCoverage:     coverage,
		HasAEOCoverage:  hasCoverage,
		TopRecs:         toRecViews(topRaw),
		AllRecs:         toRecViews(recs),
		ForecastRows:    fRows,
		IncludeAppendix: opts.IncludeAppendix,
		AuditFindings:   auditFindings,
		AuditTotal:      len(st.Findings),
		PSIFailing:      psiFailing,
		PSIAll:          psiShow,
		PSITotal:        psiTotal,
		GSCUnder:        gscUnder,
		GSCTotal:        gscTotal,
		SERPQueries:     serpList,
		SERPAI:          serpAI,
		SERPNotP1:       serpNotP1,
		SERPTotal:       serpTotal,
		Backlinks:       st.Backlinks,
		AEOData:         aeoRows,
		AEOPrompts:      aeoPrompts,
		AEOResponses:    aeoResponses,
	}
}

func toRecViews(rs []state.Recommendation) []recView {
	out := make([]recView, 0, len(rs))
	for _, r := range rs {
		out = append(out, recView{
			R:                r,
			PriorityClass:    priorityClass(r.Priority),
			LiftRange:        liftRange(r),
			EffortLabel:      effortLabel(r),
			CurrentValue:     emptyDash(r.CurrentValue),
			RecommendedValue: emptyDash(r.RecommendedValue),
		})
	}
	return out
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

func forecastTotals(st *state.State) (total, low, high int) {
	for _, r := range st.Recommendations {
		if r.ForecastedLift != nil {
			total += r.ForecastedLift.EstimatedMonthlyClicksDelta
			low += r.ForecastedLift.ConfidenceLow
			high += r.ForecastedLift.ConfidenceHigh
		}
	}
	return
}

func liftEst(r state.Recommendation) int {
	if r.ForecastedLift == nil {
		return 0
	}
	return r.ForecastedLift.EstimatedMonthlyClicksDelta
}

func liftLow(r state.Recommendation) int {
	if r.ForecastedLift == nil {
		return 0
	}
	return r.ForecastedLift.ConfidenceLow
}

func liftHigh(r state.Recommendation) int {
	if r.ForecastedLift == nil {
		return 0
	}
	return r.ForecastedLift.ConfidenceHigh
}

func liftRange(r state.Recommendation) string {
	if r.ForecastedLift == nil {
		return "—"
	}
	return fmt.Sprintf("+%s clicks/mo (range %s–%s)",
		formatInt(r.ForecastedLift.EstimatedMonthlyClicksDelta),
		formatInt(r.ForecastedLift.ConfidenceLow),
		formatInt(r.ForecastedLift.ConfidenceHigh))
}

func effortLabel(r state.Recommendation) string {
	if r.EffortMinutes <= 0 {
		return "—"
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
		return "—"
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
	}
}
