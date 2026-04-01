package client

import (
	"context"
	"fmt"
	"net/url"
)

// Lookup-by-slug/name methods for data sources.
// These list resources with a page size of 1 and filter by slug/name,
// or paginate through to find a match when the API doesn't support filtering.

type paginatedSourceResponse struct {
	Sources []Source `json:"sources"`
}

func (c *Client) GetSourceBySlug(ctx context.Context, slug string) (*Source, error) {
	var resp paginatedSourceResponse
	if err := c.Get(ctx, fmt.Sprintf("/sources?pageSize=100"), &resp); err != nil {
		return nil, err
	}
	for _, s := range resp.Sources {
		if s.Slug == slug {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("source with slug %q not found", slug)
}

type paginatedDestinationResponse struct {
	Destinations []Destination `json:"destinations"`
}

func (c *Client) GetDestinationBySlug(ctx context.Context, slug string) (*Destination, error) {
	var resp paginatedDestinationResponse
	if err := c.Get(ctx, fmt.Sprintf("/destinations?pageSize=100"), &resp); err != nil {
		return nil, err
	}
	for _, d := range resp.Destinations {
		if d.Slug == slug {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("destination with slug %q not found", slug)
}

type paginatedTransformResponse struct {
	Transforms []Transform `json:"transforms"`
}

func (c *Client) GetTransformBySlug(ctx context.Context, slug string) (*Transform, error) {
	var resp paginatedTransformResponse
	if err := c.Get(ctx, fmt.Sprintf("/transforms?pageSize=100"), &resp); err != nil {
		return nil, err
	}
	for _, t := range resp.Transforms {
		if t.Slug == slug {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("transform with slug %q not found", slug)
}

type paginatedFilterResponse struct {
	Filters []Filter `json:"filters"`
}

func (c *Client) GetFilterBySlug(ctx context.Context, slug string) (*Filter, error) {
	var resp paginatedFilterResponse
	if err := c.Get(ctx, fmt.Sprintf("/filters?pageSize=100"), &resp); err != nil {
		return nil, err
	}
	for _, f := range resp.Filters {
		if f.Slug == slug {
			return &f, nil
		}
	}
	return nil, fmt.Errorf("filter with slug %q not found", slug)
}

// Outbound lookups

func (c *Client) GetWebhookApplicationByExternalID(ctx context.Context, externalID string) (*WebhookApplication, error) {
	var resp webhookApplicationResponse
	if err := c.Get(ctx, fmt.Sprintf("/webhook-applications/by-external-id/%s", url.PathEscape(externalID)), &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

type paginatedEventTypeResponse struct {
	Data []EventType `json:"data"`
}

func (c *Client) GetEventTypeByName(ctx context.Context, name string) (*EventType, error) {
	var resp paginatedEventTypeResponse
	if err := c.Get(ctx, fmt.Sprintf("/event-types?search=%s&limit=100", url.QueryEscape(name)), &resp); err != nil {
		return nil, err
	}
	for _, et := range resp.Data {
		if et.Name == name {
			return &et, nil
		}
	}
	return nil, fmt.Errorf("event type with name %q not found", name)
}
