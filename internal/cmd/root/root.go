// Package root assembles the slaw command tree.
package root

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-app-web/internal/cmd/apps"
	"github.com/open-cli-collective/slack-app-web/internal/cmd/bottokens"
	"github.com/open-cli-collective/slack-app-web/internal/cmd/browsercmd"
	"github.com/open-cli-collective/slack-app-web/internal/cmd/cmdutil"
	"github.com/open-cli-collective/slack-app-web/internal/cmd/configtokens"
	"github.com/open-cli-collective/slack-app-web/internal/cmd/icons"
	"github.com/open-cli-collective/slack-app-web/internal/cmd/login"
	"github.com/open-cli-collective/slack-app-web/internal/cmd/me"
	"github.com/open-cli-collective/slack-app-web/internal/exitcode"
	"github.com/open-cli-collective/slack-app-web/internal/secretout"
	"github.com/open-cli-collective/slack-app-web/internal/version"
)

// NewCmd builds the root command on opts.
func NewCmd(opts *cmdutil.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slaw",
		Short: "Do the Slack app steps that have no API, in a headless browser",
		Long: `slaw drives api.slack.com in a persistent, headless Chromium profile for the
Slack app steps the Web API does not offer: generating an app configuration
token, installing or reinstalling an app, reading its bot token, and
uploading its icon. Manage manifests through the apps.manifest.* API instead.

Sign in once with 'slaw login'. Tokens are written only to a pipe or a file,
never to a terminal, and slaw never logs or screenshots a page.`,
		Version:       fmt.Sprintf("%s (commit %s, built %s)", version.Version, version.Commit, version.Date),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().StringVar(&opts.Profile, "profile", opts.Profile, "Named browser profile, one per Slack account (env SLAW_PROFILE)")
	cmd.PersistentFlags().StringVar(&opts.ProfileDir, "profile-dir", opts.ProfileDir, "Browser profile directory; overrides --profile (env SLAW_PROFILE_DIR)")
	cmd.PersistentFlags().DurationVar(&opts.Timeout, "timeout", opts.Timeout, "How long to wait for each page element")
	cmd.AddCommand(
		login.NewCmd(opts),
		me.NewCmd(opts),
		apps.NewCmd(opts),
		configtokens.NewCmd(opts),
		bottokens.NewCmd(opts),
		icons.NewCmd(opts),
		browsercmd.NewCmd(opts),
		&cobra.Command{
			Use:   "version",
			Short: "Print the slaw version",
			Args:  cobra.NoArgs,
			Run: func(_ *cobra.Command, _ []string) {
				fmt.Fprintf(opts.Stdout, "slaw %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
			},
		},
	)
	return cmd
}

// Execute runs slaw and returns the process exit code.
func Execute() int {
	opts := cmdutil.Default()
	cmd := NewCmd(opts)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(opts.Stderr, "Error:", secretout.ErrorText(err))
		code, hint := cmdutil.Classify(err)
		if hint != "" {
			fmt.Fprintln(opts.Stderr, "Next:", hint)
		}
		if code == exitcode.Generic && isUsage(err) {
			code = exitcode.Usage
		}
		return code
	}
	return 0
}
