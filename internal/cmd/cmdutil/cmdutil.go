// Package cmdutil holds what every command shares: the global options and
// the browser seam commands run against, so tests can swap in a fake.
package cmdutil

import (
	"io"
	"os"
	"time"

	"github.com/open-cli-collective/slack-app-web/internal/secretout"
	"github.com/open-cli-collective/slack-app-web/internal/slackweb"
)

// Browser is the slice of a browser session that commands use.
type Browser interface {
	SignedIn() (bool, error)
	Login(workspace string, wait time.Duration) error
	ListApps() ([]slackweb.App, error)
	Install(appID string, reinstall bool) (slackweb.InstallState, bool, error)
	BotToken(appID string) (string, error)
	CreateConfigToken(team string, replace bool, now time.Time) (slackweb.ConfigToken, error)
	GetConfigToken(team string, now time.Time) (slackweb.ConfigToken, error)
	UpdateIcon(appID, png string) error
	Close()
}

// Options are the global flags and the injectable seams.
type Options struct {
	Profile    string
	ProfileDir string
	Timeout    time.Duration

	Stdout io.Writer
	Stderr io.Writer
	// StdoutIsTerminal reports whether stdout is a terminal; secrets are
	// refused when it is.
	StdoutIsTerminal func() bool
	Now              func() time.Time
	// Open starts a browser session; tests replace it.
	Open func(slackweb.Options) (Browser, error)
}

// Default returns options wired to the real process and browser.
func Default() *Options {
	return &Options{
		Profile:          "default",
		Timeout:          30 * time.Second,
		Stdout:           os.Stdout,
		Stderr:           os.Stderr,
		StdoutIsTerminal: func() bool { return secretout.IsTerminal(os.Stdout) },
		Now:              time.Now,
		Open: func(o slackweb.Options) (Browser, error) {
			s, err := slackweb.Open(o)
			if err != nil {
				return nil, err
			}
			return s, nil
		},
	}
}

// ResolvedProfileDir is --profile-dir when set, otherwise the named
// profile's directory under the user cache.
func (o *Options) ResolvedProfileDir() (string, error) {
	if o.ProfileDir != "" {
		return o.ProfileDir, nil
	}
	return slackweb.DefaultProfileDir(o.Profile)
}

// Browser opens a session on the resolved profile.
func (o *Options) Browser(headed bool) (Browser, error) {
	dir, err := o.ResolvedProfileDir()
	if err != nil {
		return nil, err
	}
	return o.Open(slackweb.Options{ProfileDir: dir, Headed: headed, Timeout: o.Timeout})
}
