//go:build !windows

package shims

import (
	"os"
	"syscall"
)

func launch(exe string) (int, error) {
	return 0, syscall.Exec(exe, append([]string{exe}, os.Args[1:]...), os.Environ())
}

func Launch(exe string, args []string) (int, error) {
	return 0, syscall.Exec(exe, append([]string{exe}, args...), os.Environ())
}
