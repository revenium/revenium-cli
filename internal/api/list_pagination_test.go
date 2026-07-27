package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pagedServer serves a Spring HATEOAS paged collection of totalItems records,
// honouring the requested page/size and capping size the way the real API does.
// It records every request path it served.
func pagedServer(t *testing.T, totalItems, serverMaxSize int, gotPaths *[]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotPaths = append(*gotPaths, r.URL.String())

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		size, err := strconv.Atoi(r.URL.Query().Get("size"))
		if err != nil || size <= 0 {
			size = 20
		}
		if size > serverMaxSize {
			size = serverMaxSize // the cap that produced the original bug report
		}

		start := page * size
		end := start + size
		if start > totalItems {
			start = totalItems
		}
		if end > totalItems {
			end = totalItems
		}

		items := make([]map[string]interface{}, 0, end-start)
		for i := start; i < end; i++ {
			items = append(items, map[string]interface{}{"id": fmt.Sprintf("rec-%d", i)})
		}

		totalPages := (totalItems + size - 1) / size
		require.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
			"_embedded": map[string]interface{}{"aIMetricResourceList": items},
			"page": map[string]interface{}{
				"size":          size,
				"totalElements": totalItems,
				"totalPages":    totalPages,
				"number":        page,
			},
		}))
	}))
}

// TestDoListFetchAllPagesBeyondServerCap is the client-side regression guard for
// the reported bug: a month-long metrics query returned exactly 100 records
// because the server caps a page at 100 and the client never asked for page 2.
func TestDoListFetchAllPagesBeyondServerCap(t *testing.T) {
	var paths []string
	srv := pagedServer(t, 250, 100, &paths)
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", "", "", "", false)

	var got []map[string]interface{}
	require.NoError(t, c.DoList(context.Background(), "/metrics",
		ListOptions{Page: -1, PageSize: -1, FetchAll: true}, &got))

	assert.Len(t, got, 250, "must aggregate every page, not stop at the server's per-page cap")
	assert.Len(t, paths, 3, "expected 3 requests for 250 records at 100/page")
	assert.Equal(t, "rec-0", got[0]["id"])
	assert.Equal(t, "rec-249", got[249]["id"])
}

// TestDoListPageSizeDoesNotDisableFetchAll covers the exact flag combination
// from the bug report (--page-size 500). Requesting a larger batch must not
// collapse the request into a single page.
func TestDoListPageSizeDoesNotDisableFetchAll(t *testing.T) {
	var paths []string
	srv := pagedServer(t, 250, 100, &paths)
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", "", "", "", false)

	var got []map[string]interface{}
	require.NoError(t, c.DoList(context.Background(), "/metrics",
		ListOptions{Page: -1, PageSize: 500, FetchAll: true}, &got))

	assert.Len(t, got, 250, "explicit --page-size must not truncate the result set")
	assert.Contains(t, paths[0], "size=500", "requested batch size should be forwarded to the server")
}

// TestDoListPageSizeUsedAsBatchSize verifies PageSize selects the batch size for
// an aggregate fetch rather than capping the total.
func TestDoListPageSizeUsedAsBatchSize(t *testing.T) {
	var paths []string
	srv := pagedServer(t, 100, 1000, &paths)
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", "", "", "", false)

	var got []map[string]interface{}
	require.NoError(t, c.DoList(context.Background(), "/metrics",
		ListOptions{Page: -1, PageSize: 25, FetchAll: true}, &got))

	assert.Len(t, got, 100)
	assert.Len(t, paths, 4, "100 records at an explicit 25/page should take 4 requests")
}

// TestDoListExplicitPageFetchesOnlyThatPage confirms the deliberate opt-out
// still works: --page means exactly one page.
func TestDoListExplicitPageFetchesOnlyThatPage(t *testing.T) {
	var paths []string
	srv := pagedServer(t, 250, 100, &paths)
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", "", "", "", false)

	var got []map[string]interface{}
	require.NoError(t, c.DoList(context.Background(), "/metrics",
		ListOptions{Page: 1, PageSize: 100, FetchAll: false}, &got))

	assert.Len(t, got, 100, "an explicit page must return just that page")
	assert.Len(t, paths, 1, "an explicit page must issue exactly one request")
	assert.Equal(t, "rec-100", got[0]["id"], "should be the second page of records")
}

// TestDoListPlainArrayResponseStillWorks guards the non-paginated path: an
// endpoint returning a bare JSON array has no totalPages and must not loop.
func TestDoListPlainArrayResponseStillWorks(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.NoError(t, json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "a"}, {"id": "b"},
		}))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key", "", "", "", false)

	var got []map[string]interface{}
	require.NoError(t, c.DoList(context.Background(), "/plain",
		ListOptions{Page: -1, PageSize: -1, FetchAll: true}, &got))

	assert.Len(t, got, 2)
	assert.Equal(t, 1, calls, "a plain array response must not trigger pagination")
}
