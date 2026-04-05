package audit

import (
	"context"
	"strings"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/crawl"
)

func TestAuditPerfectPage(t *testing.T) {
	svc := NewService()
	result, err := svc.Run(context.Background(), Request{
		CrawlResult: crawl.Result{
			TargetURL: "https://example.com",
			Pages: []crawl.PageResult{
				{
					URL:             "https://example.com",
					StatusCode:      200,
					Title:           "Example",
					MetaDescription: "A great example site",
					Canonical:       "https://example.com",
					Headings:        []crawl.Heading{{Level: 1, Text: "Welcome"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Score != 100 {
		t.Errorf("expected score 100 for perfect page, got %.1f", result.Score)
	}
	if len(result.Issues) != 0 {
		t.Errorf("expected no issues, got %d: %+v", len(result.Issues), result.Issues)
	}
}

func TestAuditMissingTitle(t *testing.T) {
	svc := NewService()
	result, err := svc.Run(context.Background(), Request{
		CrawlResult: crawl.Result{
			Pages: []crawl.PageResult{
				{URL: "https://example.com", StatusCode: 200, Headings: []crawl.Heading{{Level: 1, Text: "H"}}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "title-missing" {
			found = true
		}
	}
	if !found {
		t.Error("expected title-missing issue")
	}
}

func TestAuditTitleTooLong(t *testing.T) {
	svc := NewService()
	longTitle := strings.Repeat("x", 61)
	result, err := svc.Run(context.Background(), Request{
		CrawlResult: crawl.Result{
			Pages: []crawl.PageResult{
				{URL: "https://example.com", StatusCode: 200, Title: longTitle, Headings: []crawl.Heading{{Level: 1, Text: "H"}}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "title-too-long" {
			found = true
		}
	}
	if !found {
		t.Error("expected title-too-long issue")
	}
}

func TestAuditMissingMetaDescription(t *testing.T) {
	svc := NewService()
	result, err := svc.Run(context.Background(), Request{
		CrawlResult: crawl.Result{
			Pages: []crawl.PageResult{
				{URL: "https://example.com", StatusCode: 200, Title: "T", Headings: []crawl.Heading{{Level: 1, Text: "H"}}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "meta-description-missing" {
			found = true
		}
	}
	if !found {
		t.Error("expected meta-description-missing issue")
	}
}

func TestAuditMissingH1(t *testing.T) {
	svc := NewService()
	result, err := svc.Run(context.Background(), Request{
		CrawlResult: crawl.Result{
			Pages: []crawl.PageResult{
				{URL: "https://example.com", StatusCode: 200, Title: "T", MetaDescription: "D"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "h1-missing" {
			found = true
		}
	}
	if !found {
		t.Error("expected h1-missing issue")
	}
}

func TestAuditMultipleH1(t *testing.T) {
	svc := NewService()
	result, err := svc.Run(context.Background(), Request{
		CrawlResult: crawl.Result{
			Pages: []crawl.PageResult{
				{
					URL: "https://example.com", StatusCode: 200, Title: "T",
					Headings: []crawl.Heading{{Level: 1, Text: "A"}, {Level: 1, Text: "B"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "h1-multiple" {
			found = true
		}
	}
	if !found {
		t.Error("expected h1-multiple issue")
	}
}

func TestAuditImageMissingAlt(t *testing.T) {
	svc := NewService()
	result, err := svc.Run(context.Background(), Request{
		CrawlResult: crawl.Result{
			Pages: []crawl.PageResult{
				{
					URL: "https://example.com", StatusCode: 200, Title: "T",
					Headings: []crawl.Heading{{Level: 1, Text: "H"}},
					Images:   []crawl.Image{{Src: "/img.png", Alt: ""}, {Src: "/ok.png", Alt: "OK"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	count := 0
	for _, issue := range result.Issues {
		if issue.Rule == "img-alt-missing" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 img-alt-missing issue, got %d", count)
	}
}

func TestAuditBrokenPage(t *testing.T) {
	svc := NewService()
	result, err := svc.Run(context.Background(), Request{
		CrawlResult: crawl.Result{
			Pages: []crawl.PageResult{
				{URL: "https://example.com/broken", StatusCode: 500},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "broken-page" && issue.Severity == SeverityError {
			found = true
		}
	}
	if !found {
		t.Error("expected broken-page error issue for 500 status")
	}
}

func TestAuditCanonicalMissing(t *testing.T) {
	svc := NewService()
	result, err := svc.Run(context.Background(), Request{
		CrawlResult: crawl.Result{
			Pages: []crawl.PageResult{
				{URL: "https://example.com", StatusCode: 200, Title: "T", Headings: []crawl.Heading{{Level: 1, Text: "H"}}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Rule == "canonical-missing" {
			found = true
		}
	}
	if !found {
		t.Error("expected canonical-missing issue")
	}
}

func TestAuditScoreDegradesWithIssues(t *testing.T) {
	svc := NewService()
	// Page with all issues: no title, no desc, no h1, no canonical, images without alt, 500 status
	result, err := svc.Run(context.Background(), Request{
		CrawlResult: crawl.Result{
			Pages: []crawl.PageResult{
				{
					URL:        "https://example.com",
					StatusCode: 500,
					Images:     []crawl.Image{{Src: "/a.png"}, {Src: "/b.png"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Score >= 100 {
		t.Errorf("expected score below 100 for page with many issues, got %.1f", result.Score)
	}
	if len(result.Issues) == 0 {
		t.Error("expected issues for broken page with missing elements")
	}
}
