package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode"
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
		// Clamped before converting to a Duration, which would overflow.
		"9223372036854775807": maxRetryAfter,
		"9300000000":          maxRetryAfter,
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

// countingServer answers every request with status (and Retry-After, when set) and counts them.
func countingServer(t *testing.T, status int, retryAfterHeader string, delay time.Duration) (*httptest.Server, *int32) {
	t.Helper()
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n, 1)
		if delay > 0 {
			time.Sleep(delay)
		}
		if retryAfterHeader != "" {
			w.Header().Set("Retry-After", retryAfterHeader)
		}
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, `{"error":"boom"}`)
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

// A POST creates. After a 5xx or a timeout the server may have created the object, and a second
// POST would create a duplicate or fail on the taken slug, so it is sent once.
func TestPostIsNotResentAfterTheServerMayHaveActed(t *testing.T) {
	waitsOf(t)

	srv, n := countingServer(t, http.StatusServiceUnavailable, "1", 0)
	c := &Client{BaseURL: srv.URL, OrganizationID: "org-1", APIKey: "key", HTTPClient: srv.Client()}
	err := c.Post(context.Background(), "/routes", map[string]string{"name": "r"}, nil)
	if !strings.Contains(fmt.Sprint(err), "API error (503)") || atomic.LoadInt32(n) != 1 {
		t.Errorf("POST with a 503: err = %v after %d sends, want the 503 after 1", err, atomic.LoadInt32(n))
	}

	srv, n = countingServer(t, http.StatusInternalServerError, "", 0)
	c = &Client{BaseURL: srv.URL, OrganizationID: "org-1", APIKey: "key", HTTPClient: srv.Client()}
	if err := c.Post(context.Background(), "/routes", nil, nil); err == nil || atomic.LoadInt32(n) != 1 {
		t.Errorf("POST with a 500: err = %v after %d sends, want 1", err, atomic.LoadInt32(n))
	}

	srv, n = countingServer(t, http.StatusCreated, "", 300*time.Millisecond)
	timeoutClient := srv.Client()
	timeoutClient.Timeout = 50 * time.Millisecond
	c = &Client{BaseURL: srv.URL, OrganizationID: "org-1", APIKey: "key", HTTPClient: timeoutClient}
	if err := c.Post(context.Background(), "/routes", nil, nil); err == nil || atomic.LoadInt32(n) != 1 {
		t.Errorf("POST that timed out: err = %v after %d sends, want 1", err, atomic.LoadInt32(n))
	}
}

// A 429 comes from the API's limiter before any handler runs, so a POST is sent again after it.
func TestPostIsResentAfter429(t *testing.T) {
	waits := waitsOf(t)
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) == 1 {
			w.Header().Set("Retry-After", "4")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{"route":{"id":"rt_1"}}`)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, OrganizationID: "org-1", APIKey: "key", HTTPClient: srv.Client()}
	if err := c.Post(context.Background(), "/routes", nil, nil); err != nil {
		t.Fatalf("POST after a 429: %v", err)
	}
	if atomic.LoadInt32(&n) != 2 || len(*waits) != 1 || (*waits)[0] != 4*time.Second {
		t.Errorf("sends = %d, waits = %v, want 2 sends and one 4s wait", atomic.LoadInt32(&n), *waits)
	}
}

// A POST whose connection never opened never reached the server, so it is sent again.
func TestPostIsResentWhenTheConnectionNeverOpened(t *testing.T) {
	waits := waitsOf(t)
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing listens there now: the dial is refused

	c := &Client{BaseURL: url, OrganizationID: "org-1", APIKey: "key", HTTPClient: &http.Client{Timeout: time.Second}}
	err := c.Post(context.Background(), "/routes", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "executing request") {
		t.Fatalf("err = %v, want the dial failure", err)
	}
	if len(*waits) != maxAttempts-1 {
		t.Errorf("waited %d times, want %d (a retry per attempt)", len(*waits), maxAttempts-1)
	}
}

// GET, PATCH and DELETE leave the same result when sent twice, so a 503 is retried after the wait
// Retry-After asks for.
func TestGetIsRetriedAfter503WithRetryAfter(t *testing.T) {
	waits := waitsOf(t)
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) == 1 {
			w.Header().Set("Retry-After", "5")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, OrganizationID: "org-1", APIKey: "key", HTTPClient: srv.Client()}
	var got struct {
		OK bool `json:"ok"`
	}
	if err := c.Get(context.Background(), "/test", &got); err != nil || !got.OK {
		t.Fatalf("GET after a 503: ok=%v err=%v", got.OK, err)
	}
	if atomic.LoadInt32(&n) != 2 || len(*waits) != 1 || (*waits)[0] != 5*time.Second {
		t.Errorf("sends = %d, waits = %v, want 2 sends and one 5s wait", atomic.LoadInt32(&n), *waits)
	}

	srv2, n2 := countingServer(t, http.StatusBadGateway, "", 0)
	c = &Client{BaseURL: srv2.URL, OrganizationID: "org-1", APIKey: "key", HTTPClient: srv2.Client()}
	if err := c.Delete(context.Background(), "/routes/rt_1"); err == nil || atomic.LoadInt32(n2) != maxAttempts {
		t.Errorf("DELETE with a 502: err = %v after %d sends, want %d", err, atomic.LoadInt32(n2), maxAttempts)
	}
}

func TestIsNotFound(t *testing.T) {
	srv, _ := countingServer(t, http.StatusNotFound, "", 0)
	c := &Client{BaseURL: srv.URL, OrganizationID: "org-1", APIKey: "key", HTTPClient: srv.Client()}
	err := c.Get(context.Background(), "/routes/rt_1", nil)
	if !IsNotFound(err) || err.Error() != "API error (404): boom" {
		t.Errorf("err = %v, IsNotFound = %v", err, IsNotFound(err))
	}
	if IsNotFound(fmt.Errorf("wrapped: %w", &StatusError{StatusCode: 500})) || !IsNotFound(fmt.Errorf("wrapped: %w", err)) {
		t.Error("IsNotFound must look through wrapping and match only 404")
	}
}

// The API trims entries with JavaScript's trim(). These are the code points it strips, listed by
// running String.fromCodePoint(c).trim() === "" over every code point in Node 22.
var jsTrimmed = []rune{
	0x9, 0xa, 0xb, 0xc, 0xd, 0x20, 0xa0, 0x1680, 0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005,
	0x2006, 0x2007, 0x2008, 0x2009, 0x200a, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000, 0xfeff,
}

func TestEmailListTrimsLikeJavaScript(t *testing.T) {
	want := map[rune]bool{}
	for _, r := range jsTrimmed {
		want[r] = true
	}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if r >= 0xd800 && r <= 0xdfff {
			continue
		}
		if isJSTrimSpace(r) != want[r] {
			t.Errorf("isJSTrimSpace(%U) = %v, want %v", r, !want[r], want[r])
		}
	}

	for in, want := range map[string][]string{
		"\ufeffops@example.com, dev@example.com\ufeff":            {"ops@example.com", "dev@example.com"},
		"\u00a0ops@example.com\u3000,\u2028dev@example.com\u2029": {"ops@example.com", "dev@example.com"},
		// NEL is trimmed by Go's unicode.IsSpace but kept by trim(), so it stays (and the API
		// rejects the address). A zero-width space is not whitespace to either.
		"\u0085ops@example.com": {"\u0085ops@example.com"},
		"\u200bops@example.com": {"\u200bops@example.com"},
	} {
		if got := ParseEmailList(in); !slices.Equal(got, want) {
			t.Errorf("ParseEmailList(%q) = %q, want %q", in, got, want)
		}
	}
}
