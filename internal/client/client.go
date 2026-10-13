// Package client is a thin HTTP wrapper over the UptimeWarden REST API.
// It is intentionally minimal — only the operations the Terraform provider
// actually needs are here. All requests carry X-Source: terraform so the
// server can set managed_by_source = 'terraform' on every mutated resource.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://app.uptimewarden.online"
	// sourceHeader is sent on every mutating request so the server knows
	// the resource is managed by Terraform and sets managed_by_source accordingly.
	sourceHeader = "terraform"
	userAgent    = "terraform-provider-uptimewarden/1.0"
	// bodyCap prevents reading an unbounded response body.
	bodyCap = 4 * 1024 * 1024 // 4 MiB
)

// Client wraps the UptimeWarden REST API.
type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

// New constructs a Client. baseURL defaults to the production URL when empty.
func New(baseURL, apiToken string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL:  baseURL,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIError is returned when the API responds with a non-2xx status. Code,
// Message and Details come from the v1 error envelope
// {success:false, error:{code, message, details?}} when the body has one.
type APIError struct {
	StatusCode int
	Body       string
	URL        string
	Code       string
	Message    string
	Details    []string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("uptimewarden API error %d on %s: %s", e.StatusCode, e.URL, e.Body)
	}
	msg := fmt.Sprintf("UptimeWarden API %d", e.StatusCode)
	if e.Code != "" {
		msg += " " + e.Code
	}
	msg += ": " + e.Message
	if len(e.Details) > 0 {
		msg += " (" + strings.Join(e.Details, "; ") + ")"
	}
	return msg
}

// parseAPIError fills Code, Message and Details from a v1 error envelope.
// details may be a string, a list of strings or anything else (rendered as
// JSON).
func parseAPIError(e *APIError) {
	var env struct {
		Error *struct {
			Code    string          `json:"code"`
			Message string          `json:"message"`
			Details json.RawMessage `json:"details"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal([]byte(e.Body), &env) != nil {
		return
	}
	if env.Error == nil {
		e.Message = env.Message
		return
	}
	e.Code, e.Message = env.Error.Code, env.Error.Message
	if len(env.Error.Details) == 0 || string(env.Error.Details) == "null" {
		return
	}
	var list []string
	var one string
	switch {
	case json.Unmarshal(env.Error.Details, &list) == nil:
		e.Details = list
	case json.Unmarshal(env.Error.Details, &one) == nil:
		e.Details = []string{one}
	default:
		e.Details = []string{string(env.Error.Details)}
	}
}

func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-Source", sourceHeader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 16384))
		apiErr := &APIError{StatusCode: resp.StatusCode, Body: string(raw), URL: url}
		parseAPIError(apiErr)
		return nil, apiErr
	}

	return resp, nil
}

func decodeJSON[T any](resp *http.Response) (T, error) {
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, bodyCap))
	if err != nil {
		var zero T
		return zero, fmt.Errorf("read response: %w", err)
	}
	var envelope struct {
		Success *bool           `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Success != nil && len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		raw = envelope.Data
	}
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, fmt.Errorf("decode response: %w", err)
	}
	return v, nil
}

// IsNotFoundErr returns true when the API returned 404, including when the
// error has been wrapped with fmt.Errorf("%w", ...).
func IsNotFoundErr(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}
