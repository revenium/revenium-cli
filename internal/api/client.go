// Package api provides an HTTP client for the Revenium API.
// It handles authentication, content negotiation, error mapping, and verbose logging.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"time"

	"github.com/revenium/revenium-cli/internal/build"
	"github.com/revenium/revenium-cli/internal/errors"
)

// Client is an HTTP client configured for the Revenium API.
type Client struct {
	BaseURL string
	// AnalyticsBaseURL is the distinct, independently configured base URL for
	// the analytics API host (e.g. https://app.revenium.ai). It is populated
	// externally from config (cfg.AnalyticsAPIURL) and never derived from
	// BaseURL by string-rewriting (see MeterBaseURL, which is the opposite,
	// rejected pattern for this field).
	AnalyticsBaseURL string
	APIKey           string
	TeamID           string
	TenantID         string
	OwnerID          string
	HTTPClient       *http.Client
	Verbose          bool
	// UseBearerAuth switches Do() to send Authorization: Bearer <key> instead
	// of x-api-key, and suppresses teamId/tenantId query-param injection.
	// Toggled by analytics-backed commands' PersistentPreRunE, never by NewClient.
	UseBearerAuth bool
	// Quiet suppresses advisory notices written to stderr (never errors).
	// Set post-construction from the root --quiet flag, matching the
	// AnalyticsBaseURL convention, so NewClient's signature stays stable.
	Quiet bool
}

// MeterBaseURL returns the metering API base URL derived from the management
// API base URL by replacing the /profitstream path segment with /meter.
func (c *Client) MeterBaseURL() string {
	return strings.Replace(c.BaseURL, "/profitstream", "/meter", 1)
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

// resolveURL builds the full request URL for path, appending teamId/tenantId
// query parameters when the client is not using bearer auth. Extracted from
// Do() so DoDownload (and any future binary method) reuses the same URL
// construction instead of duplicating it (D-03).
func (c *Client) resolveURL(path string) string {
	url := c.BaseURL + path
	if !c.UseBearerAuth {
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
	}
	return url
}

// networkError wraps a transport-level failure (HTTPClient.Do returning a
// non-nil error) in a message that names the client's actual configured
// BaseURL, rather than a hardcoded default host (WR-03). Shared by Do,
// DoDownload, and DoUploadBinary so the message only needs to be correct in
// one place.
func (c *Client) networkError(err error) error {
	return fmt.Errorf("Could not connect to %s. Check your network connection.", c.BaseURL)
}

// setCommonHeaders sets the auth header (Bearer vs x-api-key) and User-Agent
// on req. It does not set Content-Type or Accept, since those differ between
// JSON methods (Do) and binary methods (DoDownload).
func (c *Client) setCommonHeaders(req *http.Request) {
	if c.UseBearerAuth {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	} else {
		req.Header.Set("x-api-key", c.APIKey)
	}
	req.Header.Set("User-Agent", "revenium-cli/"+build.Version)
}

// Do executes an HTTP request against the Revenium API.
// If body is non-nil, it is marshaled to JSON and sent as the request body.
// If result is non-nil, the response body is decoded into it.
func (c *Client) Do(ctx context.Context, method, path string, body, result interface{}) error {
	var reqBody io.Reader
	var reqData []byte
	if body != nil {
		var err error
		reqData, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(reqData)
	}

	url := c.resolveURL(path)
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setCommonHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.Verbose {
		maskedKey := maskAPIKey(c.APIKey)
		fmt.Fprintf(os.Stderr, "> %s %s\n", method, url)
		if c.UseBearerAuth {
			fmt.Fprintf(os.Stderr, "> Authorization: Bearer %s\n", maskedKey)
		} else {
			fmt.Fprintf(os.Stderr, "> x-api-key: %s\n", maskedKey)
		}
		if len(reqData) > 0 {
			var pretty bytes.Buffer
			if json.Indent(&pretty, reqData, "> ", "  ") == nil {
				fmt.Fprintf(os.Stderr, "> Body:\n> %s\n", pretty.String())
			} else {
				fmt.Fprintf(os.Stderr, "> Body: %s\n", reqData)
			}
		}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return c.networkError(err)
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

// DoDownload executes a GET request and returns the raw response body bytes,
// with no JSON decoding. Used for binary responses (e.g. invoice PDFs) where
// the body is not JSON. It reuses resolveURL and setCommonHeaders so no
// transport setup is duplicated (D-01, D-03). It mutates no *Client field, so
// concurrent calls on a shared client are data-race free.
func (c *Client) DoDownload(ctx context.Context, path string) ([]byte, error) {
	url := c.resolveURL(path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setCommonHeaders(req)
	// The payload is binary, never JSON — do not send Content-Type or an
	// application/json Accept header.
	req.Header.Set("Accept", "*/*")

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "> %s %s\n", http.MethodGet, url)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, c.networkError(err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		if c.Verbose {
			fmt.Fprintf(os.Stderr, "< %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
		}
		return nil, mapHTTPError(resp)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "< %d %s (%d bytes)\n", resp.StatusCode, http.StatusText(resp.StatusCode), len(data))
	}

	return data, nil
}

// DoUploadBinary executes a PATCH request whose body is a multipart/form-data
// payload carrying data under a single "file" form field, with the part's
// own Content-Type set to contentType — the bytes are never handed to
// json.Marshal. Used for logo upload endpoints (D-02, corrected: `PATCH
// /v2/api/products/{id}/logo` and its sources/teams equivalents). The
// multipart/form-data shape — not a raw octet-stream body — was confirmed
// against the live dev API during the 07-03 checkpoint: FEATURES.md:348
// documents the request body as `{"file": binary}`, and a raw-body PATCH was
// rejected with 400 while a multipart "file" field succeeded with 200. It
// reuses resolveURL and setCommonHeaders so no transport setup is duplicated
// (D-03). It mutates no *Client field, so concurrent calls on a shared client
// are data-race free.
func (c *Client) DoUploadBinary(ctx context.Context, path, filename string, data []byte, contentType string) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(
		`form-data; name="file"; filename="%s"`,
		quoteEscaper.Replace(filename),
	))
	partHeader.Set("Content-Type", contentType)
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return fmt.Errorf("failed to create multipart form part: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("failed to write multipart form data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := c.resolveURL(path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, &body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setCommonHeaders(req)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// Error bodies come back as JSON even though the request payload is
	// binary, so Accept stays application/json (unlike DoDownload).
	req.Header.Set("Accept", "application/json")

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "> %s %s\n", http.MethodPatch, url)
		fmt.Fprintf(os.Stderr, "> Content-Type: %s\n", writer.FormDataContentType())
		fmt.Fprintf(os.Stderr, "> Body: %d bytes (file field content-type %s)\n", len(data), contentType)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return c.networkError(err)
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

	// WR-02: DoUploadBinary has no result parameter, so a successful response
	// body is otherwise silently discarded by the deferred io.Copy above with
	// no way for any caller to inspect it. None of the three logo-upload
	// commands (products/sources/teams) currently need server-returned data —
	// their output is built entirely from locally-known values — so surface
	// the raw body in verbose mode only, for debugging.
	if c.Verbose {
		if respBody, readErr := io.ReadAll(resp.Body); readErr == nil && len(respBody) > 0 {
			fmt.Fprintf(os.Stderr, "< Body: %s\n", respBody)
		}
	}

	return nil
}

// quoteEscaper escapes backslashes and double quotes in a filename before it
// is embedded in a multipart Content-Disposition header's filename="..."
// attribute, mirroring the unexported escapeQuotes helper that Go's own
// mime/multipart.Writer.CreateFormFile uses internally (WR-01). Without this,
// a filename containing a `"` or `\` (both legal on macOS/Linux) would
// produce a malformed header — an unescaped quote prematurely closes the
// attribute value.
var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

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
					message = formatDetails(resp.StatusCode, details)
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

// formatDetails extracts a human-readable message from the API's details field.
func formatDetails(statusCode int, details interface{}) string {
	switch d := details.(type) {
	case map[string]interface{}:
		// Extract the first value, e.g. {"error": "Expected ISO 8601 format ..."}
		for _, v := range d {
			return fmt.Sprintf("Request failed (HTTP %d): %v", statusCode, v)
		}
	case string:
		return fmt.Sprintf("Request failed (HTTP %d): %s", statusCode, d)
	}
	return fmt.Sprintf("Request failed (HTTP %d): %v", statusCode, details)
}

// DoCreate executes a POST request, automatically injecting teamId
// into the body if the client has a TeamID set and the field is not already present.
func (c *Client) DoCreate(ctx context.Context, path string, body map[string]interface{}, result interface{}) error {
	if c.TeamID != "" {
		if _, ok := body["teamId"]; !ok {
			body["teamId"] = c.TeamID
		}
	}
	if c.TenantID != "" {
		if _, ok := body["tenantId"]; !ok {
			body["tenantId"] = c.TenantID
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
	for k, v := range updates {
		existing[k] = v
	}

	return c.Do(ctx, "PUT", path, existing, result)
}

// ListOptions controls pagination behavior for list operations.
type ListOptions struct {
	// Page is the 0-based page number. -1 means not set (use default).
	Page int
	// PageSize is the number of items per page. -1 means not set (use default).
	// When FetchAll is set, this selects the per-request batch size rather than
	// capping the total number of results returned.
	PageSize int
	// FetchAll iterates through all pages and returns the aggregate result.
	// Ignored when Page is explicitly set (an explicit page means the caller
	// wants exactly that page). PageSize does NOT disable it.
	FetchAll bool
}

// DoList executes a GET request and unwraps the response into a slice.
// It handles both Spring HATEOAS paginated responses
// ({"_embedded": {"<resource>List": [...]}, "page": {...}}) and plain JSON arrays.
// When opts.FetchAll is true and no explicit page is set, it iterates through
// all pages to return the complete result set, using opts.PageSize as the batch
// size when provided.
func (c *Client) DoList(ctx context.Context, path string, opts ListOptions, result *[]map[string]interface{}) error {
	if opts.FetchAll && opts.Page < 0 {
		return c.doListAll(ctx, path, opts, result)
	}

	paginatedPath := c.buildPaginatedPath(path, opts)
	return c.doListOnePage(ctx, paginatedPath, result)
}

// buildPaginatedPath appends page and size query parameters to the path.
func (c *Client) buildPaginatedPath(path string, opts ListOptions) string {
	if opts.Page < 0 && opts.PageSize < 0 {
		return path
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	result := path
	if opts.Page >= 0 {
		result += fmt.Sprintf("%spage=%d", sep, opts.Page)
		sep = "&"
	}
	if opts.PageSize >= 0 {
		result += fmt.Sprintf("%ssize=%d", sep, opts.PageSize)
	}
	return result
}

// defaultFetchAllPageSize is the per-request batch size used when aggregating
// all pages and the caller did not specify one.
const defaultFetchAllPageSize = 100

// doListAll fetches all pages from a paginated endpoint and aggregates the results.
// opts.PageSize, when set, selects the per-request batch size; the server remains
// free to return fewer items per page than requested, which is why iteration is
// driven by the response's own totalPages rather than by the requested size.
func (c *Client) doListAll(ctx context.Context, path string, opts ListOptions, result *[]map[string]interface{}) error {
	var all []map[string]interface{}
	page := 0
	pageSize := defaultFetchAllPageSize
	if opts.PageSize > 0 {
		pageSize = opts.PageSize
	}

	for {
		opts := ListOptions{Page: page, PageSize: pageSize}
		paginatedPath := c.buildPaginatedPath(path, opts)

		items, totalPages, err := c.doListOnePageWithMeta(ctx, paginatedPath)
		if err != nil {
			return err
		}

		all = append(all, items...)

		// If response was a plain array (totalPages == -1) or last page, stop
		if totalPages < 0 || page >= totalPages-1 || len(items) == 0 {
			break
		}
		page++
	}

	*result = all
	return nil
}

// doListOnePage fetches a single page and returns the items.
//
// When the response reports more pages than the one fetched, a note is written
// to stderr. The server's own totalPages is the only way a caller can discover
// that its result set is partial: the CLI unwraps the HATEOAS envelope and
// returns a bare array, so the "page" metadata never reaches stdout. Reporting
// it on stderr keeps stdout a clean JSON array for piping while making silent
// truncation visible.
func (c *Client) doListOnePage(ctx context.Context, path string, result *[]map[string]interface{}) error {
	items, totalPages, err := c.doListOnePageWithMeta(ctx, path)
	if err != nil {
		return err
	}
	if totalPages > 1 && !c.Quiet {
		fmt.Fprintf(os.Stderr, "Note: showing 1 of %d pages. Omit --page to fetch all pages.\n", totalPages)
	}
	*result = items
	return nil
}

// doListOnePageWithMeta fetches a single page and returns items plus totalPages.
// Returns totalPages=-1 if the response was a plain array (not paginated).
func (c *Client) doListOnePageWithMeta(ctx context.Context, path string) ([]map[string]interface{}, int, error) {
	var raw json.RawMessage
	if err := c.Do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, 0, err
	}

	// Try decoding as a plain array first
	var arr []map[string]interface{}
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, -1, nil
	}

	// Try decoding as a HATEOAS wrapper object
	var wrapper map[string]interface{}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, 0, fmt.Errorf("failed to decode list response: %w", err)
	}

	// Extract pagination metadata
	totalPages := -1
	if pageInfo, ok := wrapper["page"].(map[string]interface{}); ok {
		if tp, ok := pageInfo["totalPages"].(float64); ok {
			totalPages = int(tp)
		}
	}

	embedded, ok := wrapper["_embedded"].(map[string]interface{})
	if !ok {
		return []map[string]interface{}{}, totalPages, nil
	}

	for _, v := range embedded {
		if items, ok := v.([]interface{}); ok {
			out := make([]map[string]interface{}, 0, len(items))
			for _, item := range items {
				if m, ok := item.(map[string]interface{}); ok {
					out = append(out, m)
				}
			}
			return out, totalPages, nil
		}
	}

	return []map[string]interface{}{}, totalPages, nil
}

// maskAPIKey masks all but the last 4 characters of the API key.
func maskAPIKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}
