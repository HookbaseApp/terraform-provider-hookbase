package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateSource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/api/organizations/org-1/sources" {
			t.Errorf("path = %q, want /api/organizations/org-1/sources", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req CreateSourceRequest
		json.Unmarshal(body, &req)
		if req.Name != "GitHub Webhooks" {
			t.Errorf("request Name = %q, want %q", req.Name, "GitHub Webhooks")
		}
		if req.Slug != "github-webhooks" {
			t.Errorf("request Slug = %q, want %q", req.Slug, "github-webhooks")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sourceResponse{
			Source: Source{
				ID:             "src_abc",
				OrganizationID: "org-1",
				Name:           "GitHub Webhooks",
				Slug:           "github-webhooks",
				IsActive:       true,
				IPFilterMode:   "none",
				DedupStrategy:  "none",
				CreatedAt:      "2026-01-01T00:00:00Z",
				UpdatedAt:      "2026-01-01T00:00:00Z",
			},
		})
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "key",
		HTTPClient:     srv.Client(),
	}

	src, err := c.CreateSource(context.Background(), CreateSourceRequest{
		Name: "GitHub Webhooks",
		Slug: "github-webhooks",
	})
	if err != nil {
		t.Fatalf("CreateSource returned error: %v", err)
	}
	if src.ID != "src_abc" {
		t.Errorf("ID = %q, want %q", src.ID, "src_abc")
	}
	if src.Name != "GitHub Webhooks" {
		t.Errorf("Name = %q, want %q", src.Name, "GitHub Webhooks")
	}
	if src.Slug != "github-webhooks" {
		t.Errorf("Slug = %q, want %q", src.Slug, "github-webhooks")
	}
	if !src.IsActive {
		t.Error("IsActive = false, want true")
	}
}

func TestGetSource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/organizations/org-1/sources/src_abc" {
			t.Errorf("path = %q, want /api/organizations/org-1/sources/src_abc", r.URL.Path)
		}

		provider := "github"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sourceResponse{
			Source: Source{
				ID:               "src_abc",
				OrganizationID:   "org-1",
				Name:             "GitHub Webhooks",
				Slug:             "github-webhooks",
				Provider:         &provider,
				HasSigningSecret: true,
				IsActive:         true,
				IPFilterMode:     "none",
				DedupStrategy:    "none",
				CreatedAt:        "2026-01-01T00:00:00Z",
				UpdatedAt:        "2026-01-01T00:00:00Z",
			},
		})
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "key",
		HTTPClient:     srv.Client(),
	}

	src, err := c.GetSource(context.Background(), "src_abc")
	if err != nil {
		t.Fatalf("GetSource returned error: %v", err)
	}
	if src.ID != "src_abc" {
		t.Errorf("ID = %q, want %q", src.ID, "src_abc")
	}
	if src.Provider == nil || *src.Provider != "github" {
		t.Errorf("Provider = %v, want %q", src.Provider, "github")
	}
	if !src.HasSigningSecret {
		t.Error("HasSigningSecret = false, want true")
	}
}

func TestUpdateSource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %q, want PATCH", r.Method)
		}
		if r.URL.Path != "/api/organizations/org-1/sources/src_abc" {
			t.Errorf("path = %q, want /api/organizations/org-1/sources/src_abc", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req UpdateSourceRequest
		json.Unmarshal(body, &req)
		if req.Name == nil || *req.Name != "Renamed Source" {
			t.Errorf("request Name = %v, want %q", req.Name, "Renamed Source")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sourceResponse{
			Source: Source{
				ID:             "src_abc",
				OrganizationID: "org-1",
				Name:           "Renamed Source",
				Slug:           "github-webhooks",
				IsActive:       true,
				IPFilterMode:   "none",
				DedupStrategy:  "none",
				CreatedAt:      "2026-01-01T00:00:00Z",
				UpdatedAt:      "2026-01-02T00:00:00Z",
			},
		})
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "key",
		HTTPClient:     srv.Client(),
	}

	name := "Renamed Source"
	src, err := c.UpdateSource(context.Background(), "src_abc", UpdateSourceRequest{
		Name: &name,
	})
	if err != nil {
		t.Fatalf("UpdateSource returned error: %v", err)
	}
	if src.Name != "Renamed Source" {
		t.Errorf("Name = %q, want %q", src.Name, "Renamed Source")
	}
	if src.UpdatedAt != "2026-01-02T00:00:00Z" {
		t.Errorf("UpdatedAt = %q, want %q", src.UpdatedAt, "2026-01-02T00:00:00Z")
	}
}

func TestDeleteSource(t *testing.T) {
	var gotMethod string
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "key",
		HTTPClient:     srv.Client(),
	}

	err := c.DeleteSource(context.Background(), "src_abc")
	if err != nil {
		t.Fatalf("DeleteSource returned error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/api/organizations/org-1/sources/src_abc" {
		t.Errorf("path = %q, want /api/organizations/org-1/sources/src_abc", gotPath)
	}
}

func TestRevealSourceSecret(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/organizations/org-1/sources/src_abc/reveal-secret" {
			t.Errorf("path = %q, want /api/organizations/org-1/sources/src_abc/reveal-secret", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(revealSecretResponse{
			SigningSecret: "whsec_supersecret123",
		})
	}))
	defer srv.Close()

	c := &Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "key",
		HTTPClient:     srv.Client(),
	}

	secret, err := c.RevealSourceSecret(context.Background(), "src_abc")
	if err != nil {
		t.Fatalf("RevealSourceSecret returned error: %v", err)
	}
	if secret != "whsec_supersecret123" {
		t.Errorf("secret = %q, want %q", secret, "whsec_supersecret123")
	}
}
