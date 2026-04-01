package client

import (
	"context"
	"encoding/json"
	"fmt"
)

type WebhookSubscription struct {
	ID               string          `json:"id"`
	EndpointID       string          `json:"endpointId"`
	EventTypeID      string          `json:"eventTypeId"`
	FilterExpression *string         `json:"filterExpression"`
	LabelFilters     json.RawMessage `json:"labelFilters"`
	LabelFilterMode  string          `json:"labelFilterMode"`
	TransformID      *string         `json:"transformId"`
	PriorityOverride *int            `json:"priorityOverride"`
	IsEnabled        bool            `json:"isEnabled"`
	CreatedAt        string          `json:"createdAt"`
}

type CreateWebhookSubscriptionRequest struct {
	EndpointID       string          `json:"endpointId"`
	EventTypeID      string          `json:"eventTypeId"`
	FilterExpression *string         `json:"filterExpression,omitempty"`
	LabelFilters     json.RawMessage `json:"labelFilters,omitempty"`
	LabelFilterMode  *string         `json:"labelFilterMode,omitempty"`
	TransformID      *string         `json:"transformId,omitempty"`
	IsEnabled        *bool           `json:"isEnabled,omitempty"`
	PriorityOverride *int            `json:"priorityOverride,omitempty"`
}

type UpdateWebhookSubscriptionRequest struct {
	FilterExpression *string         `json:"filterExpression,omitempty"`
	LabelFilters     json.RawMessage `json:"labelFilters,omitempty"`
	LabelFilterMode  *string         `json:"labelFilterMode,omitempty"`
	TransformID      *string         `json:"transformId,omitempty"`
	IsEnabled        *bool           `json:"isEnabled,omitempty"`
	PriorityOverride *int            `json:"priorityOverride,omitempty"`
}

type webhookSubscriptionResponse struct {
	Data WebhookSubscription `json:"data"`
}

func (c *Client) CreateWebhookSubscription(ctx context.Context, req CreateWebhookSubscriptionRequest) (*WebhookSubscription, error) {
	var resp webhookSubscriptionResponse
	if err := c.Post(ctx, "/webhook-subscriptions", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (c *Client) GetWebhookSubscription(ctx context.Context, id string) (*WebhookSubscription, error) {
	var resp webhookSubscriptionResponse
	if err := c.Get(ctx, fmt.Sprintf("/webhook-subscriptions/%s", id), &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (c *Client) UpdateWebhookSubscription(ctx context.Context, id string, req UpdateWebhookSubscriptionRequest) (*WebhookSubscription, error) {
	var resp webhookSubscriptionResponse
	if err := c.Patch(ctx, fmt.Sprintf("/webhook-subscriptions/%s", id), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (c *Client) DeleteWebhookSubscription(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/webhook-subscriptions/%s", id))
}
