// Package login implements `slaw login`, the only command that shows a
// browser window.
package login

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-app-web/internal/cmd/cmdutil"
)

// NewCmd returns the login command.
func NewCmd(opts *cmdutil.Options) *cobra.Command {
	var workspace string
	var wait time.Duration
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in to Slack in a browser window and keep the session in the profile",
		Long: `Opens a browser window at Slack's sign-in page and waits for you to sign in.
The session is kept in the browser profile, so every other command runs
headless against it. Run it again when a command reports the profile is not
signed in.`,
		Example: `  slaw login
  slaw login --workspace acme
  slaw --profile work login --wait 15m`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			ws, err := cmdutil.Workspace(workspace)
			if err != nil {
				return err
			}
			return opts.WithBrowser(true, func(b cmdutil.Browser) error {
				fmt.Fprintln(opts.Stderr, "A browser window is open at Slack's sign-in page. Sign in there; this command finishes when you are signed in.")
				if err := b.Login(ws, wait); err != nil {
					return err
				}
				dir, _ := opts.ResolvedProfileDir()
				fmt.Fprintf(opts.Stdout, "Signed in. Profile: %s\n", dir)
				return nil
			})
		},
	}
	cmd.Flags().StringVarP(&workspace, "workspace", "w", "", "Workspace subdomain to sign in to (acme for acme.slack.com)")
	cmd.Flags().DurationVar(&wait, "wait", 10*time.Minute, "How long to wait for the sign-in")
	return cmd
}
