package cmdutil

import (
	"fmt"

	"github.com/open-cli-collective/slack-app-web/internal/exitcode"
	"github.com/open-cli-collective/slack-app-web/internal/slackweb"
)

// AppID validates a positional app id.
func AppID(arg string) (string, error) {
	if !slackweb.ValidAppID(arg) {
		return "", exitcode.New(exitcode.Usage, fmt.Errorf("%q is not a Slack app id (A followed by capital letters and digits)", arg))
	}
	return arg, nil
}
