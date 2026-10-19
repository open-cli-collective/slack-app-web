// Package browsercmd implements `slaw browser install`.
package browsercmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-app-web/internal/cmd/cmdutil"
	"github.com/open-cli-collective/slack-app-web/internal/slackweb"
)

// NewCmd returns the browser command group.
func NewCmd(opts *cmdutil.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browser",
		Short: "Manage the browser slaw drives",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Download the Playwright driver and Chromium",
		Long:  "Downloads the Playwright driver and Chromium into the user cache. Run it once after installing slaw, and again after an upgrade asks for it.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := slackweb.InstallBrowser(opts.Stderr); err != nil {
				return err
			}
			fmt.Fprintln(opts.Stdout, "Installed the browser driver and Chromium")
			return nil
		},
	})
	return cmd
}
