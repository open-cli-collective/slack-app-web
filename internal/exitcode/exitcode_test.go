package exitcode

import (
	"errors"
	"fmt"
	"testing"
)

func TestOf(t *testing.T) {
	base := errors.New("boom")
	cases := []struct {
		err  error
		want int
	}{
		{nil, 0},
		{base, Generic},
		{New(Auth, base), Auth},
		{fmt.Errorf("wrapped: %w", New(NotFound, base)), NotFound},
	}
	for _, c := range cases {
		if got := Of(c.err); got != c.want {
			t.Errorf("Of(%v) = %d, want %d", c.err, got, c.want)
		}
	}
	if New(Auth, nil) != nil {
		t.Error("New(code, nil) should be nil")
	}
}
