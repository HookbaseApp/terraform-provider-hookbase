package client

import (
	"context"
	"encoding/json"
	"fmt"
)

type WebhookEndpoint struct {
	ID                      string          `json:"id"`
	ApplicationID           string          `json:"applicationId"`
	URL                     string          `json:"url"`
	Description             *string         `json:"description"`
	SecretPrefix            string          `json:"secretPrefix"`
	HasSecret               bool            `json:"hasSecret"`
	Secret                  *string         `json:"secret,omitempty"`
	SecretVersion           int             `json:"secretVersion"`
	Headers                 json.RawMessage `json:"headers"`
	TimeoutSeconds          int             `json:"timeoutSeconds"`
	IsDisabled              bool            `json:"isDisabled"`
	DisabledAt              *string         `json:"disabledAt"`
	DisabledReason          *string         `json:"disabledReason"`
	RateLimitPerSecond      int             `json:"rateLimitPerSecond"`
	SuccessStatusCodes      json.RawMessage `json:"successStatusCodes"`
	BackoffType             *string         `json:"backoffType"`
	RetryDelays             json.RawMessage `json:"retryDelays"`
	UseStaticIp             bool            `json:"useStaticIp"`
	IPAllowlistNotes        *string         `json:"ipAllowlistNotes"`
	CircuitState            string          `json:"circuitState"`
	CircuitOpenedAt         *string         `json:"circuitOpenedAt"`
	CircuitFailureCount     int             `json:"circuitFailureCount"`
	CircuitFailureThreshold int             `json:"circuitFailureThreshold"`
	CircuitSuccessThreshold int             `json:"circuitSuccessThreshold"`
	CircuitCooldownSeconds  int             `json:"circuitCooldownSeconds"`
	CreatedAt               string          `json:"createdAt"`
	UpdatedAt               string          `json:"updatedAt"`
}

type CreateWebhookEndpointRequest struct {
	ApplicationID           string          `json:"applicationId"`
	URL                     string          `json:"url"`
	Description             *string         `json:"description,omitempty"`
	Headers                 json.RawMessage `json:"headers,omitempty"`
	TimeoutSeconds          *int            `json:"timeoutSeconds,omitempty"`
	RateLimitPerSecond      *int            `json:"rateLimitPerSecond,omitempty"`
	SuccessStatusCodes      json.RawMessage `json:"successStatusCodes,omitempty"`
	BackoffType             *string         `json:"backoffType,omitempty"`
	RetryDelays             json.RawMessage `json:"retryDelays,omitempty"`
	UseStaticIp             *bool           `json:"useStaticIp,omitempty"`
	IPAllowlistNotes        *string         `json:"ipAllowlistNotes,omitempty"`
	CircuitFailureThreshold *int            `json:"circuitFailureThreshold,omitempty"`
	CircuitSuccessThreshold *int            `json:"circuitSuccessThreshold,omitempty"`
	CircuitCooldownSeconds  *int            `json:"circuitCooldownSeconds,omitempty"`
}

type UpdateWebhookEndpointRequest struct {
	URL                     *string         `json:"url,omitempty"`
	Description             *string         `json:"description,omitempty"`
	Headers                 json.RawMessage `json:"headers,omitempty"`
	TimeoutSeconds          *int            `json:"timeoutSeconds,omitempty"`
	IsDisabled              *bool           `json:"isDisabled,omitempty"`
	DisabledReason          *string         `json:"disabledReason,omitempty"`
	RateLimitPerSecond      *int            `json:"rateLimitPerSecond,omitempty"`
	SuccessStatusCodes      json.RawMessage `json:"successStatusCodes,omitempty"`
	BackoffType             *string         `json:"backoffType,omitempty"`
	RetryDelays             json.RawMessage `json:"retryDelays,omitempty"`
	UseStaticIp             *bool           `json:"useStaticIp,omitempty"`
	IPAllowlistNotes        *string         `json:"ipAllowlistNotes,omitempty"`
	CircuitFailureThreshold *int            `json:"circuitFailureThreshold,omitempty"`
	CircuitSuccessThreshold *int            `json:"circuitSuccessThreshold,omitempty"`
	CircuitCooldownSeconds  *int            `json:"circuitCooldownSeconds,omitempty"`
}

type webhookEndpointResponse struct {
	Data WebhookEndpoint `json:"data"`
}

func (c *Client) CreateWebhookEndpoint(ctx context.Context, req CreateWebhookEndpointRequest) (*WebhookEndpoint, error) {
	var resp webhookEndpointResponse
	if err := c.Post(ctx, "/webhook-endpoints", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (c *Client) GetWebhookEndpoint(ctx context.Context, id string) (*WebhookEndpoint, error) {
	var resp webhookEndpointResponse
	if err := c.Get(ctx, fmt.Sprintf("/webhook-endpoints/%s", id), &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (c *Client) UpdateWebhookEndpoint(ctx context.Context, id string, req UpdateWebhookEndpointRequest) (*WebhookEndpoint, error) {
	var resp webhookEndpointResponse
	if err := c.Patch(ctx, fmt.Sprintf("/webhook-endpoints/%s", id), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (c *Client) DeleteWebhookEndpoint(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/webhook-endpoints/%s", id))
}
