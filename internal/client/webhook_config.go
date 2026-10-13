package client

import (
	"context"
	"net/http"
)

// WebhookConfig represents a webhook delivery configuration — an outbound
// HTTP endpoint that receives alert notifications.
type WebhookConfig struct {
	PublicID      string  `json:"publicId"`
	TypeID        int64   `json:"typeId"`
	Name          string  `json:"name"`
	Description   *string `json:"description"`
	URL           string  `json:"url"`
	HTTPMethod    string  `json:"httpMethod"`
	AuthMethod    *string `json:"authMethod"`
	Timeout       int     `json:"timeout"`
	RetryCount    int     `json:"retryCount"`
	RetryDelay    int     `json:"retryDelay"`
	IsActive      bool    `json:"isActive"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

type CreateWebhookConfigRequest struct {
	TypeID      int64   `json:"typeId"`
	Name        string  `json:"name"`
	URL         string  `json:"url"`
	HTTPMethod  string  `json:"httpMethod"`
	Description *string `json:"description,omitempty"`
	AuthMethod  *string `json:"authMethod,omitempty"`
	Timeout     int     `json:"timeout,omitempty"`
	RetryCount  int     `json:"retryCount,omitempty"`
	RetryDelay  int     `json:"retryDelay,omitempty"`
	IsActive    *bool   `json:"isActive,omitempty"`
}

type UpdateWebhookConfigRequest struct {
	Name        *string `json:"name,omitempty"`
	URL         *string `json:"url,omitempty"`
	HTTPMethod  *string `json:"httpMethod,omitempty"`
	Description *string `json:"description,omitempty"`
	AuthMethod  *string `json:"authMethod,omitempty"`
	Timeout     *int    `json:"timeout,omitempty"`
	RetryCount  *int    `json:"retryCount,omitempty"`
	RetryDelay  *int    `json:"retryDelay,omitempty"`
	IsActive    *bool   `json:"isActive,omitempty"`
}

func (c *Client) GetWebhookConfig(ctx context.Context, publicID string) (*WebhookConfig, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/webhooks/configs/"+publicID, nil)
	if err != nil {
		return nil, err
	}
	w, err := decodeJSON[WebhookConfig](resp)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (c *Client) CreateWebhookConfig(ctx context.Context, req CreateWebhookConfigRequest) (*WebhookConfig, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/webhooks/configs/", req)
	if err != nil {
		return nil, err
	}
	w, err := decodeJSON[WebhookConfig](resp)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (c *Client) UpdateWebhookConfig(ctx context.Context, publicID string, req UpdateWebhookConfigRequest) (*WebhookConfig, error) {
	resp, err := c.do(ctx, http.MethodPatch, "/api/webhooks/configs/"+publicID, req)
	if err != nil {
		return nil, err
	}
	w, err := decodeJSON[WebhookConfig](resp)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (c *Client) DeleteWebhookConfig(ctx context.Context, publicID string) error {
	resp, err := c.do(ctx, http.MethodDelete, "/api/webhooks/configs/"+publicID, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
