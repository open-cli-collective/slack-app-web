package cmdutil

import (
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

// CheckSink validates sink as a usage error, before any browser opens.
func CheckSink(sink secretout.Sink) error {
	if err := sink.Check(); err != nil {
		return exitcode.New(exitcode.Usage, err)
	}
	return nil
}

// DeliverSecret writes secret to sink. A destination that fails is a
// generic failure: the secret was read, and the caller's command or path
// refused it.
func (o *Options) DeliverSecret(sink secretout.Sink, secret []byte) error {
	return sink.Write(secret, o.Stdout, o.Stderr)
}
