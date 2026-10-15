//go:build !windows

package platform

import (
	"os"
	"runtime"
)

func NativeArch() string {
	if runtime.GOARCH == "386" {
		return "x86"
	}
	return runtime.GOARCH
}

func IsTerminal(f *os.File) bool {
	i, e := f.Stat()
	return e == nil && i.Mode()&os.ModeCharDevice != 0
}
