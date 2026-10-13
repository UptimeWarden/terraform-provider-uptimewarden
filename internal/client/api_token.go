package client

import (
	"context"
	"net/http"
)

// APIToken represents a UptimeWarden API token.
// The plain token value (plainToken) is only available at creation time.
type APIToken struct {
	PublicID    string   `json:"publicId"`
	Name        string   `json:"name"`
	Prefix      string   `json:"prefix"`
	Permissions []string `json:"permissions"`
	ExpiresAt   *string  `json:"expiresAt"`
	IsActive    bool     `json:"isActive"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

// APITokenCreateResult is the POST 201 response — the only time plainToken
// is returned. Store it in Terraform state (sensitive = true).
type APITokenCreateResult struct {
	Token      APIToken `json:"token"`
	PlainToken string   `json:"plainToken"`
}

type CreateAPITokenRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	ExpiresAt   *string  `json:"expiresAt,omitempty"`
}

type UpdateAPITokenRequest struct {
	Name      *string `json:"name,omitempty"`
	IsActive  *bool   `json:"isActive,omitempty"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
}

type UpdateAPITokenPermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

func (c *Client) GetAPIToken(ctx context.Context, publicID string) (*APIToken, error) {
	type listItem struct {
		PublicID    string   `json:"publicId"`
		Name        string   `json:"name"`
		Prefix      string   `json:"prefix"`
		Permissions []string `json:"permissions"`
		ExpiresAt   *string  `json:"expiresAt"`
		IsActive    bool     `json:"isActive"`
		CreatedAt   string   `json:"createdAt"`
		UpdatedAt   string   `json:"updatedAt"`
	}
	resp, err := c.do(ctx, http.MethodGet, "/api/user/api-tokens/", nil)
	if err != nil {
		return nil, err
	}
	// API returns wrapped object with tokens array.
	var wrapper struct {
		Tokens []listItem `json:"tokens"`
	}
	if decErr := decodeJSONInto(resp, &wrapper); decErr != nil {
		return nil, decErr
	}
	for _, item := range wrapper.Tokens {
		if item.PublicID == publicID {
			return &APIToken{
				PublicID:    item.PublicID,
				Name:        item.Name,
				Prefix:      item.Prefix,
				Permissions: item.Permissions,
				ExpiresAt:   item.ExpiresAt,
				IsActive:    item.IsActive,
				CreatedAt:   item.CreatedAt,
				UpdatedAt:   item.UpdatedAt,
			}, nil
		}
	}
	return nil, &APIError{StatusCode: 404, URL: "/api/user/api-tokens/", Body: "token not found"}
}

func (c *Client) CreateAPIToken(ctx context.Context, req CreateAPITokenRequest) (*APITokenCreateResult, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/user/api-tokens/", req)
	if err != nil {
		return nil, err
	}
	r, err := decodeJSON[APITokenCreateResult](resp)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *Client) UpdateAPIToken(ctx context.Context, publicID string, req UpdateAPITokenRequest) (*APIToken, error) {
	resp, err := c.do(ctx, http.MethodPatch, "/api/user/api-tokens/"+publicID, req)
	if err != nil {
		return nil, err
	}
	t, err := decodeJSON[APIToken](resp)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (c *Client) UpdateAPITokenPermissions(ctx context.Context, publicID string, permissions []string) error {
	resp, err := c.do(ctx, http.MethodPatch, "/api/user/api-tokens/"+publicID+"/permissions",
		UpdateAPITokenPermissionsRequest{Permissions: permissions})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) DeleteAPIToken(ctx context.Context, publicID string) error {
	resp, err := c.do(ctx, http.MethodDelete, "/api/user/api-tokens/"+publicID, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
