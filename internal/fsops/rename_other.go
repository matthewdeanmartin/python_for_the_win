//go:build !windows

package fsops

import "os"

func Rename(old, new string) error { return os.Rename(old, new) }
