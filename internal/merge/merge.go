// Package merge compares crawl findings with GSC data and produces
// cross-source findings that only exist when both data sets are present.
package merge

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jakeschepis/sageo-cli/internal/common/urlnorm"
	"github.com/jakeschepis/sageo-cli/internal/state"
)

// MergedFinding is a finding that requires data from multiple sources.
type MergedFinding struct {
	Rule          string      `json:"rule"`
	URL           string      `json:"url"`
	Sources       []string    `json:"sources"`
	CrawlIssues   []string    `json:"crawl_issues,omitempty"`
	GSCData       *GSCMetrics `json:"gsc_data,omitempty"`
	Verdict       string      `json:"verdict"`
	Why           string      `json:"why"`
	Fix           string      `json:"fix"`
	Priority      string      `json:"priority"`       // "high", "medium", "low"
	PriorityScore int         `json:"priority_score"` // numeric for sorting, higher = more urgent
}

// GSCMetrics holds search performance numbers from Google Search Console.
type GSCMetrics struct {
	Impressions float64 `json:"impressions"`
	Clicks      float64 `json:"clicks"`
	CTR         float64 `json:"ctr"`
	Position    float64 `json:"position"`
}

// Run compares crawl findings with GSC data in the given state and returns
// cross-source findings. Both crawl and GSC data must be present; if either
// is missing the function returns nil.
func Run(st *state.State) []MergedFinding {
	if st.LastCrawl == "" || st.GSC == nil || st.GSC.LastPull == "" {
		return nil
	}

	// Index PSI results by URL (strategy-agnostic: use the worst-scoring entry
	// per URL so we surface the most severe performance problem).
	psiByURL := make(map[string]*state.PSIResult)
	if st.PSI != nil {
		for i := range st.PSI.Pages {
			p := &st.PSI.Pages[i]
			norm := urlnorm.Normalize(p.URL)
			if norm == "" {
				continue
			}
			if existing, ok := psiByURL[norm]; !ok || p.PerformanceScore < existing.PerformanceScore {
				psiByURL[norm] = p
			}
		}
	}

	// Index GSC top-pages by normalized URL.
	gscByURL := make(map[string]*GSCMetrics, len(st.GSC.TopPages))
	for _, row := range st.GSC.TopPages {
		norm := urlnorm.Normalize(row.Key)
		if norm == "" {
			continue
		}
		gscByURL[norm] = &GSCMetrics{
			Impressions: row.Impressions,
			Clicks:      row.Clicks,
			CTR:         row.CTR,
			Position:    row.Position,
		}
	}

	// Index crawl findings by normalized URL.
	crawlIssuesByURL := make(map[string][]string)
	crawlURLs := make(map[string]bool) // tracks every URL with a finding
	noindexURLs := make(map[string]bool)
	schemaURLs := make(map[string][]string) // URL → schema types
	thinURLs := make(map[string]bool)       // URLs with word_count < 300

	for _, f := range st.Findings {
		norm := urlnorm.Normalize(f.URL)
		if norm == "" {
			continue
		}
		crawlURLs[norm] = true
		crawlIssuesByURL[norm] = append(crawlIssuesByURL[norm], f.Rule)

		if f.Rule == "noindex-detected" || f.Rule == "x-robots-noindex" {
			noindexURLs[norm] = true
		}
	}

	// Scan findings for thin-content and schema markers. These are stored
	// as crawl findings with specific rule names.
	for _, f := range st.Findings {
		norm := urlnorm.Normalize(f.URL)
		if norm == "" {
			continue
		}
		switch {
		case f.Rule == "thin-content" || f.Rule == "low-word-count":
			thinURLs[norm] = true
		case strings.HasPrefix(f.Rule, "schema-"):
			if v, ok := f.Value.(string); ok && v != "" {
				schemaURLs[norm] = append(schemaURLs[norm], v)
			}
		}
	}

	var results []MergedFinding

	// --- Rule 1: ranking-but-not-clicking ---
	for normURL, gsc := range gscByURL {
		issues, hasCrawl := crawlIssuesByURL[normURL]
		if gsc.Impressions > 10 && gsc.Clicks == 0 && hasCrawl && len(issues) > 0 {
			results = append(results, MergedFinding{
				Rule:        "ranking-but-not-clicking",
				URL:         normURL,
				Sources:     []string{"crawl", "gsc"},
				CrawlIssues: issues,
				GSCData:     gsc,
				Verdict:     "high",
				Why:         fmt.Sprintf("Page has %d impressions but 0 clicks and %d crawl issue(s)", int(gsc.Impressions), len(issues)),
				Fix:         "Improve title and meta description to increase CTR, then fix crawl issues on this page",
			})
		}
	}

	// --- Rule 2: not-indexed ---
	for normURL := range crawlURLs {
		if noindexURLs[normURL] {
			continue // page intentionally noindexed
		}
		if _, inGSC := gscByURL[normURL]; !inGSC {
			issues := crawlIssuesByURL[normURL]
			results = append(results, MergedFinding{
				Rule:        "not-indexed",
				URL:         normURL,
				Sources:     []string{"crawl", "gsc"},
				CrawlIssues: issues,
				Verdict:     "medium",
				Why:         "Page found in crawl with no noindex directive but has zero GSC impressions — may not be indexed",
				Fix:         "Submit URL to Google via Search Console, check for crawl blocks, and ensure internal links point to this page",
			})
		}
	}

	// --- Rule 3: issues-on-high-traffic-page ---
	for normURL, gsc := range gscByURL {
		issues, hasCrawl := crawlIssuesByURL[normURL]
		if hasCrawl && len(issues) > 0 && gsc.Clicks > 0 {
			results = append(results, MergedFinding{
				Rule:        "issues-on-high-traffic-page",
				URL:         normURL,
				Sources:     []string{"crawl", "gsc"},
				CrawlIssues: issues,
				GSCData:     gsc,
				Verdict:     "high",
				Why:         fmt.Sprintf("Page gets %.0f clicks but has %d crawl issue(s) — fixing these protects existing traffic", gsc.Clicks, len(issues)),
				Fix:         "Prioritize fixing crawl issues on this page since it already receives organic traffic",
			})
		}
	}

	// --- Rule 4: thin-content-ranking-well ---
	for normURL := range thinURLs {
		gsc, inGSC := gscByURL[normURL]
		if inGSC && gsc.Position > 0 && gsc.Position < 10 {
			results = append(results, MergedFinding{
				Rule:        "thin-content-ranking-well",
				URL:         normURL,
				Sources:     []string{"crawl", "gsc"},
				CrawlIssues: crawlIssuesByURL[normURL],
				GSCData:     gsc,
				Verdict:     "medium",
				Why:         fmt.Sprintf("Page has fewer than 300 words but ranks at position %.1f — content expansion can defend this ranking", gsc.Position),
				Fix:         "Add more useful content, answer related questions, and expand the page to at least 600 words",
			})
		}
	}

	// --- Rule 5: schema-not-showing ---
	for normURL, schemas := range schemaURLs {
		gsc, inGSC := gscByURL[normURL]
		if !inGSC {
			continue
		}
		// If GSC shows impressions but no rich-result clicks/impressions,
		// and the page has schema, the schema may not be working. Since the
		// state does not carry GSC search-appearance data, flag any page
		// with schema types and GSC data where CTR is below 5% as potentially
		// having broken schema.
		if gsc.Impressions > 0 && gsc.CTR < 0.05 {
			results = append(results, MergedFinding{
				Rule:        "schema-not-showing",
				URL:         normURL,
				Sources:     []string{"crawl", "gsc"},
				CrawlIssues: crawlIssuesByURL[normURL],
				GSCData:     gsc,
				Verdict:     "low",
				Why:         fmt.Sprintf("Page has schema types [%s] but CTR is only %.1f%% — rich results may not be appearing", strings.Join(schemas, ", "), gsc.CTR*100),
				Fix:         "Validate structured data with Google's Rich Results Test and fix any errors",
			})
		}
	}

	// --- Rule 6: slow-core-web-vitals ---
	// Only fire when PSI data exists in state.
	if len(psiByURL) > 0 {
		for normURL, psiResult := range psiByURL {
			if psiResult.PerformanceScore >= 50 {
				continue
			}
			gsc, inGSC := gscByURL[normURL]
			if !inGSC || gsc.Impressions == 0 {
				continue
			}

			// Identify the worst metric to name in the fix message.
			slowMetric := slowestMetric(psiResult)

			results = append(results, MergedFinding{
				Rule:    "slow-core-web-vitals",
				URL:     normURL,
				Sources: []string{"psi", "gsc"},
				GSCData: gsc,
				Verdict: "high",
				Why: fmt.Sprintf(
					"Performance score is %.0f/100 — this page is losing ranking potential (%.0f GSC impressions)",
					psiResult.PerformanceScore, gsc.Impressions,
				),
				Fix: fmt.Sprintf(
					"Fix %s first: %s is the primary bottleneck. Use Lighthouse or PageSpeed Insights for a full audit.",
					slowMetric.name, slowMetric.detail,
				),
			})
		}
	}

	// Assign priority scores to all findings.
	for i := range results {
		results[i].PriorityScore, results[i].Priority = scoreFinding(&results[i])
	}

	// Sort by PriorityScore descending (most urgent first).
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].PriorityScore > results[j].PriorityScore
	})

	return results
}

// metricInfo holds a human-readable name and detail string for a CWV metric.
type metricInfo struct {
	name   string
	detail string
}

// slowestMetric identifies the worst Core Web Vital in a PSI result and returns
// a human-readable description for use in a fix recommendation.
func slowestMetric(p *state.PSIResult) metricInfo {
	// LCP thresholds: >4000 ms is poor; CLS >0.25 is poor.
	// We rank LCP first (most impactful for ranking) unless CLS is severely bad.
	lcpPoor := p.LCP > 4000
	clsPoor := p.CLS > 0.25

	switch {
	case lcpPoor && clsPoor:
		return metricInfo{
			name:   "LCP and CLS",
			detail: fmt.Sprintf("LCP is %.0f ms (target <2500 ms) and CLS is %.2f (target <0.10)", p.LCP, p.CLS),
		}
	case lcpPoor:
		return metricInfo{
			name:   "LCP",
			detail: fmt.Sprintf("Largest Contentful Paint is %.0f ms (target <2500 ms)", p.LCP),
		}
	case clsPoor:
		return metricInfo{
			name:   "CLS",
			detail: fmt.Sprintf("Cumulative Layout Shift is %.2f (target <0.10)", p.CLS),
		}
	default:
		// Score is poor but no single metric crossed the "poor" threshold —
		// report overall score and recommend a full Lighthouse run.
		return metricInfo{
			name:   "overall performance",
			detail: fmt.Sprintf("performance score is %.0f/100 — run Lighthouse for a detailed breakdown", p.PerformanceScore),
		}
	}
}

// scoreFinding computes a numeric priority score and label for a merged finding.
func scoreFinding(f *MergedFinding) (int, string) {
	hasCrawlIssues := len(f.CrawlIssues) > 0
	hasGSC := f.GSCData != nil

	switch {
	// Crawl issues + GSC clicks > 0 → HIGH (90-100).
	case hasCrawlIssues && hasGSC && f.GSCData.Clicks > 0:
		score := 90
		if f.GSCData.Clicks >= 10 {
			score = 100
		} else if f.GSCData.Clicks >= 5 {
			score = 95
		}
		return score, "high"

	// Crawl issues + impressions > 20 + clicks == 0 → HIGH (80-89).
	case hasCrawlIssues && hasGSC && f.GSCData.Impressions > 20 && f.GSCData.Clicks == 0:
		score := 80
		if f.GSCData.Impressions > 100 {
			score = 89
		} else if f.GSCData.Impressions > 50 {
			score = 85
		}
		return score, "high"

	// Crawl issues + impressions > 0 + position < 20 → MEDIUM (50-79).
	case hasCrawlIssues && hasGSC && f.GSCData.Impressions > 0 && f.GSCData.Position > 0 && f.GSCData.Position < 20:
		score := 50
		if f.GSCData.Position < 5 {
			score = 79
		} else if f.GSCData.Position < 10 {
			score = 65
		}
		return score, "medium"

	// Crawl issues + no GSC data → LOW (10-49).
	case hasCrawlIssues && !hasGSC:
		score := 10 + len(f.CrawlIssues)*5
		if score > 49 {
			score = 49
		}
		return score, "low"

	// No crawl issues + low CTR → MEDIUM (40-60).
	case !hasCrawlIssues && hasGSC && f.GSCData.Impressions > 0 && f.GSCData.CTR < 0.05:
		score := 40
		if f.GSCData.Impressions > 100 {
			score = 60
		} else if f.GSCData.Impressions > 50 {
			score = 50
		}
		return score, "medium"
	}

	// Default: crawl issues with some GSC data that doesn't match higher tiers.
	if hasCrawlIssues {
		return 30, "low"
	}
	return 20, "low"
}
