package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// pagedListServer serves GET <path> like the API's page/pageSize list endpoints: pageSize is
// clamped to 1-100 (default 20), items are numbered <prefix>-1..<prefix>-total, and the reply
// carries pagination {total, page, pageSize} unless omitPagination is set.
type pagedListServer struct {
	t              *testing.T
	path           string // e.g. "/sources"
	key            string // the reply's list key, e.g. "sources"
	prefix         string
	total          int
	omitPagination bool
	requests       []string
}

func (s *pagedListServer) start() *Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requests = append(s.requests, r.URL.RequestURI())
		if r.URL.Path != "/api/organizations/org-1"+s.path {
			s.t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if pageSize == 0 {
			pageSize = 20
		}
		pageSize = max(1, min(100, pageSize))

		items := []string{}
		for i := (page-1)*pageSize + 1; i <= min(page*pageSize, s.total); i++ {
			id := fmt.Sprintf("%s-%d", s.prefix, i)
			items = append(items, fmt.Sprintf(`{"id":%q,"name":%q,"slug":%q}`, id, id, id))
		}
		body := fmt.Sprintf(`{%q:[%s]`, s.key, strings.Join(items, ","))
		if !s.omitPagination {
			body += fmt.Sprintf(`,"pagination":{"total":%d,"page":%d,"pageSize":%d}`, s.total, page, pageSize)
		}
		body += "}"
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	s.t.Cleanup(srv.Close)
	return &Client{BaseURL: srv.URL, OrganizationID: "org-1", APIKey: "key", HTTPClient: srv.Client()}
}

// Each slug lookup must find a resource that sits past the first page of 100, which the
// single-page lookups could not.
func TestSlugLookupsPagePastFirst100(t *testing.T) {
	lookups := []struct {
		path, key string
		get       func(c *Client, slug string) (string, error)
	}{
		{"/sources", "sources", func(c *Client, slug string) (string, error) {
			s, err := c.GetSourceBySlug(context.Background(), slug)
			if err != nil {
				return "", err
			}
			return s.ID, nil
		}},
		{"/destinations", "destinations", func(c *Client, slug string) (string, error) {
			d, err := c.GetDestinationBySlug(context.Background(), slug)
			if err != nil {
				return "", err
			}
			return d.ID, nil
		}},
		{"/transforms", "transforms", func(c *Client, slug string) (string, error) {
			tr, err := c.GetTransformBySlug(context.Background(), slug)
			if err != nil {
				return "", err
			}
			return tr.ID, nil
		}},
		{"/filters", "filters", func(c *Client, slug string) (string, error) {
			f, err := c.GetFilterBySlug(context.Background(), slug)
			if err != nil {
				return "", err
			}
			return f.ID, nil
		}},
	}

	for _, l := range lookups {
		t.Run(l.key, func(t *testing.T) {
			t.Run("found on page 3", func(t *testing.T) {
				s := &pagedListServer{t: t, path: l.path, key: l.key, prefix: "x", total: 250}
				c := s.start()
				id, err := l.get(c, "x-237")
				if err != nil {
					t.Fatalf("lookup: %v (requests %v)", err, s.requests)
				}
				if id != "x-237" {
					t.Errorf("id = %q, want x-237", id)
				}
				if len(s.requests) != 3 {
					t.Errorf("requests = %v, want 3 pages", s.requests)
				}
			})

			t.Run("not found stops at total", func(t *testing.T) {
				s := &pagedListServer{t: t, path: l.path, key: l.key, prefix: "x", total: 200}
				c := s.start()
				if _, err := l.get(c, "missing"); err == nil || !strings.Contains(err.Error(), "not found") {
					t.Fatalf("err = %v, want not found", err)
				}
				// 200 items fill exactly two pages; total says there is no third to ask for.
				if len(s.requests) != 2 {
					t.Errorf("requests = %v, want 2", s.requests)
				}
			})

			t.Run("short page ends the walk without pagination", func(t *testing.T) {
				s := &pagedListServer{t: t, path: l.path, key: l.key, prefix: "x", total: 150, omitPagination: true}
				c := s.start()
				id, err := l.get(c, "x-150")
				if err != nil || id != "x-150" {
					t.Fatalf("id = %q, err = %v", id, err)
				}
				if _, err := l.get(c, "missing"); err == nil {
					t.Fatal("found a slug that does not exist")
				}
				if len(s.requests) != 4 {
					t.Errorf("requests = %v, want 2 per lookup", s.requests)
				}
			})
		})
	}
}

// A server that ignores page and always answers a full page must not be walked forever.
func TestSlugLookupIsBounded(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		items := make([]string, 100)
		for i := range items {
			items[i] = fmt.Sprintf(`{"id":"s-%d","slug":"s-%d"}`, i, i)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sources":[` + strings.Join(items, ",") + `]}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{BaseURL: srv.URL, OrganizationID: "org-1", APIKey: "key", HTTPClient: srv.Client()}

	if _, err := c.GetSourceBySlug(context.Background(), "missing"); err == nil {
		t.Fatal("found a slug that does not exist")
	}
	if calls > maxLookupPages {
		t.Errorf("calls = %d, want at most %d", calls, maxLookupPages)
	}
}

// Event types are listed by cursor: the lookup follows nextCursor while hasMore is set.
func TestGetEventTypeByNameFollowsCursor(t *testing.T) {
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RequestURI())
		if got := r.URL.Query().Get("search"); got != "order.created" {
			t.Errorf("search = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("cursor") {
		case "":
			_, _ = w.Write([]byte(`{"data":[{"id":"et-1","name":"order.created.v1"}],"pagination":{"hasMore":true,"nextCursor":"c 2"}}`))
		case "c 2":
			_, _ = w.Write([]byte(`{"data":[{"id":"et-2","name":"order.created"}],"pagination":{"hasMore":false,"nextCursor":null}}`))
		default:
			t.Errorf("unexpected cursor %q", r.URL.Query().Get("cursor"))
		}
	}))
	t.Cleanup(srv.Close)
	c := &Client{BaseURL: srv.URL, OrganizationID: "org-1", APIKey: "key", HTTPClient: srv.Client()}

	et, err := c.GetEventTypeByName(context.Background(), "order.created")
	if err != nil {
		t.Fatalf("GetEventTypeByName: %v (requests %v)", err, requests)
	}
	if et.ID != "et-2" {
		t.Errorf("id = %q, want et-2", et.ID)
	}
	if len(requests) != 2 {
		t.Errorf("requests = %v, want 2", requests)
	}
}
