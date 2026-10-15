//go:build !windows

package platform

func DesktopLaunch() bool { return false }
