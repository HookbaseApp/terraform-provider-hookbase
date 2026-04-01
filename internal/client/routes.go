package client

import (
	"context"
	"encoding/json"
	"fmt"
)

type Route struct {
	ID                          string          `json:"id"`
	OrganizationID              string          `json:"organizationId"`
	Name                        string          `json:"name"`
	SourceID                    string          `json:"sourceId"`
	DestinationID               string          `json:"destinationId"`
	FilterID                    *string         `json:"filterId"`
	FilterConditions            json.RawMessage `json:"filterConditions"`
	FilterLogic                 *string         `json:"filterLogic"`
	TransformID                 *string         `json:"transformId"`
	SchemaID                    *string         `json:"schemaId"`
	Priority                    int             `json:"priority"`
	IsActive                    bool            `json:"isActive"`
	NotifyOnFailure             bool            `json:"notifyOnFailure"`
	NotifyOnSuccess             bool            `json:"notifyOnSuccess"`
	NotifyOnRecovery            bool            `json:"notifyOnRecovery"`
	NotifyEmails                *string         `json:"notifyEmails"`
	FailureThreshold            *int            `json:"failureThreshold"`
	FailoverDestinationIds      []string        `json:"failoverDestinationIds"`
	FailoverAfterAttempts       *int            `json:"failoverAfterAttempts"`
	ExpectedResponse            json.RawMessage `json:"expectedResponse"`
	CircuitCooldownSeconds      *int            `json:"circuitCooldownSeconds"`
	CircuitProbeSuccessThreshold *int           `json:"circuitProbeSuccessThreshold"`
	CircuitFailureThreshold     *int            `json:"circuitFailureThreshold"`
	CreatedAt                   string          `json:"createdAt"`
	UpdatedAt                   string          `json:"updatedAt"`
}

type CreateRouteRequest struct {
	Name                        string          `json:"name"`
	SourceID                    string          `json:"sourceId"`
	DestinationID               string          `json:"destinationId"`
	FilterID                    *string         `json:"filterId,omitempty"`
	FilterConditions            json.RawMessage `json:"filterConditions,omitempty"`
	FilterLogic                 *string         `json:"filterLogic,omitempty"`
	TransformID                 *string         `json:"transformId,omitempty"`
	SchemaID                    *string         `json:"schemaId,omitempty"`
	Priority                    *int            `json:"priority,omitempty"`
	NotifyOnFailure             *bool           `json:"notifyOnFailure,omitempty"`
	NotifyOnSuccess             *bool           `json:"notifyOnSuccess,omitempty"`
	NotifyOnRecovery            *bool           `json:"notifyOnRecovery,omitempty"`
	NotifyEmails                *string         `json:"notifyEmails,omitempty"`
	FailureThreshold            *int            `json:"failureThreshold,omitempty"`
	FailoverDestinationIds      []string        `json:"failoverDestinationIds,omitempty"`
	FailoverAfterAttempts       *int            `json:"failoverAfterAttempts,omitempty"`
	ExpectedResponse            json.RawMessage `json:"expectedResponse,omitempty"`
	CircuitCooldownSeconds      *int            `json:"circuitCooldownSeconds,omitempty"`
	CircuitProbeSuccessThreshold *int           `json:"circuitProbeSuccessThreshold,omitempty"`
	CircuitFailureThreshold     *int            `json:"circuitFailureThreshold,omitempty"`
}

type UpdateRouteRequest struct {
	Name                        *string         `json:"name,omitempty"`
	SourceID                    *string         `json:"sourceId,omitempty"`
	DestinationID               *string         `json:"destinationId,omitempty"`
	FilterID                    *string         `json:"filterId,omitempty"`
	FilterConditions            json.RawMessage `json:"filterConditions,omitempty"`
	FilterLogic                 *string         `json:"filterLogic,omitempty"`
	TransformID                 *string         `json:"transformId,omitempty"`
	SchemaID                    *string         `json:"schemaId,omitempty"`
	Priority                    *int            `json:"priority,omitempty"`
	IsActive                    *bool           `json:"isActive,omitempty"`
	NotifyOnFailure             *bool           `json:"notifyOnFailure,omitempty"`
	NotifyOnSuccess             *bool           `json:"notifyOnSuccess,omitempty"`
	NotifyOnRecovery            *bool           `json:"notifyOnRecovery,omitempty"`
	NotifyEmails                *string         `json:"notifyEmails,omitempty"`
	FailureThreshold            *int            `json:"failureThreshold,omitempty"`
	FailoverDestinationIds      []string        `json:"failoverDestinationIds,omitempty"`
	FailoverAfterAttempts       *int            `json:"failoverAfterAttempts,omitempty"`
	ExpectedResponse            json.RawMessage `json:"expectedResponse,omitempty"`
	CircuitCooldownSeconds      *int            `json:"circuitCooldownSeconds,omitempty"`
	CircuitProbeSuccessThreshold *int           `json:"circuitProbeSuccessThreshold,omitempty"`
	CircuitFailureThreshold     *int            `json:"circuitFailureThreshold,omitempty"`
}

type routeResponse struct {
	Route Route `json:"route"`
}

func (c *Client) CreateRoute(ctx context.Context, req CreateRouteRequest) (*Route, error) {
	var resp routeResponse
	if err := c.Post(ctx, "/routes", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Route, nil
}

func (c *Client) GetRoute(ctx context.Context, id string) (*Route, error) {
	var resp routeResponse
	if err := c.Get(ctx, fmt.Sprintf("/routes/%s", id), &resp); err != nil {
		return nil, err
	}
	return &resp.Route, nil
}

func (c *Client) UpdateRoute(ctx context.Context, id string, req UpdateRouteRequest) (*Route, error) {
	var resp routeResponse
	if err := c.Patch(ctx, fmt.Sprintf("/routes/%s", id), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Route, nil
}

func (c *Client) DeleteRoute(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/routes/%s", id))
}
