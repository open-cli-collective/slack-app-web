package slackweb

import (
	"errors"
	"fmt"
)

// Sentinels for the outcomes a caller may branch on. Wrapped errors carry
// the specifics (which app, which workspace); test with errors.Is.
var (
	// ErrSignedOut means the profile has no Slack session.
	ErrSignedOut = errors.New("the browser profile is not signed in to Slack")
	// ErrDriverMissing means the Playwright driver or Chromium is not installed.
	ErrDriverMissing = errors.New("the browser driver or Chromium is not installed")
	// ErrAppNotFound means the app does not exist or the account cannot manage it.
	ErrAppNotFound = errors.New("app not found, or this account cannot manage it")
	// ErrNoBotToken means the app has no bot user or is not installed.
	ErrNoBotToken = errors.New("the app has no bot token: it has no bot user or is not installed")
	// ErrConfigTokenExists means the workspace already has a configuration
	// token; Slack keeps one per workspace.
	ErrConfigTokenExists = errors.New("the workspace already has an app configuration token")
	// ErrNoConfigToken means the workspace has no configuration token.
	ErrNoConfigToken = errors.New("the workspace has no app configuration token")
	// ErrWorkspaceNotOffered means the named workspace is not in the token dialog.
	ErrWorkspaceNotOffered = errors.New("the workspace is not offered for a configuration token")
	// ErrAppTokenExists means the app already has an app-level token with
	// the requested name.
	ErrAppTokenExists = errors.New("the app already has an app-level token with this name")
	// ErrAppTokenUnread means an app-level token was generated but its value
	// never reached its destination. Slack shows the value only once, so
	// the token exists and is lost; it has to be revoked.
	ErrAppTokenUnread = errors.New("the app-level token was generated but not delivered")
	// ErrNoAppToken means the app lists no app-level token with the name.
	ErrNoAppToken = errors.New("the app has no app-level token with this name")
	// ErrScopeNotOffered means the token dialog's scope picker does not
	// offer the requested scope.
	ErrScopeNotOffered = errors.New("the scope is not offered for an app-level token")
	// ErrPendingApproval means the install was submitted as a request that a
	// workspace admin must approve.
	ErrPendingApproval = errors.New("the install was requested and awaits a workspace admin's approval")
)

// UpstreamError means Slack's pages did not respond as the flow expects.
type UpstreamError struct{ Err error }

func (e *UpstreamError) Error() string { return e.Err.Error() }

func (e *UpstreamError) Unwrap() error { return e.Err }

func upstream(format string, args ...any) error {
	return &UpstreamError{Err: fmt.Errorf(format, args...)}
}
