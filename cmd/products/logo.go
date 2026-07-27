package products

import (
	"fmt"
	"mime"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/revenium/revenium-cli/cmd"
	"github.com/revenium/revenium-cli/internal/dryrun"
	"github.com/revenium/revenium-cli/internal/resource"
)

// logoCmd is the parent logo subcommand under products.
var logoCmd = &cobra.Command{
	Use:   "logo",
	Short: "Manage a product's logo image",
	Example: `  # Upload a logo
  revenium products logo set prod-123 --file ./logo.png

  # Delete a logo
  revenium products logo delete prod-123`,
}

// initLogo registers logo subcommands. Called from products.go init() to
// avoid file-ordering issues with Go's init() functions.
func initLogo() {
	logoCmd.AddCommand(newLogoSetCmd())
	logoCmd.AddCommand(newLogoDeleteCmd())
}

// newLogoSetCmd returns the `revenium products logo set <id>` command.
// Endpoint: PATCH /v2/api/products/{url.PathEscape(id)}/logo, a
// multipart/form-data body with the file under a "file" field (D-02 as
// corrected by the 07-03 live-API checkpoint; D-06, D-07 unaffected) — never
// a GET-then-PUT round trip.
func newLogoSetCmd() *cobra.Command {
	var filePath string

	c := &cobra.Command{
		Use:         "set <id>",
		Short:       "Upload a product's logo image",
		Annotations: map[string]string{"mutating": "true"},
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Upload a logo
  revenium products logo set prod-123 --file ./logo.png`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			path := fmt.Sprintf("/v2/api/products/%s/logo", url.PathEscape(id))

			// Per D-07, the only client-side validation is that the file is
			// readable — no mime/size checks. A zero-byte file is accepted
			// and forwarded; the API is trusted to reject unsupported
			// formats. The read happens before the dry-run branch below so
			// the summary can report a real byte count; this is a read-only
			// local operation, so the dry-run contract of making no remote
			// changes still holds.
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", filePath, err)
			}

			contentType := mime.TypeByExtension(filepath.Ext(filePath))
			if contentType == "" {
				contentType = "application/octet-stream"
			}

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "update", "product logo", path, map[string]interface{}{
					"file":         filePath,
					"bytes":        len(data),
					"content_type": contentType,
				})
			}

			if err := cmd.APIClient.DoUploadBinary(c.Context(), path, filepath.Base(filePath), data, contentType); err != nil {
				return err
			}

			if cmd.Output.IsJSON() {
				return cmd.Output.RenderJSON(map[string]interface{}{
					"id":           id,
					"file":         filePath,
					"bytes":        len(data),
					"content_type": contentType,
				})
			}
			if !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.OutOrStdout(), "Uploaded logo for product %s from %s (%d bytes, %s).\n", id, filePath, len(data), contentType)
			}
			return nil
		},
	}

	c.Flags().StringVar(&filePath, "file", "", "Path to the logo image file to upload")
	_ = c.MarkFlagRequired("file")

	return c
}

// newLogoDeleteCmd returns the `revenium products logo delete <id>` command.
// Endpoint: DELETE /v2/api/products/{url.PathEscape(id)}/logo — a plain,
// body-less call via cmd.APIClient.Do, not DoUploadBinary. It follows the
// same mutating/dry-run/confirmation shape as cmd/products/delete.go so
// logo delete is never special-cased as low stakes (D-08).
func newLogoDeleteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:         "delete <id>",
		Short:       "Delete a product's logo image",
		Annotations: map[string]string{"mutating": "true"},
		Args:        cobra.MatchAll(cobra.ExactArgs(1), cmd.ValidResourceID),
		Example: `  # Delete a logo (with confirmation)
  revenium products logo delete prod-123

  # Delete without confirmation
  revenium products logo delete prod-123 --yes`,
		RunE: func(c *cobra.Command, args []string) error {
			id := args[0]
			path := fmt.Sprintf("/v2/api/products/%s/logo", url.PathEscape(id))

			if cmd.DryRun() {
				return dryrun.Render(cmd.Output, "delete", "product logo", path, nil)
			}

			yes, _ := c.Flags().GetBool("yes")

			ok, err := resource.ConfirmDelete("product logo", id, yes, cmd.Output.IsJSON())
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}

			// Whatever status the API returns for a repeat delete (e.g. a
			// second call after the logo is already gone) is surfaced
			// unchanged by mapHTTPError — no local special-casing of a
			// synthesized success (MEDIA-03 idempotency probe).
			if err := cmd.APIClient.Do(c.Context(), "DELETE", path, nil, nil); err != nil {
				return err
			}

			if !cmd.Output.IsQuiet() {
				fmt.Fprintf(c.OutOrStdout(), "Deleted logo for product %s.\n", id)
			}
			return nil
		},
	}

	return c
}
