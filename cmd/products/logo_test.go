package products

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	"github.com/revenium/revenium-cli/internal/dryrun"
	apierrors "github.com/revenium/revenium-cli/internal/errors"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordedRequest captures the shape of one request a recording test server
// observed, so tests can assert an exact method/path/body/Content-Type
// sequence and — critically — an exact request COUNT (proving no
// GET-then-PUT round trip is in play, PITFALLS.md B5). For a
// multipart/form-data request (the live-confirmed logo upload shape, 07-03),
// Body and ContentType are populated from the "file" form field's own part —
// not the outer multipart envelope — so downstream assertions read the same
// as they would for a raw body.
type recordedRequest struct {
	Method      string
	Path        string
	Body        []byte
	ContentType string
}

// newRecordingLogoServer returns an httptest.Server that appends every
// request it receives to records and always responds with statusCode/body.
func newRecordingLogoServer(records *[]recordedRequest, statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outerContentType := r.Header.Get("Content-Type")
		var fileBody []byte
		var fileContentType string

		if mediaType, params, err := mime.ParseMediaType(outerContentType); err == nil && strings.HasPrefix(mediaType, "multipart/") {
			mr := multipart.NewReader(r.Body, params["boundary"])
			part, perr := mr.NextPart()
			if perr == nil && part.FormName() == "file" {
				fileBody, _ = io.ReadAll(part)
				fileContentType = part.Header.Get("Content-Type")
			}
		} else {
			fileBody, _ = io.ReadAll(r.Body)
			fileContentType = outerContentType
		}

		*records = append(*records, recordedRequest{
			Method:      r.Method,
			Path:        r.URL.Path,
			Body:        fileBody,
			ContentType: fileContentType,
		})
		w.WriteHeader(statusCode)
		if body != "" {
			fmt.Fprint(w, body)
		}
	}))
}

func TestProductLogoSetPatchesRawBytes(t *testing.T) {
	var records []recordedRequest
	srv := newRecordingLogoServer(&records, http.StatusOK, "")
	defer srv.Close()

	fakePNG := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x01, 0x02, 0x03}
	filePath := filepath.Join(t.TempDir(), "logo.png")
	require.NoError(t, os.WriteFile(filePath, fakePNG, 0o600))

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newLogoSetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"prod-1", "--file", filePath})
	err := c.Execute()

	require.NoError(t, err)
	require.Len(t, records, 1, "logo set must issue exactly one request — no GET-then-PUT round trip")
	assert.Equal(t, http.MethodPatch, records[0].Method)
	assert.Equal(t, "/v2/api/products/prod-1/logo", records[0].Path)
	assert.Equal(t, fakePNG, records[0].Body)
	assert.Equal(t, "image/png", records[0].ContentType)
}

func TestProductLogoSetInfersOctetStreamFallback(t *testing.T) {
	var records []recordedRequest
	srv := newRecordingLogoServer(&records, http.StatusOK, "")
	defer srv.Close()

	filePath := filepath.Join(t.TempDir(), "logo.unknownext")
	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0o600))

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newLogoSetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"prod-1", "--file", filePath})
	err := c.Execute()

	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "application/octet-stream", records[0].ContentType)
}

func TestProductLogoSetMissingFileFlag(t *testing.T) {
	var records []recordedRequest
	srv := newRecordingLogoServer(&records, http.StatusOK, "")
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newLogoSetCmd()
	c.SetOut(&buf)
	c.SetErr(&buf)
	c.SetArgs([]string{"prod-1"})
	err := c.Execute()

	require.Error(t, err)
	assert.Empty(t, records, "no HTTP request should be issued when --file is omitted")
}

func TestProductLogoSetUnreadableFile(t *testing.T) {
	var records []recordedRequest
	srv := newRecordingLogoServer(&records, http.StatusOK, "")
	defer srv.Close()

	missingPath := filepath.Join(t.TempDir(), "does-not-exist.png")

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newLogoSetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"prod-1", "--file", missingPath})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), missingPath)
	assert.Empty(t, records, "no HTTP request should be issued when --file is unreadable")
}

func TestProductLogoSetZeroByteFile(t *testing.T) {
	var records []recordedRequest
	srv := newRecordingLogoServer(&records, http.StatusBadRequest, `{"message":"empty file"}`)
	defer srv.Close()

	filePath := filepath.Join(t.TempDir(), "empty.png")
	require.NoError(t, os.WriteFile(filePath, []byte{}, 0o600))

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newLogoSetCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"prod-1", "--file", filePath})
	err := c.Execute()

	require.Error(t, err)
	var apiErr *apierrors.APIError
	require.True(t, errors.As(err, &apiErr), "a zero-byte file's 400 response must surface as *errors.APIError, not a client-side rejection")
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.Len(t, records, 1)
	assert.Equal(t, []byte{}, records[0].Body)
}

func TestProductLogoSetDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	// The exact path/action/resource/body shape newLogoSetCmd's RunE passes
	// to dryrun.Render in the cmd.DryRun() branch. cmd.DryRun() has no
	// exported setter (same constraint documented in
	// cmd/guardrails/budget_rules_delete_test.go TestBudgetRulesDeleteDryRun),
	// so this test pins the dry-run output contract directly rather than
	// exercising the RunE gate end-to-end.
	path := "/v2/api/products/prod-1/logo"
	body := map[string]interface{}{
		"file":         "/tmp/logo.png",
		"bytes":        11,
		"content_type": "image/png",
	}

	err := dryrun.Render(out, "update", "product logo", path, body)

	require.NoError(t, err)
	rendered := buf.String()
	assert.Contains(t, rendered, "Dry run: update product logo")
	assert.Contains(t, rendered, path)
	assert.Contains(t, rendered, "11")
	assert.Contains(t, rendered, "image/png")
	assert.NotContains(t, rendered, "\x89PNG", "dry-run summary must never contain raw file bytes")
	assert.Contains(t, rendered, "No changes were made.")
}

func TestProductLogoSetReportsPathAndSize(t *testing.T) {
	var records []recordedRequest
	srv := newRecordingLogoServer(&records, http.StatusOK, "")
	defer srv.Close()

	fakePNG := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A}
	filePath := filepath.Join(t.TempDir(), "logo.png")
	require.NoError(t, os.WriteFile(filePath, fakePNG, 0o600))

	// Table mode: output contains file path and byte count.
	var tableBuf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&tableBuf, &tableBuf, false, false)

	c := newLogoSetCmd()
	c.SetOut(&tableBuf)
	c.SetArgs([]string{"prod-1", "--file", filePath})
	require.NoError(t, c.Execute())

	assert.Contains(t, tableBuf.String(), filePath)
	assert.Contains(t, tableBuf.String(), fmt.Sprintf("%d bytes", len(fakePNG)))

	// Quiet mode: empty buffer, but the PATCH still occurs.
	records = nil
	var quietBuf bytes.Buffer
	cmd.Output = output.NewWithWriter(&quietBuf, &quietBuf, false, true)

	c2 := newLogoSetCmd()
	c2.SetOut(&quietBuf)
	c2.SetArgs([]string{"prod-1", "--file", filePath})
	require.NoError(t, c2.Execute())

	assert.Empty(t, quietBuf.String())
	assert.Len(t, records, 1, "quiet mode must still issue the PATCH")
}

func TestProductLogoSetIsIdempotent(t *testing.T) {
	var records []recordedRequest
	srv := newRecordingLogoServer(&records, http.StatusOK, "")
	defer srv.Close()

	fakePNG := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A}
	filePath := filepath.Join(t.TempDir(), "logo.png")
	require.NoError(t, os.WriteFile(filePath, fakePNG, 0o600))

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	for i := 0; i < 2; i++ {
		c := newLogoSetCmd()
		c.SetOut(&buf)
		c.SetArgs([]string{"prod-1", "--file", filePath})
		require.NoError(t, c.Execute())
	}

	require.Len(t, records, 2)
	for _, r := range records {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/v2/api/products/prod-1/logo", r.Path)
		assert.Equal(t, fakePNG, r.Body)
	}
}

func TestProductLogoDeleteWithYes(t *testing.T) {
	var records []recordedRequest
	srv := newRecordingLogoServer(&records, http.StatusOK, `{"message":"Deleted"}`)
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newLogoDeleteCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"prod-1", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, http.MethodDelete, records[0].Method)
	assert.Equal(t, "/v2/api/products/prod-1/logo", records[0].Path)
	assert.Contains(t, buf.String(), "Deleted logo for product prod-1.")
}

func TestProductLogoDeleteQuiet(t *testing.T) {
	var records []recordedRequest
	srv := newRecordingLogoServer(&records, http.StatusOK, `{"message":"Deleted"}`)
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, true)

	c := newLogoDeleteCmd()
	c.Flags().Bool("yes", false, "Skip confirmation prompts")
	c.SetOut(&buf)
	c.SetArgs([]string{"prod-1", "--yes"})
	err := c.Execute()

	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Empty(t, buf.String())
}

func TestProductLogoDeleteJSONMode(t *testing.T) {
	var records []recordedRequest
	srv := newRecordingLogoServer(&records, http.StatusOK, `{"message":"Deleted"}`)
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	c := newLogoDeleteCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"prod-1"})
	err := c.Execute()

	require.NoError(t, err)
	require.Len(t, records, 1, "delete should proceed without prompt in JSON mode")
}

func TestProductLogoDeleteDryRun(t *testing.T) {
	var buf bytes.Buffer
	out := output.NewWithWriter(&buf, &buf, false, false)

	// Same decoupled-assertion approach as TestProductLogoSetDryRun: pins
	// the exact path/action/resource/body shape newLogoDeleteCmd's RunE
	// passes to dryrun.Render in the cmd.DryRun() branch.
	path := "/v2/api/products/prod-1/logo"

	err := dryrun.Render(out, "delete", "product logo", path, nil)

	require.NoError(t, err)
	rendered := buf.String()
	assert.Contains(t, rendered, "Dry run: delete product logo")
	assert.Contains(t, rendered, path)
	assert.Contains(t, rendered, "No changes were made.")
}

func TestProductLogoDeleteRepeatSurfacesAPIStatus(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"message":"Deleted"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message":"not found"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c1 := newLogoDeleteCmd()
	c1.Flags().Bool("yes", false, "Skip confirmation prompts")
	c1.SetOut(&buf)
	c1.SetArgs([]string{"prod-1", "--yes"})
	require.NoError(t, c1.Execute())

	c2 := newLogoDeleteCmd()
	c2.Flags().Bool("yes", false, "Skip confirmation prompts")
	c2.SetOut(&buf)
	c2.SetArgs([]string{"prod-1", "--yes"})
	err := c2.Execute()

	require.Error(t, err)
	var apiErr *apierrors.APIError
	require.True(t, errors.As(err, &apiErr), "a repeat delete's 404 must surface unchanged, with no local special-casing")
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

func TestProductLogoCommandsIssueOneRequestEach(t *testing.T) {
	var records []recordedRequest
	srv := newRecordingLogoServer(&records, http.StatusOK, `{"message":"ok"}`)
	defer srv.Close()

	filePath := filepath.Join(t.TempDir(), "logo.png")
	require.NoError(t, os.WriteFile(filePath, []byte{0x89, 'P', 'N', 'G'}, 0o600))

	var setBuf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&setBuf, &setBuf, false, false)

	setCmd := newLogoSetCmd()
	setCmd.SetOut(&setBuf)
	setCmd.SetArgs([]string{"prod-1", "--file", filePath})
	require.NoError(t, setCmd.Execute())

	var deleteBuf bytes.Buffer
	cmd.Output = output.NewWithWriter(&deleteBuf, &deleteBuf, false, false)
	deleteCmd := newLogoDeleteCmd()
	deleteCmd.Flags().Bool("yes", false, "Skip confirmation prompts")
	deleteCmd.SetOut(&deleteBuf)
	deleteCmd.SetArgs([]string{"prod-1", "--yes"})
	require.NoError(t, deleteCmd.Execute())

	require.Len(t, records, 2, "no GET and no PUT anywhere — one request per invocation")
	assert.Equal(t, []string{http.MethodPatch, http.MethodDelete}, []string{records[0].Method, records[1].Method})

	// Each invocation writes exactly one output line (MEDIA-03 ordering
	// probe's single-confirmation-line contract).
	assert.Equal(t, 1, strings.Count(strings.TrimRight(setBuf.String(), "\n"), "\n")+1)
	assert.Equal(t, 1, strings.Count(strings.TrimRight(deleteBuf.String(), "\n"), "\n")+1)
}

func TestProductLogoParentRegistersBothChildren(t *testing.T) {
	logoCmd.ResetCommands()
	initLogo()

	children := logoCmd.Commands()
	names := make([]string, 0, len(children))
	for _, child := range children {
		names = append(names, child.Name())
		assert.Equal(t, "true", child.Annotations["mutating"], "child %q must carry the mutating annotation", child.Name())
	}
	assert.ElementsMatch(t, []string{"set", "delete"}, names)
	assert.Empty(t, logoCmd.Annotations["mutating"], "the logo parent itself must carry no mutating marker")
}
