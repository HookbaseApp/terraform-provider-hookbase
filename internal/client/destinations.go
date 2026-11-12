package client

import (
	"context"
	"encoding/json"
	"fmt"
)

type FieldMapping struct {
	Source  string  `json:"source"`
	Target  string  `json:"target"`
	Type    string  `json:"type"`
	Default *string `json:"default,omitempty"`
}

// Throttle configures delivery throttling for a destination. Mode is one of
// "off", "rate", or "concurrency". RateLimit+RateUnit are required when Mode
// is "rate"; MaxConcurrency is required when Mode is "concurrency"; QueueLimit
// is optional in both.
type Throttle struct {
	Mode           string  `json:"mode"`
	RateLimit      *int    `json:"rateLimit,omitempty"`
	RateUnit       *string `json:"rateUnit,omitempty"`
	MaxConcurrency *int    `json:"maxConcurrency,omitempty"`
	QueueLimit     *int    `json:"queueLimit,omitempty"`
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
	Throttle           *Throttle         `json:"throttle,omitempty"`
	Type               string            `json:"type"`
	Config             json.RawMessage   `json:"config"`
	BatchSize          *int              `json:"batchSize"`
	BatchWindowSeconds *int              `json:"batchWindowSeconds"`
	FieldMapping       []FieldMapping    `json:"fieldMapping"`
	UseStaticIP        bool              `json:"useStaticIp"`
	IsActive           bool              `json:"isActive"`
	// HasSigningSecret: the destination signs its deliveries (http destinations). The secret itself
	// is in the create reply only (SigningSecret, null for other types); after that it takes
	// RevealDestinationSecret.
	HasSigningSecret bool    `json:"hasSigningSecret"`
	SigningSecret    *string `json:"signingSecret,omitempty"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
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
	Throttle           *Throttle         `json:"throttle,omitempty"`
	Type               *string           `json:"type,omitempty"`
	Config             json.RawMessage   `json:"config,omitempty"`
	BatchSize          *int              `json:"batchSize,omitempty"`
	BatchWindowSeconds *int              `json:"batchWindowSeconds,omitempty"`
	FieldMapping       []FieldMapping    `json:"fieldMapping,omitempty"`
	UseStaticIP        *bool             `json:"useStaticIp,omitempty"`
}

// UpdateDestinationRequest is the PATCH body; a nil field is left out and keeps the stored value.
// Headers is a pointer so that a pointer to a nil map sends JSON null (clear), and Config clears
// with json.RawMessage("null"). The API refuses signingSecret here, so there is no field for it.
type UpdateDestinationRequest struct {
	Name               *string            `json:"name,omitempty"`
	URL                *string            `json:"url,omitempty"`
	Method             *string            `json:"method,omitempty"`
	Headers            *map[string]string `json:"headers,omitempty"`
	AuthType           *string            `json:"authType,omitempty"`
	AuthConfig         map[string]string  `json:"authConfig,omitempty"`
	TimeoutMs          *int               `json:"timeoutMs,omitempty"`
	Throttle           *Throttle          `json:"throttle,omitempty"`
	Type               *string            `json:"type,omitempty"`
	Config             json.RawMessage    `json:"config,omitempty"`
	BatchSize          *int               `json:"batchSize,omitempty"`
	BatchWindowSeconds *int               `json:"batchWindowSeconds,omitempty"`
	FieldMapping       []FieldMapping     `json:"fieldMapping,omitempty"`
	UseStaticIP        *bool              `json:"useStaticIp,omitempty"`
	IsActive           *bool              `json:"isActive,omitempty"`
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

// UpdateDestination sends the PATCH. The API answers {success: true} with no destination, so
// callers read it back with GetDestination.
func (c *Client) UpdateDestination(ctx context.Context, id string, req UpdateDestinationRequest) error {
	return c.Patch(ctx, fmt.Sprintf("/destinations/%s", id), req, nil)
}

func (c *Client) DeleteDestination(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/destinations/%s", id))
}

type revealDestinationSecretResponse struct {
	SigningSecret string `json:"signingSecret"`
	// PreviousSecretExpiresAt is when a rotation's grace period ends (both secrets sign until then),
	// or null.
	PreviousSecretExpiresAt *string `json:"previousSecretExpiresAt"`
}

// RevealDestinationSecret reads an http destination's signing secret. The API audit-logs every
// call, answers 400 for a type that has no secret and 404 when none is stored, and needs a key
// with write scope.
func (c *Client) RevealDestinationSecret(ctx context.Context, id string) (string, error) {
	var resp revealDestinationSecretResponse
	if err := c.Get(ctx, fmt.Sprintf("/destinations/%s/reveal-secret", id), &resp); err != nil {
		return "", err
	}
	return resp.SigningSecret, nil
}
