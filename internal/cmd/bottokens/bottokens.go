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
	cmd.AddCommand(&cobra.Command{
		Use:   "get <app-id>",
		Short: "Write an app's bot token to a pipe",
		Long: `Reads the app's Bot User OAuth Token from its OAuth & Permissions page and
writes it to stdout with no trailing newline. stdout must be a pipe or a file:
the command refuses to print a token to a terminal.`,
		Example: `  slaw bot-tokens get A0123456789 | op item create --category "API Credential" --title "My bot" credential[password]=-
  slaw bot-tokens get A0123456789 > token.txt`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			appID, err := cmdutil.AppID(args[0])
			if err != nil {
				return err
			}
			if opts.StdoutIsTerminal() {
				return secretout.ErrTerminal
			}
			return opts.WithBrowser(false, func(b cmdutil.Browser) error {
				token, err := b.BotToken(appID)
				if err != nil {
					return err
				}
				return secretout.Write(opts.Stdout, opts.StdoutIsTerminal(), []byte(token))
			})
		},
	})
	return cmd
}
