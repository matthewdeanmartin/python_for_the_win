//go:build !windows

package platform

import "fmt"

func (Native) IsElevated() (bool, error)        { return false, fmt.Errorf("Windows probe unavailable") }
func (Native) FreeBytes(string) (uint64, error) { return 0, fmt.Errorf("Windows probe unavailable") }
func (Native) OwnsConsole() (bool, error)       { return false, fmt.Errorf("Windows probe unavailable") }
func (Native) IsNetworkPath(string) (bool, error) {
	return false, fmt.Errorf("Windows probe unavailable")
}
func (Native) IsAppExecLink(string) (bool, error) {
	return false, fmt.Errorf("Windows probe unavailable")
}
