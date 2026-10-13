package client

import (
	"context"
	"fmt"
	"net/http"
)

// Monitor is the API representation of an UptimeWarden monitor.
type Monitor struct {
	PublicID        string `json:"publicId"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	URL             string `json:"url"`
	Interval        int    `json:"interval"`
	Timeout         int    `json:"timeout"`
	IsActive        bool   `json:"isActive"`
	Region          string `json:"region,omitempty"`
	ManagedBySource string `json:"managedBySource,omitempty"`
	CreatedAt       string `json:"createdAt,omitempty"`
	UpdatedAt       string `json:"updatedAt,omitempty"`
}

// CreateMonitorRequest is the body for POST /api/v1/monitors.
type CreateMonitorRequest struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	URL             string `json:"url"`
	Interval        int    `json:"interval,omitempty"`
	Timeout         int    `json:"timeout,omitempty"`
	Region          string `json:"region,omitempty"`
	IsActive        *bool  `json:"isActive,omitempty"`
	ManagedBySource string `json:"managedBySource"`
}

// UpdateMonitorRequest is the body for PATCH /api/v1/monitors/{id}.
// Timeout and Region are pointers: nil leaves the stored value unchanged,
// which is what the provider sends when the attribute is not configured.
type UpdateMonitorRequest struct {
	Name     string  `json:"name,omitempty"`
	URL      string  `json:"url,omitempty"`
	Interval int     `json:"interval,omitempty"`
	Timeout  *int    `json:"timeout,omitempty"`
	Region   *string `json:"region,omitempty"`
	IsActive *bool   `json:"isActive,omitempty"`
}

// GetMonitor fetches a single monitor by its public ID.
func (c *Client) GetMonitor(ctx context.Context, publicID string) (*Monitor, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/monitors/"+publicID, nil)
	if err != nil {
		return nil, err
	}
	m, err := decodeJSON[Monitor](resp)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// CreateMonitor creates a new monitor.
func (c *Client) CreateMonitor(ctx context.Context, req CreateMonitorRequest) (*Monitor, error) {
	req.ManagedBySource = "terraform"
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/monitors", req)
	if err != nil {
		return nil, err
	}
	m, err := decodeJSON[Monitor](resp)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// UpdateMonitor updates a monitor by its public ID.
func (c *Client) UpdateMonitor(ctx context.Context, publicID string, req UpdateMonitorRequest) (*Monitor, error) {
	resp, err := c.do(ctx, http.MethodPatch, "/api/v1/monitors/"+publicID, req)
	if err != nil {
		return nil, fmt.Errorf("update monitor %s: %w", publicID, err)
	}
	m, err := decodeJSON[Monitor](resp)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// DeleteMonitor deletes a monitor by its public ID.
func (c *Client) DeleteMonitor(ctx context.Context, publicID string) error {
	resp, err := c.do(ctx, http.MethodDelete, "/api/v1/monitors/"+publicID, nil)
	if err != nil {
		return fmt.Errorf("delete monitor %s: %w", publicID, err)
	}
	resp.Body.Close()
	return nil
}
