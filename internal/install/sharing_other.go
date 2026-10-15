//go:build !windows

package install

func sharingError(e error) error { return e }
