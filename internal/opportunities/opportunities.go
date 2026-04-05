package opportunities

import (
	"github.com/jakeschepis/sageo-cli/internal/gsc"
	"github.com/jakeschepis/sageo-cli/internal/serp"
)

// Type classifies an opportunity.
type Type string

const (
	TypePage    Type = "page"
	TypeKeyword Type = "keyword"
	TypeAnswer  Type = "answer"
)

// Opportunity represents an actionable SEO improvement signal.
type Opportunity struct {
	Type           Type     `json:"type"`
	Target         string   `json:"target"`
	Evidence       []string `json:"evidence"`
	Confidence     float64  `json:"confidence"`
	ImpactEstimate string   `json:"impact_estimate"`
	EffortEstimate string   `json:"effort_estimate"`
	Sources        []string `json:"sources"`
	EstimatedCost  float64  `json:"estimated_cost"`
}

// MergeInput holds all data sources for opportunity detection.
type MergeInput struct {
	GSCSeeds    []gsc.OpportunitySeed
	SERPResults map[string]*serp.AnalyzeResponse
}

// Merge combines GSC seed data with optional SERP evidence into ranked opportunities.
func Merge(input MergeInput) []Opportunity {
	var opps []Opportunity

	for _, seed := range input.GSCSeeds {
		opp := fromGSCSeed(seed)

		// Enrich with SERP data if available
		if serpResp, ok := input.SERPResults[seed.Query]; ok && serpResp != nil {
			enrichWithSERP(&opp, seed, serpResp)
		}

		opps = append(opps, opp)
	}

	return opps
}

func fromGSCSeed(seed gsc.OpportunitySeed) Opportunity {
	opp := Opportunity{
		Type:    TypeKeyword,
		Target:  seed.Query,
		Sources: []string{"gsc"},
	}

	// Position-based opportunity classification
	if seed.Position <= 10 && seed.CTR < 0.03 {
		opp.Evidence = append(opp.Evidence, "low CTR despite first-page ranking")
		opp.ImpactEstimate = "high"
		opp.EffortEstimate = "low"
		opp.Confidence = 0.8
	} else if seed.Position > 10 && seed.Position <= 20 {
		opp.Evidence = append(opp.Evidence, "ranking on page 2, close to first page")
		opp.ImpactEstimate = "medium"
		opp.EffortEstimate = "medium"
		opp.Confidence = 0.6
	} else if seed.Impressions > 100 && seed.Position > 20 {
		opp.Evidence = append(opp.Evidence, "high impressions with poor ranking")
		opp.ImpactEstimate = "medium"
		opp.EffortEstimate = "high"
		opp.Confidence = 0.5
	} else {
		opp.Evidence = append(opp.Evidence, "underperforming query")
		opp.ImpactEstimate = "low"
		opp.EffortEstimate = "medium"
		opp.Confidence = 0.4
	}

	// Page-level opportunity if specific page is involved
	if seed.Page != "" {
		opp.Type = TypePage
		opp.Target = seed.Page
		opp.Evidence = append(opp.Evidence, "query: "+seed.Query)
	}

	return opp
}

func enrichWithSERP(opp *Opportunity, seed gsc.OpportunitySeed, serpResp *serp.AnalyzeResponse) {
	opp.Sources = append(opp.Sources, "serpapi")
	opp.EstimatedCost = 0.01 // one SERP query used for validation

	// Check if the seed page appears in current SERP results
	pageFound := false
	for _, result := range serpResp.OrganicResults {
		if result.Link == seed.Page {
			pageFound = true
			if result.Position < int(seed.Position) {
				opp.Evidence = append(opp.Evidence, "SERP position improved since GSC data")
				opp.Confidence += 0.1
			} else if result.Position > int(seed.Position) {
				opp.Evidence = append(opp.Evidence, "SERP position declined since GSC data")
				opp.Confidence += 0.05
			}
			break
		}
	}

	if !pageFound && len(serpResp.OrganicResults) > 0 {
		opp.Evidence = append(opp.Evidence, "page not found in current SERP top results")
	}

	// Check for featured snippets / answer boxes (simple heuristic)
	for _, result := range serpResp.OrganicResults {
		if result.Position == 0 {
			opp.Type = TypeAnswer
			opp.Evidence = append(opp.Evidence, "answer box detected for this query")
			break
		}
	}

	// Cap confidence at 1.0
	if opp.Confidence > 1.0 {
		opp.Confidence = 1.0
	}
}
