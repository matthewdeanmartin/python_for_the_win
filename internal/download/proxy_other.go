//go:build !windows

package download

func systemProxyReader() ProxyReader { return nil }
