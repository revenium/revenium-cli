package invoices

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
)

// newDownloadCmd returns the `revenium invoices download <id>` command.
// Endpoint: GET /v2/api/invoices/{url.PathEscape(id)}/download.
func newDownloadCmd() *cobra.Command {
	var outputPath string

	c := &cobra.Command{
		Use:   "download <id>",
		Short: "Download an invoice's binary file (PDF/CSV) to disk",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Download an invoice to the default filename in the current directory
  revenium invoices download inv-123

  # Download an invoice to a specific path
  revenium invoices download inv-123 --output ./my-invoice.pdf`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			path := fmt.Sprintf("/v2/api/invoices/%s/download", url.PathEscape(id))

			data, err := cmd.APIClient.DoDownload(c.Context(), path)
			if err != nil {
				return err
			}

			// Empty-payload guard: a 2xx response with zero bytes is not a
			// valid invoice download. Fail before any filesystem write so no
			// file is ever created for a non-invoice payload.
			if len(data) == 0 {
				return fmt.Errorf("invoice %s: download returned no data, no file was written", id)
			}

			dest := "invoice-" + sanitizeFilenameID(id) + defaultExtension(data)
			if c.Flags().Changed("output") {
				dest = outputPath
				// WR-05: this local --output means "destination path," not
				// the persistent root --output ("json"/"table" format) that
				// every other command uses. A value that looks like a format
				// token is very likely a user reaching for that convention
				// out of habit, so warn rather than silently writing a file
				// literally named "json" or "table".
				if outputPath == "json" || outputPath == "table" {
					fmt.Fprintf(c.ErrOrStderr(), "warning: --output here means destination file path, not output format; writing to file named %q. Use --json for JSON output.\n", outputPath)
				}
			}

			// WR-06: invoices download is not a mutating command, so it never
			// routes through resource.ConfirmDelete-style prompting, and
			// os.WriteFile below would otherwise clobber a pre-existing file
			// at dest with no confirmation at all — a gap inconsistent with
			// the extra care already given to invoices as financial
			// documents (0600 permissions, no implicit directory creation).
			if err := confirmOverwrite(c, dest); err != nil {
				return err
			}

			// Invoices are financial documents, so 0600 rather than 0644 keeps
			// the downloaded file from being group/world readable (T-07-03).
			// No parent directories are created implicitly for either the
			// derived name or an explicit --output value: a bad path fails
			// loudly via the os.WriteFile error below (T-07-01).
			if err := os.WriteFile(dest, data, 0o600); err != nil {
				return err
			}

			if cmd.Output.IsJSON() {
				return cmd.Output.RenderJSON(map[string]interface{}{
					"id":    id,
					"path":  dest,
					"bytes": len(data),
				})
			}
			if !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.OutOrStdout(), "Downloaded invoice %s to %s (%d bytes).\n", id, dest, len(data))
			}
			return nil
		},
	}

	// This local --output flag intentionally shadows rootCmd's persistent
	// --output (format) flag: cobra's flag merge keeps this local definition,
	// so root's outputFormat variable stays empty on this command and --json
	// remains the way to select JSON output here.
	c.Flags().StringVarP(&outputPath, "output", "o", "", "Destination file path. Defaults to ./invoice-<id>.pdf in the current directory; the API may return CSV, so choose a different extension if needed.")

	return c
}

// sanitizeFilenameID replaces path separators in id with "-" before it is
// used to build the derived default filename. internal/validate.ResourceID
// rejects "../" and percent-encoding but permits a bare "/" or "\", so
// without this an id such as "a/b" would steer the write into a
// non-existent (or unintended) subdirectory (T-07-02). The HTTP request
// path continues to use url.PathEscape on the unsanitized id — only the
// on-disk filename is affected.
func sanitizeFilenameID(id string) string {
	r := strings.NewReplacer("/", "-", "\\", "-")
	return r.Replace(id)
}

// confirmOverwrite prompts before clobbering a pre-existing file at dest,
// mirroring the --yes/JSON-mode/non-TTY bypass behavior of
// resource.ConfirmDelete (WR-06). Returns nil if the write may proceed, or
// an error describing why it was aborted.
func confirmOverwrite(c *cobra.Command, dest string) error {
	if cmd.YesMode() || cmd.Output.IsJSON() {
		return nil
	}
	if _, statErr := os.Stat(dest); statErr != nil {
		// Missing (or otherwise unreadable): nothing to overwrite, or let
		// os.WriteFile itself surface the real problem below.
		return nil
	}
	if !term.IsTerminal(os.Stdin.Fd()) {
		return nil
	}
	fmt.Fprintf(c.ErrOrStderr(), "File %s already exists. Overwrite? [y/N] ", dest)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if scanErr := scanner.Err(); scanErr != nil {
			return scanErr
		}
		return fmt.Errorf("aborted: %s already exists", dest)
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if answer != "y" && answer != "yes" {
		return fmt.Errorf("aborted: %s already exists", dest)
	}
	return nil
}

// defaultExtension sniffs data's content to pick a default file extension
// for the derived filename (WR-04). DoDownload returns only raw bytes, with
// no access to the response's Content-Type header, so this is content-based
// rather than header-based: without it, a CSV invoice downloaded with no
// --output would be written as "invoice-<id>.pdf" while actually containing
// CSV text, misleading any downstream tooling that dispatches on extension.
func defaultExtension(data []byte) string {
	contentType := http.DetectContentType(data)
	switch {
	case strings.HasPrefix(contentType, "application/pdf"):
		return ".pdf"
	case strings.HasPrefix(contentType, "text/"):
		// http.DetectContentType has no distinct signature for CSV — plain
		// delimited text is sniffed as text/plain — so any non-PDF,
		// text-like payload is treated as CSV, matching the two formats the
		// API is documented to return.
		return ".csv"
	default:
		return ".pdf"
	}
}
