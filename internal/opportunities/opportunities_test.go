package opportunities

import (
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/gsc"
	"github.com/jakeschepis/sageo-cli/internal/serp"
)

func TestMergeGSCOnly(t *testing.T) {
	input := MergeInput{
		GSCSeeds: []gsc.OpportunitySeed{
			{
				Query:       "seo tool",
				Page:        "https://example.com/seo",
				Clicks:      10,
				Impressions: 500,
				CTR:         0.02,
				Position:    8,
			},
		},
	}

	opps := Merge(input)
	if len(opps) != 1 {
		t.Fatalf("expected 1 opportunity, got %d", len(opps))
	}

	opp := opps[0]
	if opp.Type != TypePage {
		t.Fatalf("expected type 'page', got %q", opp.Type)
	}
	if opp.Target != "https://example.com/seo" {
		t.Fatalf("expected target 'https://example.com/seo', got %q", opp.Target)
	}
	if len(opp.Sources) != 1 || opp.Sources[0] != "gsc" {
		t.Fatalf("expected sources [gsc], got %v", opp.Sources)
	}
	if opp.Confidence == 0 {
		t.Fatal("expected non-zero confidence")
	}
}

func TestMergeWithSERP(t *testing.T) {
	input := MergeInput{
		GSCSeeds: []gsc.OpportunitySeed{
			{
				Query:       "seo tool",
				Page:        "https://example.com/seo",
				Clicks:      10,
				Impressions: 500,
				CTR:         0.02,
				Position:    8,
			},
		},
		SERPResults: map[string]*serp.AnalyzeResponse{
			"seo tool": {
				Query: "seo tool",
				OrganicResults: []serp.OrganicResult{
					{Position: 1, Title: "Top Result", Link: "https://competitor.com", Domain: "competitor.com"},
					{Position: 5, Title: "Our Page", Link: "https://example.com/seo", Domain: "example.com"},
				},
			},
		},
	}

	opps := Merge(input)
	if len(opps) != 1 {
		t.Fatalf("expected 1 opportunity, got %d", len(opps))
	}

	opp := opps[0]
	if len(opp.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %v", opp.Sources)
	}

	hasSERP := false
	for _, s := range opp.Sources {
		if s == "serpapi" {
			hasSERP = true
		}
	}
	if !hasSERP {
		t.Fatal("expected serpapi in sources")
	}
}

func TestMergeEmpty(t *testing.T) {
	opps := Merge(MergeInput{})
	if opps != nil {
		t.Fatalf("expected nil for empty input, got %v", opps)
	}
}

func TestMergeLowCTRFirstPage(t *testing.T) {
	input := MergeInput{
		GSCSeeds: []gsc.OpportunitySeed{
			{
				Query:       "best tool",
				Page:        "https://example.com/best",
				Clicks:      5,
				Impressions: 1000,
				CTR:         0.005,
				Position:    5,
			},
		},
	}

	opps := Merge(input)
	if len(opps) != 1 {
		t.Fatalf("expected 1 opportunity, got %d", len(opps))
	}

	if opps[0].ImpactEstimate != "high" {
		t.Fatalf("expected high impact for low CTR first-page, got %q", opps[0].ImpactEstimate)
	}
	if opps[0].EffortEstimate != "low" {
		t.Fatalf("expected low effort for first-page CTR fix, got %q", opps[0].EffortEstimate)
	}
}

func TestMergePage2Ranking(t *testing.T) {
	input := MergeInput{
		GSCSeeds: []gsc.OpportunitySeed{
			{
				Query:       "near miss",
				Page:        "https://example.com/near",
				Clicks:      2,
				Impressions: 100,
				CTR:         0.02,
				Position:    15,
			},
		},
	}

	opps := Merge(input)
	if len(opps) != 1 {
		t.Fatalf("expected 1 opportunity, got %d", len(opps))
	}
	if opps[0].ImpactEstimate != "medium" {
		t.Fatalf("expected medium impact for page-2, got %q", opps[0].ImpactEstimate)
	}
}
