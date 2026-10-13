package client

import (
	"context"
	"net/http"
)

// EscalationPolicyStep is one step in an escalation chain.
type EscalationPolicyStep struct {
	StepOrder   int      `json:"stepOrder"`
	TargetType  string   `json:"targetType"` // "user" | "schedule"
	TargetID    int64    `json:"targetId"`
	WaitMinutes int      `json:"waitMinutes"`
	Channels    []string `json:"channels"`
}

// EscalationPolicy is an on-call escalation policy.
type EscalationPolicy struct {
	PublicID  string                 `json:"publicId"`
	Name      string                 `json:"name"`
	Steps     []EscalationPolicyStep `json:"steps"`
	CreatedAt string                 `json:"createdAt"`
	UpdatedAt string                 `json:"updatedAt"`
}

type EscalationPolicyRequest struct {
	Name  string                      `json:"name"`
	Steps []EscalationPolicyStepInput `json:"steps"`
}

type EscalationPolicyStepInput struct {
	StepOrder   int      `json:"stepOrder"`
	TargetType  string   `json:"targetType"`
	TargetID    int64    `json:"targetId"`
	WaitMinutes *int     `json:"waitMinutes,omitempty"`
	Channels    []string `json:"channels,omitempty"`
}

func (c *Client) GetEscalationPolicy(ctx context.Context, publicID string) (*EscalationPolicy, error) {
	// List all and find by publicId — the API has no single-resource GET.
	type listItem struct {
		PublicID  string                 `json:"publicId"`
		Name      string                 `json:"name"`
		Steps     []EscalationPolicyStep `json:"steps"`
		CreatedAt string                 `json:"createdAt"`
		UpdatedAt string                 `json:"updatedAt"`
	}
	resp, err := c.do(ctx, http.MethodGet, "/api/on-call/policies", nil)
	if err != nil {
		return nil, err
	}
	items, err := decodeJSON[[]listItem](resp)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.PublicID == publicID {
			return &EscalationPolicy{
				PublicID:  item.PublicID,
				Name:      item.Name,
				Steps:     item.Steps,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			}, nil
		}
	}
	return nil, &APIError{StatusCode: 404, URL: "/api/on-call/policies", Body: "not found"}
}

func (c *Client) CreateEscalationPolicy(ctx context.Context, req EscalationPolicyRequest) (*EscalationPolicy, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/on-call/policies", req)
	if err != nil {
		return nil, err
	}
	p, err := decodeJSON[EscalationPolicy](resp)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) UpdateEscalationPolicy(ctx context.Context, publicID string, req EscalationPolicyRequest) error {
	resp, err := c.do(ctx, http.MethodPut, "/api/on-call/policies/"+publicID, req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) DeleteEscalationPolicy(ctx context.Context, publicID string) error {
	resp, err := c.do(ctx, http.MethodDelete, "/api/on-call/policies/"+publicID, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
