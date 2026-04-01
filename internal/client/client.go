package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL        string
	OrganizationID string
	APIKey         string
	HTTPClient     *http.Client
}

type APIError struct {
	Error   string      `json:"error"`
	Code    string      `json:"code,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

func New(apiURL, organizationID, apiKey string) *Client {
	return &Client{
		BaseURL:        apiURL,
		OrganizationID: organizationID,
		APIKey:         apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
	}
}

func (c *Client) orgURL(path string) string {
	if c.OrganizationID != "" {
		return fmt.Sprintf("%s/api/organizations/%s%s", c.BaseURL, c.OrganizationID, path)
	}
	return fmt.Sprintf("%s/api%s", c.BaseURL, path)
}

func (c *Client) doRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var respBody []byte
	for attempt := 0; attempt < 3; attempt++ {
		var bodyReader io.Reader
		if body != nil {
			jsonBody, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("marshaling request body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonBody)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "terraform-provider-hookbase/1.0")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			if attempt < 2 {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return fmt.Errorf("executing request: %w", err)
		}

		respBody, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			if attempt < 2 {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if attempt < 2 {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
		}

		if resp.StatusCode >= 400 {
			var apiErr APIError
			if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error != "" {
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, apiErr.Error)
			}
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
		}

		break
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return nil
}

func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	return c.doRequest(ctx, http.MethodGet, c.orgURL(path), nil, result)
}

func (c *Client) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, http.MethodPost, c.orgURL(path), body, result)
}

func (c *Client) Patch(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, http.MethodPatch, c.orgURL(path), body, result)
}

func (c *Client) Delete(ctx context.Context, path string) error {
	return c.doRequest(ctx, http.MethodDelete, c.orgURL(path), nil, nil)
}
