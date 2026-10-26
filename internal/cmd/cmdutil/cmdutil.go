// Package cmdutil holds what every command shares: the global options and
// the browser seam commands run against, so tests can swap in a fake.
package cmdutil

import (
	"io"
	"os"
	"time"

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
	CreateAppToken(appID, name string, scopes []string) (string, error)
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
	Now    func() time.Time
	// Open starts a browser session; tests replace it.
	Open func(slackweb.Options) (Browser, error)
}

// Default returns options wired to the real process and browser, with the
// profile defaults read from SLAW_PROFILE and SLAW_PROFILE_DIR.
func Default() *Options {
	profile := os.Getenv("SLAW_PROFILE")
	if profile == "" {
		profile = "default"
	}
	return &Options{
		Profile:    profile,
		ProfileDir: os.Getenv("SLAW_PROFILE_DIR"),
		Timeout:    30 * time.Second,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Now:        time.Now,
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

// WithBrowser opens a session on the resolved profile, runs fn, and closes
// the session however fn returns. A session left open keeps the profile
// locked against every later run, so commands reach the browser only
// through here.
func (o *Options) WithBrowser(headed bool, fn func(Browser) error) error {
	dir, err := o.ResolvedProfileDir()
	if err != nil {
		return err
	}
	b, err := o.Open(slackweb.Options{ProfileDir: dir, Headed: headed, Timeout: o.Timeout})
	if err != nil {
		return err
	}
	defer b.Close()
	return fn(b)
}
