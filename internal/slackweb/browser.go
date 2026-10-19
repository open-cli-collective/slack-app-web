// Package slackweb drives api.slack.com in a persistent Chromium profile for
// the Slack app steps that have no API: signing in, generating an app
// configuration token, installing an app, reading its bot token and
// uploading its icon. It never logs, prints or screenshots page content, so
// a token on a page cannot leak through it.
package slackweb

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"

	"github.com/open-cli-collective/slack-app-web/internal/exitcode"
)

const appsURL = "https://api.slack.com/apps"

// ErrSignedOut means the profile has no Slack session.
var ErrSignedOut = exitcode.New(exitcode.Auth, errors.New("the browser profile is not signed in to Slack: run `slaw login`"))

// Options configures a browser session.
type Options struct {
	// ProfileDir is the persistent Chromium profile.
	ProfileDir string
	// Headed shows the window; only login needs it.
	Headed bool
	// Timeout bounds each wait for an element.
	Timeout time.Duration
}

// Session is one Chromium instance on a persistent profile.
type Session struct {
	pw      *playwright.Playwright
	ctx     playwright.BrowserContext
	page    playwright.Page
	timeout float64
}

// DefaultProfileDir is the profile directory for a named profile under the
// user cache directory.
func DefaultProfileDir(profile string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "slack-app-web", "profiles", profile), nil
}

// InstallBrowser downloads the Playwright driver and Chromium, with
// progress on progress.
func InstallBrowser(progress io.Writer) error {
	return playwright.Install(&playwright.RunOptions{
		Browsers: []string{"chromium"},
		Stdout:   progress,
		Stderr:   progress,
	})
}

// Open starts Chromium on the profile.
func Open(opts Options) (*Session, error) {
	if opts.ProfileDir == "" {
		return nil, exitcode.New(exitcode.Usage, errors.New("no browser profile directory"))
	}
	if err := os.MkdirAll(opts.ProfileDir, 0o700); err != nil {
		return nil, err
	}
	pw, err := playwright.Run(&playwright.RunOptions{SkipInstallBrowsers: true, Verbose: false, Stdout: io.Discard, Stderr: io.Discard})
	if err != nil {
		return nil, exitcode.New(exitcode.Generic, fmt.Errorf("the browser driver is not installed: run `slaw browser install` (%w)", err))
	}
	headless := !opts.Headed
	ctx, err := pw.Chromium.LaunchPersistentContext(opts.ProfileDir, playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: &headless,
		Viewport: &playwright.Size{Width: 1400, Height: 1000},
	})
	if err != nil {
		_ = pw.Stop()
		if strings.Contains(err.Error(), "ProcessSingleton") || strings.Contains(err.Error(), "SingletonLock") {
			return nil, fmt.Errorf("the profile %s is in use by another browser; close it and retry", opts.ProfileDir)
		}
		if strings.Contains(err.Error(), "Executable doesn't exist") {
			return nil, fmt.Errorf("chromium is not installed: run `slaw browser install`")
		}
		return nil, fmt.Errorf("could not start chromium: %w", err)
	}
	if err := ctx.AddInitScript(playwright.Script{Content: playwright.String(clipboardHook)}); err != nil {
		_ = ctx.Close()
		_ = pw.Stop()
		return nil, err
	}
	page := firstPage(ctx)
	if page == nil {
		if page, err = ctx.NewPage(); err != nil {
			_ = ctx.Close()
			_ = pw.Stop()
			return nil, err
		}
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Session{pw: pw, ctx: ctx, page: page, timeout: float64(timeout.Milliseconds())}, nil
}

func firstPage(ctx playwright.BrowserContext) playwright.Page {
	if pages := ctx.Pages(); len(pages) > 0 {
		return pages[0]
	}
	return nil
}

// Close shuts the browser down.
func (s *Session) Close() {
	if s == nil {
		return
	}
	_ = s.ctx.Close()
	_ = s.pw.Stop()
}

func (s *Session) goto_(url string) error {
	if _, err := s.page.Goto(url, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded, Timeout: &s.timeout}); err != nil {
		return exitcode.New(exitcode.Upstream, fmt.Errorf("could not load %s: %w", url, err))
	}
	return nil
}

// visible waits up to the session timeout for loc and reports whether it
// appeared.
func (s *Session) visible(loc playwright.Locator, timeout float64) bool {
	return loc.First().WaitFor(playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateVisible, Timeout: &timeout}) == nil
}

// SignedIn loads Your Apps and reports whether the profile has a session.
func (s *Session) SignedIn() (bool, error) {
	if err := s.goto_(appsURL); err != nil {
		return false, err
	}
	return s.visible(s.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Generate Token"}), s.timeout), nil
}

// requireSignedIn fails with ErrSignedOut when the profile has no session.
func (s *Session) requireSignedIn() error {
	ok, err := s.SignedIn()
	if err != nil {
		return err
	}
	if !ok {
		return ErrSignedOut
	}
	return nil
}

// copied returns what Copy buttons have written since the last reset.
func (s *Session) copied() ([]string, error) {
	v, err := s.page.Evaluate(`() => window.__slawCopied || []`)
	if err != nil {
		return nil, err
	}
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, it := range items {
		if str, ok := it.(string); ok {
			out = append(out, str)
		}
	}
	return out, nil
}

func (s *Session) resetCopied() error {
	_, err := s.page.Evaluate(`() => { window.__slawCopied = []; }`)
	return err
}
