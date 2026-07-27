package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/internal/output"
)

// newListFlagCmd builds a throwaway command carrying the standard list flags,
// with the given args already parsed.
func newListFlagCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	c := &cobra.Command{Use: "x", RunE: func(*cobra.Command, []string) error { return nil }}
	AddListFlags(c)
	c.SetArgs(args)
	c.SetOut(&bytes.Buffer{})
	c.SetErr(&bytes.Buffer{})
	if err := c.Execute(); err != nil {
		t.Fatalf("parsing %v: %v", args, err)
	}
	return c
}

func setOutputMode(t *testing.T, jsonMode bool) {
	t.Helper()
	prev := Output
	t.Cleanup(func() { Output = prev })
	var buf bytes.Buffer
	Output = output.NewWithWriter(&buf, &buf, jsonMode, false)
}

// TestListOptsFetchAllRegardlessOfOutputMode is the regression guard for the
// silent-truncation bug: `metrics ai --output json --page-size 500` returned
// only the first page (100 records) of a month-long query, with no indication
// the result was partial. JSON mode must aggregate all pages exactly like table
// mode, and --page-size must not opt out of aggregation.
func TestListOptsFetchAllRegardlessOfOutputMode(t *testing.T) {
	tests := []struct {
		name         string
		jsonMode     bool
		args         []string
		wantFetchAll bool
		wantPage     int
		wantPageSize int
	}{
		{
			name:         "json with explicit page-size still fetches all",
			jsonMode:     true,
			args:         []string{"--page-size", "500"},
			wantFetchAll: true,
			wantPage:     -1,
			wantPageSize: 500,
		},
		{
			name:         "json with no paging flags fetches all",
			jsonMode:     true,
			wantFetchAll: true,
			wantPage:     -1,
			wantPageSize: -1,
		},
		{
			name:         "table with explicit page-size still fetches all",
			jsonMode:     false,
			args:         []string{"--page-size", "500"},
			wantFetchAll: true,
			wantPage:     -1,
			wantPageSize: 500,
		},
		{
			name:         "table with no paging flags fetches all",
			jsonMode:     false,
			wantFetchAll: true,
			wantPage:     -1,
			wantPageSize: -1,
		},
		{
			name:         "explicit page opts out of fetch-all in json mode",
			jsonMode:     true,
			args:         []string{"--page", "2"},
			wantFetchAll: false,
			wantPage:     2,
			wantPageSize: -1,
		},
		{
			name:         "explicit page opts out of fetch-all in table mode",
			jsonMode:     false,
			args:         []string{"--page", "0"},
			wantFetchAll: false,
			wantPage:     0,
			wantPageSize: -1,
		},
		{
			name:         "explicit page plus page-size is a single sized page",
			jsonMode:     true,
			args:         []string{"--page", "3", "--page-size", "50"},
			wantFetchAll: false,
			wantPage:     3,
			wantPageSize: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setOutputMode(t, tt.jsonMode)
			opts := ListOptsFromFlags(newListFlagCmd(t, tt.args...))

			if opts.FetchAll != tt.wantFetchAll {
				t.Errorf("FetchAll = %v, want %v", opts.FetchAll, tt.wantFetchAll)
			}
			if opts.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", opts.Page, tt.wantPage)
			}
			if opts.PageSize != tt.wantPageSize {
				t.Errorf("PageSize = %d, want %d", opts.PageSize, tt.wantPageSize)
			}
		})
	}
}

// TestListOptsUnsetFlagsAreNotTreatedAsValues guards the default-value trap:
// --page and --page-size carry non-negative cobra defaults (0 and 20), so
// reading them without checking Changed() would make every command look
// explicitly paged and silently disable aggregation.
func TestListOptsUnsetFlagsAreNotTreatedAsValues(t *testing.T) {
	setOutputMode(t, false)
	opts := ListOptsFromFlags(newListFlagCmd(t))

	if opts.Page != -1 {
		t.Errorf("unset --page leaked its default: Page = %d, want -1", opts.Page)
	}
	if opts.PageSize != -1 {
		t.Errorf("unset --page-size leaked its default: PageSize = %d, want -1", opts.PageSize)
	}
	if !opts.FetchAll {
		t.Error("FetchAll = false with no paging flags set, want true")
	}
}
