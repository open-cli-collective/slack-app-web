// Package apptokens implements `slaw app-tokens create` and `revoke`.
package apptokens

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-app-web/internal/cmd/cmdutil"
	"github.com/open-cli-collective/slack-app-web/internal/secretout"
	"github.com/open-cli-collective/slack-app-web/internal/slackweb"
)

// NewCmd returns the app-tokens command group.
func NewCmd(opts *cmdutil.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app-tokens",
		Short: "Generate and revoke app-level tokens",
		Long: `An app-level token (xapp-...) acts for an app across every workspace it is
installed in. Socket Mode needs one with the connections:write scope.`,
	}
	cmd.AddCommand(newCreateCmd(opts), newRevokeCmd(opts))
	return cmd
}

func newCreateCmd(opts *cmdutil.Options) *cobra.Command {
	var name string
	var scopes []string
	var sink secretout.Sink
	create := &cobra.Command{
		Use:   "create <app-id>",
		Short: "Generate an app-level token and deliver it to a file or a command",
		Long: `Generates an app-level token with the given name and scopes on the app's
Basic Information page. Slack shows a new token's value once, so an app that
already lists a token with the name is refused rather than given a second
one; revoke it with 'slaw app-tokens revoke' or pick another name.

The token is delivered, with no trailing newline, to exactly one destination:

  --to-file <path>      a new file with mode 0600 (--overwrite replaces one)
  --to-command <cmd>    a shell command that reads the token on stdin

The token is never written to slaw's own output.`,
		Example: `  slaw app-tokens create A0123456789 --name socket-mode --scope connections:write --to-command 'slck set-credential --key app_token --stdin'
  slaw app-tokens create A0123456789 --name socket-mode --scope connections:write --scope authorizations:read --to-file ./app-token`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			appID, err := cmdutil.AppID(args[0])
			if err != nil {
				return err
			}
			name = strings.TrimSpace(name)
			if name == "" {
				return cmdutil.Usage("--name is required")
			}
			if err := cmdutil.Scopes(scopes); err != nil {
				return err
			}
			if err := cmdutil.CheckSink(sink); err != nil {
				return err
			}
			return opts.WithBrowser(false, func(b cmdutil.Browser) error {
				token, err := b.CreateAppToken(appID, name, scopes)
				if err != nil {
					return err
				}
				if err := opts.DeliverSecret(sink, []byte(token)); err != nil {
					return fmt.Errorf("%q: %w: %w", name, slackweb.ErrAppTokenUnread, err)
				}
				return nil
			})
		},
	}
	create.Flags().StringVarP(&name, "name", "n", "", "Token name, unique among the app's app-level tokens (required)")
	create.Flags().StringArrayVarP(&scopes, "scope", "s", nil, "Scope to grant; repeat for several (at least one, such as connections:write)")
	cmdutil.AddSinkFlags(create, &sink)
	return create
}

func newRevokeCmd(opts *cmdutil.Options) *cobra.Command {
	var name string
	revoke := &cobra.Command{
		Use:   "revoke <app-id>",
		Short: "Revoke an app-level token by name",
		Long: `Revokes the app's app-level token with the given name on its Basic
Information page. Every copy of the token stops working at once. The token's
value is not needed, so this also clears a token whose value was lost.`,
		Example: `  slaw app-tokens revoke A0123456789 --name socket-mode`,
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			appID, err := cmdutil.AppID(args[0])
			if err != nil {
				return err
			}
			name = strings.TrimSpace(name)
			if name == "" {
				return cmdutil.Usage("--name is required")
			}
			return opts.WithBrowser(false, func(b cmdutil.Browser) error {
				if err := b.RevokeAppToken(appID, name); err != nil {
					return err
				}
				fmt.Fprintf(opts.Stdout, "Revoked the app-level token %q of %s\n", name, appID)
				return nil
			})
		},
	}
	revoke.Flags().StringVarP(&name, "name", "n", "", "Name of the token to revoke (required)")
	return revoke
}
