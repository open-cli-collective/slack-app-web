package cmdutil

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/open-cli-collective/slack-app-web/internal/exitcode"
	"github.com/open-cli-collective/slack-app-web/internal/secretout"
)

func TestSinkMistakesAreUsageErrorsNamingTheFlag(t *testing.T) {
	existing := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(existing, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		sink secretout.Sink
		want error
		flag string
	}{
		{secretout.Sink{}, secretout.ErrNoSink, "--to-command"},
		{secretout.Sink{File: "f", Command: "c"}, secretout.ErrTwoSinks, "--to-file"},
		{secretout.Sink{Command: "c", Overwrite: true}, secretout.ErrOverwriteWithoutFile, "--overwrite"},
		{secretout.Sink{File: existing}, secretout.ErrExists, "--overwrite"},
	}
	for _, c := range cases {
		err := CheckSink(c.sink)
		if !errors.Is(err, c.want) || exitcode.Of(err) != exitcode.Usage || !strings.Contains(err.Error(), c.flag) {
			t.Errorf("CheckSink(%+v) = %v (exit %d), want %v naming %s", c.sink, err, exitcode.Of(err), c.want, c.flag)
		}
	}
}

// A file that appears while the secret is read is the same mistake as one
// found before the browser opens, and exits the same way.
func TestAFileThatAppearsLaterIsStillAUsageError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	sink := secretout.Sink{File: path}
	if err := CheckSink(sink); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := &Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	err := opts.DeliverSecret(sink, []byte("xoxb-later"))
	if !errors.Is(err, secretout.ErrExists) || exitcode.Of(err) != exitcode.Usage {
		t.Fatalf("err = %v (exit %d), want a usage error", err, exitcode.Of(err))
	}
}

func TestAFailingDestinationIsNotAUsageError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	opts := &Options{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
	err := opts.DeliverSecret(secretout.Sink{Command: "cat >/dev/null; exit 4"}, []byte("xoxb-x"))
	if err == nil || exitcode.Of(err) == exitcode.Usage {
		t.Fatalf("err = %v (exit %d), want a non-usage failure", err, exitcode.Of(err))
	}
	dir := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	if os.Getuid() == 0 {
		return
	}
	err = CheckSink(secretout.Sink{File: filepath.Join(dir, "secret")})
	if err == nil || exitcode.Of(err) == exitcode.Usage {
		t.Fatalf("unreadable directory: err = %v (exit %d), want a non-usage failure", err, exitcode.Of(err))
	}
}
