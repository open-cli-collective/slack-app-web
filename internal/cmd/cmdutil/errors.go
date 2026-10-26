package cmdutil

import (
	"errors"
	"fmt"

	"github.com/open-cli-collective/slack-app-web/internal/exitcode"
	"github.com/open-cli-collective/slack-app-web/internal/slackweb"
)

// Classify maps an error to its exit code and the next step to suggest.
func Classify(err error) (code int, hint string) {
	var upstream *slackweb.UpstreamError
	switch {
	case errors.Is(err, slackweb.ErrSignedOut):
		return exitcode.Auth, "run `slaw login`"
	case errors.Is(err, slackweb.ErrDriverMissing):
		return exitcode.Generic, "run `slaw browser install`"
	case errors.Is(err, slackweb.ErrAppNotFound), errors.Is(err, slackweb.ErrNoBotToken),
		errors.Is(err, slackweb.ErrWorkspaceNotOffered):
		return exitcode.NotFound, ""
	case errors.Is(err, slackweb.ErrNoConfigToken):
		return exitcode.NotFound, "run `slaw config-tokens create`"
	case errors.Is(err, slackweb.ErrConfigTokenExists):
		return exitcode.Usage, "`slaw config-tokens get` reads it; `slaw config-tokens create --replace` deletes it and generates a new one"
	case errors.Is(err, slackweb.ErrAppTokenUnread):
		// A destination that failed keeps its own code; a token Slack never
		// showed is an upstream failure.
		code := exitcode.Upstream
		var coded *exitcode.Error
		if errors.As(err, &coded) {
			code = coded.Code
		}
		return code, "run `slaw app-tokens revoke` with the name exactly as the app lists it, then run the command again"
	case errors.Is(err, slackweb.ErrAppTokenExists):
		return exitcode.Usage, "pass another --name, or run `slaw app-tokens revoke` with the name exactly as the app lists it"
	case errors.Is(err, slackweb.ErrNoAppToken):
		return exitcode.NotFound, ""
	case errors.Is(err, slackweb.ErrScopeNotOffered):
		return exitcode.Usage, ""
	case errors.Is(err, slackweb.ErrPendingApproval):
		return exitcode.Generic, ""
	case errors.As(err, &upstream):
		return exitcode.Upstream, ""
	}
	return exitcode.Of(err), ""
}

// Usage marks err as a usage error.
func Usage(format string, args ...any) error {
	return exitcode.New(exitcode.Usage, fmt.Errorf(format, args...))
}
