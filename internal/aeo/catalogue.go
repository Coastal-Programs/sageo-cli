// Package aeo exposes the DataForSEO AI Optimization model catalogue and
// helpers for consuming it from CLI commands.
package aeo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/dataforseo"
)

// CacheTTL is how long the on-disk model catalogue is considered fresh.
const CacheTTL = 7 * 24 * time.Hour

// CacheFilename is the JSON file name used for the persisted catalogue.
const CacheFilename = "aeo_models.json"

// SupportedEngines enumerates the engines exposed by DataForSEO's AI
// optimization API. The order mirrors the rest of the CLI surface.
var SupportedEngines = []string{"chatgpt", "claude", "gemini", "perplexity"}

// Model describes a single (engine, model_name) entry from the DataForSEO
// /v3/ai_optimization/<engine>/llm_responses/models endpoint.
type Model struct {
	Engine      string    `json:"engine"`
	ModelName   string    `json:"model_name"`
	DisplayName string    `json:"display_name"`
	IsDefault   bool      `json:"is_default"`
	FetchedAt   time.Time `json:"fetched_at"`
}

// EngineToPath maps the command-line engine alias (e.g. "chatgpt") to the
// DataForSEO URL path segment (e.g. "chat_gpt").
func EngineToPath(engine string) string {
	switch engine {
	case "chatgpt":
		return "chat_gpt"
	case "claude":
		return "claude"
	case "gemini":
		return "gemini"
	case "perplexity":
		return "perplexity"
	default:
		return ""
	}
}

// modelsEndpoint returns the full endpoint path for an engine's models call.
func modelsEndpoint(engine string) (string, error) {
	seg := EngineToPath(engine)
	if seg == "" {
		return "", fmt.Errorf("unsupported engine %q: valid values: chatgpt, claude, gemini, perplexity", engine)
	}
	return "/v3/ai_optimization/" + seg + "/llm_responses/models", nil
}

// modelsEnvelope is the decoded shape of an llm_responses/models response.
type modelsEnvelope struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
	Tasks         []struct {
		StatusCode    int    `json:"status_code"`
		StatusMessage string `json:"status_message"`
		Result        []struct {
			Items []struct {
				ModelName        string `json:"model_name"`
				ModelDisplayName string `json:"model_display_name"`
				IsDefault        bool   `json:"is_default"`
			} `json:"items"`
		} `json:"result"`
	} `json:"tasks"`
}

// FetchModels calls the engine-specific llm_responses/models endpoint and
// returns the parsed catalogue.
func FetchModels(client *dataforseo.Client, engine string) ([]Model, error) {
	endpoint, err := modelsEndpoint(engine)
	if err != nil {
		return nil, err
	}

	raw, err := client.Post(endpoint, []map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("fetching %s models: %w", engine, err)
	}

	var envelope modelsEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode %s models response: %w", engine, err)
	}
	if envelope.StatusCode != 20000 {
		return nil, fmt.Errorf("dataforseo %s models error %d: %s", engine, envelope.StatusCode, envelope.StatusMessage)
	}
	if len(envelope.Tasks) == 0 {
		return nil, fmt.Errorf("dataforseo %s models: no tasks returned", engine)
	}
	task := envelope.Tasks[0]
	if task.StatusCode != 20000 {
		return nil, fmt.Errorf("dataforseo %s models task error %d: %s", engine, task.StatusCode, task.StatusMessage)
	}

	now := time.Now().UTC()
	models := make([]Model, 0)
	for _, result := range task.Result {
		for _, item := range result.Items {
			models = append(models, Model{
				Engine:      engine,
				ModelName:   item.ModelName,
				DisplayName: item.ModelDisplayName,
				IsDefault:   item.IsDefault,
				FetchedAt:   now,
			})
		}
	}
	return models, nil
}

// FetchAllModels fans out to all supported engines in parallel. Per-engine
// errors are returned as an aggregated error but successful engines are still
// included in the returned map.
func FetchAllModels(client *dataforseo.Client) (map[string][]Model, error) {
	type result struct {
		engine string
		models []Model
		err    error
	}

	results := make([]result, len(SupportedEngines))
	var wg sync.WaitGroup
	for i, engine := range SupportedEngines {
		wg.Add(1)
		go func(i int, engine string) {
			defer wg.Done()
			m, err := FetchModels(client, engine)
			results[i] = result{engine: engine, models: m, err: err}
		}(i, engine)
	}
	wg.Wait()

	out := make(map[string][]Model, len(SupportedEngines))
	var errs []string
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", r.engine, r.err))
			continue
		}
		out[r.engine] = r.models
	}
	if len(errs) > 0 {
		return out, fmt.Errorf("failed to fetch models for %d engine(s): %v", len(errs), errs)
	}
	return out, nil
}

// cacheFile is the on-disk JSON shape used by LoadCached/SaveCached.
type cacheFile struct {
	FetchedAt time.Time          `json:"fetched_at"`
	Models    map[string][]Model `json:"models"`
	Meta      map[string]any     `json:"meta,omitempty"`
}

// cachePath returns the full path to the catalogue JSON file inside cacheDir.
func cachePath(cacheDir string) string {
	return filepath.Join(cacheDir, CacheFilename)
}

// LoadCached reads the persisted catalogue. An empty map (and nil error) is
// returned when the cache file is missing or expired.
func LoadCached(cacheDir string) (map[string][]Model, error) {
	path := cachePath(cacheDir)
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string][]Model{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading aeo models cache: %w", err)
	}

	var cf cacheFile
	if err := json.Unmarshal(body, &cf); err != nil {
		return nil, fmt.Errorf("decoding aeo models cache: %w", err)
	}

	if cf.FetchedAt.IsZero() || time.Since(cf.FetchedAt) > CacheTTL {
		return map[string][]Model{}, nil
	}
	if cf.Models == nil {
		return map[string][]Model{}, nil
	}
	return cf.Models, nil
}

// SaveCached writes the catalogue to <cacheDir>/aeo_models.json.
func SaveCached(cacheDir string, m map[string][]Model) error {
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return fmt.Errorf("creating aeo cache dir: %w", err)
	}
	cf := cacheFile{
		FetchedAt: time.Now().UTC(),
		Models:    m,
	}
	body, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding aeo models cache: %w", err)
	}
	if err := os.WriteFile(cachePath(cacheDir), body, 0o600); err != nil {
		return fmt.Errorf("writing aeo models cache: %w", err)
	}
	return nil
}

// DefaultModelName returns the model_name marked is_default for an engine, or
// the first catalogue entry if no default is flagged. An error is returned if
// the engine has no catalogue entries at all.
func DefaultModelName(catalogue map[string][]Model, engine string) (string, error) {
	models := catalogue[engine]
	if len(models) == 0 {
		return "", fmt.Errorf("no cached models for engine %q", engine)
	}
	for _, m := range models {
		if m.IsDefault && m.ModelName != "" {
			return m.ModelName, nil
		}
	}
	if models[0].ModelName == "" {
		return "", fmt.Errorf("cached catalogue for %q has no usable model_name", engine)
	}
	return models[0].ModelName, nil
}
