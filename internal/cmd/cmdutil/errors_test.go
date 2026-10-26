package cmdutil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/open-cli-collective/slack-app-web/internal/exitcode"
	"github.com/open-cli-collective/slack-app-web/internal/secretout"
	"github.com/open-cli-collective/slack-app-web/internal/slackweb"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		err      error
		code     int
		wantHint bool
	}{
		{fmt.Errorf("x: %w", slackweb.ErrSignedOut), exitcode.Auth, true},
		{slackweb.ErrDriverMissing, exitcode.Generic, true},
		{fmt.Errorf("A0EXAMPLE01: %w", slackweb.ErrAppNotFound), exitcode.NotFound, false},
		{slackweb.ErrNoBotToken, exitcode.NotFound, false},
		{slackweb.ErrNoConfigToken, exitcode.NotFound, true},
		{slackweb.ErrConfigTokenExists, exitcode.Usage, true},
		{fmt.Errorf("%q: %w", "socket-mode", slackweb.ErrAppTokenExists), exitcode.Usage, true},
		{fmt.Errorf("%q: %w: %w", "socket-mode", slackweb.ErrAppTokenUnread, errors.New("timeout")), exitcode.Upstream, true},
		{fmt.Errorf("%q: %w: %w", "socket-mode", slackweb.ErrAppTokenUnread, exitcode.New(exitcode.Usage, errors.New("file exists"))), exitcode.Usage, true},
		{slackweb.ErrScopeNotOffered, exitcode.Usage, false},
		{slackweb.ErrPendingApproval, exitcode.Generic, false},
		{CheckSink(secretout.Sink{}), exitcode.Usage, false},
		{&slackweb.UpstreamError{Err: errors.New("x")}, exitcode.Upstream, false},
		{Usage("bad"), exitcode.Usage, false},
		{errors.New("other"), exitcode.Generic, false},
	}
	for _, c := range cases {
		code, hint := Classify(c.err)
		if code != c.code || (hint != "") != c.wantHint {
			t.Errorf("Classify(%v) = %d, %q; want %d, hint %v", c.err, code, hint, c.code, c.wantHint)
		}
	}
}
