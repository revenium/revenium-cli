package invoices

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/api"
	apierrors "github.com/revenium/revenium-cli/internal/errors"
	"github.com/revenium/revenium-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDownloadInvoiceWritesFile is the end-to-end tracer test: a served
// non-JSON payload (PDF magic-byte prefix) written via --output round-trips
// byte-for-byte, and the resolved path is reported on stdout.
func TestDownloadInvoiceWritesFile(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4\nfake-invoice-content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v2/api/invoices/inv-1/download", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write(pdfBytes)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	dest := filepath.Join(t.TempDir(), "invoice-out.pdf")

	c := newDownloadCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"inv-1", "--output", dest})
	err := c.Execute()

	require.NoError(t, err)
	got, readErr := os.ReadFile(dest)
	require.NoError(t, readErr)
	assert.Equal(t, pdfBytes, got)
	assert.Contains(t, buf.String(), dest)
}

// TestDownloadInvoiceDerivesDefaultFilename asserts that with --output
// absent, the file lands at ./invoice-<id>.pdf in the process working
// directory, and the confirmation line is the only thing written to stdout.
func TestDownloadInvoiceDerivesDefaultFilename(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4\ndefault-filename-content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(pdfBytes)
	}))
	defer srv.Close()

	t.Chdir(t.TempDir())

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDownloadCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"inv-1"})
	err := c.Execute()

	require.NoError(t, err)
	got, readErr := os.ReadFile("invoice-inv-1.pdf")
	require.NoError(t, readErr)
	assert.Equal(t, pdfBytes, got)
	assert.Equal(t, fmt.Sprintf("Downloaded invoice inv-1 to invoice-inv-1.pdf (%d bytes).\n", len(pdfBytes)), buf.String())
}

// TestDownloadInvoiceSanitizesIDInFilename asserts a "/" in the resource id
// is replaced before it is used to build the derived filename (T-07-02),
// while the HTTP request path still carries the percent-escaped id.
func TestDownloadInvoiceSanitizesIDInFilename(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4\nsanitized-id-content")
	var gotEscapedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// r.URL.Path is always the decoded form ("a/b"); the percent-escaped
		// id shows up in the escaped path / raw request line instead.
		gotEscapedPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
		w.Write(pdfBytes)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	c := newDownloadCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"a/b"})
	err := c.Execute()

	require.NoError(t, err)
	assert.Equal(t, "/v2/api/invoices/a%2Fb/download", gotEscapedPath)

	entries, readErr := os.ReadDir(".")
	require.NoError(t, readErr)
	require.Len(t, entries, 1)
	assert.Equal(t, "invoice-a-b.pdf", entries[0].Name())

	got, readErr := os.ReadFile("invoice-a-b.pdf")
	require.NoError(t, readErr)
	assert.Equal(t, pdfBytes, got)
}

// TestDownloadInvoiceFileMode asserts the written file is 0600, not the
// default 0644, since invoices are financial documents (T-07-03).
func TestDownloadInvoiceFileMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("%PDF-1.4\nmode-check"))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	dest := filepath.Join(t.TempDir(), "invoice-mode.pdf")

	c := newDownloadCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"inv-1", "--output", dest})
	err := c.Execute()
	require.NoError(t, err)

	info, statErr := os.Stat(dest)
	require.NoError(t, statErr)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

// TestDownloadInvoiceErrorWritesNoFile asserts a non-2xx response returns
// the same *errors.APIError every other client method returns, and that no
// file is created at the destination path.
func TestDownloadInvoiceErrorWritesNoFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message":"not found"}`)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	dest := filepath.Join(t.TempDir(), "invoice-404.pdf")

	c := newDownloadCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"inv-1", "--output", dest})
	err := c.Execute()

	require.Error(t, err)
	var apiErr *apierrors.APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)

	_, statErr := os.Stat(dest)
	assert.True(t, os.IsNotExist(statErr), "no file should be written on a non-2xx response")
}

// TestDownloadInvoiceEmptyBodyWritesNoFile asserts a 200 response with a
// zero-length body is treated as an error, not a successful (empty) file.
func TestDownloadInvoiceEmptyBodyWritesNoFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	dest := filepath.Join(t.TempDir(), "invoice-empty.pdf")

	c := newDownloadCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"inv-1", "--output", dest})
	err := c.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "inv-1")

	_, statErr := os.Stat(dest)
	assert.True(t, os.IsNotExist(statErr), "no file should be written for an empty body")
}

// TestDownloadInvoiceInterruptedBodyWritesNoFile asserts a transfer that is
// interrupted mid-body (server claims more bytes than it sends, then closes
// the connection) returns an error and writes no file (MEDIA-02 concurrency
// probe).
func TestDownloadInvoiceInterruptedBodyWritesNoFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("expected ResponseWriter to support hijacking")
		}
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			t.Fatalf("hijack failed: %v", err)
		}
		defer conn.Close()
		// Claim a Content-Length far larger than the bytes actually sent,
		// then close the connection early so io.ReadAll observes an
		// unexpected EOF.
		fmt.Fprint(bufrw, "HTTP/1.1 200 OK\r\nContent-Length: 100000\r\n\r\nshort-body")
		bufrw.Flush()
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	dest := filepath.Join(t.TempDir(), "invoice-interrupted.pdf")

	c := newDownloadCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"inv-1", "--output", dest})
	err := c.Execute()

	require.Error(t, err)

	_, statErr := os.Stat(dest)
	assert.True(t, os.IsNotExist(statErr), "no file should be written for an interrupted transfer")
}

// TestDownloadInvoiceIsIdempotent asserts re-running the download command
// for the same id against the same server response produces a byte-identical
// file at the same path, with no additional side effect (MEDIA-02).
func TestDownloadInvoiceIsIdempotent(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4\nidempotent-content")
	var requestCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write(pdfBytes)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "invoice-idempotent.pdf")

	for i := 0; i < 2; i++ {
		var buf bytes.Buffer
		cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
		cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

		c := newDownloadCmd()
		c.SetOut(&buf)
		c.SetArgs([]string{"inv-1", "--output", dest})
		err := c.Execute()
		require.NoError(t, err)
	}

	assert.Equal(t, 2, requestCount)
	got, readErr := os.ReadFile(dest)
	require.NoError(t, readErr)
	assert.Equal(t, pdfBytes, got)
}

// TestDownloadInvoiceJSONMode asserts JSON mode renders the resolved path
// and byte count as JSON.
func TestDownloadInvoiceJSONMode(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4\njson-mode-content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(pdfBytes)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, true, false)

	dest := filepath.Join(t.TempDir(), "invoice-json.pdf")

	c := newDownloadCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"inv-1", "--output", dest})
	err := c.Execute()
	require.NoError(t, err)

	assert.Contains(t, buf.String(), dest)
	assert.Contains(t, buf.String(), fmt.Sprintf("%d", len(pdfBytes)))
	assert.Contains(t, buf.String(), "inv-1")
}

// TestDownloadInvoiceQuietMode asserts quiet mode still writes the file but
// produces no stdout output.
func TestDownloadInvoiceQuietMode(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4\nquiet-mode-content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(pdfBytes)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, true)

	dest := filepath.Join(t.TempDir(), "invoice-quiet.pdf")

	c := newDownloadCmd()
	c.SetOut(&buf)
	c.SetArgs([]string{"inv-1", "--output", dest})
	err := c.Execute()
	require.NoError(t, err)

	got, readErr := os.ReadFile(dest)
	require.NoError(t, readErr)
	assert.Equal(t, pdfBytes, got)
	assert.Empty(t, buf.String())
}

// TestDownloadInvoiceLocalOutputFlagShadowsPersistent proves the local
// --output destination flag wins cobra's flag merge over a persistent
// --output (format) flag defined on a parent command, mirroring rootCmd's
// real persistent --output flag.
func TestDownloadInvoiceLocalOutputFlagShadowsPersistent(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4\nshadow-content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(pdfBytes)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	cmd.APIClient = api.NewClient(srv.URL, "test-key", "", "", "", false)
	cmd.Output = output.NewWithWriter(&buf, &buf, false, false)

	dest := filepath.Join(t.TempDir(), "invoice-shadow.pdf")

	parent := &cobra.Command{Use: "revenium"}
	var format string
	parent.PersistentFlags().StringVarP(&format, "output", "o", "table", "Output format")

	child := newDownloadCmd()
	parent.AddCommand(child)

	parent.SetOut(&buf)
	parent.SetArgs([]string{"download", "inv-1", "--output", dest})
	err := parent.Execute()

	require.NoError(t, err)
	got, readErr := os.ReadFile(dest)
	require.NoError(t, readErr)
	assert.Equal(t, pdfBytes, got)
}
