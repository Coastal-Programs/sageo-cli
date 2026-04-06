package audit

import (
	"fmt"

	"github.com/jakeschepis/sageo-cli/internal/crawl"
)

const (
	maxTitleLength       = 60
	maxDescriptionLength = 160
	maxResponseTimeMs    = 2000
	minWordCount         = 300
)

func checkTitle(page crawl.PageResult) []Issue {
	var issues []Issue
	if page.Title == "" {
		issues = append(issues, Issue{
			Rule:     "title-missing",
			Severity: SeverityError,
			URL:      page.URL,
			Message:  "Page is missing a title tag",
			Why:      "The title tag is the most important on-page SEO element — Google uses it as the search result headline",
			Fix:      "Add a unique, descriptive <title> under 60 characters",
		})
	} else if len(page.Title) > maxTitleLength {
		issues = append(issues, Issue{
			Rule:     "title-too-long",
			Severity: SeverityWarning,
			URL:      page.URL,
			Message:  fmt.Sprintf("Title exceeds %d characters (%d)", maxTitleLength, len(page.Title)),
			Detail:   page.Title,
			Why:      "Google truncates titles over ~60 characters in search results",
			Fix:      "Shorten to under 60 characters, front-load important keywords",
		})
	}
	return issues
}

func checkMetaDescription(page crawl.PageResult) []Issue {
	var issues []Issue
	if page.MetaDescription == "" {
		issues = append(issues, Issue{
			Rule:     "meta-description-missing",
			Severity: SeverityWarning,
			URL:      page.URL,
			Message:  "Page is missing a meta description",
			Why:      "Google often uses the meta description as the search result snippet",
			Fix:      "Add a compelling meta description under 160 characters",
		})
	} else if len(page.MetaDescription) > maxDescriptionLength {
		issues = append(issues, Issue{
			Rule:     "meta-description-too-long",
			Severity: SeverityWarning,
			URL:      page.URL,
			Message:  fmt.Sprintf("Meta description exceeds %d characters (%d)", maxDescriptionLength, len(page.MetaDescription)),
			Detail:   page.MetaDescription,
			Why:      "Google truncates descriptions over ~160 characters — the message gets cut off",
			Fix:      "Shorten to under 160 characters, include a clear call to action",
		})
	}
	return issues
}

func checkH1(page crawl.PageResult) []Issue {
	var issues []Issue
	h1Count := 0
	for _, h := range page.Headings {
		if h.Level == 1 {
			h1Count++
		}
	}
	if h1Count == 0 {
		issues = append(issues, Issue{
			Rule:     "h1-missing",
			Severity: SeverityError,
			URL:      page.URL,
			Message:  "Page is missing an H1 heading",
			Why:      "The H1 tells Google what the page is about — missing it weakens topical relevance",
			Fix:      "Add a single H1 tag with the primary keyword for the page",
		})
	} else if h1Count > 1 {
		issues = append(issues, Issue{
			Rule:     "h1-multiple",
			Severity: SeverityWarning,
			URL:      page.URL,
			Message:  fmt.Sprintf("Page has %d H1 headings (should have exactly 1)", h1Count),
			Why:      "Multiple H1s dilute the page focus and confuse search engines",
			Fix:      "Keep one H1 and convert others to H2 or H3",
		})
	}
	return issues
}

func checkImageAlt(page crawl.PageResult) []Issue {
	var issues []Issue
	for _, img := range page.Images {
		if img.Alt == "" {
			issues = append(issues, Issue{
				Rule:     "img-alt-missing",
				Severity: SeverityWarning,
				URL:      page.URL,
				Message:  "Image is missing alt text",
				Detail:   img.Src,
				Why:      "Alt text is required for accessibility and helps Google understand images",
				Fix:      "Add descriptive alt text that describes the image content",
			})
		}
	}
	return issues
}

func checkCanonical(page crawl.PageResult) []Issue {
	var issues []Issue
	if page.Canonical == "" {
		issues = append(issues, Issue{
			Rule:     "canonical-missing",
			Severity: SeverityInfo,
			URL:      page.URL,
			Message:  "Page is missing a canonical tag",
			Why:      "Without a canonical, Google may treat URL variants as duplicate content",
			Fix:      "Add <link rel=\"canonical\" href=\"...\"> pointing to the preferred URL",
		})
	}
	return issues
}

func checkStatusCode(page crawl.PageResult) []Issue {
	var issues []Issue
	if page.StatusCode >= 400 {
		severity := SeverityWarning
		if page.StatusCode >= 500 {
			severity = SeverityError
		}
		issues = append(issues, Issue{
			Rule:     "broken-page",
			Severity: severity,
			URL:      page.URL,
			Message:  fmt.Sprintf("Page returned HTTP %d", page.StatusCode),
		})
	}
	return issues
}

func checkViewport(page crawl.PageResult) []Issue {
	if page.Viewport == "" {
		return []Issue{{
			Rule:     "viewport-missing",
			Severity: SeverityError,
			URL:      page.URL,
			Message:  "Page is missing a viewport meta tag",
			Why:      "Without a viewport tag, mobile devices render desktop layout — Google penalises this",
			Fix:      "Add <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">",
		}}
	}
	return nil
}

func checkOpenGraph(page crawl.PageResult) []Issue {
	var issues []Issue
	if page.OGTitle == "" {
		issues = append(issues, Issue{
			Rule:     "og-title-missing",
			Severity: SeverityWarning,
			URL:      page.URL,
			Message:  "Page is missing og:title tag",
			Why:      "Social shares will use a generic or wrong title without og:title",
			Fix:      "Add <meta property=\"og:title\" content=\"Your Page Title\">",
		})
	}
	if page.OGDescription == "" {
		issues = append(issues, Issue{
			Rule:     "og-description-missing",
			Severity: SeverityWarning,
			URL:      page.URL,
			Message:  "Page is missing og:description tag",
			Why:      "Social shares won't show a description preview",
			Fix:      "Add <meta property=\"og:description\" content=\"Short description\">",
		})
	}
	if page.OGImage == "" {
		issues = append(issues, Issue{
			Rule:     "og-image-missing",
			Severity: SeverityWarning,
			URL:      page.URL,
			Message:  "Page is missing og:image tag",
			Why:      "Social shares appear without a preview image — lower click rates",
			Fix:      "Add <meta property=\"og:image\" content=\"https://...\">",
		})
	}
	return issues
}

func checkResponseTime(page crawl.PageResult) []Issue {
	if page.ResponseTime > maxResponseTimeMs {
		return []Issue{{
			Rule:     "slow-response",
			Severity: SeverityWarning,
			URL:      page.URL,
			Message:  fmt.Sprintf("Response time %dms exceeds %dms target", page.ResponseTime, maxResponseTimeMs),
			Why:      "Slow pages rank lower and lose visitors to bounce",
			Fix:      "Check server response time, enable caching, compress assets",
		}}
	}
	return nil
}

func checkWordCount(page crawl.PageResult) []Issue {
	if page.WordCount < minWordCount && page.WordCount > 0 {
		return []Issue{{
			Rule:     "thin-content",
			Severity: SeverityWarning,
			URL:      page.URL,
			Message:  fmt.Sprintf("Page has only %d words (target: %d+)", page.WordCount, minWordCount),
			Why:      "Thin content pages struggle to rank — Google prefers comprehensive pages",
			Fix:      "Add more relevant content, answer related questions, expand on the topic",
		}}
	}
	return nil
}

func checkSchema(page crawl.PageResult) []Issue {
	if len(page.SchemaTypes) == 0 {
		return []Issue{{
			Rule:     "schema-missing",
			Severity: SeverityInfo,
			URL:      page.URL,
			Message:  "Page has no structured data (JSON-LD)",
			Why:      "Structured data enables rich results in Google — stars, FAQs, breadcrumbs",
			Fix:      "Add JSON-LD structured data relevant to the page type",
		}}
	}
	return nil
}

func checkMetaRobots(page crawl.PageResult) []Issue {
	if page.MetaRobots != "" {
		lower := page.MetaRobots
		if contains(lower, "noindex") {
			return []Issue{{
				Rule:     "noindex-detected",
				Severity: SeverityError,
				URL:      page.URL,
				Message:  "Page has noindex directive — Google will not index this page",
				Why:      "This page will not appear in search results",
				Fix:      "Remove noindex if this page should be searchable",
			}}
		}
	}
	if page.XRobotsTag != "" && contains(page.XRobotsTag, "noindex") {
		return []Issue{{
			Rule:     "x-robots-noindex",
			Severity: SeverityError,
			URL:      page.URL,
			Message:  "X-Robots-Tag header contains noindex",
			Why:      "This page will not appear in search results",
			Fix:      "Remove the X-Robots-Tag noindex header on the server",
		}}
	}
	return nil
}

func checkLang(page crawl.PageResult) []Issue {
	if page.Lang == "" {
		return []Issue{{
			Rule:     "lang-missing",
			Severity: SeverityWarning,
			URL:      page.URL,
			Message:  "HTML lang attribute is missing",
			Why:      "Helps search engines understand the page language for correct regional results",
			Fix:      "Add lang attribute to <html> tag, e.g. <html lang=\"en\">",
		}}
	}
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
