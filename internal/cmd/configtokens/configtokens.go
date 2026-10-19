// Package configtokens implements `slaw config-tokens`.
package configtokens

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-app-web/internal/cmd/cmdutil"
	"github.com/open-cli-collective/slack-app-web/internal/secretout"
	"github.com/open-cli-collective/slack-app-web/internal/slackweb"
)

const contract = `The pair is written to stdout as one JSON object:

  {"token": "xoxe.xoxp-...", "refresh_token": "xoxe-...", "exp": 1700000000}

The field names are those of Slack's tooling.tokens.rotate. exp is unix
seconds; the access token lasts 12 hours and the refresh token renews it.
stdout must be a pipe or a file: tokens are never printed to a terminal.`

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
	create := &cobra.Command{
		Use:   "create",
		Short: "Generate an app configuration token pair and write it to a pipe",
		Long: `Generates an app configuration token for a workspace. A workspace that
already has one is refused unless --replace is set, which deletes the
existing token first and so invalidates every copy of it.

` + contract,
		Example: `  slaw config-tokens create | <command that stores the pair>
  slaw config-tokens create --team "Example Co" --replace > pair.json`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return emit(opts, func(b cmdutil.Browser) (slackweb.ConfigToken, error) {
				return b.CreateConfigToken(team, replace, opts.Now())
			})
		},
	}
	create.Flags().StringVarP(&team, "team", "t", "", "Workspace name as the Your Apps page shows it; needed only when the account has several")
	create.Flags().BoolVar(&replace, "replace", false, "Delete the workspace's existing token first")
	return create
}

func newGetCmd(opts *cmdutil.Options) *cobra.Command {
	var team string
	get := &cobra.Command{
		Use:   "get",
		Short: "Read the workspace's existing configuration token pair and write it to a pipe",
		Long: `Reads the token pair the Your Apps page holds for the workspace.

` + contract,
		Example: `  slaw config-tokens get | <command that stores the pair>`,
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return emit(opts, func(b cmdutil.Browser) (slackweb.ConfigToken, error) {
				return b.GetConfigToken(team, opts.Now())
			})
		},
	}
	get.Flags().StringVarP(&team, "team", "t", "", "Workspace name as the Your Apps page shows it; needed only when the account has several")
	return get
}

func emit(opts *cmdutil.Options, read func(cmdutil.Browser) (slackweb.ConfigToken, error)) error {
	if opts.StdoutIsTerminal() {
		return secretout.ErrTerminal
	}
	b, err := opts.Browser(false)
	if err != nil {
		return err
	}
	defer b.Close()
	pair, err := read(b)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(pair) //nolint:gosec // emitting the pair is this command's purpose; secretout guards where it goes
	if err != nil {
		return err
	}
	return secretout.Write(opts.Stdout, opts.StdoutIsTerminal(), raw)
}
