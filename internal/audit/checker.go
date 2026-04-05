package audit

import (
	"fmt"

	"github.com/jakeschepis/sageo-cli/internal/crawl"
)

const (
	maxTitleLength       = 60
	maxDescriptionLength = 160
)

func checkTitle(page crawl.PageResult) []Issue {
	var issues []Issue
	if page.Title == "" {
		issues = append(issues, Issue{
			Rule:     "title-missing",
			Severity: SeverityError,
			URL:      page.URL,
			Message:  "Page is missing a title tag",
		})
	} else if len(page.Title) > maxTitleLength {
		issues = append(issues, Issue{
			Rule:     "title-too-long",
			Severity: SeverityWarning,
			URL:      page.URL,
			Message:  fmt.Sprintf("Title exceeds %d characters (%d)", maxTitleLength, len(page.Title)),
			Detail:   page.Title,
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
		})
	} else if len(page.MetaDescription) > maxDescriptionLength {
		issues = append(issues, Issue{
			Rule:     "meta-description-too-long",
			Severity: SeverityWarning,
			URL:      page.URL,
			Message:  fmt.Sprintf("Meta description exceeds %d characters (%d)", maxDescriptionLength, len(page.MetaDescription)),
			Detail:   page.MetaDescription,
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
		})
	} else if h1Count > 1 {
		issues = append(issues, Issue{
			Rule:     "h1-multiple",
			Severity: SeverityWarning,
			URL:      page.URL,
			Message:  fmt.Sprintf("Page has %d H1 headings (should have exactly 1)", h1Count),
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
