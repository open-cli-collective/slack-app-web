// Package me implements `slaw me`, the scripted health check.
package me

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-app-web/internal/cmd/cmdutil"
	"github.com/open-cli-collective/slack-app-web/internal/slackweb"
)

// NewCmd returns the me command.
func NewCmd(opts *cmdutil.Options) *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Report whether the browser profile is signed in to Slack",
		Long:  "Loads Your Apps headless and reports whether the profile has a Slack session. Exits 3 when it does not.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			dir, err := opts.ResolvedProfileDir()
			if err != nil {
				return err
			}
			b, err := opts.Browser(false)
			if err != nil {
				return err
			}
			defer b.Close()
			ok, err := b.SignedIn()
			if err != nil {
				return err
			}
			signed := "no"
			if ok {
				signed = "yes"
			}
			fmt.Fprintf(opts.Stdout, "Profile: %s\nSigned in: %s\n", dir, signed)
			if !ok {
				return slackweb.ErrSignedOut
			}
			return nil
		},
	}
}
