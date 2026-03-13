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
	TenantID   string
	OwnerID    string
	HTTPClient *http.Client
	Verbose    bool
}

// NewClient creates a new API client.
func NewClient(baseURL, apiKey, teamID, tenantID, ownerID string, verbose bool) *Client {
	return &Client{
		BaseURL:  baseURL,
		APIKey:   apiKey,
		TeamID:   teamID,
		TenantID: tenantID,
		OwnerID:  ownerID,
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
	if c.TenantID != "" {
		if strings.Contains(url, "?") {
			url += "&tenantId=" + c.TenantID
		} else {
			url += "?tenantId=" + c.TenantID
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
		// Include API error details when available
		if bodyStr != "" {
			var apiErr map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &apiErr); err == nil {
				if details, ok := apiErr["details"]; ok {
					message = fmt.Sprintf("Request failed (HTTP %d): %v", resp.StatusCode, details)
					break
				}
				if msg, ok := apiErr["message"].(string); ok {
					message = fmt.Sprintf("Request failed (HTTP %d): %s", resp.StatusCode, msg)
					break
				}
			}
		}
		message = fmt.Sprintf("Request failed (HTTP %d).", resp.StatusCode)
	}

	return &errors.APIError{
		StatusCode: resp.StatusCode,
		Message:    message,
		Body:       bodyStr,
	}
}

// DoCreate executes a POST request, automatically injecting teamId
// into the body if the client has a TeamID set and the field is not already present.
func (c *Client) DoCreate(ctx context.Context, path string, body map[string]interface{}, result interface{}) error {
	if c.TeamID != "" {
		if _, ok := body["teamId"]; !ok {
			body["teamId"] = c.TeamID
		}
	}
	return c.Do(ctx, "POST", path, body, result)
}

// DoCreateWithOwner is like DoCreate but also injects ownerId into the body.
func (c *Client) DoCreateWithOwner(ctx context.Context, path string, body map[string]interface{}, result interface{}) error {
	if c.OwnerID != "" {
		if _, ok := body["ownerId"]; !ok {
			body["ownerId"] = c.OwnerID
		}
	}
	return c.DoCreate(ctx, path, body, result)
}

// DoUpdate fetches the existing resource via GET, merges the provided updates into it,
// and sends a PUT request with the merged data. It also ensures teamId, ownerId, and
// organizationIds are set from nested objects if not present as flat fields.
func (c *Client) DoUpdate(ctx context.Context, path string, updates map[string]interface{}, result interface{}) error {
	var existing map[string]interface{}
	if err := c.Do(ctx, "GET", path, nil, &existing); err != nil {
		return err
	}

	// Extract flat IDs from nested objects if not already present.
	// The API returns nested objects (e.g. "team": {"id": "x"}) in GET responses
	// but expects flat IDs (e.g. "teamId": "x") in PUT requests.
	nestedToFlat := map[string]string{
		"team":         "teamId",
		"owner":        "ownerId",
		"organization": "organizationId",
		"product":      "productId",
		"client":       "clientId",
	}
	for nested, flat := range nestedToFlat {
		if _, ok := existing[flat]; !ok {
			if obj, ok := existing[nested].(map[string]interface{}); ok {
				if id, ok := obj["id"].(string); ok {
					existing[flat] = id
				}
			}
		}
	}
	// Extract IDs from nested array objects (e.g. "organizations" -> "organizationIds", "teams" -> "teamIds")
	nestedArrayToFlat := map[string]string{
		"organizations": "organizationIds",
		"teams":         "teamIds",
	}
	for nested, flat := range nestedArrayToFlat {
		if _, ok := existing[flat]; !ok {
			if items, ok := existing[nested].([]interface{}); ok {
				ids := make([]string, 0, len(items))
				for _, item := range items {
					if m, ok := item.(map[string]interface{}); ok {
						if id, ok := m["id"].(string); ok {
							ids = append(ids, id)
						}
					}
				}
				if len(ids) > 0 {
					existing[flat] = ids
				}
			}
		}
	}
	// Map label to clientEmailAddress if not present (subscriptions)
	if _, ok := existing["clientEmailAddress"]; !ok {
		if label, ok := existing["label"].(string); ok && label != "" {
			if _, hasClient := existing["client"]; hasClient {
				existing["clientEmailAddress"] = label
			}
		}
	}

	for k, v := range updates {
		existing[k] = v
	}

	return c.Do(ctx, "PUT", path, existing, result)
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
