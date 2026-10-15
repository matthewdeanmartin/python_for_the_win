//go:build !windows

package platform

import "errors"

var errUnsupported = errors.New("probe unavailable: requires Windows")

func (Native) IsElevated() (bool, error)        { return false, errUnsupported }
func (Native) FreeBytes(string) (uint64, error) { return 0, errUnsupported }
func (Native) OwnsConsole() (bool, error)       { return false, errUnsupported }
func (Native) IsNetworkPath(string) (bool, error) {
	return false, errUnsupported
}
func (Native) IsAppExecLink(string) (bool, error) {
	return false, errUnsupported
}
