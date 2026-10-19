// Package bottokens implements `slaw bot-tokens get`.
package bottokens

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-app-web/internal/cmd/cmdutil"
	"github.com/open-cli-collective/slack-app-web/internal/secretout"
)

// NewCmd returns the bot-tokens command group.
func NewCmd(opts *cmdutil.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bot-tokens",
		Short: "Read an installed app's bot token",
	}
	var sink secretout.Sink
	get := &cobra.Command{
		Use:   "get <app-id>",
		Short: "Deliver an app's bot token to a file or a command",
		Long: `Reads the app's Bot User OAuth Token from its OAuth & Permissions page and
delivers it, with no trailing newline, to exactly one destination:

  --to-file <path>      a new file with mode 0600 (--overwrite replaces one)
  --to-command <cmd>    a shell command that reads the token on stdin

The token is never written to slaw's own output.`,
		Example: `  slaw bot-tokens get A0123456789 --to-command 'aws secretsmanager put-secret-value --secret-id <id> --secret-string file:///dev/stdin'
  slaw bot-tokens get A0123456789 --to-file ./bot-token`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			appID, err := cmdutil.AppID(args[0])
			if err != nil {
				return err
			}
			if err := cmdutil.CheckSink(sink); err != nil {
				return err
			}
			return opts.WithBrowser(false, func(b cmdutil.Browser) error {
				token, err := b.BotToken(appID)
				if err != nil {
					return err
				}
				return opts.DeliverSecret(sink, []byte(token))
			})
		},
	}
	cmdutil.AddSinkFlags(get, &sink)
	cmd.AddCommand(get)
	return cmd
}
