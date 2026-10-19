package secretout

import (
	"bytes"
	"errors"
	"testing"
)

func TestWriteRefusesATerminal(t *testing.T) {
	var buf bytes.Buffer
	err := Write(&buf, true, []byte("xoxb-secret"))
	if !errors.Is(err, ErrTerminal) {
		t.Fatalf("err = %v, want ErrTerminal", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("wrote %d bytes to a terminal", buf.Len())
	}
}

func TestWriteSendsExactBytesToAPipe(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, false, []byte("xoxb-secret")); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "xoxb-secret" {
		t.Fatalf("wrote %q", got)
	}
}

func TestErrorTextRedactsTokensAndCallLogs(t *testing.T) {
	err := errors.New("timeout waiting for <input value=\"xoxb-1-2-abc\"> and xoxe.xoxp-1-Mi0\nCall log:\n  xoxe-1-My0t")
	if got, want := ErrorText(err), "timeout waiting for <input value=\"[redacted]\"> and [redacted]"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
