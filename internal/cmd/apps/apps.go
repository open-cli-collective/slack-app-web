// Package apps implements `slaw apps`: listing the account's apps and
// installing one to its workspace.
package apps

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-app-web/internal/cmd/cmdutil"
	"github.com/open-cli-collective/slack-app-web/internal/output"
	"github.com/open-cli-collective/slack-app-web/internal/slackweb"
)

// NewCmd returns the apps command group.
func NewCmd(opts *cmdutil.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps",
		Short: "List and install Slack apps",
	}
	cmd.AddCommand(newListCmd(opts), newInstallCmd(opts))
	return cmd
}

func newListCmd(opts *cmdutil.Options) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List the apps this account can manage",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return opts.WithBrowser(false, func(b cmdutil.Browser) error {
				apps, err := b.ListApps()
				if err != nil {
					return err
				}
				rows := make([][]string, 0, len(apps))
				for _, a := range apps {
					rows = append(rows, []string{a.ID, a.Name, a.Workspace})
				}
				output.Table(opts.Stdout, []string{"ID", "NAME", "WORKSPACE"}, rows)
				return nil
			})
		},
	}
}

func newInstallCmd(opts *cmdutil.Options) *cobra.Command {
	var reinstall bool
	cmd := &cobra.Command{
		Use:   "install <app-id>",
		Short: "Install an app to its workspace, or reinstall it",
		Long: `Installs the app to its workspace and approves the permission request.
An app that is already installed is left as it is unless --reinstall is set;
reinstall after changing an app's scopes so they take effect. Where the
workspace requires admin approval, this submits the request and exits 1.`,
		Example: `  slaw apps install A0123456789
  slaw apps install A0123456789 --reinstall`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			appID, err := cmdutil.AppID(args[0])
			if err != nil {
				return err
			}
			return opts.WithBrowser(false, func(b cmdutil.Browser) error {
				state, acted, err := b.Install(appID, reinstall)
				if err != nil {
					return err
				}
				switch {
				case state.PendingApproval:
					return fmt.Errorf("%s: %w", appID, slackweb.ErrPendingApproval)
				case !acted:
					fmt.Fprintf(opts.Stdout, "%s is already installed\n", appID)
					if state.ReinstallNeeded {
						fmt.Fprintf(opts.Stderr, "Its scopes changed since the last install: run with --reinstall for them to take effect.\n")
					}
				case !state.Installed:
					return &slackweb.UpstreamError{Err: fmt.Errorf("%s does not show as installed after the install", appID)}
				case reinstall:
					fmt.Fprintf(opts.Stdout, "Reinstalled %s\n", appID)
				default:
					fmt.Fprintf(opts.Stdout, "Installed %s\n", appID)
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&reinstall, "reinstall", false, "Reinstall an app that is already installed")
	return cmd
}
