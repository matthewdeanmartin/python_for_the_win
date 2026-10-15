package fsops

import (
	"errors"
	"golang.org/x/sys/windows"
	"testing"
	"time"
)

func TestRetryRename(t *testing.T) {
	for _, cause := range []error{windows.ERROR_SHARING_VIOLATION, windows.ERROR_ACCESS_DENIED, windows.ERROR_FILE_NOT_FOUND} {
		count := 0
		waited := time.Duration(0)
		e := retryRename(func() error { count++; return cause }, func(d time.Duration) { waited += d })
		want := 6
		if cause == windows.ERROR_FILE_NOT_FOUND {
			want = 1
		}
		if count != want || !errors.Is(e, cause) || waited > 2*time.Second {
			t.Fatal(count, waited, e)
		}
	}
	count := 0
	e := retryRename(func() error {
		count++
		if count < 3 {
			return windows.ERROR_SHARING_VIOLATION
		}
		return nil
	}, func(time.Duration) {})
	if e != nil || count != 3 {
		t.Fatal(count, e)
	}
}
