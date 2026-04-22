package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	DirName  = ".sageo"
	FileName = "state.json"
)

// GSCRow is a single row of GSC search analytics data.
type GSCRow struct {
	Key         string  `json:"key"`
	Clicks      float64 `json:"clicks"`
	Impressions float64 `json:"impressions"`
	CTR         float64 `json:"ctr"`
	Position    float64 `json:"position"`
}

// GSCData holds the most recent GSC pull saved to state.
type GSCData struct {
	LastPull    string   `json:"last_pull,omitempty"`
	Property    string   `json:"property,omitempty"`
	TopPages    []GSCRow `json:"top_pages,omitempty"`
	TopKeywords []GSCRow `json:"top_keywords,omitempty"`
}

// Finding is a single interpreted audit result.
type Finding struct {
	Rule    string      `json:"rule"`
	URL     string      `json:"url"`
	Value   interface{} `json:"value"`
	Verdict string      `json:"verdict"`
	Why     string      `json:"why"`
	Fix     string      `json:"fix"`
}

// HistoryEntry records an action taken by the agent or user.
type HistoryEntry struct {
	Timestamp string `json:"ts"`
	Action    string `json:"action"`
	Detail    string `json:"detail,omitempty"`
}

// PSIResult holds a single PageSpeed Insights result for one URL.
type PSIResult struct {
	URL              string  `json:"url"`
	PerformanceScore float64 `json:"performance_score"`
	LCP              float64 `json:"lcp_ms"`
	CLS              float64 `json:"cls"`
	Strategy         string  `json:"strategy"`
}

// PSIData holds all PageSpeed Insights results saved to state.
type PSIData struct {
	LastRun string      `json:"last_run,omitempty"`
	Pages   []PSIResult `json:"pages,omitempty"`
}

// SERPFeatureRecord is a SERP feature stored in state.
type SERPFeatureRecord struct {
	Type     string `json:"type"`
	Position int    `json:"position,omitempty"`
	Title    string `json:"title,omitempty"`
	URL      string `json:"url,omitempty"`
	Domain   string `json:"domain,omitempty"`
}

// SERPQueryResult stores SERP data for a single query.
type SERPQueryResult struct {
	Query            string              `json:"query"`
	HasAIOverview    bool                `json:"has_ai_overview"`
	Features         []SERPFeatureRecord `json:"features,omitempty"`
	RelatedQuestions []string            `json:"related_questions,omitempty"`
	TopDomains       []string            `json:"top_domains,omitempty"`
	OurPosition      int                 `json:"our_position"` // -1 if not found in results, 0+ for actual position
}

// SERPData holds all SERP results saved to state.
type SERPData struct {
	LastRun string            `json:"last_run,omitempty"`
	Queries []SERPQueryResult `json:"queries,omitempty"`
}

// LabsKeyword stores keyword intelligence from DataForSEO Labs.
type LabsKeyword struct {
	Keyword      string  `json:"keyword"`
	SearchVolume int     `json:"search_volume"`
	Difficulty   float64 `json:"difficulty"` // 0-100
	CPC          float64 `json:"cpc,omitempty"`
	Intent       string  `json:"intent,omitempty"`   // informational, navigational, commercial, transactional
	Position     int     `json:"position,omitempty"` // current ranking position if from ranked-keywords
}

// LabsData holds Labs intelligence saved to state.
type LabsData struct {
	LastRun     string        `json:"last_run,omitempty"`
	Target      string        `json:"target,omitempty"` // domain that was analyzed
	Keywords    []LabsKeyword `json:"keywords,omitempty"`
	Competitors []string      `json:"competitors,omitempty"` // top competitor domains
}

// BacklinksData holds backlink profile data saved to state.
type BacklinksData struct {
	LastRun               string   `json:"last_run,omitempty"`
	Target                string   `json:"target,omitempty"`
	TotalBacklinks        int64    `json:"total_backlinks"`
	TotalReferringDomains int64    `json:"total_referring_domains"`
	BrokenBacklinks       int64    `json:"broken_backlinks"`
	Rank                  int      `json:"rank"`
	DoFollow              int64    `json:"dofollow"`
	NoFollow              int64    `json:"nofollow"`
	SpamScore             float64  `json:"spam_score"`
	TopReferrers          []string `json:"top_referrers,omitempty"` // top 10 referring domain names
	GapDomains            []string `json:"gap_domains,omitempty"`   // top 20 gap domains from backlinks gap analysis
}

// AEOResponseResult is a single engine's response for a prompt.
type AEOResponseResult struct {
	Engine    string    `json:"engine"`
	ModelName string    `json:"model_name"`
	Response  string    `json:"response"`
	FetchedAt time.Time `json:"fetched_at"`
}

// AEOPromptResult stores all engine responses captured for a single prompt.
type AEOPromptResult struct {
	Prompt    string              `json:"prompt"`
	Results   []AEOResponseResult `json:"results"`
	FetchedAt string              `json:"fetched_at"`
}

// AEOData holds AEO LLM response history saved to state.
type AEOData struct {
	LastRun   string            `json:"last_run,omitempty"`
	Responses []AEOPromptResult `json:"responses,omitempty"`
}

// LocalMentionMatch mirrors internal/aeo/mentions.Match for on-disk storage.
// Defined here to avoid import cycles (mentions imports state for
// AEOResponseResult).
type LocalMentionMatch struct {
	Engine    string   `json:"engine"`
	ModelName string   `json:"model_name"`
	Prompt    string   `json:"prompt"`
	Term      string   `json:"term"`
	Count     int      `json:"count"`
	Contexts  []string `json:"contexts,omitempty"`
}

// TopPageEntry is a page that AI engines cite for a given term.
type TopPageEntry struct {
	URL      string  `json:"url"`
	Domain   string  `json:"domain,omitempty"`
	Mentions int     `json:"mentions"`
	Share    float64 `json:"share,omitempty"`
}

// MentionsData holds brand mention data, both locally detected (Layer A) and
// sourced from the DataForSEO LLM Mentions API (Layer B), keyed by term.
type MentionsData struct {
	Term         string              `json:"term"`
	LocalMatches []LocalMentionMatch `json:"local_matches,omitempty"`
	DomainShare  map[string]float64  `json:"domain_share,omitempty"`
	TopPages     []TopPageEntry      `json:"top_pages,omitempty"`
	FetchedAt    time.Time           `json:"fetched_at"`
}

// State is the single project file the AI reads and writes.
type State struct {
	Site            string           `json:"site"`
	Initialized     string           `json:"initialized"`
	LastCrawl       string           `json:"last_crawl,omitempty"`
	Score           float64          `json:"score,omitempty"`
	PagesCrawled    int              `json:"pages_crawled,omitempty"`
	Findings        []Finding        `json:"findings,omitempty"`
	MergedFindings  json.RawMessage  `json:"merged_findings,omitempty"`
	LastAnalysis    string           `json:"last_analysis,omitempty"`
	GSC             *GSCData         `json:"gsc,omitempty"`
	PSI             *PSIData         `json:"psi,omitempty"`
	SERP            *SERPData        `json:"serp,omitempty"`
	Labs            *LabsData        `json:"labs,omitempty"`
	Backlinks       *BacklinksData   `json:"backlinks,omitempty"`
	AEO             *AEOData         `json:"aeo,omitempty"`
	BrandTerms      []string         `json:"brand_terms,omitempty"`
	Mentions        []MentionsData   `json:"mentions,omitempty"`
	Recommendations []Recommendation `json:"recommendations,omitempty"`
	History         []HistoryEntry   `json:"history,omitempty"`

	// PipelineCursor records the last stage that completed successfully in
	// an autonomous `sageo run` invocation. `--resume` picks up after this.
	PipelineCursor string `json:"pipeline_cursor,omitempty"`
	// PipelineRuns is an audit log of autonomous pipeline invocations.
	PipelineRuns []PipelineRun `json:"pipeline_runs,omitempty"`
}

// PipelineRun records a single autonomous pipeline execution.
type PipelineRun struct {
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
	StagesRun    []string  `json:"stages_run,omitempty"`
	TotalCostUSD float64   `json:"total_cost_usd"`
	Outcome      string    `json:"outcome"` // success | partial | failed
	FailedStage  string    `json:"failed_stage,omitempty"`
	Error        string    `json:"error,omitempty"`
}

// UpsertRecommendations inserts or replaces recommendations by ID.
// If an incoming recommendation has no CreatedAt, it is set to now.
func (s *State) UpsertRecommendations(recs []Recommendation) {
	now := time.Now().UTC()
	for _, r := range recs {
		if r.CreatedAt.IsZero() {
			r.CreatedAt = now
		}
		replaced := false
		for i, existing := range s.Recommendations {
			if existing.ID == r.ID {
				// Preserve the original CreatedAt on update.
				if !existing.CreatedAt.IsZero() {
					r.CreatedAt = existing.CreatedAt
				}
				s.Recommendations[i] = r
				replaced = true
				break
			}
		}
		if !replaced {
			s.Recommendations = append(s.Recommendations, r)
		}
	}
}

// Path returns the state.json path relative to a working directory.
func Path(dir string) string {
	return filepath.Join(dir, DirName, FileName)
}

// Init creates a new .sageo/state.json for a site.
func Init(dir string, siteURL string) (*State, error) {
	sageoDir := filepath.Join(dir, DirName)
	if err := os.MkdirAll(sageoDir, 0755); err != nil {
		return nil, fmt.Errorf("create .sageo dir: %w", err)
	}

	path := filepath.Join(sageoDir, FileName)
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("state.json already exists — run sageo status to view")
	}

	s := &State{
		Site:        siteURL,
		Initialized: time.Now().UTC().Format(time.RFC3339),
		Findings:    []Finding{},
		History:     []HistoryEntry{},
	}

	if err := s.Save(dir); err != nil {
		return nil, err
	}
	return s, nil
}

// Load reads state.json from disk.
func Load(dir string) (*State, error) {
	data, err := os.ReadFile(Path(dir))
	if err != nil {
		return nil, fmt.Errorf("read state.json: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse state.json: %w", err)
	}
	return &s, nil
}

// Save writes state.json to disk.
func (s *State) Save(dir string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return os.WriteFile(Path(dir), data, 0644)
}

// Exists returns true if .sageo/state.json exists in the given directory.
func Exists(dir string) bool {
	_, err := os.Stat(Path(dir))
	return err == nil
}

// UpdateAudit replaces findings and score from an audit run.
func (s *State) UpdateAudit(score float64, pagesCrawled int, findings []Finding) {
	s.LastCrawl = time.Now().UTC().Format(time.RFC3339)
	s.Score = score
	s.PagesCrawled = pagesCrawled
	s.Findings = findings
}

// Sources returns which data sources have contributed to this state.
// used contains sources with data present; missing contains sources not yet populated.
func (s *State) Sources() (used []string, missing []string) {
	if s.LastCrawl != "" {
		used = append(used, "crawl")
	}

	if s.GSC != nil && s.GSC.LastPull != "" {
		used = append(used, "gsc")
	} else {
		missing = append(missing, "gsc")
	}

	if s.PSI != nil && s.PSI.LastRun != "" {
		used = append(used, "psi")
	} else {
		missing = append(missing, "psi")
	}

	if s.SERP != nil && s.SERP.LastRun != "" {
		used = append(used, "serp")
	} else {
		missing = append(missing, "serp")
	}

	if s.Labs != nil && s.Labs.LastRun != "" {
		used = append(used, "labs")
	} else {
		missing = append(missing, "labs")
	}

	if s.Backlinks != nil && s.Backlinks.LastRun != "" {
		used = append(used, "backlinks")
	} else {
		missing = append(missing, "backlinks")
	}

	return used, missing
}

// UpsertPSI replaces the PSI data in state.
func (s *State) UpsertPSI(data PSIData) {
	data.LastRun = time.Now().UTC().Format(time.RFC3339)
	s.PSI = &data
}

// UpsertAEOResponses stores AEO LLM results for a prompt, replacing any
// prior entry for the same prompt (upsert by prompt text).
func (s *State) UpsertAEOResponses(prompt string, results []AEOResponseResult) {
	if s.AEO == nil {
		s.AEO = &AEOData{}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	s.AEO.LastRun = now
	entry := AEOPromptResult{
		Prompt:    prompt,
		Results:   results,
		FetchedAt: now,
	}
	for i, existing := range s.AEO.Responses {
		if existing.Prompt == prompt {
			s.AEO.Responses[i] = entry
			return
		}
	}
	s.AEO.Responses = append(s.AEO.Responses, entry)
}

// UpsertMentions inserts or replaces a MentionsData entry by term.
func (s *State) UpsertMentions(data MentionsData) {
	if data.FetchedAt.IsZero() {
		data.FetchedAt = time.Now().UTC()
	}
	for i, existing := range s.Mentions {
		if existing.Term == data.Term {
			s.Mentions[i] = data
			return
		}
	}
	s.Mentions = append(s.Mentions, data)
}

const maxHistoryEntries = 200

// AddHistory appends an entry to the history log, keeping at most the last
// maxHistoryEntries entries to prevent unbounded state growth.
func (s *State) AddHistory(action, detail string) {
	s.History = append(s.History, HistoryEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    action,
		Detail:    detail,
	})
	if len(s.History) > maxHistoryEntries {
		s.History = s.History[len(s.History)-maxHistoryEntries:]
	}
}
