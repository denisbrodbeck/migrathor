package migrathor

import (
	"fmt"
	"testing"
)

func TestDriverError(t *testing.T) {
	uerr := fmt.Errorf("some driver error")
	err := &DriverError{"you won't believe what happened next", uerr}

	got := err.Error()
	want := "you won't believe what happened next: some driver error"
	if got != want {
		t.Errorf("Error messages did not match\ngot  %q\nwant %q\n", got, want)
	}
	got = UnderlyingError(err).Error()
	want = "some driver error"
	if got != want {
		t.Errorf("underlying error does not match\ngot  %q\nwant %q\n", got, want)
	}
	got = UnderlyingError(fmt.Errorf("not DriverError")).Error()
	want = "not DriverError"
	if got != want {
		t.Errorf("underlying error returned something wrong\ngot  %q\nwant %q\n", got, want)
	}
}
