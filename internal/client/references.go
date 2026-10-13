package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// EscalationPolicy is a row of GET /api/v1/escalation-policies.
type EscalationPolicy struct {
	PublicID  string `json:"publicId"`
	Name      string `json:"name"`
	StepCount int64  `json:"stepCount"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// Webhook is a row of GET /api/v1/webhooks.
type Webhook struct {
	PublicID            string   `json:"publicId"`
	Name                string   `json:"name"`
	Description         *string  `json:"description"`
	MessageFormat       *string  `json:"messageFormat"`
	IsActive            bool     `json:"isActive"`
	EnabledAlertTypes   []string `json:"enabledAlertTypes"`
	EnabledMonitorTypes []string `json:"enabledMonitorTypes"`
	CreatedAt           string   `json:"createdAt"`
}

// Region is an active region from GET /api/v1/regions.
type Region struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsDefault   bool    `json:"isDefault"`
}

// RegionAllowance is the caller's plan allowance from GET /api/v1/regions.
// -1 means unlimited.
type RegionAllowance struct {
	RegionSelection          bool  `json:"regionSelection"`
	MaxRegions               int64 `json:"maxRegions"`
	MaxVerificationRegions   int64 `json:"maxVerificationRegions"`
	DefaultConfirmingRegions int64 `json:"defaultConfirmingRegions"`
	AllowConsensusOverride   bool  `json:"allowConsensusOverride"`
}

// Regions is the full GET /api/v1/regions answer.
type Regions struct {
	Regions []Region
	Plan    *RegionAllowance
}

// ListEscalationPolicies returns the caller's escalation policies.
func (c *Client) ListEscalationPolicies(ctx context.Context) ([]EscalationPolicy, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/escalation-policies", nil)
	if err != nil {
		return nil, err
	}
	return decodeJSON[[]EscalationPolicy](resp)
}

// ListWebhooks returns the caller's webhook configurations.
func (c *Client) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/webhooks", nil)
	if err != nil {
		return nil, err
	}
	return decodeJSON[[]Webhook](resp)
}

// ListRegions returns the active regions and the caller's plan allowance.
func (c *Client) ListRegions(ctx context.Context) (*Regions, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/regions", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, bodyCap))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	var body struct {
		Data []Region         `json:"data"`
		Plan *RegionAllowance `json:"plan"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &Regions{Regions: body.Data, Plan: body.Plan}, nil
}
