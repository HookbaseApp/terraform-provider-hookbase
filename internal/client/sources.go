package client

import (
	"context"
	"fmt"
)

type Source struct {
	ID             string  `json:"id"`
	OrganizationID string  `json:"organizationId"`
	Name           string  `json:"name"`
	Slug           string  `json:"slug"`
	Provider       *string `json:"provider"`
	Description    *string `json:"description"`
	// SigningSecret is present only on the create reply (null when the provider takes no secret or
	// the vendor issues it after registration); GET redacts it — see RevealSourceSecret.
	SigningSecret      *string `json:"signingSecret,omitempty"`
	HasSigningSecret   bool    `json:"hasSigningSecret"`
	SigningSecretLast4 *string `json:"signingSecretLast4"`
	// VerifyToken is Meta's (WhatsApp) "Verify token", returned in full only on create; read it
	// later with GetSourceVerifyToken.
	VerifyToken *string `json:"verifyToken,omitempty"`
	// VerifiesWithoutSecret: the vendor signs with a key it publishes (wise, aws-sns), so no
	// secret is stored or needed.
	VerifiesWithoutSecret bool `json:"verifiesWithoutSecret"`
	// AwaitingSigningSecret: the vendor issues the secret only after the ingest URL is registered
	// (asana, notion) and it hasn't been set yet.
	AwaitingSigningSecret   bool     `json:"awaitingSigningSecret"`
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
	VerifyToken             *string  `json:"verifyToken,omitempty"`
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

// UpdateSourceRequest is the PATCH body. A nil field is omitted and leaves the stored value alone.
// The API treats JSON null as "clear" for its nullable fields, which a plain omitempty pointer
// can never send, so those fields are double pointers (and pointers to slices): a non-nil outer
// pointer to a nil value marshals as null. Use Set/Null to build them.
type UpdateSourceRequest struct {
	Name                    *string   `json:"name,omitempty"`
	Description             **string  `json:"description,omitempty"`
	Provider                **string  `json:"provider,omitempty"`
	SigningSecret           *string   `json:"signingSecret,omitempty"`
	VerifyToken             *string   `json:"verifyToken,omitempty"`
	RejectInvalidSignatures *bool     `json:"rejectInvalidSignatures,omitempty"`
	RateLimitPerMinute      **int     `json:"rateLimitPerMinute,omitempty"`
	IsActive                *bool     `json:"isActive,omitempty"`
	IPFilterMode            *string   `json:"ipFilterMode,omitempty"`
	IPAllowlist             *[]string `json:"ipAllowlist,omitempty"`
	IPDenylist              *[]string `json:"ipDenylist,omitempty"`
	EncryptFields           *[]string `json:"encryptFields,omitempty"`
	MaskFields              *[]string `json:"maskFields,omitempty"`
	DedupEnabled            *bool     `json:"dedupEnabled,omitempty"`
	DedupStrategy           *string   `json:"dedupStrategy,omitempty"`
	DedupWindowHours        *int      `json:"dedupWindowHours,omitempty"`
	DedupCustomHeader       **string  `json:"dedupCustomHeader,omitempty"`
	TransientMode           *bool     `json:"transientMode,omitempty"`
	AllowedMethods          *[]string `json:"allowedMethods,omitempty"`
}

// Set returns a nullable update field carrying v.
func Set[T any](v T) **T {
	p := &v
	return &p
}

// Null returns a nullable update field that sends JSON null, clearing the stored value.
func Null[T any]() **T {
	var p *T
	return &p
}

type sourceResponse struct {
	Source Source `json:"source"`
}

type revealSecretResponse struct {
	SigningSecret string `json:"signingSecret"`
}

type verifyTokenResponse struct {
	VerifyToken *string `json:"verifyToken"`
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

// GetSourceVerifyToken reads a WhatsApp (Meta) source's verify token. The API answers 400 for a
// provider that has none, so callers should only ask for sources whose provider uses one. An
// empty string means none is stored.
func (c *Client) GetSourceVerifyToken(ctx context.Context, id string) (string, error) {
	var resp verifyTokenResponse
	if err := c.Get(ctx, fmt.Sprintf("/sources/%s/verify-token", id), &resp); err != nil {
		return "", err
	}
	if resp.VerifyToken == nil {
		return "", nil
	}
	return *resp.VerifyToken, nil
}
