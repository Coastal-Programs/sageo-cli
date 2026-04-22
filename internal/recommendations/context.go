package recommendations

import "strings"

// ContextForURL builds a best-effort PageContext for the given target URL
// from whatever data is present in s. Crawl page bodies are not persisted
// in state, so Title / MetaDescription / H1 / H2s / BodyExcerpt are left
// empty unless callers populate them from a fresh crawl. SERP and GSC
// data are used to fill competitor titles, PAA, and target keywords.
func ContextForURL(s *State, targetURL, targetQuery string) PageContext {
	if s == nil {
		return PageContext{}
	}
	ctx := PageContext{}

	if s.SERP != nil && targetQuery != "" {
		for _, q := range s.SERP.Queries {
			if !strings.EqualFold(q.Query, targetQuery) {
				continue
			}
			ctx.TopPAAQuestions = append(ctx.TopPAAQuestions, q.RelatedQuestions...)
			for _, f := range q.Features {
				if f.Title != "" {
					ctx.TopCompetitorTitles = append(ctx.TopCompetitorTitles, f.Title)
				}
			}
			break
		}
	}

	if s.GSC != nil {
		for _, row := range s.GSC.TopKeywords {
			if row.Key == "" {
				continue
			}
			ctx.TargetKeywords = append(ctx.TargetKeywords, row.Key)
			if len(ctx.TargetKeywords) >= 10 {
				break
			}
		}
	}

	return ctx
}
