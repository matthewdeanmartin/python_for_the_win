package download

import (
	"errors"
	"golang.org/x/sys/windows/registry"
)

func systemProxyReader() ProxyReader { return readStaticProxy }
func readStaticProxy() (StaticProxy, error) {
	k, e := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if errors.Is(e, registry.ErrNotExist) {
		return StaticProxy{}, nil
	}
	if e != nil {
		return StaticProxy{}, e
	}
	defer func() { _ = k.Close() }() // Read-only handle; read errors are checked above.
	enabled, _, e := k.GetIntegerValue("ProxyEnable")
	if e != nil && !errors.Is(e, registry.ErrNotExist) {
		return StaticProxy{}, e
	}
	server, _, e := k.GetStringValue("ProxyServer")
	if e != nil && !errors.Is(e, registry.ErrNotExist) {
		return StaticProxy{}, e
	}
	bypass, _, e := k.GetStringValue("ProxyOverride")
	if e != nil && !errors.Is(e, registry.ErrNotExist) {
		return StaticProxy{}, e
	}
	auto, _, e := k.GetStringValue("AutoConfigURL")
	if e != nil && !errors.Is(e, registry.ErrNotExist) {
		return StaticProxy{}, e
	}
	return StaticProxy{Enabled: enabled != 0, Server: server, Bypass: bypass, Auto: auto != ""}, nil
}
