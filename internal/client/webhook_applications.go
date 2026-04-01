package client

import (
	"context"
	"encoding/json"
	"fmt"
)

type WebhookApplication struct {
	ID                 string          `json:"id"`
	OrganizationID     string          `json:"organizationId"`
	ExternalID         *string         `json:"externalId"`
	Name               string          `json:"name"`
	Metadata           json.RawMessage `json:"metadata"`
	RateLimitPerSecond int             `json:"rateLimitPerSecond"`
	RateLimitPerMinute int             `json:"rateLimitPerMinute"`
	RateLimitPerHour   int             `json:"rateLimitPerHour"`
	IsDisabled         bool            `json:"isDisabled"`
	DisabledAt         *string         `json:"disabledAt"`
	DisabledReason     *string         `json:"disabledReason"`
	CreatedAt          string          `json:"createdAt"`
	UpdatedAt          string          `json:"updatedAt"`
}

type CreateWebhookApplicationRequest struct {
	ExternalID         *string         `json:"externalId,omitempty"`
	Name               string          `json:"name"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	RateLimitPerSecond *int            `json:"rateLimitPerSecond,omitempty"`
	RateLimitPerMinute *int            `json:"rateLimitPerMinute,omitempty"`
	RateLimitPerHour   *int            `json:"rateLimitPerHour,omitempty"`
}

type UpdateWebhookApplicationRequest struct {
	Name               *string         `json:"name,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	RateLimitPerSecond *int            `json:"rateLimitPerSecond,omitempty"`
	RateLimitPerMinute *int            `json:"rateLimitPerMinute,omitempty"`
	RateLimitPerHour   *int            `json:"rateLimitPerHour,omitempty"`
	IsDisabled         *bool           `json:"isDisabled,omitempty"`
	DisabledReason     *string         `json:"disabledReason,omitempty"`
}

type webhookApplicationResponse struct {
	Data WebhookApplication `json:"data"`
}

func (c *Client) CreateWebhookApplication(ctx context.Context, req CreateWebhookApplicationRequest) (*WebhookApplication, error) {
	var resp webhookApplicationResponse
	if err := c.Post(ctx, "/webhook-applications", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (c *Client) GetWebhookApplication(ctx context.Context, id string) (*WebhookApplication, error) {
	var resp webhookApplicationResponse
	if err := c.Get(ctx, fmt.Sprintf("/webhook-applications/%s", id), &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (c *Client) UpdateWebhookApplication(ctx context.Context, id string, req UpdateWebhookApplicationRequest) (*WebhookApplication, error) {
	var resp webhookApplicationResponse
	if err := c.Patch(ctx, fmt.Sprintf("/webhook-applications/%s", id), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (c *Client) DeleteWebhookApplication(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/webhook-applications/%s", id))
}
