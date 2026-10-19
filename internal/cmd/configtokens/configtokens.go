// Package configtokens implements `slaw config-tokens`.
package configtokens

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-app-web/internal/cmd/cmdutil"
	"github.com/open-cli-collective/slack-app-web/internal/secretout"
	"github.com/open-cli-collective/slack-app-web/internal/slackweb"
)

const contract = `The pair is delivered as one JSON object:

  {"token": "xoxe.xoxp-...", "refresh_token": "xoxe-...", "exp": 1700000000}

to exactly one destination:

  --to-file <path>      a new file with mode 0600 (--overwrite replaces one)
  --to-command <cmd>    a shell command that reads the pair on stdin

The field names are those of Slack's tooling.tokens.rotate. exp is unix
seconds; the access token lasts 12 hours and the refresh token renews it.
The pair is never written to slaw's own output.`

// NewCmd returns the config-tokens command group.
func NewCmd(opts *cmdutil.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config-tokens",
		Short: "Generate or read app configuration tokens",
		Long: `An app configuration token is the credential Slack's apps.manifest.* API
takes. Slack keeps one per workspace.`,
	}
	cmd.AddCommand(newCreateCmd(opts), newGetCmd(opts))
	return cmd
}

func newCreateCmd(opts *cmdutil.Options) *cobra.Command {
	var team string
	var replace bool
	var sink secretout.Sink
	create := &cobra.Command{
		Use:   "create",
		Short: "Generate an app configuration token pair and deliver it to a file or a command",
		Long: `Generates an app configuration token for a workspace. A workspace that
already has one is refused unless --replace is set, which deletes the
existing token first and so invalidates every copy of it.

` + contract,
		Example: `  slaw config-tokens create --to-command '<command that stores the pair from stdin>'
  slaw config-tokens create --team "Example Co" --replace --to-file ./pair.json`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return emit(opts, sink, func(b cmdutil.Browser) (slackweb.ConfigToken, error) {
				return b.CreateConfigToken(team, replace, opts.Now())
			})
		},
	}
	create.Flags().StringVarP(&team, "team", "t", "", "Workspace name as the Your Apps page shows it; needed only when the account has several")
	create.Flags().BoolVar(&replace, "replace", false, "Delete the workspace's existing token first")
	cmdutil.AddSinkFlags(create, &sink)
	return create
}

func newGetCmd(opts *cmdutil.Options) *cobra.Command {
	var team string
	var sink secretout.Sink
	get := &cobra.Command{
		Use:   "get",
		Short: "Read the workspace's existing configuration token pair and deliver it to a file or a command",
		Long: `Reads the token pair the Your Apps page holds for the workspace.

` + contract,
		Example: `  slaw config-tokens get --to-command 'slck set-credential --key app_config_token --stdin'
  slaw config-tokens get --to-file ./pair.json`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return emit(opts, sink, func(b cmdutil.Browser) (slackweb.ConfigToken, error) {
				return b.GetConfigToken(team, opts.Now())
			})
		},
	}
	get.Flags().StringVarP(&team, "team", "t", "", "Workspace name as the Your Apps page shows it; needed only when the account has several")
	cmdutil.AddSinkFlags(get, &sink)
	return get
}

func emit(opts *cmdutil.Options, sink secretout.Sink, read func(cmdutil.Browser) (slackweb.ConfigToken, error)) error {
	if err := cmdutil.CheckSink(sink); err != nil {
		return err
	}
	return opts.WithBrowser(false, func(b cmdutil.Browser) error {
		pair, err := read(b)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(pair) //nolint:gosec // delivering the pair is this command's purpose; the sink is the caller's
		if err != nil {
			return err
		}
		return opts.DeliverSecret(sink, raw)
	})
}
