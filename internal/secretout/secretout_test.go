package secretout

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const secret = "xoxb-secret-value"

func TestCheckNeedsExactlyOneSink(t *testing.T) {
	if err := (Sink{}).Check(); !errors.Is(err, ErrNoSink) {
		t.Fatalf("no sink: err = %v", err)
	}
	if !strings.Contains(ErrNoSink.Error(), "--to-file") || !strings.Contains(ErrNoSink.Error(), "--to-command") {
		t.Fatalf("ErrNoSink does not name both flags: %q", ErrNoSink)
	}
	if err := (Sink{File: "f", Command: "c"}).Check(); !errors.Is(err, ErrTwoSinks) {
		t.Fatalf("two sinks: err = %v", err)
	}
	if err := (Sink{Command: "c", Overwrite: true}).Check(); err == nil {
		t.Fatal("--overwrite without --to-file was accepted")
	}
}

func TestToFileCreatesAnOwnerOnlyFileWithExactBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	var out, errOut bytes.Buffer
	if err := (Sink{File: path}).Write([]byte(secret), &out, &errOut); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != secret {
		t.Fatalf("file holds %q, err %v", got, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("mode %v, err %v", info.Mode().Perm(), err)
		}
	}
	assertNoSecret(t, out.String()+errOut.String())
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatalf("left %d entries in the directory, want only the token file", len(entries))
	}
}

func TestToFileRefusesToReplaceAFileUnlessOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (Sink{File: path}).Check(); err == nil || !strings.Contains(err.Error(), "--overwrite") {
		t.Fatalf("Check err = %v, want a refusal naming --overwrite", err)
	}
	if err := writeFile(path, []byte(secret), false); err == nil {
		t.Fatal("writeFile replaced a file that appeared after Check")
	}
	if got, _ := os.ReadFile(path); string(got) != "old" {
		t.Fatalf("file now holds %q", got)
	}
	if err := (Sink{File: path, Overwrite: true}).Write([]byte(secret), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != secret {
		t.Fatalf("overwrite left %q", got)
	}
}

func TestToCommandGetsTheSecretOnStdinAndPassesItsOutputThrough(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	dir := t.TempDir()
	var out, errOut bytes.Buffer
	cmd := "cat > " + filepath.Join(dir, "got") + "; echo stored; echo note >&2"
	if err := (Sink{Command: cmd}).Write([]byte(secret), &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "got")); string(got) != secret {
		t.Fatalf("command read %q", got)
	}
	if out.String() != "stored\n" || errOut.String() != "note\n" {
		t.Fatalf("stdout %q stderr %q", out.String(), errOut.String())
	}
}

func TestToCommandFailureNamesTheStatusNotTheSecret(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	var out, errOut bytes.Buffer
	err := (Sink{Command: "cat >/dev/null; exit 7"}).Write([]byte(secret), &out, &errOut)
	if err == nil || !strings.Contains(err.Error(), "status 7") {
		t.Fatalf("err = %v, want the exit status", err)
	}
	assertNoSecret(t, err.Error()+out.String()+errOut.String())
}

func TestErrorTextRedactsTokensAndCallLogs(t *testing.T) {
	err := errors.New("timeout waiting for <input value=\"xoxb-1-2-abc\"> and xoxe.xoxp-1-Mi0\nCall log:\n  xoxe-1-My0t")
	if got, want := ErrorText(err), "timeout waiting for <input value=\"[redacted]\"> and [redacted]"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func assertNoSecret(t *testing.T, s string) {
	t.Helper()
	if strings.Contains(s, secret) {
		t.Fatalf("secret reached output: %q", s)
	}
}
