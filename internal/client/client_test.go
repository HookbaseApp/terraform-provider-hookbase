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
		json.NewEncoder(w).Encode(payload{ID: "src_1", Name: "My Source"})
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
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(respBody{ID: "src_new", Name: "Created"})
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
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respBody{ID: "src_1", Name: "Updated"})
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
		json.NewEncoder(w).Encode(APIError{Error: "Bad Request"})
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
		json.NewEncoder(w).Encode(APIError{Error: "Not Found"})
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
		fmt.Fprint(w, `{"ok":true}`)
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
		fmt.Fprint(w, `{"ok":true}`)
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
