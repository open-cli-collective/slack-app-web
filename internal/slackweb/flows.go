package slackweb

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/mxschmitt/playwright-go"
)

// Login opens the Slack sign-in page and waits, up to wait, for a person to
// sign in. workspace, when set, starts at that workspace's sign-in page
// (<workspace>.slack.com).
func (s *Session) Login(workspace string, wait time.Duration) error {
	start := "https://slack.com/signin"
	if workspace != "" {
		start = "https://" + workspace + ".slack.com/"
	}
	if err := s.goto_(start); err != nil {
		return s.explain(err)
	}
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		cookies, err := s.ctx.Cookies("https://slack.com", "https://api.slack.com")
		if err != nil {
			return s.explain(err)
		}
		for _, c := range cookies {
			// "d" is Slack's session cookie.
			if c.Name == "d" && c.Value != "" {
				return s.requireSignedIn()
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("%w: no sign-in within %s", ErrSignedOut, wait)
}

// ListApps returns the apps on the Your Apps page.
func (s *Session) ListApps() ([]App, error) {
	if err := s.requireSignedIn(); err != nil {
		return nil, err
	}
	rows := s.page.Locator("table").First().Locator("tr")
	if !s.visible(rows, s.timeout) {
		return nil, nil
	}
	v, err := rows.EvaluateAll(`rows => rows.map(r => [...r.querySelectorAll('th,td')].map(c => c.innerText))`)
	if err != nil {
		return nil, err
	}
	return parseAppTable(toRows(v)), nil
}

func toRows(v any) [][]string {
	raw, _ := v.([]any)
	rows := make([][]string, 0, len(raw))
	for _, r := range raw {
		cells, _ := r.([]any)
		row := make([]string, 0, len(cells))
		for _, c := range cells {
			str, _ := c.(string)
			row = append(row, str)
		}
		rows = append(rows, row)
	}
	return rows
}

// InstallState is what the install page shows for an app.
type InstallState struct {
	Installed       bool
	ReinstallNeeded bool
	PendingApproval bool
}

var (
	installedText       = regexp.MustCompile(`(?i)(bot|user) user oauth token`)
	reinstallNeededText = regexp.MustCompile(`(?i)reinstall your app`)
	installButton       = regexp.MustCompile(`(?i)^install to `)
	reinstallButton     = regexp.MustCompile(`(?i)^reinstall to `)
	requestButton       = regexp.MustCompile(`(?i)^request to install`)
	allowButton         = regexp.MustCompile(`(?i)^allow$`)
)

func (s *Session) installState() (InstallState, error) {
	body, err := s.page.Locator("body").InnerText()
	if err != nil {
		return InstallState{}, err
	}
	return InstallState{
		Installed:       installedText.MatchString(body),
		ReinstallNeeded: reinstallNeededText.MatchString(body),
	}, nil
}

func (s *Session) loadInstallPage(appID string) error {
	if err := s.requireSignedIn(); err != nil {
		return err
	}
	if err := s.goto_(appsURL + "/" + appID + "/install-on-team"); err != nil {
		return err
	}
	// The tokens or the install button render after the page loads; wait
	// for one of them before reading the install state.
	ready := s.page.GetByText(installedText).
		Or(s.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: installButton})).
		Or(s.page.GetByRole("link", playwright.PageGetByRoleOptions{Name: installButton})).
		Or(s.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: requestButton})).
		Or(s.page.GetByRole("link", playwright.PageGetByRoleOptions{Name: requestButton}))
	if !s.visible(ready, s.timeout) {
		return fmt.Errorf("%s: %w", appID, ErrAppNotFound)
	}
	return nil
}

// Install installs appID to its workspace, or reinstalls it when reinstall
// is set. An installed app is left alone unless reinstall is set.
func (s *Session) Install(appID string, reinstall bool) (InstallState, bool, error) {
	if err := s.loadInstallPage(appID); err != nil {
		return InstallState{}, false, err
	}
	state, err := s.installState()
	if err != nil {
		return state, false, err
	}
	if state.Installed && !reinstall {
		return state, false, nil
	}
	pattern := installButton
	if state.Installed {
		pattern = reinstallButton
	}
	button := s.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: pattern}).
		Or(s.page.GetByRole("link", playwright.PageGetByRoleOptions{Name: pattern}))
	request := s.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: requestButton}).
		Or(s.page.GetByRole("link", playwright.PageGetByRoleOptions{Name: requestButton}))
	switch {
	case s.visible(button, s.timeout):
		if err := button.First().Click(); err != nil {
			return state, false, err
		}
	case s.visible(request, 1000):
		if err := request.First().Click(); err != nil {
			return state, false, err
		}
		state.PendingApproval = true
	default:
		return state, false, upstream("no install button on the app's install page")
	}
	allow := s.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: allowButton})
	if s.visible(allow, s.timeout) {
		if err := allow.First().Click(); err != nil {
			return state, false, err
		}
		if err := s.page.WaitForURL(regexp.MustCompile(`api\.slack\.com/apps/`), playwright.PageWaitForURLOptions{Timeout: &s.timeout}); err != nil {
			return state, false, upstream("the install did not return to the app page")
		}
	}
	if err := s.loadInstallPage(appID); err != nil {
		return state, true, err
	}
	after, err := s.installState()
	after.PendingApproval = state.PendingApproval
	return after, true, err
}

// BotToken reads appID's bot token from its OAuth & Permissions page.
func (s *Session) BotToken(appID string) (string, error) {
	if err := s.loadInstallPage(appID); err != nil {
		return "", err
	}
	if err := s.goto_(appsURL + "/" + appID + "/oauth"); err != nil {
		return "", err
	}
	label := s.page.GetByText(regexp.MustCompile(`(?i)bot user oauth token`))
	if !s.visible(label, s.timeout) {
		return "", fmt.Errorf("%s: %w", appID, ErrNoBotToken)
	}
	v, err := s.page.Locator("input").EvaluateAll(`inputs => inputs.map(i => i.value)`)
	if err != nil {
		return "", err
	}
	if token := firstMatch(botTokenPattern, toStrings(v)); token != "" {
		return token, nil
	}
	if err := s.resetCopied(); err != nil {
		return "", err
	}
	copyButtons := s.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: regexp.MustCompile(`(?i)copy`)})
	n, err := copyButtons.Count()
	if err != nil {
		return "", err
	}
	for i := 0; i < n; i++ {
		if err := copyButtons.Nth(i).Click(); err != nil {
			// The page has several Copy buttons; one that cannot be
			// clicked just moves the search to the next.
			continue
		}
		copied, err := s.copied()
		if err != nil {
			return "", err
		}
		if token := firstMatch(botTokenPattern, copied); token != "" {
			return token, nil
		}
	}
	return "", errors.New("the bot token was not found on the OAuth & Permissions page")
}

func toStrings(v any) []string {
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, it := range items {
		if str, ok := it.(string); ok {
			out = append(out, str)
		}
	}
	return out
}

// CreateConfigToken generates an app configuration token for the workspace
// named team (the name the Your Apps page shows), or for the only workspace
// when team is empty, and reads the pair through the Copy buttons. Slack
// keeps one token per workspace: an existing one is deleted first when
// replace is set, and is otherwise an ErrConfigTokenExists.
func (s *Session) CreateConfigToken(team string, replace bool, now time.Time) (ConfigToken, error) {
	if err := s.requireSignedIn(); err != nil {
		return ConfigToken{}, err
	}
	if row, ok := s.configTokenRow(team, 2000); ok {
		if !replace {
			return ConfigToken{}, ErrConfigTokenExists
		}
		if err := s.deleteConfigToken(row); err != nil {
			return ConfigToken{}, err
		}
	}
	generateToken := s.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Generate Token"}).First()
	if enabled, _ := generateToken.IsEnabled(); !enabled {
		return ConfigToken{}, ErrConfigTokenExists
	}
	if err := generateToken.Click(); err != nil {
		return ConfigToken{}, err
	}
	dialog := s.page.GetByRole("dialog")
	if !s.visible(dialog, s.timeout) {
		return ConfigToken{}, errors.New("the Generate Token dialog did not open")
	}
	if err := s.chooseWorkspace(dialog, team); err != nil {
		return ConfigToken{}, err
	}
	generate := dialog.GetByRole("button", playwright.LocatorGetByRoleOptions{Name: regexp.MustCompile(`(?i)^generate$`)})
	if err := generate.First().Click(); err != nil {
		return ConfigToken{}, err
	}
	row, ok := s.configTokenRow(team, s.timeout)
	if !ok {
		return ConfigToken{}, errors.New("no configuration token row appeared after Generate")
	}
	return s.readConfigToken(row, now)
}

// GetConfigToken reads the workspace's existing configuration token pair.
func (s *Session) GetConfigToken(team string, now time.Time) (ConfigToken, error) {
	if err := s.requireSignedIn(); err != nil {
		return ConfigToken{}, err
	}
	row, ok := s.configTokenRow(team, 5000)
	if !ok {
		return ConfigToken{}, ErrNoConfigToken
	}
	return s.readConfigToken(row, now)
}

// configTokenRow finds the token table row for team, or the first row when
// team is empty.
func (s *Session) configTokenRow(team string, timeout float64) (playwright.Locator, bool) {
	row := s.page.Locator("tr").Filter(playwright.LocatorFilterOptions{Has: s.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Copy access token"})})
	if team != "" {
		row = row.Filter(playwright.LocatorFilterOptions{HasText: team})
	}
	if !s.visible(row, timeout) {
		return nil, false
	}
	return row.First(), true
}

func (s *Session) deleteConfigToken(row playwright.Locator) error {
	if err := row.GetByRole("button", playwright.LocatorGetByRoleOptions{Name: "Delete token"}).First().Click(); err != nil {
		return err
	}
	// The confirmation is a popover, not a dialog, with one button named
	// "Revoke Token".
	confirm := s.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Revoke Token", Exact: playwright.Bool(true)})
	if !s.visible(confirm, 5000) {
		return upstream("no Revoke Token confirmation appeared")
	}
	if n, err := confirm.Count(); err != nil {
		return err
	} else if n != 1 {
		return upstream("expected one Revoke Token button, found %d", n)
	}
	if err := confirm.Click(); err != nil {
		return err
	}
	if err := row.WaitFor(playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateDetached, Timeout: &s.timeout}); err != nil {
		return errors.New("the existing configuration token was not deleted")
	}
	return nil
}

func (s *Session) readConfigToken(row playwright.Locator, now time.Time) (ConfigToken, error) {
	rowText, err := row.InnerText()
	if err != nil {
		return ConfigToken{}, err
	}
	access, err := s.copyFrom(row, "Copy access token", configAccessPattern)
	if err != nil {
		return ConfigToken{}, err
	}
	refresh, err := s.copyFrom(row, "Copy refresh token", configRefreshPattern)
	if err != nil {
		return ConfigToken{}, err
	}
	return ConfigToken{Token: access, RefreshToken: refresh, Exp: parseExpiry(rowText, now)}, nil
}

func (s *Session) chooseWorkspace(dialog playwright.Locator, team string) error {
	sel := dialog.Locator("select")
	if n, _ := sel.Count(); n > 0 {
		if team == "" {
			return nil
		}
		_, err := sel.First().SelectOption(playwright.SelectOptionValues{Labels: &[]string{team}})
		return err
	}
	// The picker starts empty even when the account has one workspace, and
	// Generate stays disabled until one is chosen.
	if err := dialog.GetByRole("combobox").Or(dialog.GetByText(regexp.MustCompile(`(?i)select a workspace`))).First().Click(); err != nil {
		return err
	}
	if team == "" {
		option := s.page.GetByRole("option")
		if !s.visible(option, s.timeout) {
			return fmt.Errorf("the workspace picker offered no workspace: %w", ErrWorkspaceNotOffered)
		}
		n, err := settledCount(option)
		if err != nil {
			return err
		}
		if n > 1 {
			return fmt.Errorf("the account has %d workspaces; name one with --team", n)
		}
		return option.First().Click()
	}
	option := s.page.GetByRole("option", playwright.PageGetByRoleOptions{Name: team})
	if !s.visible(option, s.timeout) {
		return fmt.Errorf("%q: %w", team, ErrWorkspaceNotOffered)
	}
	return option.First().Click()
}

// settledCount counts loc once the count has held for a moment, so a list
// that renders its options in steps is not counted half-drawn.
func settledCount(loc playwright.Locator) (int, error) {
	const (
		step   = 100 * time.Millisecond
		steady = 3
		limit  = 50
	)
	last, same := -1, 0
	for i := 0; i < limit; i++ {
		n, err := loc.Count()
		if err != nil {
			return 0, err
		}
		if n == last {
			same++
			if same >= steady {
				return n, nil
			}
		} else {
			last, same = n, 0
		}
		time.Sleep(step)
	}
	return last, nil
}

func (s *Session) copyFrom(row playwright.Locator, label string, pattern *regexp.Regexp) (string, error) {
	if err := s.resetCopied(); err != nil {
		return "", err
	}
	if err := row.GetByRole("button", playwright.LocatorGetByRoleOptions{Name: label}).First().Click(); err != nil {
		return "", err
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		copied, err := s.copied()
		if err != nil {
			return "", err
		}
		if token := firstMatch(pattern, copied); token != "" {
			return token, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return "", fmt.Errorf("%q did not yield a token", label)
}

// UpdateIcon uploads png as appID's icon on its Basic Information page and
// confirms the app's avatar changed. The page saves an icon as soon as the
// file is chosen; an older layout asks to confirm a crop first.
func (s *Session) UpdateIcon(appID, png string) error {
	if err := s.requireSignedIn(); err != nil {
		return err
	}
	if err := s.goto_(appsURL + "/" + appID + "/general"); err != nil {
		return err
	}
	input := s.page.Locator(`input[type="file"]`)
	if err := input.First().WaitFor(playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateAttached, Timeout: &s.timeout}); err != nil {
		return fmt.Errorf("%s: %w", appID, ErrAppNotFound)
	}
	before := s.avatar()
	if err := input.First().SetInputFiles(png); err != nil {
		return err
	}
	confirm := s.page.GetByRole("dialog").GetByRole("button", playwright.LocatorGetByRoleOptions{Name: regexp.MustCompile(`(?i)^(save|crop|upload|done)`)})
	deadline := time.Now().Add(time.Duration(s.timeout) * time.Millisecond)
	for time.Now().Before(deadline) {
		if after := s.avatar(); after != "" && after != before {
			return nil
		}
		if ok, _ := confirm.First().IsVisible(); ok {
			if enabled, _ := confirm.First().IsEnabled(); enabled {
				// A failed click is retried on the next pass; the avatar
				// changing is what decides success.
				_ = confirm.First().Click()
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return upstream("the icon of %s did not change after the upload", appID)
}

// avatar is the URL of the app icon the settings page shows.
func (s *Session) avatar() string {
	v, err := s.page.Evaluate(`() => { const i = [...document.querySelectorAll('img')].find(i => /avatars\.slack-edge\.com/.test(i.src)); return i ? i.src : ''; }`)
	if err != nil {
		return ""
	}
	src, _ := v.(string)
	return src
}

var (
	appTokensHeading = regexp.MustCompile(`(?i)^app-level tokens$`)
	generateAppToken = regexp.MustCompile(`(?i)generate token and scopes`)
	addScopeButton   = regexp.MustCompile(`(?i)^add scope`)
	tokenNameField   = regexp.MustCompile(`(?i)token name`)
	generateButton   = regexp.MustCompile(`(?i)^generate$`)
	doneButton       = regexp.MustCompile(`(?i)^(done|close)$`)
	revokeAppToken   = regexp.MustCompile(`(?i)^revoke token$`)
	confirmRevoke    = regexp.MustCompile(`(?i)^yes, i.m sure$`)
)

// CreateAppToken generates an app-level token named name with scopes on
// appID's Basic Information page and returns it. Slack shows the token's
// value in the dialog that follows Generate; a name the app already lists
// is an ErrAppTokenExists, checked before anything is created.
func (s *Session) CreateAppToken(appID, name string, scopes []string) (string, error) {
	if err := s.requireSignedIn(); err != nil {
		return "", err
	}
	if err := s.goto_(appsURL + "/" + appID + "/general"); err != nil {
		return "", err
	}
	open := s.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: generateAppToken}).
		Or(s.page.GetByRole("link", playwright.PageGetByRoleOptions{Name: generateAppToken}))
	if !s.visible(open, s.timeout) {
		return "", fmt.Errorf("%s: %w", appID, ErrAppNotFound)
	}
	section, err := s.appTokensSection(open)
	if err != nil {
		return "", err
	}
	if tokenNameListed(section, name) {
		return "", fmt.Errorf("%q: %w", name, ErrAppTokenExists)
	}
	if err := open.First().Click(); err != nil {
		return "", err
	}
	dialog := s.page.GetByRole("dialog")
	if !s.visible(dialog, s.timeout) {
		return "", upstream("the Generate Token and Scopes dialog did not open")
	}
	field := dialog.GetByLabel(tokenNameField).Or(dialog.GetByRole("textbox"))
	if !s.visible(field, s.timeout) {
		return "", upstream("the token dialog has no name field")
	}
	if err := field.First().Fill(name); err != nil {
		return "", err
	}
	for _, scope := range scopes {
		if err := s.addAppTokenScope(dialog, scope); err != nil {
			return "", err
		}
	}
	if err := s.resetCopied(); err != nil {
		return "", err
	}
	generate := dialog.GetByRole("button", playwright.LocatorGetByRoleOptions{Name: generateButton})
	if !s.visible(generate, s.timeout) {
		return "", upstream("the token dialog has no Generate button")
	}
	if err := generate.First().Click(); err != nil {
		return "", err
	}
	token, err := s.readAppToken()
	if err != nil {
		return "", fmt.Errorf("%q: %w: %w", name, ErrAppTokenUnread, err)
	}
	done := s.page.GetByRole("dialog").GetByRole("button", playwright.LocatorGetByRoleOptions{Name: doneButton})
	if s.visible(done, 5000) {
		// The token is already read; a dialog left open does not undo it.
		_ = done.First().Click()
	}
	return token, nil
}

// RevokeAppToken revokes appID's app-level token named name on the Basic
// Information page. The token's name opens its details, where Revoke asks
// for confirmation in an alert dialog. Revoking needs only the name, so it
// also clears a token whose value was lost. ErrNoAppToken when the app lists
// no token with the name.
func (s *Session) RevokeAppToken(appID, name string) error {
	if err := s.requireSignedIn(); err != nil {
		return err
	}
	if err := s.goto_(appsURL + "/" + appID + "/general"); err != nil {
		return err
	}
	open := s.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: generateAppToken}).
		Or(s.page.GetByRole("link", playwright.PageGetByRoleOptions{Name: generateAppToken}))
	if !s.visible(open, s.timeout) {
		return fmt.Errorf("%s: %w", appID, ErrAppNotFound)
	}
	section, err := s.appTokensSection(open)
	if err != nil {
		return err
	}
	if !tokenNameListed(section, name) {
		return fmt.Errorf("%q: %w", name, ErrNoAppToken)
	}
	entry := s.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: name, Exact: playwright.Bool(true)})
	if n, err := entry.Count(); err != nil {
		return err
	} else if n != 1 {
		return upstream("expected one token named %q to open, found %d", name, n)
	}
	if err := entry.Click(); err != nil {
		return err
	}
	revoke := s.page.GetByRole("dialog").GetByRole("button", playwright.LocatorGetByRoleOptions{Name: revokeAppToken})
	if !s.visible(revoke, s.timeout) {
		return upstream("the token's details have no Revoke button")
	}
	if err := revoke.First().Click(); err != nil {
		return err
	}
	confirm := s.page.GetByRole("alertdialog").GetByRole("button", playwright.LocatorGetByRoleOptions{Name: confirmRevoke})
	if !s.visible(confirm, s.timeout) {
		return upstream("no revoke confirmation appeared")
	}
	if n, err := confirm.Count(); err != nil {
		return err
	} else if n != 1 {
		return upstream("expected one revoke confirmation button, found %d", n)
	}
	if err := confirm.Click(); err != nil {
		return err
	}
	if err := entry.WaitFor(playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateDetached, Timeout: &s.timeout}); err != nil {
		return upstream("the token %q is still listed after revoking it", name)
	}
	return nil
}

// appTokensSection returns the text of the smallest element holding both
// the App-Level Tokens heading and the button that opens the dialog, which
// is where the page lists the app's existing tokens by name. The text holds
// names and scopes, never token values.
func (s *Session) appTokensSection(open playwright.Locator) (string, error) {
	heading := s.page.GetByText(appTokensHeading)
	if !s.visible(heading, s.timeout) {
		return "", upstream("the Basic Information page has no App-Level Tokens section")
	}
	// Every element that holds both is an ancestor of both, so in document
	// order the last one is the innermost.
	section := s.page.Locator("div, section").
		Filter(playwright.LocatorFilterOptions{Has: heading}).
		Filter(playwright.LocatorFilterOptions{Has: open}).
		Last()
	return section.InnerText()
}

// addAppTokenScope opens the Add Scope picker in the token dialog, picks
// scope, and confirms the dialog now lists it.
func (s *Session) addAppTokenScope(dialog playwright.Locator, scope string) error {
	add := dialog.GetByRole("button", playwright.LocatorGetByRoleOptions{Name: addScopeButton})
	if !s.visible(add, s.timeout) {
		return upstream("the token dialog has no Add Scope button")
	}
	if err := add.First().Click(); err != nil {
		return err
	}
	exact := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(scope) + `\b`)
	option := s.page.GetByRole("option", playwright.PageGetByRoleOptions{Name: exact}).
		Or(s.page.GetByRole("menuitem", playwright.PageGetByRoleOptions{Name: exact})).
		Or(s.page.GetByRole("menuitemcheckbox", playwright.PageGetByRoleOptions{Name: exact}))
	if !s.visible(option, s.timeout) {
		return fmt.Errorf("%q: %w", scope, ErrScopeNotOffered)
	}
	if err := option.First().Click(); err != nil {
		return err
	}
	if !s.visible(dialog.GetByText(scope, playwright.LocatorGetByTextOptions{Exact: playwright.Bool(true)}), s.timeout) {
		return upstream("the scope %s was not added to the token dialog", scope)
	}
	return nil
}

// readAppToken waits for the token Slack shows after Generate, in the
// dialog's text or one of its inputs, falling back to its Copy button.
func (s *Session) readAppToken() (string, error) {
	copyButton := s.page.GetByRole("dialog").GetByRole("button", playwright.LocatorGetByRoleOptions{Name: regexp.MustCompile(`(?i)copy`)})
	deadline := time.Now().Add(time.Duration(s.timeout) * time.Millisecond)
	for time.Now().Before(deadline) {
		v, err := s.page.Evaluate(`() => [...document.querySelectorAll('[role="dialog"]')].flatMap(d => [d.textContent || '', ...[...d.querySelectorAll('input, textarea')].map(i => i.value)])`)
		if err != nil {
			return "", err
		}
		if token := firstMatch(appTokenPattern, toStrings(v)); token != "" {
			return token, nil
		}
		if ok, _ := copyButton.First().IsVisible(); ok {
			// A failed click is retried on the next pass; finding the
			// token is what decides success.
			_ = copyButton.First().Click()
			copied, err := s.copied()
			if err != nil {
				return "", err
			}
			if token := firstMatch(appTokenPattern, copied); token != "" {
				return token, nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return "", upstream("no app-level token appeared after Generate")
}
