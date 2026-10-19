package slackweb

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// App is one row of the Your Apps table.
type App struct {
	Name      string
	ID        string
	Workspace string
}

// ConfigToken is an app configuration token pair. Its JSON field names are
// Slack's own, from tooling.tokens.rotate, so the pair can be handed to any
// tool that rotates it.
type ConfigToken struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	Exp          int64  `json:"exp"`
}

// configTokenLifetime is how long Slack keeps an app configuration access
// token valid, and the expiry assumed when the page does not state one.
const configTokenLifetime = 12 * time.Hour

var (
	appIDPattern         = regexp.MustCompile(`^A[A-Z0-9]{6,}$`)
	configAccessPattern  = regexp.MustCompile(`xoxe\.xoxp-[A-Za-z0-9-]+`)
	configRefreshPattern = regexp.MustCompile(`xoxe-[0-9]-[A-Za-z0-9-]+`)
	botTokenPattern      = regexp.MustCompile(`xoxb-[A-Za-z0-9-]+`)
	expiresInPattern     = regexp.MustCompile(`(?i)expires in (\d+) (minute|hour|day)s?`)
)

// parseAppTable reads the Your Apps table: the first row is the header, and
// columns are found by header text so a reordered or widened table still
// parses. Rows without a valid app id are skipped.
func parseAppTable(rows [][]string) []App {
	if len(rows) == 0 {
		return nil
	}
	name, id, workspace := -1, -1, -1
	for i, h := range rows[0] {
		switch h = strings.ToLower(strings.TrimSpace(h)); {
		case strings.Contains(h, "app name"):
			name = i
		case strings.Contains(h, "app id"):
			id = i
		case strings.Contains(h, "workspace"):
			workspace = i
		}
	}
	if id < 0 {
		return nil
	}
	cell := func(row []string, i int) string {
		if i < 0 || i >= len(row) {
			return ""
		}
		return strings.Join(strings.Fields(row[i]), " ")
	}
	var apps []App
	for _, row := range rows[1:] {
		appID := cell(row, id)
		if !appIDPattern.MatchString(appID) {
			continue
		}
		apps = append(apps, App{Name: cell(row, name), ID: appID, Workspace: cell(row, workspace)})
	}
	return apps
}

// parseExpiry reads "Expires in N hours" from the token row and returns the
// expiry as unix seconds, falling back to the documented 12 hours.
func parseExpiry(text string, now time.Time) int64 {
	m := expiresInPattern.FindStringSubmatch(text)
	if m == nil {
		return now.Add(configTokenLifetime).Unix()
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return now.Add(configTokenLifetime).Unix()
	}
	unit := time.Hour
	switch strings.ToLower(m[2]) {
	case "minute":
		unit = time.Minute
	case "day":
		unit = 24 * time.Hour
	}
	return now.Add(time.Duration(n) * unit).Unix()
}

// firstMatch returns the first match of pattern across candidates.
func firstMatch(pattern *regexp.Regexp, candidates []string) string {
	for _, c := range candidates {
		if m := pattern.FindString(c); m != "" {
			return m
		}
	}
	return ""
}

// ValidAppID reports whether id looks like a Slack app id.
func ValidAppID(id string) bool { return appIDPattern.MatchString(id) }

// clipboardHook replaces the page's clipboard writes with a capture into
// window.__slawCopied, so a Copy button hands its value to the CLI without
// touching the system clipboard.
const clipboardHook = `(() => {
  window.__slawCopied = [];
  const keep = (t) => { window.__slawCopied.push(String(t)); };
  if (navigator.clipboard) { navigator.clipboard.writeText = async (t) => keep(t); }
  const exec = document.execCommand.bind(document);
  document.execCommand = (cmd, ...rest) => {
    if (cmd === 'copy') {
      const s = window.getSelection && window.getSelection().toString();
      const a = document.activeElement;
      keep(s || (a && a.value) || '');
      return true;
    }
    return exec(cmd, ...rest);
  };
})();`
