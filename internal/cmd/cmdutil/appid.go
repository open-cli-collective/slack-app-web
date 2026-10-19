package cmdutil

import (
	"regexp"

	"github.com/open-cli-collective/slack-app-web/internal/slackweb"
)

var workspacePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*$`)

// AppID validates a positional app id.
func AppID(arg string) (string, error) {
	if !slackweb.ValidAppID(arg) {
		return "", Usage("%q is not a Slack app id (A followed by capital letters and digits)", arg)
	}
	return arg, nil
}

// Workspace validates a workspace subdomain before it becomes part of a URL.
func Workspace(arg string) (string, error) {
	if arg != "" && !workspacePattern.MatchString(arg) {
		return "", Usage("%q is not a Slack workspace subdomain (letters, digits and hyphens, as in acme for acme.slack.com)", arg)
	}
	return arg, nil
}
