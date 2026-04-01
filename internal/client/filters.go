package client

import (
	"context"
	"encoding/json"
	"fmt"
)

type Filter struct {
	ID             string          `json:"id"`
	OrganizationID string          `json:"organizationId"`
	Name           string          `json:"name"`
	Slug           string          `json:"slug"`
	Description    *string         `json:"description"`
	Conditions     json.RawMessage `json:"conditions"`
	Logic          string          `json:"logic"`
	CreatedAt      string          `json:"createdAt"`
	UpdatedAt      string          `json:"updatedAt"`
}

type CreateFilterRequest struct {
	Name        string          `json:"name"`
	Slug        string          `json:"slug,omitempty"`
	Description *string         `json:"description,omitempty"`
	Conditions  json.RawMessage `json:"conditions"`
	Logic       *string         `json:"logic,omitempty"`
}

type UpdateFilterRequest struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Conditions  json.RawMessage `json:"conditions,omitempty"`
	Logic       *string         `json:"logic,omitempty"`
}

type filterResponse struct {
	Filter Filter `json:"filter"`
}

func (c *Client) CreateFilter(ctx context.Context, req CreateFilterRequest) (*Filter, error) {
	var resp filterResponse
	if err := c.Post(ctx, "/filters", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Filter, nil
}

func (c *Client) GetFilter(ctx context.Context, id string) (*Filter, error) {
	var resp filterResponse
	if err := c.Get(ctx, fmt.Sprintf("/filters/%s", id), &resp); err != nil {
		return nil, err
	}
	return &resp.Filter, nil
}

func (c *Client) UpdateFilter(ctx context.Context, id string, req UpdateFilterRequest) (*Filter, error) {
	var resp filterResponse
	if err := c.Patch(ctx, fmt.Sprintf("/filters/%s", id), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Filter, nil
}

func (c *Client) DeleteFilter(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/filters/%s", id))
}
