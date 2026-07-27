package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	apierrors "github.com/revenium/revenium-cli/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://api.example.com", "test-key", "", "", "", true)

	assert.Equal(t, "https://api.example.com", c.BaseURL)
	assert.Equal(t, "test-key", c.APIKey)
	assert.True(t, c.Verbose)
	assert.NotNil(t, c.HTTPClient)
}

func TestClientSetsAuthHeader(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("x-api-key")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-secret-key", "", "", "", false)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.NoError(t, err)
	assert.Equal(t, "my-secret-key", gotHeader)
}

func TestClientSetsBearerAuthHeader(t *testing.T) {
	var gotAuth, gotAPIKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAPIKey = r.Header.Get("x-api-key")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-secret-key", "", "", "", false)
	c.UseBearerAuth = true
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.NoError(t, err)
	assert.Equal(t, "Bearer my-secret-key", gotAuth)
	assert.Empty(t, gotAPIKey)
}

func TestClientSuppressesTeamTenantParamsWithBearerAuth(t *testing.T) {
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-secret-key", "team-1", "tenant-1", "", false)
	c.UseBearerAuth = true
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.NoError(t, err)
	assert.NotContains(t, gotRawQuery, "teamId=")
	assert.NotContains(t, gotRawQuery, "tenantId=")
}

func TestClientSetsContentType(t *testing.T) {
	var gotContentType, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", "", "", "", false)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.NoError(t, err)
	assert.Equal(t, "application/json", gotContentType)
	assert.Equal(t, "application/json", gotAccept)
}

func TestClientSetsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", "", "", "", false)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(gotUA, "revenium-cli/"), "User-Agent should start with revenium-cli/, got: %s", gotUA)
}

func TestClientTimeout(t *testing.T) {
	c := NewClient("https://api.example.com", "key", "", "", "", false)
	assert.Equal(t, 30*time.Second, c.HTTPClient.Timeout)
}

func TestErrorMapping401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"unauthorized"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "bad-key", "", "", "", false)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid API key")
	assert.Contains(t, err.Error(), "revenium config set key")
}

func TestErrorMapping403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"error":"forbidden"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", "", "", "", false)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Access denied")
}

func TestErrorMapping404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":"not found"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", "", "", "", false)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Resource not found")
}

func TestErrorMapping500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"internal"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", "", "", "", false)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Revenium API error")
}

func TestSuccessfulRequest(t *testing.T) {
	type result struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result{Name: "test-source", ID: 42})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", "", "", "", false)
	var got result
	err := c.Do(context.Background(), http.MethodGet, "/sources/42", nil, &got)

	require.NoError(t, err)
	assert.Equal(t, "test-source", got.Name)
	assert.Equal(t, 42, got.ID)
}

// TestDoUpdateFlattensNestedIDsAndMerges is a regression guard (D-05) proving
// DoUpdate's GET-merge-PUT cycle — nested-ID flattening, nested-array-ID
// flattening, and the updates merge — is unchanged for a representative
// non-subscriptions resource.
func TestDoUpdateFlattensNestedIDsAndMerges(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "org-1",
				"name": "Old Org",
				"team": map[string]interface{}{"id": "team-1"},
				"organizations": []interface{}{
					map[string]interface{}{"id": "org-a"},
					map[string]interface{}{"id": "org-b"},
				},
			})
			return
		}
		assert.Equal(t, http.MethodPut, r.Method)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"org-1","name":"New Org"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", "", "", "", false)
	updates := map[string]interface{}{"name": "New Org"}
	var result map[string]interface{}
	err := c.DoUpdate(context.Background(), "/orgs/org-1", updates, &result)

	require.NoError(t, err)
	assert.Equal(t, "team-1", receivedBody["teamId"], "nested team object must flatten to teamId")
	assert.ElementsMatch(t, []interface{}{"org-a", "org-b"}, receivedBody["organizationIds"], "nested organizations array must flatten to organizationIds")
	assert.Equal(t, "New Org", receivedBody["name"], "updates must be merged into the fetched object")
}

// TestDoUpdateDoesNotAutoFillClientEmailAddress is the D-05 negative
// regression: the subscriptions-specific clientEmailAddress backfill has been
// REMOVED from the shared DoUpdate and relocated into cmd/subscriptions.
// A GET response with a nested "client" object + "label" but no flat
// clientEmailAddress key must NOT have clientEmailAddress auto-filled.
func TestDoUpdateDoesNotAutoFillClientEmailAddress(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     "sub-1",
				"label":  "x@y.com",
				"client": map[string]interface{}{"id": "client-1"},
			})
			return
		}
		assert.Equal(t, http.MethodPut, r.Method)
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"sub-1"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", "", "", "", false)
	updates := map[string]interface{}{}
	var result map[string]interface{}
	err := c.DoUpdate(context.Background(), "/v2/api/subscriptions/sub-1", updates, &result)

	require.NoError(t, err)
	_, hasClientEmail := receivedBody["clientEmailAddress"]
	assert.False(t, hasClientEmail, "DoUpdate must no longer auto-fill clientEmailAddress (D-05 — relocated to cmd/subscriptions)")
}

// TestDoDownloadReturnsRawBytes asserts DoDownload round-trips a non-JSON
// binary payload unchanged, and that the request carries the auth header
// without an application/json Content-Type (D-01).
func TestDoDownloadReturnsRawBytes(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4\nnot-real-pdf-content")
	var gotAPIKey, gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("x-api-key")
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write(pdfBytes)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", "", "", "", false)
	data, err := c.DoDownload(context.Background(), "/download")

	require.NoError(t, err)
	assert.Equal(t, pdfBytes, data)
	assert.Equal(t, "test-key", gotAPIKey)
	assert.NotEqual(t, "application/json", gotContentType)
}

// TestDoDownloadMapsHTTPError asserts 401 and 404 responses each return the
// same *errors.APIError DoDownload's sibling Do() method returns, carrying
// the corresponding StatusCode, so internal/errors.ExitCodeFor needs no new
// cases for the download path.
func TestDoDownloadMapsHTTPError(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
	}{
		{"401", http.StatusUnauthorized},
		{"404", http.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				fmt.Fprint(w, `{"message":"error"}`)
			}))
			defer srv.Close()

			c := NewClient(srv.URL, "key", "", "", "", false)
			data, err := c.DoDownload(context.Background(), "/download")

			require.Error(t, err)
			assert.Nil(t, data)
			var apiErr *apierrors.APIError
			require.True(t, errors.As(err, &apiErr))
			assert.Equal(t, tc.statusCode, apiErr.StatusCode)
		})
	}
}

// TestDoDownloadRespectsCancelledContext asserts a context cancelled before
// the call returns an error and no bytes, rather than a partial body.
func TestDoDownloadRespectsCancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("should never be read"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", "", "", "", false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	data, err := c.DoDownload(ctx, "/download")

	require.Error(t, err)
	assert.Nil(t, data)
}

// TestDoDownloadIsRaceFreeAcrossGoroutines runs N concurrent DoDownload
// calls on one shared *Client against one httptest server. It exists to be
// meaningful under `go test -race` (MEDIA-01 concurrency probe): DoDownload
// must mutate no *Client field.
func TestDoDownloadIsRaceFreeAcrossGoroutines(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4\nconcurrent-content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(pdfBytes)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", "", "", "", false)

	const n = 20
	var wg sync.WaitGroup
	errs := make([]error, n)
	results := make([][]byte, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			data, err := c.DoDownload(context.Background(), "/download")
			errs[idx] = err
			results[idx] = data
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		require.NoError(t, errs[i])
		assert.Equal(t, pdfBytes, results[i])
	}
}

// TestDoDownloadAppendsTeamAndTenantParams asserts a client with TeamID and
// TenantID set produces a request whose RawQuery contains both, proving
// DoDownload reuses resolveURL rather than duplicating URL construction.
func TestDoDownloadAppendsTeamAndTenantParams(t *testing.T) {
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", "team-1", "tenant-1", "", false)
	_, err := c.DoDownload(context.Background(), "/download")

	require.NoError(t, err)
	assert.Contains(t, gotRawQuery, "teamId=team-1")
	assert.Contains(t, gotRawQuery, "tenantId=tenant-1")
}

// TestDoUploadBinarySendsMultipartFileField asserts DoUploadBinary sends
// method PATCH, a multipart/form-data body with the payload under a "file"
// form field whose own part carries the caller's Content-Type, the auth
// header, and Accept: application/json (D-02, corrected against the live API
// during the 07-03 checkpoint: a raw octet-stream body was rejected with 400,
// multipart/form-data with a "file" field succeeded with 200).
func TestDoUploadBinarySendsMultipartFileField(t *testing.T) {
	payload := []byte{0x89, 'P', 'N', 'G', 0x00, 0x01, 0x02}
	var gotMethod, gotAPIKey, gotAccept, gotOuterContentType string
	var gotFieldContentType string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAPIKey = r.Header.Get("x-api-key")
		gotAccept = r.Header.Get("Accept")
		gotOuterContentType = r.Header.Get("Content-Type")

		_, params, err := mime.ParseMediaType(gotOuterContentType)
		require.NoError(t, err)
		mr := multipart.NewReader(r.Body, params["boundary"])
		part, err := mr.NextPart()
		require.NoError(t, err)
		assert.Equal(t, "file", part.FormName())
		gotFieldContentType = part.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(part)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", "", "", "", false)
	err := c.DoUploadBinary(context.Background(), "/upload", "logo.png", payload, "image/png")

	require.NoError(t, err)
	assert.Equal(t, http.MethodPatch, gotMethod)
	assert.Equal(t, payload, gotBody)
	assert.True(t, strings.HasPrefix(gotOuterContentType, "multipart/form-data;"))
	assert.Equal(t, "image/png", gotFieldContentType)
	assert.Equal(t, "test-key", gotAPIKey)
	assert.Equal(t, "application/json", gotAccept)
}

// TestDoUploadBinaryMapsHTTPError asserts 404 and 415 responses each return
// an *errors.APIError with the matching StatusCode.
func TestDoUploadBinaryMapsHTTPError(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
	}{
		{"404", http.StatusNotFound},
		{"415", http.StatusUnsupportedMediaType},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				fmt.Fprint(w, `{"message":"error"}`)
			}))
			defer srv.Close()

			c := NewClient(srv.URL, "key", "", "", "", false)
			err := c.DoUploadBinary(context.Background(), "/upload", "logo.png", []byte("data"), "image/png")

			require.Error(t, err)
			var apiErr *apierrors.APIError
			require.True(t, errors.As(err, &apiErr))
			assert.Equal(t, tc.statusCode, apiErr.StatusCode)
		})
	}
}

// TestDoUploadBinaryRespectsCancelledContext asserts a pre-cancelled context
// returns an error rather than sending a partial upload.
func TestDoUploadBinaryRespectsCancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", "", "", "", false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.DoUploadBinary(ctx, "/upload", "logo.png", []byte("data"), "image/png")

	require.Error(t, err)
}

// TestDoUploadBinaryIsRaceFreeAcrossGoroutines runs N concurrent
// DoUploadBinary calls on one shared *Client, meaningful under
// `go test -race` (MEDIA-01 concurrency probe): DoUploadBinary must mutate
// no *Client field.
func TestDoUploadBinaryIsRaceFreeAcrossGoroutines(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", "", "", "", false)

	const n = 20
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = c.DoUploadBinary(context.Background(), "/upload", "logo.png", []byte("concurrent-data"), "image/png")
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		require.NoError(t, errs[i])
	}
}

// TestDoUploadBinaryAppendsTeamAndTenantParams asserts a client with TeamID
// and TenantID set produces a request whose RawQuery contains both, proving
// DoUploadBinary reuses resolveURL rather than duplicating URL construction.
func TestDoUploadBinaryAppendsTeamAndTenantParams(t *testing.T) {
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "key", "team-1", "tenant-1", "", false)
	err := c.DoUploadBinary(context.Background(), "/upload", "logo.png", []byte("data"), "image/png")

	require.NoError(t, err)
	assert.Contains(t, gotRawQuery, "teamId=team-1")
	assert.Contains(t, gotRawQuery, "tenantId=tenant-1")
}

func TestVerboseLogging(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	c := NewClient(srv.URL, "my-api-key-1234", "", "", "", true)
	err := c.Do(context.Background(), http.MethodGet, "/test", nil, nil)

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	require.NoError(t, err)
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/test")
	assert.Contains(t, output, "200")
	// API key should be masked in verbose output
	assert.NotContains(t, output, "my-api-key-1234")
	assert.Contains(t, output, "1234") // last 4 chars shown
}
