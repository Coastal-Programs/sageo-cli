// Package llm defines a provider abstraction for large language model
// completions. Drivers (anthropic, openai) implement Provider and are
// selected via the registry.
package llm

import "context"

// Provider is the minimal interface implemented by every LLM driver.
type Provider interface {
	// Name returns a short identifier such as "anthropic" or "openai".
	Name() string
	// Complete issues a single completion request and returns the text
	// response plus token/cost accounting.
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

// CompletionRequest is a provider-agnostic completion request.
type CompletionRequest struct {
	System      string
	User        string
	MaxTokens   int
	Temperature float64
	// JSONSchema is an optional JSON Schema (as a string) that callers can
	// supply when they want the driver to request structured output. It is
	// advisory — drivers that do not support schemas will ignore it and
	// embed the schema hint in the user prompt instead.
	JSONSchema *string
}

// CompletionResponse is a provider-agnostic completion response.
type CompletionResponse struct {
	Text         string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	Model        string
}
