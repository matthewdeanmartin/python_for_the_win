package fsops

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"time"
)

// Rename tolerates short-lived scanner/process handle release delays, never a persistent lock.
func Rename(old, new string) error {
	return retryRename(func() error { return os.Rename(old, new) }, time.Sleep)
}
func retryRename(fn func() error, sleep func(time.Duration)) error {
	var err error
	for attempt := 0; attempt < 6; attempt++ {
		err = fn()
		if err == nil || !errors.Is(err, windows.ERROR_SHARING_VIOLATION) && !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return err
		}
		if attempt < 5 {
			sleep(time.Duration(50*(1<<attempt)) * time.Millisecond)
		}
	}
	return err
}
