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

// State is the single project file the AI reads and writes.
type State struct {
	Site           string          `json:"site"`
	Initialized    string          `json:"initialized"`
	LastCrawl      string          `json:"last_crawl,omitempty"`
	Score          float64         `json:"score,omitempty"`
	PagesCrawled   int             `json:"pages_crawled,omitempty"`
	Findings       []Finding       `json:"findings,omitempty"`
	MergedFindings json.RawMessage `json:"merged_findings,omitempty"`
	LastAnalysis   string          `json:"last_analysis,omitempty"`
	GSC            *GSCData        `json:"gsc,omitempty"`
	PSI            *PSIData        `json:"psi,omitempty"`
	History        []HistoryEntry  `json:"history,omitempty"`
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

	return used, missing
}

// AddHistory appends an entry to the history log.
func (s *State) AddHistory(action, detail string) {
	s.History = append(s.History, HistoryEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    action,
		Detail:    detail,
	})
}
