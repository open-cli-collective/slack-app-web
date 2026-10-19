// Package secretout delivers a secret to a destination the caller names: a
// file created for the owner alone, or the stdin of a command. A secret never
// goes to slaw's own stdout or stderr. A terminal check cannot tell an agent
// transcript or a CI log from a pipe into a secret store, so the destination
// has to be explicit.
package secretout

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ErrNoSink is returned when neither --to-file nor --to-command is set.
var ErrNoSink = errors.New("name where the secret goes with --to-file <path> or --to-command <command>; it is never printed")

// ErrTwoSinks is returned when both --to-file and --to-command are set.
var ErrTwoSinks = errors.New("--to-file and --to-command are exclusive; pass one")

// Sink is the destination of a secret.
type Sink struct {
	// File is a path to create with mode 0600.
	File string
	// Overwrite lets File replace an existing file.
	Overwrite bool
	// Command is a shell command that receives the secret on stdin.
	Command string
}

// Check validates the sink before any secret is read, so a mistake costs no
// browser session: exactly one destination, and a file that may be written.
func (s Sink) Check() error {
	switch {
	case s.File == "" && s.Command == "":
		return ErrNoSink
	case s.File != "" && s.Command != "":
		return ErrTwoSinks
	case s.Overwrite && s.File == "":
		return errors.New("--overwrite applies only to --to-file")
	}
	if s.File != "" && !s.Overwrite {
		if _, err := os.Lstat(s.File); err == nil {
			return fmt.Errorf("%s exists; pass --overwrite to replace it", s.File)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

// Write delivers secret to the sink. The command's own stdout and stderr go
// to stdout and stderr; the secret is only ever on the command's stdin.
func (s Sink) Write(secret []byte, stdout, stderr io.Writer) error {
	if err := s.Check(); err != nil {
		return err
	}
	if s.File != "" {
		return writeFile(s.File, secret, s.Overwrite)
	}
	return runCommand(s.Command, secret, stdout, stderr)
}

// writeFile writes secret to a temporary file in path's directory with mode
// 0600 and moves it into place, so a reader never sees a partial secret and a
// failure leaves no file behind. Without overwrite the move is a hard link,
// which fails when path exists, so a file created after Check is not replaced.
func writeFile(path string, secret []byte, overwrite bool) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(tmp.Name())
	}()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(secret); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if overwrite {
		return os.Rename(tmp.Name(), path)
	}
	if err := os.Link(tmp.Name(), path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%s exists; pass --overwrite to replace it", path)
		}
		return err
	}
	return nil
}

func runCommand(command string, secret []byte, stdout, stderr io.Writer) error {
	shell, flag := "sh", "-c"
	if runtime.GOOS == "windows" {
		shell, flag = "cmd", "/C"
	}
	cmd := exec.Command(shell, flag, command) //nolint:gosec // running the caller's own command is the purpose of --to-command
	cmd.Stdin = bytes.NewReader(secret)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return fmt.Errorf("--to-command exited with status %d", exit.ExitCode())
		}
		return fmt.Errorf("--to-command could not run: %w", err)
	}
	return nil
}
