// Package version carries the build identity injected by ldflags.
package version

// Set by -ldflags at build time.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)
