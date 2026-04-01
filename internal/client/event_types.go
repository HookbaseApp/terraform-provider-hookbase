package client

import (
	"context"
	"fmt"
)

type EventType struct {
	ID                string  `json:"id"`
	OrganizationID    string  `json:"organizationId"`
	Name              string  `json:"name"`
	DisplayName       string  `json:"displayName"`
	Description       *string `json:"description"`
	Category          *string `json:"category"`
	Schema            *string `json:"schema"`
	SchemaVersion     int     `json:"schemaVersion"`
	ExamplePayload    *string `json:"examplePayload"`
	DocumentationURL  *string `json:"documentationUrl"`
	DefaultPriority   int     `json:"defaultPriority"`
	IsEnabled         bool    `json:"isEnabled"`
	IsDeprecated      bool    `json:"isDeprecated"`
	DeprecatedAt      *string `json:"deprecatedAt"`
	DeprecatedMessage *string `json:"deprecatedMessage"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
}

type CreateEventTypeRequest struct {
	Name             string  `json:"name"`
	DisplayName      *string `json:"displayName,omitempty"`
	Description      *string `json:"description,omitempty"`
	Category         *string `json:"category,omitempty"`
	Schema           *string `json:"schema,omitempty"`
	ExamplePayload   *string `json:"examplePayload,omitempty"`
	DocumentationURL *string `json:"documentationUrl,omitempty"`
	IsEnabled        *bool   `json:"isEnabled,omitempty"`
	DefaultPriority  *int    `json:"defaultPriority,omitempty"`
}

type UpdateEventTypeRequest struct {
	DisplayName      *string `json:"displayName,omitempty"`
	Description      *string `json:"description,omitempty"`
	Category         *string `json:"category,omitempty"`
	Schema           *string `json:"schema,omitempty"`
	ExamplePayload   *string `json:"examplePayload,omitempty"`
	DocumentationURL *string `json:"documentationUrl,omitempty"`
	IsEnabled        *bool   `json:"isEnabled,omitempty"`
	IsDeprecated     *bool   `json:"isDeprecated,omitempty"`
	DeprecatedMessage *string `json:"deprecatedMessage,omitempty"`
	DefaultPriority  *int    `json:"defaultPriority,omitempty"`
}

type eventTypeResponse struct {
	Data EventType `json:"data"`
}

func (c *Client) CreateEventType(ctx context.Context, req CreateEventTypeRequest) (*EventType, error) {
	var resp eventTypeResponse
	if err := c.Post(ctx, "/event-types", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (c *Client) GetEventType(ctx context.Context, id string) (*EventType, error) {
	var resp eventTypeResponse
	if err := c.Get(ctx, fmt.Sprintf("/event-types/%s", id), &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (c *Client) UpdateEventType(ctx context.Context, id string, req UpdateEventTypeRequest) (*EventType, error) {
	var resp eventTypeResponse
	if err := c.Patch(ctx, fmt.Sprintf("/event-types/%s", id), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

func (c *Client) DeleteEventType(ctx context.Context, id string) error {
	return c.Delete(ctx, fmt.Sprintf("/event-types/%s", id))
}
