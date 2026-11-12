package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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

// Retries. A 429 gets more tries than other failures, and waits as long as the API's Retry-After
// says: the API's global limiter counts requests per IP in fixed one-minute windows (120 a
// minute), so a refresh of many resources, which Terraform runs in parallel, can exhaust a window
// and only the next one helps. Parallel requests that all waited for the same window can hit the
// limit again as it opens, hence the extra tries.
const (
	maxAttempts          = 3
	maxRateLimitAttempts = 6
	maxRetryAfter        = 60 * time.Second
)

// wait sleeps for d or until ctx is done. A variable so tests can record the waits.
var wait = func(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// backoff is the wait before retry number attempt+1 when the API gives no Retry-After.
func backoff(attempt int) time.Duration {
	return time.Duration(attempt+1) * time.Second
}

// retryAfter is the wait a 429 or 503 asks for in its Retry-After header (seconds or an HTTP
// date), capped at maxRetryAfter, or fallback when there is none.
func retryAfter(h http.Header, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(h.Get("Retry-After"))
	if v == "" {
		return fallback
	}
	d := fallback
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		d = time.Duration(secs) * time.Second
	} else if at, err := http.ParseTime(v); err == nil {
		d = time.Until(at)
	}
	switch {
	case d < 0:
		return 0
	case d > maxRetryAfter:
		return maxRetryAfter
	}
	return d
}

func (c *Client) doRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var jsonBody []byte
	if body != nil {
		var err error
		if jsonBody, err = json.Marshal(body); err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
	}

	var respBody []byte
	for attempt := 0; ; attempt++ {
		var bodyReader io.Reader
		if body != nil {
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
			if attempt+1 < maxAttempts && ctx.Err() == nil {
				if werr := wait(ctx, backoff(attempt)); werr == nil {
					continue
				}
			}
			return fmt.Errorf("executing request: %w", err)
		}

		respBody, err = io.ReadAll(resp.Body)
		// Close's error is not actionable here: the body has already been read, and the
		// ReadAll error below is the one that tells us whether the response is usable.
		_ = resp.Body.Close()
		if err != nil {
			if attempt+1 < maxAttempts {
				if werr := wait(ctx, backoff(attempt)); werr == nil {
					continue
				}
			}
			return fmt.Errorf("reading response: %w", err)
		}

		var retryIn time.Duration
		retry := false
		switch {
		case resp.StatusCode == http.StatusTooManyRequests:
			retry = attempt+1 < maxRateLimitAttempts
			retryIn = retryAfter(resp.Header, backoff(attempt))
		case resp.StatusCode == http.StatusServiceUnavailable:
			retry = attempt+1 < maxAttempts
			retryIn = retryAfter(resp.Header, backoff(attempt))
		case resp.StatusCode >= 500:
			retry = attempt+1 < maxAttempts
			retryIn = backoff(attempt)
		}
		if retry {
			if err := wait(ctx, retryIn); err != nil {
				return fmt.Errorf("waiting to retry after API error (%d): %w", resp.StatusCode, err)
			}
			continue
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
