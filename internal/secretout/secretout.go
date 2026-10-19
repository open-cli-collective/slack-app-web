// Package secretout writes secret values to stdout only when stdout is not a
// terminal, so a token reaches a pipe or a file and never a screen, a
// terminal scrollback or an agent transcript.
package secretout

import (
	"errors"
	"io"
	"os"

	"golang.org/x/term"
)

// ErrTerminal is returned when stdout is a terminal.
var ErrTerminal = errors.New("refusing to print a secret to a terminal: pipe stdout into a secret store, for example `| op item create ...` or `> file`")

// IsTerminal reports whether f is attached to a terminal.
func IsTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd())) //nolint:gosec // Fd fits in int on every supported platform
}

// Write writes secret to w unless isTerminal is true. It adds no trailing
// newline, so the bytes a pipe receives are exactly the secret.
func Write(w io.Writer, isTerminal bool, secret []byte) error {
	if isTerminal {
		return ErrTerminal
	}
	_, err := w.Write(secret)
	return err
}
