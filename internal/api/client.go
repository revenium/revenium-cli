// Package api provides an HTTP client for the Revenium API.
// It handles authentication, content negotiation, error mapping, and verbose logging.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/revenium/revenium-cli/internal/build"
	"github.com/revenium/revenium-cli/internal/errors"
)

// Client is an HTTP client configured for the Revenium API.
type Client struct {
	BaseURL    string
	APIKey     string
	TeamID     string
	HTTPClient *http.Client
	Verbose    bool
}

// NewClient creates a new API client with the given base URL, API key, team ID, and verbose setting.
func NewClient(baseURL, apiKey, teamID string, verbose bool) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		TeamID:  teamID,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Verbose: verbose,
	}
}

// Do executes an HTTP request against the Revenium API.
// If body is non-nil, it is marshaled to JSON and sent as the request body.
// If result is non-nil, the response body is decoded into it.
func (c *Client) Do(ctx context.Context, method, path string, body, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := c.BaseURL + path
	if c.TeamID != "" {
		if strings.Contains(url, "?") {
			url += "&teamId=" + c.TeamID
		} else {
			url += "?teamId=" + c.TeamID
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "revenium-cli/"+build.Version)

	if c.Verbose {
		maskedKey := maskAPIKey(c.APIKey)
		fmt.Fprintf(os.Stderr, "> %s %s\n", method, url)
		fmt.Fprintf(os.Stderr, "> x-api-key: %s\n", maskedKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("Could not connect to api.revenium.ai. Check your network connection.")
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "< %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	if resp.StatusCode >= 400 {
		return mapHTTPError(resp)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// mapHTTPError reads the response body and returns an appropriate APIError.
func mapHTTPError(resp *http.Response) error {
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	var message string
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		message = "Invalid API key. Run `revenium config set key <your-key>` to fix."
	case resp.StatusCode == http.StatusForbidden:
		message = "Access denied. Your API key may not have permission for this operation."
	case resp.StatusCode == http.StatusNotFound:
		message = "Resource not found."
	case resp.StatusCode >= 500:
		message = "Revenium API error. Try again later or contact support."
	default:
		message = fmt.Sprintf("Request failed (HTTP %d).", resp.StatusCode)
	}

	return &errors.APIError{
		StatusCode: resp.StatusCode,
		Message:    message,
		Body:       bodyStr,
	}
}

// DoList executes a GET request and unwraps the response into a slice.
// It handles both Spring HATEOAS paginated responses
// ({"_embedded": {"<resource>List": [...]}, "page": {...}}) and plain JSON arrays.
func (c *Client) DoList(ctx context.Context, path string, result *[]map[string]interface{}) error {
	var raw json.RawMessage
	if err := c.Do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return err
	}

	// Try decoding as a plain array first
	var arr []map[string]interface{}
	if err := json.Unmarshal(raw, &arr); err == nil {
		*result = arr
		return nil
	}

	// Try decoding as a HATEOAS wrapper object
	var wrapper map[string]interface{}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return fmt.Errorf("failed to decode list response: %w", err)
	}

	embedded, ok := wrapper["_embedded"].(map[string]interface{})
	if !ok {
		*result = []map[string]interface{}{}
		return nil
	}

	for _, v := range embedded {
		if items, ok := v.([]interface{}); ok {
			out := make([]map[string]interface{}, 0, len(items))
			for _, item := range items {
				if m, ok := item.(map[string]interface{}); ok {
					out = append(out, m)
				}
			}
			*result = out
			return nil
		}
	}

	*result = []map[string]interface{}{}
	return nil
}

// maskAPIKey masks all but the last 4 characters of the API key.
func maskAPIKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}
