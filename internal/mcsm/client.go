// Package mcsm implements an HTTP API client for the MCSManager panel,
// following the official API documentation.
// Authentication: the apikey is sent as a URL query parameter;
// all requests carry the fixed headers required by the API spec.
package mcsm

import (
	"bytes"
	"context"
	stdjson "encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.gh.ink/json"
)

// RawMessage is a raw JSON byte slice.  go.gh.ink/json does not export this type,
// but its backends (sonic/goccy) are fully compatible with the standard library's
// RawMessage semantics.
type RawMessage = stdjson.RawMessage

// Client is the panel API client.
type Client struct {
	BaseURL string // e.g. https://panel.example.com, without trailing slash
	APIKey  string
	HTTP    *http.Client
}

// New creates a new Client.
func New(baseURL, apiKey string, timeout time.Duration) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: timeout},
	}
}

// envelope is the panel's uniform response wrapper {status, data, time}.
type envelope struct {
	Status int        `json:"status"`
	Data   RawMessage `json:"data"`
	Time   int64      `json:"time"`
}

// APIError represents a non-200 business status returned by the panel.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	desc := map[int]string{
		400: "bad request parameters",
		403: "insufficient permissions",
		500: "internal server error",
	}[e.Status]
	if desc == "" {
		desc = "request failed"
	}
	if e.Message != "" {
		return fmt.Sprintf("API error %d (%s): %s", e.Status, desc, e.Message)
	}
	return fmt.Sprintf("API error %d (%s)", e.Status, desc)
}

// Do performs an API request.  query may be nil; body is serialized to JSON
// when non-nil; out receives the response data when non-nil.
// Returns the raw data for --json output.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body, out any) (RawMessage, error) {
	if query == nil {
		query = url.Values{}
	}
	query.Set("apikey", c.APIKey)
	fullURL := c.BaseURL + path + "?" + query.Encode()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("serialize request body: %w", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reader)
	if err != nil {
		return nil, err
	}
	// These headers are required per the API documentation.
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s failed: %w", path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("response is not valid panel JSON (HTTP %d): %s", resp.StatusCode, truncate(raw, 200))
	}
	if env.Status != 200 {
		msg := ""
		var s string
		if json.Unmarshal(env.Data, &s) == nil {
			msg = s
		} else if len(env.Data) > 0 {
			msg = truncate(env.Data, 200)
		}
		return nil, &APIError{Status: env.Status, Message: msg}
	}
	if out != nil {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return nil, fmt.Errorf("parse response data: %w", err)
		}
	}
	return env.Data, nil
}

func truncate(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n]) + "..."
	}
	return string(b)
}
