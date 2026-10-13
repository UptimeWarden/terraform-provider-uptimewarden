package client

import (
	"context"
	"net/http"
)

// StatusPage represents a public or private UptimeWarden status page.
type StatusPage struct {
	PublicID           string  `json:"publicId"`
	Name               string  `json:"name"`
	Title              string  `json:"title"`
	Slug               string  `json:"slug"`
	Description        *string `json:"description"`
	Domain             *string `json:"domain"`
	LogoURL            *string `json:"logoUrl"`
	FaviconURL         *string `json:"faviconUrl"`
	CustomHTML         *string `json:"customHtml"`
	IsPublic           bool    `json:"isPublic"`
	AllowSubscriptions bool    `json:"allowSubscriptions"`
	ShowHistory        bool    `json:"showHistory"`
	HistoryDays        int     `json:"historyDays"`
	ShowResponseTime   bool    `json:"showResponseTime"`
	ShowUptime         bool    `json:"showUptime"`
	HidePoweredBy      bool    `json:"hidePoweredBy"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          string  `json:"updatedAt"`
}

type CreateStatusPageRequest struct {
	Name               string  `json:"name"`
	Title              string  `json:"title"`
	Slug               string  `json:"slug"`
	Description        *string `json:"description,omitempty"`
	Domain             *string `json:"domain,omitempty"`
	LogoURL            *string `json:"logoUrl,omitempty"`
	FaviconURL         *string `json:"faviconUrl,omitempty"`
	CustomHTML         *string `json:"customHtml,omitempty"`
	IsPublic           *bool   `json:"isPublic,omitempty"`
	AllowSubscriptions *bool   `json:"allowSubscriptions,omitempty"`
	ShowHistory        *bool   `json:"showHistory,omitempty"`
	HistoryDays        *int    `json:"historyDays,omitempty"`
	ShowResponseTime   *bool   `json:"showResponseTime,omitempty"`
	ShowUptime         *bool   `json:"showUptime,omitempty"`
	HidePoweredBy      *bool   `json:"hidePoweredBy,omitempty"`
}

type UpdateStatusPageRequest struct {
	Name               *string `json:"name,omitempty"`
	Title              *string `json:"title,omitempty"`
	Slug               *string `json:"slug,omitempty"`
	Description        *string `json:"description,omitempty"`
	Domain             *string `json:"domain,omitempty"`
	LogoURL            *string `json:"logoUrl,omitempty"`
	FaviconURL         *string `json:"faviconUrl,omitempty"`
	CustomHTML         *string `json:"customHtml,omitempty"`
	IsPublic           *bool   `json:"isPublic,omitempty"`
	AllowSubscriptions *bool   `json:"allowSubscriptions,omitempty"`
	ShowHistory        *bool   `json:"showHistory,omitempty"`
	HistoryDays        *int    `json:"historyDays,omitempty"`
	ShowResponseTime   *bool   `json:"showResponseTime,omitempty"`
	ShowUptime         *bool   `json:"showUptime,omitempty"`
	HidePoweredBy      *bool   `json:"hidePoweredBy,omitempty"`
}

func (c *Client) GetStatusPage(ctx context.Context, publicID string) (*StatusPage, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/status-pages/"+publicID, nil)
	if err != nil {
		return nil, err
	}
	s, err := decodeJSON[StatusPage](resp)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) CreateStatusPage(ctx context.Context, req CreateStatusPageRequest) (*StatusPage, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/status-pages", req)
	if err != nil {
		return nil, err
	}
	s, err := decodeJSON[StatusPage](resp)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) UpdateStatusPage(ctx context.Context, publicID string, req UpdateStatusPageRequest) (*StatusPage, error) {
	resp, err := c.do(ctx, http.MethodPatch, "/api/status-pages/"+publicID, req)
	if err != nil {
		return nil, err
	}
	s, err := decodeJSON[StatusPage](resp)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) DeleteStatusPage(ctx context.Context, publicID string) error {
	resp, err := c.do(ctx, http.MethodDelete, "/api/status-pages/"+publicID, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
