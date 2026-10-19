package output

import (
	"bytes"
	"testing"
)

func TestTable(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, []string{"NAME", "ID"}, [][]string{{"a | b", "A1"}, {"two\nlines", ""}})
	want := "NAME | ID\na b | A1\ntwo lines | -\n"
	if got := buf.String(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
