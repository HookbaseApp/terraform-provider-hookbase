package client

import (
	"context"
	"encoding/json"
	"fmt"
)

type FieldMapping struct {
	Source  string  `json:"source"`
	Target string  `json:"target"`
	Type   string  `json:"type"`
	Default *string `json:"default,omitempty"`
}

type Destination struct {
	ID                 string            `json:"id"`
	OrganizationID     string            `json:"organizationId"`
	Name               string            `json:"name"`
	Slug               string            `json:"slug"`
	URL                string            `json:"url"`
	Method             string            `json:"method"`
	Headers            map[string]string `json:"headers"`
	AuthType           *string           `json:"authType"`
	AuthConfig         map[string]string `json:"authConfig"`
	TimeoutMs          *int              `json:"timeoutMs"`
	RateLimitPerMinute *int              `json:"rateLimitPerMinute"`
	Type               string            `json:"type"`
	Config             json.RawMessage   `json:"config"`
	BatchSize          *int              `json:"batchSize"`
	BatchWindowSeconds *int              `json:"batchWindowSeconds"`
	FieldMapping       []FieldMapping    `json:"fieldMapping"`
	UseStaticIP        bool              `json:"useStaticIp"`
	IsActive           bool              `json:"isActive"`
	CreatedAt          string            `json:"createdAt"`
	UpdatedAt          string            `json:"updatedAt"`
}

type CreateDestinationRequest struct {
	Name               string            `json:"name"`
	Slug               string            `json:"slug"`
	URL                string            `json:"url"`
	Method             *string           `json:"method,omitempty"`
	Headers            map[string]string `json:"headers,omitempty"`
	AuthType           *string           `json:"authType,omitempty"`
	AuthConfig         map[string]string `json:"authConfig,omitempty"`
	TimeoutMs          *int              `json:"timeoutMs,omitempty"`
	RateLimitPerMinute *int              `json:"rateLimitPerMinute,omitempty"`
	Type               *string           `json:"type,omitempty"`
	Config             json.RawMessage   `json:"config,omitempty"`
	BatchSize          *int              `json:"batchSize,omitempty"`
	BatchWindowSeconds *int              `json:"batchWindowSeconds,omitempty"`
	FieldMapping       []FieldMapping    `json:"fieldMapping,omitempty"`
	UseStaticIP        *bool             `json:"useStaticIp,omitempty"`
}

type UpdateDestinationRequest struct {
	Name               *string           `json:"name,omitempty"`
	URL                *string           `json:"url,omitempty"`
	Method             *string           `json:"method,omitempty"`
	Headers            map[string]string `json:"headers,omitempty"`
	AuthType           *string           `json:"authType,omitempty"`
	AuthConfig         map[string]string `json:"authConfig,omitempty"`
	TimeoutMs          *int              `json:"timeoutMs,omitempty"`
	RateLimitPerMinute *int              `json:"rateLimitPerMinute,omitempty"`
	Type               *string           `json:"type,omitempty"`
	Config             json.RawMessage   `json:"config,omitempty"`
	BatchSize          *int              `json:"batchSize,omitempty"`
	BatchWindowSeconds *int              `json:"batchWindowSeconds,omitempty"`
	FieldMapping       []FieldMapping    `json:"fieldMapping,omitempty"`
	UseStaticIP        *bool             `json:"useStaticIp,omitempty"`
	IsActive           *bool             `json:"isActive,omitempty"`
}

type destinationResponse struct {
	Destination Destination `json:"destination"`
}

func (c *Client) CreateDestination(ctx context.Context, req CreateDestinationRequest) (*Destination, error) {
	var resp destinationResponse
	if err := c.Post(ctx, "/destinations", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Destination, nil
}

func (c *Client) GetDestination(ctx context.Context, id string) (*Destination, error) {
	var resp destinationResponse
	if err := c.Get(ctx, fmt.Sprintf("/destinations/%s", id), &resp); err != nil {
		return nil, err
	}
	return &resp.Destination, nil
}

func (c *Client) UpdateDestination(ctx context.Context, id string, req UpdateDestinationRequest) (*Destination, error) {
	var resp destinationResponse
	if err := c.Patch(ctx, fmt.Sprintf("/destinations/%s", id), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Destination, nil
}

func (c *Client) DeleteDestination(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/destinations/%s", id))
}
