// Package pdf renders a styled PDF summary of the Sageo audit, recommendations,
// and forecast from a populated state.State.
//
// PDF library: codeberg.org/go-pdf/fpdf — active successor to the archived
// github.com/jung-kurt/gofpdf (and the later-archived github.com/go-pdf/fpdf).
// Pure-Go, no CGO, no headless browser required.
package pdf

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"codeberg.org/go-pdf/fpdf"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

// Options controls PDF rendering.
type Options struct {
	IncludeAppendix bool
	LogoPath        string
	BrandColorHex   string

	// disableCompression is used by tests to assert on raw text content.
	disableCompression bool
}

const (
	defaultBrandHex = "#1E40AF"
	pageWidthMM     = 210.0
	marginLeft      = 15.0
	marginRight     = 15.0
	marginTop       = 15.0
	contentWidth    = pageWidthMM - marginLeft - marginRight
)

// Render writes a styled PDF report for st into out.
func Render(st *state.State, out io.Writer, opts Options) error {
	_, _, err := RenderWithStats(st, out, opts)
	return err
}

// RenderWithStats renders and returns the total page count and bytes written.
func RenderWithStats(st *state.State, out io.Writer, opts Options) (pages int, bytesWritten int64, err error) {
	if st == nil {
		return 0, 0, fmt.Errorf("pdf: state is nil")
	}
	if out == nil {
		return 0, 0, fmt.Errorf("pdf: writer is nil")
	}
	cw := &countingWriter{w: out}
	brandR, brandG, brandB := parseHex(opts.BrandColorHex, defaultBrandHex)

	defer func() {
		bytesWritten = cw.n
	}()

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginLeft, marginTop, marginRight)
	pdf.SetAutoPageBreak(true, 18)
	pdf.AliasNbPages("")
	if opts.disableCompression {
		pdf.SetCompression(false)
	}

	onCover := true
	pdf.SetHeaderFunc(func() {
		if onCover {
			return
		}
		pdf.SetFillColor(brandR, brandG, brandB)
		pdf.Rect(0, 0, pageWidthMM, 10, "F")
		pdf.SetY(3)
		pdf.SetX(marginLeft)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetTextColor(255, 255, 255)
		title := fmt.Sprintf("Sageo SEO Report  —  %s", safeSite(st.Site))
		pdf.CellFormat(contentWidth, 4, asciiSafe(title), "", 0, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetY(marginTop + 2)
	})
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.SetTextColor(120, 120, 120)
		pdf.CellFormat(contentWidth, 8,
			fmt.Sprintf("Page %d of {nb}", pdf.PageNo()),
			"", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	})

	pdf.AddPage()
	renderCover(pdf, st, brandR, brandG, brandB, opts.LogoPath)
	onCover = false
	pdf.AddPage()
	renderExecutiveSummary(pdf, st, brandR, brandG, brandB)
	renderWhatsBroken(pdf, st, brandR, brandG, brandB)
	renderRecommendations(pdf, st, brandR, brandG, brandB)
	renderForecastSummary(pdf, st, brandR, brandG, brandB)
	if opts.IncludeAppendix {
		renderAppendix(pdf, st, brandR, brandG, brandB)
	}

	pages = pdf.PageCount()
	if err := pdf.Output(cw); err != nil {
		return pages, cw.n, fmt.Errorf("pdf: output: %w", err)
	}
	if err := pdf.Error(); err != nil {
		return pages, cw.n, fmt.Errorf("pdf: render: %w", err)
	}
	return pages, cw.n, nil
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

// ---------------------------------------------------------------------------
// Section renderers
// ---------------------------------------------------------------------------

func renderCover(pdf *fpdf.Fpdf, st *state.State, br, bg, bb int, logoPath string) {
	// Brand block filling the top third.
	pdf.SetFillColor(br, bg, bb)
	pdf.Rect(0, 0, pageWidthMM, 90, "F")

	if logoPath != "" {
		pdf.ImageOptions(logoPath, marginLeft, 15, 40, 0, false,
			fpdf.ImageOptions{ReadDpi: true}, 0, "")
	}

	pdf.SetTextColor(255, 255, 255)
	pdf.SetY(45)
	pdf.SetX(marginLeft)
	pdf.SetFont("Helvetica", "B", 28)
	pdf.CellFormat(contentWidth, 12, "SEO Performance Report", "", 1, "L", false, 0, "")

	pdf.SetX(marginLeft)
	pdf.SetFont("Helvetica", "", 14)
	pdf.CellFormat(contentWidth, 8, asciiSafe(safeSite(st.Site)), "", 1, "L", false, 0, "")

	pdf.SetX(marginLeft)
	pdf.SetFont("Helvetica", "", 11)
	pdf.CellFormat(contentWidth, 7,
		"Generated "+time.Now().UTC().Format("2 Jan 2006"),
		"", 1, "L", false, 0, "")

	pdf.SetTextColor(0, 0, 0)

	// Headline stats below the brand block.
	pdf.SetY(105)
	recCount := len(st.Recommendations)
	totalLift := totalForecast(st)

	drawHeadlineCard(pdf, marginLeft, pdf.GetY(), (contentWidth-6)/2, 35,
		"Recommendations", fmt.Sprintf("%d", recCount),
		"prioritised actions", br, bg, bb)
	drawHeadlineCard(pdf, marginLeft+(contentWidth-6)/2+6, pdf.GetY(), (contentWidth-6)/2, 35,
		"Forecast monthly clicks", fmt.Sprintf("+%s", formatInt(totalLift)),
		"estimated uplift", br, bg, bb)

	pdf.SetY(155)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(90, 90, 90)
	pdf.MultiCell(contentWidth, 5,
		asciiSafe("This report summarises what is underperforming on your site, the prioritised changes we recommend, and the forecasted traffic impact of implementing them."),
		"", "L", false)
	pdf.SetTextColor(0, 0, 0)
}

func drawHeadlineCard(pdf *fpdf.Fpdf, x, y, w, h float64, label, value, caption string, br, bg, bb int) {
	pdf.SetDrawColor(220, 220, 220)
	pdf.SetLineWidth(0.3)
	pdf.Rect(x, y, w, h, "D")

	pdf.SetXY(x+4, y+4)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(120, 120, 120)
	pdf.CellFormat(w-8, 5, strings.ToUpper(label), "", 2, "L", false, 0, "")

	pdf.SetXY(x+4, y+12)
	pdf.SetFont("Helvetica", "B", 22)
	pdf.SetTextColor(br, bg, bb)
	pdf.CellFormat(w-8, 10, value, "", 2, "L", false, 0, "")

	pdf.SetXY(x+4, y+25)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(120, 120, 120)
	pdf.CellFormat(w-8, 5, caption, "", 2, "L", false, 0, "")

	pdf.SetTextColor(0, 0, 0)
}

func renderExecutiveSummary(pdf *fpdf.Fpdf, st *state.State, br, bg, bb int) {
	h1(pdf, "Executive Summary", br, bg, bb)

	issues := len(st.Findings)
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(contentWidth, 5, asciiSafe(fmt.Sprintf(
		"Audit score: %.0f / 100.   Pages crawled: %d.   Issues found: %d.   Recommendations: %d.",
		st.Score, st.PagesCrawled, issues, len(st.Recommendations))),
		"", "L", false)
	pdf.Ln(2)

	h2(pdf, "Top 5 recommendations")
	recs := sortedRecs(st.Recommendations)
	top := recs
	if len(top) > 5 {
		top = top[:5]
	}
	if len(top) == 0 {
		pdf.SetFont("Helvetica", "I", 10)
		pdf.MultiCell(contentWidth, 5, "No recommendations generated yet. Run `sageo run` first.", "", "L", false)
	} else {
		pdf.SetFont("Helvetica", "", 10)
		for _, r := range top {
			drawPriorityBadge(pdf, r.Priority)
			pdf.SetX(marginLeft + 14)
			line := fmt.Sprintf("%s on %s", string(r.ChangeType), truncate(r.TargetURL, 55))
			pdf.SetFont("Helvetica", "B", 10)
			pdf.CellFormat(contentWidth-14, 5, asciiSafe(line), "", 1, "L", false, 0, "")
			if r.Rationale != "" {
				pdf.SetX(marginLeft + 14)
				pdf.SetFont("Helvetica", "", 9)
				pdf.SetTextColor(80, 80, 80)
				pdf.MultiCell(contentWidth-14, 4.5, asciiSafe(truncate(r.Rationale, 160)), "", "L", false)
				pdf.SetTextColor(0, 0, 0)
			}
			pdf.Ln(1)
		}
	}

	pdf.Ln(3)
	h2(pdf, "Forecast uplift")
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(contentWidth, 5, asciiSafe(fmt.Sprintf(
		"Total forecasted monthly click delta across all recommendations: +%s.",
		formatInt(totalForecast(st)))), "", "L", false)

	pdf.Ln(2)
	h2(pdf, "AEO coverage")
	pct := aeoBrandCoverage(st)
	pdf.SetFont("Helvetica", "", 10)
	if pct < 0 {
		pdf.MultiCell(contentWidth, 5, "No AEO data captured. Run `sageo aeo ask` against the target prompts.", "", "L", false)
	} else {
		pdf.MultiCell(contentWidth, 5, asciiSafe(fmt.Sprintf(
			"Brand mentioned in %.0f%% of prompts across all tested AI models.", pct)),
			"", "L", false)
	}
}

func renderWhatsBroken(pdf *fpdf.Fpdf, st *state.State, br, bg, bb int) {
	sections := []struct {
		title string
		draw  func()
	}{
		{"Crawl & audit issues", func() { drawAuditSection(pdf, st) }},
		{"PageSpeed (PSI)", func() { drawPSISection(pdf, st) }},
		{"GSC underperforming queries", func() { drawGSCSection(pdf, st) }},
		{"SERP competitive gaps", func() { drawSERPSection(pdf, st) }},
		{"Backlinks profile", func() { drawBacklinksSection(pdf, st) }},
		{"AEO visibility", func() { drawAEOSection(pdf, st) }},
	}
	for _, s := range sections {
		pdf.AddPage()
		h1(pdf, "What's broken: "+s.title, br, bg, bb)
		s.draw()
	}
}

func drawAuditSection(pdf *fpdf.Fpdf, st *state.State) {
	pdf.SetFont("Helvetica", "", 10)
	if len(st.Findings) == 0 {
		pdf.MultiCell(contentWidth, 5, "No audit findings recorded.", "", "L", false)
		return
	}
	pdf.MultiCell(contentWidth, 5, asciiSafe(fmt.Sprintf(
		"%d audit findings across %d crawled pages. Items below are the top fixes blocking crawl, indexing, or basic on-page quality.",
		len(st.Findings), st.PagesCrawled)), "", "L", false)
	pdf.Ln(2)

	drawTableHeader(pdf, []string{"Rule", "URL", "Verdict"}, []float64{50, 95, 35})
	n := len(st.Findings)
	if n > 20 {
		n = 20
	}
	for i := 0; i < n; i++ {
		f := st.Findings[i]
		drawTableRow(pdf, []string{f.Rule, truncate(f.URL, 60), f.Verdict}, []float64{50, 95, 35})
	}
}

func drawPSISection(pdf *fpdf.Fpdf, st *state.State) {
	pdf.SetFont("Helvetica", "", 10)
	if st.PSI == nil || len(st.PSI.Pages) == 0 {
		pdf.MultiCell(contentWidth, 5, "No PageSpeed Insights data captured.", "", "L", false)
		return
	}
	var failing []state.PSIResult
	for _, p := range st.PSI.Pages {
		if p.PerformanceScore < 0.9 || p.LCP > 2500 || p.CLS > 0.1 {
			failing = append(failing, p)
		}
	}
	pdf.MultiCell(contentWidth, 5, asciiSafe(fmt.Sprintf(
		"%d pages measured, %d failing Core Web Vitals thresholds (perf<0.9, LCP>2.5s, or CLS>0.1).",
		len(st.PSI.Pages), len(failing))), "", "L", false)
	pdf.Ln(2)

	drawTableHeader(pdf, []string{"URL", "Perf", "LCP ms", "CLS", "Device"},
		[]float64{90, 20, 25, 20, 25})
	list := failing
	if len(list) == 0 {
		list = st.PSI.Pages
	}
	if len(list) > 20 {
		list = list[:20]
	}
	for _, p := range list {
		drawTableRow(pdf, []string{
			truncate(p.URL, 55),
			fmt.Sprintf("%.0f", p.PerformanceScore*100),
			fmt.Sprintf("%.0f", p.LCP),
			fmt.Sprintf("%.2f", p.CLS),
			p.Strategy,
		}, []float64{90, 20, 25, 20, 25})
	}
}

func drawGSCSection(pdf *fpdf.Fpdf, st *state.State) {
	pdf.SetFont("Helvetica", "", 10)
	if st.GSC == nil || len(st.GSC.TopKeywords) == 0 {
		pdf.MultiCell(contentWidth, 5, "No Google Search Console data captured.", "", "L", false)
		return
	}
	// "Underperforming" = high impressions but low CTR or position > 10.
	var under []state.GSCRow
	for _, r := range st.GSC.TopKeywords {
		if r.Impressions >= 100 && (r.Position > 10 || r.CTR < 0.02) {
			under = append(under, r)
		}
	}
	sort.Slice(under, func(i, j int) bool { return under[i].Impressions > under[j].Impressions })

	pdf.MultiCell(contentWidth, 5, asciiSafe(fmt.Sprintf(
		"%d queries tracked; %d are underperforming (impressions over 100 but position >10 or CTR <2%%).",
		len(st.GSC.TopKeywords), len(under))), "", "L", false)
	pdf.Ln(2)

	drawTableHeader(pdf, []string{"Query", "Impr", "Clicks", "CTR", "Pos"},
		[]float64{90, 25, 25, 20, 20})
	if len(under) > 20 {
		under = under[:20]
	}
	for _, r := range under {
		drawTableRow(pdf, []string{
			truncate(r.Key, 55),
			formatInt(int(r.Impressions)),
			formatInt(int(r.Clicks)),
			fmt.Sprintf("%.1f%%", r.CTR*100),
			fmt.Sprintf("%.1f", r.Position),
		}, []float64{90, 25, 25, 20, 20})
	}
}

func drawSERPSection(pdf *fpdf.Fpdf, st *state.State) {
	pdf.SetFont("Helvetica", "", 10)
	if st.SERP == nil || len(st.SERP.Queries) == 0 {
		pdf.MultiCell(contentWidth, 5, "No SERP data captured.", "", "L", false)
		return
	}
	aiCount := 0
	notRanking := 0
	for _, q := range st.SERP.Queries {
		if q.HasAIOverview {
			aiCount++
		}
		if q.OurPosition <= 0 || q.OurPosition > 10 {
			notRanking++
		}
	}
	pdf.MultiCell(contentWidth, 5, asciiSafe(fmt.Sprintf(
		"%d queries analysed. %d triggered AI Overviews. %d not ranking on page 1.",
		len(st.SERP.Queries), aiCount, notRanking)), "", "L", false)
	pdf.Ln(2)

	drawTableHeader(pdf, []string{"Query", "Our pos", "AIO", "Top domains"},
		[]float64{70, 20, 15, 75})
	n := len(st.SERP.Queries)
	if n > 20 {
		n = 20
	}
	for i := 0; i < n; i++ {
		q := st.SERP.Queries[i]
		pos := "-"
		if q.OurPosition > 0 {
			pos = fmt.Sprintf("%d", q.OurPosition)
		}
		aio := "no"
		if q.HasAIOverview {
			aio = "yes"
		}
		domains := ""
		if len(q.TopDomains) > 0 {
			d := q.TopDomains
			if len(d) > 3 {
				d = d[:3]
			}
			domains = strings.Join(d, ", ")
		}
		drawTableRow(pdf, []string{truncate(q.Query, 40), pos, aio, truncate(domains, 45)},
			[]float64{70, 20, 15, 75})
	}
}

func drawBacklinksSection(pdf *fpdf.Fpdf, st *state.State) {
	pdf.SetFont("Helvetica", "", 10)
	if st.Backlinks == nil || st.Backlinks.LastRun == "" {
		pdf.MultiCell(contentWidth, 5, "No backlinks data captured.", "", "L", false)
		return
	}
	bl := st.Backlinks
	pdf.MultiCell(contentWidth, 5, asciiSafe(fmt.Sprintf(
		"%s backlinks from %s referring domains. %s broken. Spam score %.0f.",
		formatInt64(bl.TotalBacklinks), formatInt64(bl.TotalReferringDomains),
		formatInt64(bl.BrokenBacklinks), bl.SpamScore)), "", "L", false)
	pdf.Ln(2)

	drawTableHeader(pdf, []string{"Metric", "Value"}, []float64{80, 100})
	rows := [][2]string{
		{"Total backlinks", formatInt64(bl.TotalBacklinks)},
		{"Referring domains", formatInt64(bl.TotalReferringDomains)},
		{"DoFollow", formatInt64(bl.DoFollow)},
		{"NoFollow", formatInt64(bl.NoFollow)},
		{"Broken backlinks", formatInt64(bl.BrokenBacklinks)},
		{"Spam score", fmt.Sprintf("%.0f", bl.SpamScore)},
		{"Gap domains identified", fmt.Sprintf("%d", len(bl.GapDomains))},
	}
	for _, r := range rows {
		drawTableRow(pdf, []string{r[0], r[1]}, []float64{80, 100})
	}
}

func drawAEOSection(pdf *fpdf.Fpdf, st *state.State) {
	pdf.SetFont("Helvetica", "", 10)
	if st.AEO == nil || len(st.AEO.Responses) == 0 {
		pdf.MultiCell(contentWidth, 5, "No AEO data captured. Run `sageo aeo ask` to test AI visibility.", "", "L", false)
		return
	}
	pct := aeoBrandCoverage(st)
	pdf.MultiCell(contentWidth, 5, asciiSafe(fmt.Sprintf(
		"%d prompts tested across %d responses. Brand-term coverage: %.0f%%.",
		len(st.AEO.Responses), aeoResponseCount(st), pct)), "", "L", false)
	pdf.Ln(2)

	drawTableHeader(pdf, []string{"Prompt", "Engines", "Brand mentioned"},
		[]float64{100, 40, 40})
	n := len(st.AEO.Responses)
	if n > 20 {
		n = 20
	}
	for i := 0; i < n; i++ {
		p := st.AEO.Responses[i]
		mentioned := promptMentionsBrand(p, st.BrandTerms)
		m := "no"
		if mentioned {
			m = "yes"
		}
		drawTableRow(pdf, []string{truncate(p.Prompt, 65),
			fmt.Sprintf("%d", len(p.Results)), m}, []float64{100, 40, 40})
	}
}

func renderRecommendations(pdf *fpdf.Fpdf, st *state.State, br, bg, bb int) {
	pdf.AddPage()
	h1(pdf, "Recommendations", br, bg, bb)
	recs := sortedRecs(st.Recommendations)
	if len(recs) == 0 {
		pdf.SetFont("Helvetica", "I", 10)
		pdf.MultiCell(contentWidth, 5, "No recommendations generated yet.", "", "L", false)
		return
	}
	for _, r := range recs {
		drawRecommendationCard(pdf, r, br, bg, bb)
	}
}

func drawRecommendationCard(pdf *fpdf.Fpdf, r state.Recommendation, br, bg, bb int) {
	// Soft page-break guard — if less than 55mm remaining, new page.
	if pdf.GetY() > 230 {
		pdf.AddPage()
	}
	startY := pdf.GetY() + 2
	pdf.SetY(startY)

	drawPriorityBadge(pdf, r.Priority)
	pdf.SetX(marginLeft + 14)
	pdf.SetFont("Helvetica", "B", 11)
	heading := fmt.Sprintf("%s on %s", string(r.ChangeType), truncate(r.TargetURL, 55))
	pdf.CellFormat(contentWidth-14, 6, asciiSafe(heading), "", 1, "L", false, 0, "")

	// Side-by-side current / recommended values.
	if r.CurrentValue != "" || r.RecommendedValue != "" {
		pdf.Ln(1)
		colW := (contentWidth - 4) / 2
		y := pdf.GetY()

		pdf.SetX(marginLeft)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(120, 120, 120)
		pdf.CellFormat(colW, 4, "CURRENT", "", 0, "L", false, 0, "")
		pdf.SetX(marginLeft + colW + 4)
		pdf.CellFormat(colW, 4, "RECOMMENDED", "", 1, "L", false, 0, "")

		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetXY(marginLeft, y+5)
		pdf.MultiCell(colW, 4.5, asciiSafe(emptyDash(r.CurrentValue)), "", "L", false)
		endLeft := pdf.GetY()

		pdf.SetXY(marginLeft+colW+4, y+5)
		pdf.MultiCell(colW, 4.5, asciiSafe(emptyDash(r.RecommendedValue)), "", "L", false)
		endRight := pdf.GetY()

		if endLeft > endRight {
			pdf.SetY(endLeft)
		} else {
			pdf.SetY(endRight)
		}
	}

	if r.Rationale != "" {
		pdf.Ln(1)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(120, 120, 120)
		pdf.CellFormat(contentWidth, 4, "RATIONALE", "", 1, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Helvetica", "", 9)
		pdf.MultiCell(contentWidth, 4.5, asciiSafe(r.Rationale), "", "L", false)
	}

	if len(r.Evidence) > 0 {
		pdf.Ln(0.5)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(120, 120, 120)
		pdf.CellFormat(contentWidth, 4, "EVIDENCE", "", 1, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Helvetica", "", 9)
		for _, e := range r.Evidence {
			line := fmt.Sprintf("- [%s] %s", e.Source, evidenceLine(e))
			pdf.MultiCell(contentWidth, 4.5, asciiSafe(line), "", "L", false)
		}
	}

	// Forecast + effort row.
	pdf.Ln(0.5)
	pdf.SetFont("Helvetica", "", 9)
	lift := "-"
	if r.ForecastedLift != nil {
		lift = fmt.Sprintf("+%s clicks/mo (range %s..%s)",
			formatInt(r.ForecastedLift.EstimatedMonthlyClicksDelta),
			formatInt(r.ForecastedLift.ConfidenceLow),
			formatInt(r.ForecastedLift.ConfidenceHigh))
	}
	effort := "-"
	if r.EffortMinutes > 0 {
		effort = fmt.Sprintf("%d min", r.EffortMinutes)
	}
	pdf.MultiCell(contentWidth, 4.5,
		asciiSafe(fmt.Sprintf("Forecast: %s   |   Effort: %s", lift, effort)),
		"", "L", false)

	// Separator.
	pdf.Ln(1)
	pdf.SetDrawColor(230, 230, 230)
	pdf.Line(marginLeft, pdf.GetY(), marginLeft+contentWidth, pdf.GetY())
	pdf.Ln(3)
}

func renderForecastSummary(pdf *fpdf.Fpdf, st *state.State, br, bg, bb int) {
	pdf.AddPage()
	h1(pdf, "Forecast Summary", br, bg, bb)
	recs := sortedByForecast(st.Recommendations)
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(contentWidth, 5, asciiSafe(fmt.Sprintf(
		"Top recommendations ranked by forecasted monthly click delta. Grand total across all %d recommendations: +%s.",
		len(st.Recommendations), formatInt(totalForecast(st)))),
		"", "L", false)
	pdf.Ln(2)

	drawTableHeader(pdf, []string{"#", "Change", "Target URL", "Pri", "Clicks/mo"},
		[]float64{10, 40, 90, 15, 25})
	limit := len(recs)
	if limit > 20 {
		limit = 20
	}
	for i := 0; i < limit; i++ {
		r := recs[i]
		delta := 0
		if r.ForecastedLift != nil {
			delta = r.ForecastedLift.EstimatedMonthlyClicksDelta
		}
		drawTableRow(pdf, []string{
			fmt.Sprintf("%d", i+1),
			string(r.ChangeType),
			truncate(r.TargetURL, 55),
			fmt.Sprintf("%d", r.Priority),
			fmt.Sprintf("+%s", formatInt(delta)),
		}, []float64{10, 40, 90, 15, 25})
	}

	pdf.Ln(2)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(contentWidth-25, 6, "Grand total (all recommendations)", "T", 0, "R", false, 0, "")
	pdf.CellFormat(25, 6, fmt.Sprintf("+%s", formatInt(totalForecast(st))), "T", 1, "R", false, 0, "")
}

func renderAppendix(pdf *fpdf.Fpdf, st *state.State, br, bg, bb int) {
	pdf.AddPage()
	h1(pdf, "Appendix A: Top GSC queries", br, bg, bb)
	if st.GSC == nil || len(st.GSC.TopKeywords) == 0 {
		pdf.SetFont("Helvetica", "I", 10)
		pdf.MultiCell(contentWidth, 5, "No GSC data.", "", "L", false)
	} else {
		drawTableHeader(pdf, []string{"Query", "Impr", "Clicks", "CTR", "Pos"},
			[]float64{90, 25, 25, 20, 20})
		n := len(st.GSC.TopKeywords)
		if n > 50 {
			n = 50
		}
		for i := 0; i < n; i++ {
			r := st.GSC.TopKeywords[i]
			drawTableRow(pdf, []string{
				truncate(r.Key, 55),
				formatInt(int(r.Impressions)),
				formatInt(int(r.Clicks)),
				fmt.Sprintf("%.1f%%", r.CTR*100),
				fmt.Sprintf("%.1f", r.Position),
			}, []float64{90, 25, 25, 20, 20})
		}
	}

	pdf.AddPage()
	h1(pdf, "Appendix B: Crawl error list", br, bg, bb)
	errs := 0
	for _, f := range st.Findings {
		if f.Verdict == "fail" || f.Verdict == "error" {
			errs++
		}
	}
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(contentWidth, 5, asciiSafe(fmt.Sprintf("%d failing findings out of %d total.", errs, len(st.Findings))), "", "L", false)
	pdf.Ln(2)
	drawTableHeader(pdf, []string{"Rule", "URL", "Verdict"}, []float64{50, 95, 35})
	for _, f := range st.Findings {
		if f.Verdict == "fail" || f.Verdict == "error" {
			drawTableRow(pdf, []string{f.Rule, truncate(f.URL, 60), f.Verdict}, []float64{50, 95, 35})
		}
	}

	pdf.AddPage()
	h1(pdf, "Appendix C: Full audit findings", br, bg, bb)
	drawTableHeader(pdf, []string{"Rule", "URL", "Verdict", "Why"}, []float64{35, 70, 20, 55})
	for _, f := range st.Findings {
		drawTableRow(pdf, []string{f.Rule, truncate(f.URL, 40), f.Verdict, truncate(f.Why, 35)},
			[]float64{35, 70, 20, 55})
	}
}

// ---------------------------------------------------------------------------
// Primitive helpers
// ---------------------------------------------------------------------------

func h1(pdf *fpdf.Fpdf, text string, br, bg, bb int) {
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetTextColor(br, bg, bb)
	pdf.CellFormat(contentWidth, 9, asciiSafe(text), "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(1)
}

func h2(pdf *fpdf.Fpdf, text string) {
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(contentWidth, 6, asciiSafe(text), "", 1, "L", false, 0, "")
}

// drawPriorityBadge draws a colour-coded badge at the current Y and leaves X
// positioned to the right of the badge. The badge also includes the numeric
// priority so colour is never the sole signal.
func drawPriorityBadge(pdf *fpdf.Fpdf, priority int) {
	r, g, b := priorityColor(priority)
	y := pdf.GetY()
	pdf.SetFillColor(r, g, b)
	pdf.Rect(marginLeft, y, 12, 6, "F")
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetXY(marginLeft, y+1)
	pdf.CellFormat(12, 4, fmt.Sprintf("P%d", priority), "", 0, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	// Move caret back to the same line for follow-up content.
	pdf.SetXY(marginLeft+14, y)
}

func priorityColor(p int) (int, int, int) {
	switch {
	case p >= 80:
		return 185, 28, 28 // red
	case p >= 60:
		return 202, 138, 4 // amber
	default:
		return 107, 114, 128 // grey
	}
}

func drawTableHeader(pdf *fpdf.Fpdf, cols []string, widths []float64) {
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(240, 240, 240)
	pdf.SetTextColor(60, 60, 60)
	for i, c := range cols {
		pdf.CellFormat(widths[i], 6, asciiSafe(c), "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(0, 0, 0)
}

func drawTableRow(pdf *fpdf.Fpdf, cols []string, widths []float64) {
	pdf.SetFont("Helvetica", "", 9)
	for i, c := range cols {
		pdf.CellFormat(widths[i], 5.5, asciiSafe(c), "1", 0, "L", false, 0, "")
	}
	pdf.Ln(-1)
}

// ---------------------------------------------------------------------------
// Data helpers
// ---------------------------------------------------------------------------

func sortedRecs(in []state.Recommendation) []state.Recommendation {
	out := make([]state.Recommendation, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Priority > out[j].Priority
	})
	return out
}

func sortedByForecast(in []state.Recommendation) []state.Recommendation {
	out := make([]state.Recommendation, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		a := 0
		b := 0
		if out[i].ForecastedLift != nil {
			a = out[i].ForecastedLift.EstimatedMonthlyClicksDelta
		}
		if out[j].ForecastedLift != nil {
			b = out[j].ForecastedLift.EstimatedMonthlyClicksDelta
		}
		return a > b
	})
	return out
}

func totalForecast(st *state.State) int {
	total := 0
	for _, r := range st.Recommendations {
		if r.ForecastedLift != nil {
			total += r.ForecastedLift.EstimatedMonthlyClicksDelta
		}
	}
	return total
}

// aeoBrandCoverage returns the percentage of prompts where any brand term was
// mentioned in any engine's response. Returns -1 when no AEO data exists or
// brand terms are absent.
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

func aeoResponseCount(st *state.State) int {
	if st.AEO == nil {
		return 0
	}
	n := 0
	for _, p := range st.AEO.Responses {
		n += len(p.Results)
	}
	return n
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

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func safeSite(s string) string {
	if s == "" {
		return "(no site)"
	}
	return s
}

func formatInt(n int) string {
	return formatInt64(int64(n))
}

func formatInt64(n int64) string {
	if n < 0 {
		return "-" + formatInt64(-n)
	}
	s := fmt.Sprintf("%d", n)
	// Insert thousands separators.
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

func parseHex(hex, fallback string) (int, int, int) {
	h := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(h) != 6 {
		h = strings.TrimPrefix(fallback, "#")
	}
	var r, g, b int
	if _, err := fmt.Sscanf(h, "%02x%02x%02x", &r, &g, &b); err != nil {
		return parseHex(fallback, defaultBrandHex)
	}
	return r, g, b
}

// asciiSafe strips non-ASCII characters that Helvetica's built-in WinAnsi
// encoding cannot render. gofpdf would otherwise emit replacement chars or
// fail silently — this keeps output deterministic without requiring a TTF.
func asciiSafe(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 128 {
			b.WriteRune(r)
			continue
		}
		switch r {
		case '—', '–':
			b.WriteString("-")
		case '“', '”':
			b.WriteRune('"')
		case '‘', '’':
			b.WriteRune('\'')
		case '…':
			b.WriteString("...")
		case '•':
			b.WriteString("- ")
		case '→':
			b.WriteString("->")
		default:
			b.WriteRune('?')
		}
	}
	return b.String()
}
