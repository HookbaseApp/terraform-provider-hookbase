package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type Route struct {
	ID                           string          `json:"id"`
	OrganizationID               string          `json:"organizationId"`
	Name                         string          `json:"name"`
	SourceID                     string          `json:"sourceId"`
	DestinationID                string          `json:"destinationId"`
	FilterID                     *string         `json:"filterId"`
	FilterConditions             json.RawMessage `json:"filterConditions"`
	FilterLogic                  *string         `json:"filterLogic"`
	TransformID                  *string         `json:"transformId"`
	SchemaID                     *string         `json:"schemaId"`
	Priority                     int             `json:"priority"`
	IsActive                     bool            `json:"isActive"`
	NotifyOnFailure              bool            `json:"notifyOnFailure"`
	NotifyOnSuccess              bool            `json:"notifyOnSuccess"`
	NotifyOnRecovery             bool            `json:"notifyOnRecovery"`
	NotifyEmails                 EmailList       `json:"notifyEmails"`
	FailureThreshold             *int            `json:"failureThreshold"`
	FailoverDestinationIds       []string        `json:"failoverDestinationIds"`
	FailoverAfterAttempts        *int            `json:"failoverAfterAttempts"`
	ExpectedResponse             json.RawMessage `json:"expectedResponse"`
	CircuitCooldownSeconds       *int            `json:"circuitCooldownSeconds"`
	CircuitProbeSuccessThreshold *int            `json:"circuitProbeSuccessThreshold"`
	CircuitFailureThreshold      *int            `json:"circuitFailureThreshold"`
	// HoldOnCircuitOpen: deliveries wait while the route's circuit is open instead of failing
	// (API 0134). Nil from an API that predates it, which behaved as true.
	HoldOnCircuitOpen *bool  `json:"holdOnCircuitOpen"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

// EmailList is a route's notifyEmails as GET answers it. The API stores the addresses as a JSON
// array and answers them as one string joined with ", ". An API from before that change answers
// the column as it is stored: its JSON text inside a string ("[\"a@example.com\"]"), or the array
// itself. All of these decode to the addresses the way ParseEmailList reads them; null and ""
// decode to none. Requests still send the comma-separated string.
type EmailList []string

func (l *EmailList) UnmarshalJSON(b []byte) error {
	var s *string
	if err := json.Unmarshal(b, &s); err == nil {
		switch {
		case s == nil:
			*l = nil
		case strings.HasPrefix(strings.TrimSpace(*s), "["):
			var items []string
			if json.Unmarshal([]byte(*s), &items) == nil {
				*l = ParseEmailList(strings.Join(items, ","))
				return nil
			}
			*l = ParseEmailList(*s)
		default:
			*l = ParseEmailList(*s)
		}
		return nil
	}
	var items []string
	if err := json.Unmarshal(b, &items); err != nil {
		return fmt.Errorf("notifyEmails: want a string or a list of strings, got %s", b)
	}
	*l = ParseEmailList(strings.Join(items, ","))
	return nil
}

// ParseEmailList reads a comma-separated list of addresses the way the API does
// (normalizeNotifyEmails in api/src/utils/notify-emails.ts): each entry trimmed, empty entries
// dropped, and an address listed again in any letter case dropped, keeping the first spelling and
// the order. It returns nil when no address is left.
func ParseEmailList(s string) EmailList {
	var out EmailList
	seen := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		key := strings.ToLower(part)
		if part == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, part)
	}
	return out
}

type CreateRouteRequest struct {
	Name                         string          `json:"name"`
	SourceID                     string          `json:"sourceId"`
	DestinationID                string          `json:"destinationId"`
	FilterID                     *string         `json:"filterId,omitempty"`
	FilterConditions             json.RawMessage `json:"filterConditions,omitempty"`
	FilterLogic                  *string         `json:"filterLogic,omitempty"`
	TransformID                  *string         `json:"transformId,omitempty"`
	SchemaID                     *string         `json:"schemaId,omitempty"`
	Priority                     *int            `json:"priority,omitempty"`
	NotifyOnFailure              *bool           `json:"notifyOnFailure,omitempty"`
	NotifyOnSuccess              *bool           `json:"notifyOnSuccess,omitempty"`
	NotifyOnRecovery             *bool           `json:"notifyOnRecovery,omitempty"`
	NotifyEmails                 *string         `json:"notifyEmails,omitempty"`
	FailureThreshold             *int            `json:"failureThreshold,omitempty"`
	FailoverDestinationIds       []string        `json:"failoverDestinationIds,omitempty"`
	FailoverAfterAttempts        *int            `json:"failoverAfterAttempts,omitempty"`
	ExpectedResponse             json.RawMessage `json:"expectedResponse,omitempty"`
	CircuitCooldownSeconds       *int            `json:"circuitCooldownSeconds,omitempty"`
	CircuitProbeSuccessThreshold *int            `json:"circuitProbeSuccessThreshold,omitempty"`
	CircuitFailureThreshold      *int            `json:"circuitFailureThreshold,omitempty"`
	HoldOnCircuitOpen            *bool           `json:"holdOnCircuitOpen,omitempty"`
}

// UpdateRouteRequest is the PATCH body; a nil field is left out and keeps the stored value. The
// API clears filterId, transformId, schemaId and notifyEmails given "", filterConditions and
// failoverDestinationIds given [] (FailoverDestinationIds is a pointer so [] can be sent), and
// expectedResponse given null. The circuit_* settings are not taken here: see
// UpdateRouteCircuitConfig.
type UpdateRouteRequest struct {
	Name                   *string         `json:"name,omitempty"`
	SourceID               *string         `json:"sourceId,omitempty"`
	DestinationID          *string         `json:"destinationId,omitempty"`
	FilterID               *string         `json:"filterId,omitempty"`
	FilterConditions       json.RawMessage `json:"filterConditions,omitempty"`
	FilterLogic            *string         `json:"filterLogic,omitempty"`
	TransformID            *string         `json:"transformId,omitempty"`
	SchemaID               *string         `json:"schemaId,omitempty"`
	Priority               *int            `json:"priority,omitempty"`
	IsActive               *bool           `json:"isActive,omitempty"`
	NotifyOnFailure        *bool           `json:"notifyOnFailure,omitempty"`
	NotifyOnSuccess        *bool           `json:"notifyOnSuccess,omitempty"`
	NotifyOnRecovery       *bool           `json:"notifyOnRecovery,omitempty"`
	NotifyEmails           *string         `json:"notifyEmails,omitempty"`
	FailureThreshold       *int            `json:"failureThreshold,omitempty"`
	FailoverDestinationIds *[]string       `json:"failoverDestinationIds,omitempty"`
	FailoverAfterAttempts  *int            `json:"failoverAfterAttempts,omitempty"`
	ExpectedResponse       json.RawMessage `json:"expectedResponse,omitempty"`
	HoldOnCircuitOpen      *bool           `json:"holdOnCircuitOpen,omitempty"`
}

// RouteCircuitConfig is the body of PATCH /routes/:id/circuit-config. The API needs the
// circuit_breaker plan feature, checks the ranges (cooldown 30-3600 s, failure threshold 1-50,
// probe success threshold 1-10), and has no way to clear a value.
type RouteCircuitConfig struct {
	CircuitCooldownSeconds       *int `json:"circuitCooldownSeconds,omitempty"`
	CircuitProbeSuccessThreshold *int `json:"circuitProbeSuccessThreshold,omitempty"`
	CircuitFailureThreshold      *int `json:"circuitFailureThreshold,omitempty"`
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

// UpdateRoute sends the PATCH. The API answers {success: true} with no route, so callers read it
// back with GetRoute.
func (c *Client) UpdateRoute(ctx context.Context, id string, req UpdateRouteRequest) error {
	return c.Patch(ctx, fmt.Sprintf("/routes/%s", id), req, nil)
}

// UpdateRouteCircuitConfig changes a route's circuit breaker settings, which PATCH /routes/:id
// ignores.
func (c *Client) UpdateRouteCircuitConfig(ctx context.Context, id string, req RouteCircuitConfig) error {
	return c.Patch(ctx, fmt.Sprintf("/routes/%s/circuit-config", id), req, nil)
}

func (c *Client) DeleteRoute(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/routes/%s", id))
}
