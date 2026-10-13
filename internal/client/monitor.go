package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Monitor is the v1 monitor resource as the API returns it: a JSON object
// keyed by the API's camelCase field names. It is kept as a map because the
// resource has some sixty fields and the provider maps them table-driven
// (internal/provider/monitor_fields.go).
type Monitor map[string]any

// PublicID returns the monitor's public id ("" when absent).
func (m Monitor) PublicID() string {
	s, _ := m["publicId"].(string)
	return s
}

// GetMonitor fetches a single monitor by its public ID.
func (c *Client) GetMonitor(ctx context.Context, publicID string) (Monitor, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/monitors/"+url.PathEscape(publicID), nil)
	if err != nil {
		return nil, err
	}
	return decodeJSON[Monitor](resp)
}

// CreateMonitor creates a monitor. body uses the API's field names; the
// client sets managedBySource to "terraform".
func (c *Client) CreateMonitor(ctx context.Context, body map[string]any) (Monitor, error) {
	out := make(map[string]any, len(body)+1)
	for k, v := range body {
		out[k] = v
	}
	out["managedBySource"] = "terraform"
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/monitors", out)
	if err != nil {
		return nil, err
	}
	return decodeJSON[Monitor](resp)
}

// UpdateMonitor PATCHes a monitor. Only the fields in body change.
func (c *Client) UpdateMonitor(ctx context.Context, publicID string, body map[string]any) (Monitor, error) {
	resp, err := c.do(ctx, http.MethodPatch, "/api/v1/monitors/"+url.PathEscape(publicID), body)
	if err != nil {
		return nil, fmt.Errorf("update monitor %s: %w", publicID, err)
	}
	return decodeJSON[Monitor](resp)
}

// DeleteMonitor deletes a monitor by its public ID.
func (c *Client) DeleteMonitor(ctx context.Context, publicID string) error {
	resp, err := c.do(ctx, http.MethodDelete, "/api/v1/monitors/"+url.PathEscape(publicID), nil)
	if err != nil {
		return fmt.Errorf("delete monitor %s: %w", publicID, err)
	}
	resp.Body.Close()
	return nil
}
