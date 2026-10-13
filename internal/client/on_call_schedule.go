package client

import (
	"context"
	"net/http"
)

// OnCallScheduleMember maps a user to a day-of-week on-call slot.
type OnCallScheduleMember struct {
	UserID    int64 `json:"userId"`
	DayOfWeek int   `json:"dayOfWeek"` // 0=Sunday … 6=Saturday
}

// OnCallSchedule is a weekly on-call rotation schedule.
type OnCallSchedule struct {
	PublicID  string                 `json:"publicId"`
	Name      string                 `json:"name"`
	Timezone  string                 `json:"timezone"`
	Members   []OnCallScheduleMember `json:"members"`
	CreatedAt string                 `json:"createdAt"`
	UpdatedAt string                 `json:"updatedAt"`
}

type OnCallScheduleRequest struct {
	Name     string                      `json:"name"`
	Timezone string                      `json:"timezone,omitempty"`
	Members  []OnCallScheduleMemberInput `json:"members,omitempty"`
}

type OnCallScheduleMemberInput struct {
	UserID    int64 `json:"userId"`
	DayOfWeek int   `json:"dayOfWeek"`
}

func (c *Client) GetOnCallSchedule(ctx context.Context, publicID string) (*OnCallSchedule, error) {
	type listItem struct {
		PublicID  string                 `json:"publicId"`
		Name      string                 `json:"name"`
		Timezone  string                 `json:"timezone"`
		Members   []OnCallScheduleMember `json:"members"`
		CreatedAt string                 `json:"createdAt"`
		UpdatedAt string                 `json:"updatedAt"`
	}
	resp, err := c.do(ctx, http.MethodGet, "/api/on-call/schedules", nil)
	if err != nil {
		return nil, err
	}
	items, err := decodeJSON[[]listItem](resp)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.PublicID == publicID {
			return &OnCallSchedule{
				PublicID:  item.PublicID,
				Name:      item.Name,
				Timezone:  item.Timezone,
				Members:   item.Members,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			}, nil
		}
	}
	return nil, &APIError{StatusCode: 404, URL: "/api/on-call/schedules", Body: "not found"}
}

func (c *Client) CreateOnCallSchedule(ctx context.Context, req OnCallScheduleRequest) (*OnCallSchedule, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/on-call/schedules", req)
	if err != nil {
		return nil, err
	}
	s, err := decodeJSON[OnCallSchedule](resp)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) UpdateOnCallSchedule(ctx context.Context, publicID string, req OnCallScheduleRequest) error {
	resp, err := c.do(ctx, http.MethodPut, "/api/on-call/schedules/"+publicID, req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) DeleteOnCallSchedule(ctx context.Context, publicID string) error {
	resp, err := c.do(ctx, http.MethodDelete, "/api/on-call/schedules/"+publicID, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
