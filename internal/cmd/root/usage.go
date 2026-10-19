package root

import "strings"

// isUsage recognizes cobra's own argument and flag errors, which arrive as
// plain errors.
func isUsage(err error) bool {
	msg := err.Error()
	for _, s := range []string{"unknown command", "unknown flag", "unknown shorthand flag", "accepts ", "requires at least", "invalid argument", "flag needs an argument"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}
