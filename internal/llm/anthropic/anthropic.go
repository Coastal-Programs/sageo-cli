// Package anthropic implements an llm.Provider backed by the Anthropic
// Messages API.
//
// API reference: https://docs.anthropic.com/en/api/messages
//
// Request shape (POST https://api.anthropic.com/v1/messages):
//
//	{
//	  "model": "claude-sonnet-4-6",
//	  "max_tokens": 1024,
//	  "system": "...",
//	  "messages": [{"role": "user", "content": "..."}]
//	}
//
// Required headers:
//
//	x-api-key: <key>
//	anthropic-version: 2023-06-01
//	content-type: application/json
//
// Response shape (truncated to fields we consume):
//
//	{
//	  "model": "claude-sonnet-4-6",
//	  "content": [{"type": "text", "text": "..."}],
//	  "usage": {"input_tokens": 123, "output_tokens": 45}
//	}
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/llm"
)

// DefaultModel is the current flagship Sonnet snapshot.
// Verify latest at https://docs.claude.com/en/docs/about-claude/models/overview
const DefaultModel = "claude-sonnet-4-6"

// Pricing per 1 million tokens (USD).
// Source: https://www.anthropic.com/pricing
// Sonnet 4.6 maintains Sonnet 4.5 pricing ($3 input / $15 output per MTok).
const (
	inputPricePerMTok  = 3.00
	outputPricePerMTok = 15.00
)

// DefaultBaseURL is the Anthropic API endpoint root.
const DefaultBaseURL = "https://api.anthropic.com"

// APIVersion is the value sent in the anthropic-version header.
const APIVersion = "2023-06-01"

// maxResponseSize caps bytes read from a Messages API response.
const maxResponseSize = 10 * 1024 * 1024 // 10 MB

// HTTPClient is satisfied by *http.Client and test stubs.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is the Anthropic driver.
type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient HTTPClient
}

// Option configures the client.
type Option func(*Client)

// WithBaseURL overrides the API endpoint (used by tests).
func WithBaseURL(url string) Option { return func(c *Client) { c.baseURL = url } }

// WithHTTPClient overrides the HTTP client (used by tests).
func WithHTTPClient(hc HTTPClient) Option { return func(c *Client) { c.httpClient = hc } }

// WithModel overrides the default model id.
func WithModel(model string) Option { return func(c *Client) { c.model = model } }

func init() {
	llm.Register("anthropic", func(cfg *config.Config) (llm.Provider, error) {
		return New(cfg)
	})
}

// New builds a client with credentials loaded from cfg.
func New(cfg *config.Config, opts ...Option) (*Client, error) {
	if cfg == nil || cfg.AnthropicAPIKey == "" {
		return nil, fmt.Errorf("anthropic: api key not configured (set SAGEO_ANTHROPIC_API_KEY)")
	}
	c := &Client{
		apiKey:     cfg.AnthropicAPIKey,
		model:      DefaultModel,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// Name returns the provider identifier.
func (c *Client) Name() string { return "anthropic" }

type messageContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesRequest struct {
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens"`
	System      string           `json:"system,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	Messages    []messageContent `json:"messages"`
}

type messagesResponseContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type messagesResponseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type messagesResponse struct {
	Model   string                    `json:"model"`
	Content []messagesResponseContent `json:"content"`
	Usage   messagesResponseUsage     `json:"usage"`
	Error   *apiError                 `json:"error,omitempty"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Complete sends req to the Messages API and returns the first text block.
func (c *Client) Complete(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	body := messagesRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    req.System,
		Messages: []messageContent{
			{Role: "user", Content: req.User},
		},
	}
	if req.Temperature > 0 {
		t := req.Temperature
		body.Temperature = &t
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return llm.CompletionResponse{}, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(buf))
	if err != nil {
		return llm.CompletionResponse{}, fmt.Errorf("anthropic: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", APIVersion)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return llm.CompletionResponse{}, fmt.Errorf("anthropic: http call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return llm.CompletionResponse{}, fmt.Errorf("anthropic: read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var envelope messagesResponse
		_ = json.Unmarshal(raw, &envelope)
		if envelope.Error != nil {
			return llm.CompletionResponse{}, fmt.Errorf("anthropic: %s (%d): %s", envelope.Error.Type, resp.StatusCode, envelope.Error.Message)
		}
		return llm.CompletionResponse{}, fmt.Errorf("anthropic: http %d: %s", resp.StatusCode, string(raw))
	}

	var parsed messagesResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return llm.CompletionResponse{}, fmt.Errorf("anthropic: parse response: %w", err)
	}

	var text string
	for _, block := range parsed.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}

	return llm.CompletionResponse{
		Text:         text,
		InputTokens:  parsed.Usage.InputTokens,
		OutputTokens: parsed.Usage.OutputTokens,
		CostUSD:      Cost(parsed.Usage.InputTokens, parsed.Usage.OutputTokens),
		Model:        parsed.Model,
	}, nil
}

// Cost computes USD cost from token counts using the published per-token
// pricing constants.
func Cost(inputTokens, outputTokens int) float64 {
	return (float64(inputTokens)/1_000_000.0)*inputPricePerMTok +
		(float64(outputTokens)/1_000_000.0)*outputPricePerMTok
}
