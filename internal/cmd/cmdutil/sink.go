package cmdutil

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-app-web/internal/exitcode"
	"github.com/open-cli-collective/slack-app-web/internal/secretout"
)

// AddSinkFlags registers the flags that name where a secret goes. Every
// command that reads a secret takes exactly one of --to-file and
// --to-command; none writes a secret to its own output.
func AddSinkFlags(cmd *cobra.Command, sink *secretout.Sink) {
	cmd.Flags().StringVar(&sink.File, "to-file", "", "Write the secret to this path, created with mode 0600")
	cmd.Flags().BoolVar(&sink.Overwrite, "overwrite", false, "Let --to-file replace an existing file")
	cmd.Flags().StringVar(&sink.Command, "to-command", "", "Run this shell command with the secret on its stdin")
}

// CheckSink validates sink before any browser opens.
func CheckSink(sink secretout.Sink) error {
	return sinkError(sink.Check())
}

// DeliverSecret writes secret to sink.
func (o *Options) DeliverSecret(sink secretout.Sink, secret []byte) error {
	return sinkError(sink.Write(secret, o.Stdout, o.Stderr))
}

// sinkError marks a destination the caller named wrongly as a usage error
// and names the flag that fixes it, wherever it is caught: an existing file
// found before the browser opens and one that appears while the secret is
// read are the same mistake. Other failures (a directory slaw cannot stat, a
// command that fails) keep their own exit code.
func sinkError(err error) error {
	var hint string
	switch {
	case err == nil:
		return nil
	case errors.Is(err, secretout.ErrNoSink), errors.Is(err, secretout.ErrTwoSinks):
		hint = "pass one of --to-file <path> or --to-command <command>"
	case errors.Is(err, secretout.ErrOverwriteWithoutFile):
		hint = "--overwrite goes with --to-file"
	case errors.Is(err, secretout.ErrExists):
		hint = "pass --overwrite to replace it"
	default:
		return err
	}
	return exitcode.New(exitcode.Usage, fmt.Errorf("%w; %s", err, hint))
}
