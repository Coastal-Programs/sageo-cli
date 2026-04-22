// Package openai implements an llm.Provider backed by the OpenAI Chat
// Completions API.
//
// API reference: https://platform.openai.com/docs/api-reference/chat/create
//
// Request shape (POST https://api.openai.com/v1/chat/completions):
//
//	{
//	  "model": "gpt-5",
//	  "messages": [
//	    {"role": "system", "content": "..."},
//	    {"role": "user",   "content": "..."}
//	  ],
//	  "max_completion_tokens": 1024,
//	  "temperature": 0.7
//	}
//
// Required headers:
//
//	Authorization: Bearer <key>
//	Content-Type: application/json
//
// Response shape (truncated to fields we consume):
//
//	{
//	  "model": "gpt-5",
//	  "choices": [{"message": {"role": "assistant", "content": "..."}}],
//	  "usage": {"prompt_tokens": 123, "completion_tokens": 45, "total_tokens": 168}
//	}
package openai

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

// DefaultModel is the current flagship chat completions model.
// Verify latest at https://platform.openai.com/docs/models
const DefaultModel = "gpt-5"

// Pricing per 1 million tokens (USD).
// Source: https://openai.com/api/pricing/ (GPT-5 standard tier).
const (
	inputPricePerMTok  = 1.25
	outputPricePerMTok = 10.00
)

// DefaultBaseURL is the OpenAI API endpoint root.
const DefaultBaseURL = "https://api.openai.com"

const maxResponseSize = 10 * 1024 * 1024 // 10 MB

// HTTPClient is satisfied by *http.Client and test stubs.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is the OpenAI driver.
type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient HTTPClient
}

// Option configures the client.
type Option func(*Client)

func WithBaseURL(url string) Option       { return func(c *Client) { c.baseURL = url } }
func WithHTTPClient(hc HTTPClient) Option { return func(c *Client) { c.httpClient = hc } }
func WithModel(model string) Option       { return func(c *Client) { c.model = model } }

func init() {
	llm.Register("openai", func(cfg *config.Config) (llm.Provider, error) {
		return New(cfg)
	})
}

// New builds a client with credentials loaded from cfg.
func New(cfg *config.Config, opts ...Option) (*Client, error) {
	if cfg == nil || cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("openai: api key not configured (set SAGEO_OPENAI_API_KEY)")
	}
	c := &Client{
		apiKey:     cfg.OpenAIAPIKey,
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
func (c *Client) Name() string { return "openai" }

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type       string          `json:"type"`
	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
}

type chatRequest struct {
	Model               string          `json:"model"`
	Messages            []chatMessage   `json:"messages"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
	Temperature         *float64        `json:"temperature,omitempty"`
	ResponseFormat      *responseFormat `json:"response_format,omitempty"`
}

type chatResponseChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponseUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type chatResponse struct {
	Model   string               `json:"model"`
	Choices []chatResponseChoice `json:"choices"`
	Usage   chatResponseUsage    `json:"usage"`
	Error   *apiError            `json:"error,omitempty"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Complete sends req to the Chat Completions API and returns the first
// choice's message content.
func (c *Client) Complete(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}

	messages := make([]chatMessage, 0, 2)
	if req.System != "" {
		messages = append(messages, chatMessage{Role: "system", Content: req.System})
	}
	messages = append(messages, chatMessage{Role: "user", Content: req.User})

	body := chatRequest{
		Model:               c.model,
		Messages:            messages,
		MaxCompletionTokens: maxTokens,
	}
	if req.Temperature > 0 {
		t := req.Temperature
		body.Temperature = &t
	}
	if req.JSONSchema != nil && *req.JSONSchema != "" {
		body.ResponseFormat = &responseFormat{
			Type:       "json_schema",
			JSONSchema: json.RawMessage(*req.JSONSchema),
		}
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return llm.CompletionResponse{}, fmt.Errorf("openai: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(buf))
	if err != nil {
		return llm.CompletionResponse{}, fmt.Errorf("openai: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return llm.CompletionResponse{}, fmt.Errorf("openai: http call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return llm.CompletionResponse{}, fmt.Errorf("openai: read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var envelope chatResponse
		_ = json.Unmarshal(raw, &envelope)
		if envelope.Error != nil {
			return llm.CompletionResponse{}, fmt.Errorf("openai: %s (%d): %s", envelope.Error.Type, resp.StatusCode, envelope.Error.Message)
		}
		return llm.CompletionResponse{}, fmt.Errorf("openai: http %d: %s", resp.StatusCode, string(raw))
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return llm.CompletionResponse{}, fmt.Errorf("openai: parse response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return llm.CompletionResponse{}, fmt.Errorf("openai: empty choices in response")
	}

	return llm.CompletionResponse{
		Text:         parsed.Choices[0].Message.Content,
		InputTokens:  parsed.Usage.PromptTokens,
		OutputTokens: parsed.Usage.CompletionTokens,
		CostUSD:      Cost(parsed.Usage.PromptTokens, parsed.Usage.CompletionTokens),
		Model:        parsed.Model,
	}, nil
}

// Cost computes USD cost from token counts using the published per-token
// pricing constants.
func Cost(inputTokens, outputTokens int) float64 {
	return (float64(inputTokens)/1_000_000.0)*inputPricePerMTok +
		(float64(outputTokens)/1_000_000.0)*outputPricePerMTok
}
