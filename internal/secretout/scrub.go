package secretout

import (
	"os"
	"regexp"
	"strings"
)

var tokenPattern = regexp.MustCompile(`xox[a-z]?[.-][A-Za-z0-9.-]+`)

// ErrorText renders err for stderr with anything shaped like a Slack token
// redacted: browser errors carry call logs that quote page elements, which
// can hold a token's value. Only the first line is kept unless
// SLAW_FULL_ERRORS=1, which keeps the (still redacted) call log.
func ErrorText(err error) string {
	msg := err.Error()
	if os.Getenv("SLAW_FULL_ERRORS") != "1" {
		msg, _, _ = strings.Cut(msg, "\n")
	}
	return tokenPattern.ReplaceAllString(msg, "[redacted]")
}
