package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	c := New("https://api.hookbase.app", "org-123", "whr_testkey")

	if c.BaseURL != "https://api.hookbase.app" {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, "https://api.hookbase.app")
	}
	if c.OrganizationID != "org-123" {
		t.Errorf("OrganizationID = %q, want %q", c.OrganizationID, "org-123")
	}
	if c.APIKey != "whr_testkey" {
		t.Errorf("APIKey = %q, want %q", c.APIKey, "whr_testkey")
	}
	if c.HTTPClient == nil {
		t.Fatal("HTTPClient is nil")
	}
	if c.HTTPClient.Timeout != 30*time.Second {
		t.Errorf("HTTPClient.Timeout = %v, want %v", c.HTTPClient.Timeout, 30*time.Second)
	}
}

func TestSuccessfulGet(t *testing.T) {
	type payload struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(payload{ID: "src_1", Name: "My Source"})
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "key",
		HTTPClient:     srv.Client(),
	}

	var got payload
	err := c.Get(context.Background(), "/sources", &got)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.ID != "src_1" {
		t.Errorf("ID = %q, want %q", got.ID, "src_1")
	}
	if got.Name != "My Source" {
		t.Errorf("Name = %q, want %q", got.Name, "My Source")
	}
}

func TestSuccessfulPost(t *testing.T) {
	type reqBody struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	type respBody struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	var receivedBody reqBody
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(respBody{ID: "src_new", Name: "Created"})
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "key",
		HTTPClient:     srv.Client(),
	}

	var got respBody
	err := c.Post(context.Background(), "/sources", reqBody{Name: "Test", Slug: "test"}, &got)
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}
	if got.ID != "src_new" {
		t.Errorf("response ID = %q, want %q", got.ID, "src_new")
	}
	if receivedBody.Name != "Test" {
		t.Errorf("request body Name = %q, want %q", receivedBody.Name, "Test")
	}
	if receivedBody.Slug != "test" {
		t.Errorf("request body Slug = %q, want %q", receivedBody.Slug, "test")
	}
}

func TestPatchRequest(t *testing.T) {
	type reqBody struct {
		Name string `json:"name"`
	}
	type respBody struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	var gotMethod string
	var receivedBody reqBody
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(respBody{ID: "src_1", Name: "Updated"})
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "key",
		HTTPClient:     srv.Client(),
	}

	var got respBody
	err := c.Patch(context.Background(), "/sources/src_1", reqBody{Name: "Updated"}, &got)
	if err != nil {
		t.Fatalf("Patch returned error: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
	if receivedBody.Name != "Updated" {
		t.Errorf("request body Name = %q, want %q", receivedBody.Name, "Updated")
	}
	if got.Name != "Updated" {
		t.Errorf("response Name = %q, want %q", got.Name, "Updated")
	}
}

func TestDeleteRequest(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "key",
		HTTPClient:     srv.Client(),
	}

	err := c.Delete(context.Background(), "/sources/src_1")
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

func TestAPIErrorHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(APIError{Error: "Bad Request"})
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "key",
		HTTPClient:     srv.Client(),
	}

	err := c.Get(context.Background(), "/sources", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error %q should contain status code 400", err.Error())
	}
	if !strings.Contains(err.Error(), "Bad Request") {
		t.Errorf("error %q should contain 'Bad Request'", err.Error())
	}
}

func TestNotFoundHandling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(APIError{Error: "Not Found"})
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "key",
		HTTPClient:     srv.Client(),
	}

	err := c.Get(context.Background(), "/sources/nonexistent", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error %q should contain status code 404", err.Error())
	}
}

func TestRetryOn429(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "key",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	type result struct {
		OK bool `json:"ok"`
	}
	var got result
	err := c.Get(context.Background(), "/test", &got)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !got.OK {
		t.Error("expected OK=true after retry")
	}
	if atomic.LoadInt32(&attempts) < 2 {
		t.Errorf("expected at least 2 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestRetryOn500(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "key",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	type result struct {
		OK bool `json:"ok"`
	}
	var got result
	err := c.Get(context.Background(), "/test", &got)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !got.OK {
		t.Error("expected OK=true after retry")
	}
	if atomic.LoadInt32(&attempts) < 2 {
		t.Errorf("expected at least 2 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "whr_secretkey",
		HTTPClient:     srv.Client(),
	}

	_ = c.Get(context.Background(), "/test", nil)

	expected := "Bearer whr_secretkey"
	if gotAuth != expected {
		t.Errorf("Authorization header = %q, want %q", gotAuth, expected)
	}
}

func TestOrgURLConstruction(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-abc-123",
		APIKey:         "key",
		HTTPClient:     srv.Client(),
	}

	_ = c.Get(context.Background(), "/sources/src_1", nil)

	expected := "/api/organizations/org-abc-123/sources/src_1"
	if gotPath != expected {
		t.Errorf("request path = %q, want %q", gotPath, expected)
	}
}

// A route's notifyEmails decodes from the API's joined string and from the stored JSON array an
// older API answers, trimmed and with empty entries dropped.
func TestEmailListDecodes(t *testing.T) {
	for raw, want := range map[string][]string{
		`null`:                               nil,
		`""`:                                 nil,
		`" , "`:                              nil,
		`"a@example.com, b@example.com"`:     {"a@example.com", "b@example.com"},
		`"a@example.com,,b@example.com,"`:    {"a@example.com", "b@example.com"},
		`["a@example.com"," b@example.com"]`: {"a@example.com", "b@example.com"},
		// Before the fix, GET answered the JSONB column as its text.
		`"[\"a@example.com\", \"b@example.com\"]"`: {"a@example.com", "b@example.com"},
		// Repeats in any letter case count once, first spelling kept, as the API stores them.
		`"A@example.com, b@example.com, a@EXAMPLE.com"`: {"A@example.com", "b@example.com"},
		`[]`: nil,
	} {
		var r Route
		if err := json.Unmarshal([]byte(`{"notifyEmails":`+raw+`}`), &r); err != nil {
			t.Errorf("%s: %v", raw, err)
			continue
		}
		if fmt.Sprint([]string(r.NotifyEmails)) != fmt.Sprint(want) || len(r.NotifyEmails) != len(want) {
			t.Errorf("%s decoded to %q, want %q", raw, r.NotifyEmails, want)
		}
	}
	var r Route
	if err := json.Unmarshal([]byte(`{"notifyEmails":42}`), &r); err == nil {
		t.Error("a number decoded without error")
	}
}

// The API's global limiter answers 429 with Retry-After until its one-minute window ends. The
// client waits as asked and tries more often than for other errors, instead of giving up after
// about three seconds.
func TestRetryOn429HonorsRetryAfter(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) <= 3 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = fmt.Fprint(w, `{"error":"Rate limit exceeded. Please slow down."}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, OrganizationID: "org-1", APIKey: "key", HTTPClient: srv.Client()}

	var got struct {
		OK bool `json:"ok"`
	}
	start := time.Now()
	if err := c.Get(context.Background(), "/test", &got); err != nil {
		t.Fatalf("Get after three 429s: %v", err)
	}
	if !got.OK || atomic.LoadInt32(&attempts) != 4 {
		t.Errorf("ok=%v after %d attempts, want true after 4", got.OK, atomic.LoadInt32(&attempts))
	}
	if elapsed := time.Since(start); elapsed < 3*time.Second {
		t.Errorf("retried after %v, want Retry-After's 1s three times", elapsed)
	}
}

// waitsOf replaces the client's wait with one that records the durations and returns at once.
func waitsOf(t *testing.T) *[]time.Duration {
	t.Helper()
	var waits []time.Duration
	orig := wait
	wait = func(ctx context.Context, d time.Duration) error {
		waits = append(waits, d)
		return ctx.Err()
	}
	t.Cleanup(func() { wait = orig })
	return &waits
}

func TestRetryOn429GivesUpAndCapsTheWait(t *testing.T) {
	waits := waitsOf(t)
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = fmt.Fprint(w, `{"error":"Rate limit exceeded. Please slow down."}`)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, OrganizationID: "org-1", APIKey: "key", HTTPClient: srv.Client()}

	err := c.Get(context.Background(), "/test", nil)
	if err == nil || !strings.Contains(err.Error(), "API error (429): Rate limit exceeded") {
		t.Fatalf("err = %v, want the API's 429", err)
	}
	if n := atomic.LoadInt32(&attempts); n != maxRateLimitAttempts {
		t.Errorf("attempts = %d, want %d", n, maxRateLimitAttempts)
	}
	for _, d := range *waits {
		if d != maxRetryAfter {
			t.Errorf("waited %v, want Retry-After capped at %v", d, maxRetryAfter)
		}
	}
	if len(*waits) != maxRateLimitAttempts-1 {
		t.Errorf("waited %d times, want %d", len(*waits), maxRateLimitAttempts-1)
	}
}

func TestRetryAfterHeader(t *testing.T) {
	fallback := 2 * time.Second
	future := time.Now().Add(30 * time.Second).UTC().Format(http.TimeFormat)
	for value, want := range map[string]time.Duration{
		"":      fallback,
		"7":     7 * time.Second,
		"0":     0,
		"3600":  maxRetryAfter,
		"-5":    fallback,
		"soon":  fallback,
		future:  30 * time.Second,
		"  12 ": 12 * time.Second,
	} {
		h := http.Header{}
		if value != "" {
			h.Set("Retry-After", value)
		}
		got := retryAfter(h, fallback)
		if value == future {
			// An HTTP date has one-second resolution.
			if got < 28*time.Second || got > 30*time.Second {
				t.Errorf("Retry-After %q = %v, want about 30s", value, got)
			}
			continue
		}
		if got != want {
			t.Errorf("Retry-After %q = %v, want %v", value, got, want)
		}
	}
}

// A cancelled operation stops waiting for the rate limit window instead of sleeping it out.
func TestRetryWaitStopsOnCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, OrganizationID: "org-1", APIKey: "key", HTTPClient: srv.Client()}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := c.Get(ctx, "/test", nil)
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("err = %v, want the context's", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("returned after %v, want soon after the cancel", elapsed)
	}
}
