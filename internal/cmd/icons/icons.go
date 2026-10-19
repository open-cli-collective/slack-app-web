// Package icons implements `slaw icons update`.
package icons

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-app-web/internal/cmd/cmdutil"
)

// NewCmd returns the icons command group.
func NewCmd(opts *cmdutil.Options) *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "icons",
		Short: "Change an app's icon",
	}
	update := &cobra.Command{
		Use:     "update <app-id>",
		Short:   "Upload a PNG as an app's icon",
		Long:    "Uploads the PNG on the app's Basic Information page and saves it. Slack wants a square image between 512 and 2000 pixels.",
		Example: `  slaw icons update A0123456789 --file icon.png`,
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			appID, err := cmdutil.AppID(args[0])
			if err != nil {
				return err
			}
			png, err := checkPNG(file)
			if err != nil {
				return err
			}
			return opts.WithBrowser(false, func(b cmdutil.Browser) error {
				if err := b.UpdateIcon(appID, png); err != nil {
					return err
				}
				fmt.Fprintf(opts.Stdout, "Updated the icon of %s\n", appID)
				return nil
			})
		},
	}
	update.Flags().StringVarP(&file, "file", "F", "", "PNG file to upload (required)")
	cmd.AddCommand(update)
	return cmd
}

var pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

func checkPNG(path string) (string, error) {
	if path == "" {
		return "", cmdutil.Usage("--file is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	f, err := os.Open(abs)
	if err != nil {
		return "", cmdutil.Usage("%w", err)
	}
	defer f.Close()
	head := make([]byte, len(pngSignature))
	if _, err := f.Read(head); err != nil || !strings.HasPrefix(string(head), string(pngSignature)) {
		return "", cmdutil.Usage("%s is not a PNG", path)
	}
	return abs, nil
}
