package client

import (
	"context"
	"fmt"
)

type Source struct {
	ID                      string   `json:"id"`
	OrganizationID          string   `json:"organizationId"`
	Name                    string   `json:"name"`
	Slug                    string   `json:"slug"`
	Provider                *string  `json:"provider"`
	Description             *string  `json:"description"`
	SigningSecret           *string  `json:"signingSecret,omitempty"`
	HasSigningSecret        bool     `json:"hasSigningSecret"`
	SigningSecretLast4      *string  `json:"signingSecretLast4"`
	RejectInvalidSignatures bool     `json:"rejectInvalidSignatures"`
	RateLimitPerMinute      *int     `json:"rateLimitPerMinute"`
	IsActive                bool     `json:"isActive"`
	IPFilterMode            string   `json:"ipFilterMode"`
	IPAllowlist             []string `json:"ipAllowlist"`
	IPDenylist              []string `json:"ipDenylist"`
	EncryptFields           []string `json:"encryptFields"`
	MaskFields              []string `json:"maskFields"`
	DedupEnabled            bool     `json:"dedupEnabled"`
	DedupStrategy           string   `json:"dedupStrategy"`
	DedupWindowHours        int      `json:"dedupWindowHours"`
	DedupCustomHeader       *string  `json:"dedupCustomHeader"`
	TransientMode           bool     `json:"transientMode"`
	AllowedMethods          []string `json:"allowedMethods"`
	IngestURL               *string  `json:"ingestUrl,omitempty"`
	CreatedAt               string   `json:"createdAt"`
	UpdatedAt               string   `json:"updatedAt"`
}

type CreateSourceRequest struct {
	Name                    string   `json:"name"`
	Slug                    string   `json:"slug"`
	Provider                *string  `json:"provider,omitempty"`
	Description             *string  `json:"description,omitempty"`
	SigningSecret           *string  `json:"signingSecret,omitempty"`
	RejectInvalidSignatures *bool    `json:"rejectInvalidSignatures,omitempty"`
	RateLimitPerMinute      *int     `json:"rateLimitPerMinute,omitempty"`
	IPFilterMode            *string  `json:"ipFilterMode,omitempty"`
	IPAllowlist             []string `json:"ipAllowlist,omitempty"`
	IPDenylist              []string `json:"ipDenylist,omitempty"`
	EncryptFields           []string `json:"encryptFields,omitempty"`
	MaskFields              []string `json:"maskFields,omitempty"`
	DedupEnabled            *bool    `json:"dedupEnabled,omitempty"`
	DedupStrategy           *string  `json:"dedupStrategy,omitempty"`
	DedupWindowHours        *int     `json:"dedupWindowHours,omitempty"`
	DedupCustomHeader       *string  `json:"dedupCustomHeader,omitempty"`
	TransientMode           *bool    `json:"transientMode,omitempty"`
	AllowedMethods          []string `json:"allowedMethods,omitempty"`
}

type UpdateSourceRequest struct {
	Name                    *string  `json:"name,omitempty"`
	Description             *string  `json:"description,omitempty"`
	RejectInvalidSignatures *bool    `json:"rejectInvalidSignatures,omitempty"`
	RateLimitPerMinute      *int     `json:"rateLimitPerMinute,omitempty"`
	IsActive                *bool    `json:"isActive,omitempty"`
	IPFilterMode            *string  `json:"ipFilterMode,omitempty"`
	IPAllowlist             []string `json:"ipAllowlist,omitempty"`
	IPDenylist              []string `json:"ipDenylist,omitempty"`
	EncryptFields           []string `json:"encryptFields,omitempty"`
	MaskFields              []string `json:"maskFields,omitempty"`
	DedupEnabled            *bool    `json:"dedupEnabled,omitempty"`
	DedupStrategy           *string  `json:"dedupStrategy,omitempty"`
	DedupWindowHours        *int     `json:"dedupWindowHours,omitempty"`
	DedupCustomHeader       *string  `json:"dedupCustomHeader,omitempty"`
	TransientMode           *bool    `json:"transientMode,omitempty"`
	AllowedMethods          []string `json:"allowedMethods,omitempty"`
}

type sourceResponse struct {
	Source Source `json:"source"`
}

type revealSecretResponse struct {
	SigningSecret string `json:"signingSecret"`
}

func (c *Client) CreateSource(ctx context.Context, req CreateSourceRequest) (*Source, error) {
	var resp sourceResponse
	if err := c.Post(ctx, "/sources", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Source, nil
}

func (c *Client) GetSource(ctx context.Context, id string) (*Source, error) {
	var resp sourceResponse
	if err := c.Get(ctx, fmt.Sprintf("/sources/%s", id), &resp); err != nil {
		return nil, err
	}
	return &resp.Source, nil
}

func (c *Client) UpdateSource(ctx context.Context, id string, req UpdateSourceRequest) (*Source, error) {
	var resp sourceResponse
	if err := c.Patch(ctx, fmt.Sprintf("/sources/%s", id), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Source, nil
}

func (c *Client) DeleteSource(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/sources/%s", id))
}

func (c *Client) RevealSourceSecret(ctx context.Context, id string) (string, error) {
	var resp revealSecretResponse
	if err := c.Get(ctx, fmt.Sprintf("/sources/%s/reveal-secret", id), &resp); err != nil {
		return "", err
	}
	return resp.SigningSecret, nil
}
