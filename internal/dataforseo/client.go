package dataforseo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.dataforseo.com"

// HTTPClient is an interface for HTTP operations (supports testing).
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is a shared DataForSEO API client using HTTP Basic Auth.
type Client struct {
	login      string
	password   string
	baseURL    string
	httpClient HTTPClient
}

// Option configures the DataForSEO client.
type Option func(*Client)

// WithBaseURL overrides the default DataForSEO base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithHTTPClient overrides the default HTTP client (useful for testing).
func WithHTTPClient(hc HTTPClient) Option {
	return func(c *Client) { c.httpClient = hc }
}

// New creates a DataForSEO client with Basic Auth credentials.
func New(login, password string, opts ...Option) *Client {
	c := &Client{
		login:      login,
		password:   password,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Post sends a POST request to the given endpoint path (e.g. "/v3/serp/google/organic/live/regular")
// with the provided body serialized as JSON. Returns the raw response bytes.
func (c *Client) Post(endpoint string, body any) ([]byte, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+basicAuth(c.login, c.password))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dataforseo returned status %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

func basicAuth(login, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(login + ":" + password))
}
