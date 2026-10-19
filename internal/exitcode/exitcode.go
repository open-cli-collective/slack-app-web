// Package exitcode maps command errors onto the family exit-code taxonomy
// (cli-common scriptability.md §3.2).
package exitcode

import "errors"

// Exit codes.
const (
	Generic  = 1
	Usage    = 2
	Auth     = 3
	NotFound = 4
	Upstream = 5
)

// Error is an error that carries the process exit code it should produce.
type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }

func (e *Error) Unwrap() error { return e.Err }

// New wraps err with code.
func New(code int, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Err: err}
}

// Of returns the exit code for err: its Error code when it carries one,
// otherwise Generic.
func Of(err error) int {
	if err == nil {
		return 0
	}
	var coded *Error
	if errors.As(err, &coded) {
		return coded.Code
	}
	return Generic
}
