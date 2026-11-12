package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Lookup-by-slug/name methods for data sources. The API has no filter by slug on its list
// endpoints, so each lookup walks the list a page at a time until it finds a match.

// lookupPageSize is the largest page the list endpoints serve (they clamp pageSize to 100).
const lookupPageSize = 100

// maxLookupPages bounds a walk (100,000 items at the full page size), so a server that ignores
// the page parameter can't keep a lookup going forever.
const maxLookupPages = 1000

// listPagination is the pagination object of the page/pageSize list endpoints (sources,
// destinations, transforms, filters, routes).
type listPagination struct {
	Total    *int `json:"total"`
	Page     int  `json:"page"`
	PageSize int  `json:"pageSize"`
}

// findInPages walks GET path?page=N&pageSize=100, reading the items under key, and returns the
// first one match accepts, or nil once the list is exhausted. The list ends at pagination.total,
// or at a page shorter than the page size the server reports (or, without one, than requested),
// which also covers a server that sends no pagination object.
func findInPages[T any](ctx context.Context, c *Client, path, key string, match func(*T) bool) (*T, error) {
	seen := 0
	for page := 1; page <= maxLookupPages; page++ {
		var resp map[string]json.RawMessage
		if err := c.Get(ctx, fmt.Sprintf("%s?page=%d&pageSize=%d", path, page, lookupPageSize), &resp); err != nil {
			return nil, err
		}

		var items []T
		if raw := resp[key]; len(raw) > 0 {
			if err := json.Unmarshal(raw, &items); err != nil {
				return nil, fmt.Errorf("unmarshaling %s: %w", key, err)
			}
		}
		for i := range items {
			if match(&items[i]) {
				return &items[i], nil
			}
		}
		seen += len(items)

		var p listPagination
		if raw := resp["pagination"]; len(raw) > 0 {
			// A malformed pagination object is treated as absent: the short-page rule still ends the walk.
			_ = json.Unmarshal(raw, &p)
		}
		pageSize := lookupPageSize
		if p.PageSize > 0 {
			pageSize = p.PageSize
		}
		if len(items) == 0 || len(items) < pageSize || (p.Total != nil && seen >= *p.Total) {
			return nil, nil
		}
	}
	return nil, nil
}

func (c *Client) GetSourceBySlug(ctx context.Context, slug string) (*Source, error) {
	s, err := findInPages(ctx, c, "/sources", "sources", func(s *Source) bool { return s.Slug == slug })
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("source with slug %q not found", slug)
	}
	return s, nil
}

func (c *Client) GetDestinationBySlug(ctx context.Context, slug string) (*Destination, error) {
	d, err := findInPages(ctx, c, "/destinations", "destinations", func(d *Destination) bool { return d.Slug == slug })
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, fmt.Errorf("destination with slug %q not found", slug)
	}
	return d, nil
}

func (c *Client) GetTransformBySlug(ctx context.Context, slug string) (*Transform, error) {
	t, err := findInPages(ctx, c, "/transforms", "transforms", func(t *Transform) bool { return t.Slug == slug })
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("transform with slug %q not found", slug)
	}
	return t, nil
}

func (c *Client) GetFilterBySlug(ctx context.Context, slug string) (*Filter, error) {
	f, err := findInPages(ctx, c, "/filters", "filters", func(f *Filter) bool { return f.Slug == slug })
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("filter with slug %q not found", slug)
	}
	return f, nil
}

// Outbound lookups

func (c *Client) GetWebhookApplicationByExternalID(ctx context.Context, externalID string) (*WebhookApplication, error) {
	var resp webhookApplicationResponse
	if err := c.Get(ctx, fmt.Sprintf("/webhook-applications/by-external-id/%s", url.PathEscape(externalID)), &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

type cursorPagination struct {
	HasMore    bool    `json:"hasMore"`
	NextCursor *string `json:"nextCursor"`
}

type paginatedEventTypeResponse struct {
	Data       []EventType      `json:"data"`
	Pagination cursorPagination `json:"pagination"`
}

// GetEventTypeByName finds an event type by its exact name. The list's search is a substring
// match ("order" also finds "order.created"), so the walk follows nextCursor until a name matches
// exactly or the list runs out.
func (c *Client) GetEventTypeByName(ctx context.Context, name string) (*EventType, error) {
	cursor := ""
	for page := 0; page < maxLookupPages; page++ {
		path := fmt.Sprintf("/event-types?search=%s&limit=%d", url.QueryEscape(name), lookupPageSize)
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		var resp paginatedEventTypeResponse
		if err := c.Get(ctx, path, &resp); err != nil {
			return nil, err
		}
		for i := range resp.Data {
			if resp.Data[i].Name == name {
				return &resp.Data[i], nil
			}
		}
		next := resp.Pagination.NextCursor
		if !resp.Pagination.HasMore || next == nil || *next == "" || *next == cursor {
			break
		}
		cursor = *next
	}
	return nil, fmt.Errorf("event type with name %q not found", name)
}
