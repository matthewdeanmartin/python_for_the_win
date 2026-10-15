package download

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/net/http/httpproxy"
)

// StaticProxy is a read-only snapshot. PAC/WPAD are never evaluated.
type StaticProxy struct {
	Enabled        bool
	Server, Bypass string
	Auto           bool
}
type ProxyReader func() (StaticProxy, error)

func (p StaticProxy) URL(target *url.URL) (*url.URL, error) {
	if !p.Enabled || strings.TrimSpace(p.Server) == "" {
		return nil, nil
	}
	host := strings.ToLower(target.Hostname())
	for _, rule := range strings.Split(p.Bypass, ";") {
		rule = strings.ToLower(strings.TrimSpace(rule))
		if rule == "" {
			continue
		}
		if rule == "<local>" && !strings.Contains(host, ".") && net.ParseIP(host) == nil {
			return nil, nil
		}
		if wildcardHost(rule, host) || wildcardHost(rule, strings.ToLower(target.Host)) {
			return nil, nil
		}
	}
	server := strings.TrimSpace(p.Server)
	if strings.Contains(server, "=") {
		server = ""
		for _, part := range strings.Split(p.Server, ";") {
			pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(pair) == 2 && strings.EqualFold(pair[0], target.Scheme) {
				server = strings.TrimSpace(pair[1])
				break
			}
		}
	}
	if server == "" {
		return nil, nil
	}
	if !strings.Contains(server, "://") {
		server = "http://" + server
	}
	u, e := url.Parse(server)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("invalid or unsupported Windows static proxy setting")
	}
	return u, nil
}
func wildcardHost(pattern, value string) bool {
	// Wildcards match hostnames only, not filesystem paths or URLs.
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == value
	}
	if !strings.HasPrefix(value, parts[0]) {
		return false
	}
	value = strings.TrimPrefix(value, parts[0])
	for _, part := range parts[1 : len(parts)-1] {
		i := strings.Index(value, part)
		if i < 0 {
			return false
		}
		value = value[i+len(part):]
	}
	return strings.HasSuffix(value, parts[len(parts)-1])
}

type proxyFallback struct {
	direct   *http.Transport
	read     ProxyReader
	once     sync.Once
	settings StaticProxy
	err      error
}

func (t *proxyFallback) RoundTrip(req *http.Request) (*http.Response, error) {
	response, e := t.direct.RoundTrip(req)
	if e == nil || req.Context().Err() != nil {
		return response, e
	}
	t.once.Do(func() { t.settings, t.err = t.read() })
	if t.err != nil {
		return nil, fmt.Errorf("direct connection failed; Windows proxy settings unavailable (PAC/WPAD unsupported): %w", e)
	}
	proxy, pe := t.settings.URL(req.URL)
	if pe != nil {
		return nil, fmt.Errorf("direct connection failed; %s (PAC/WPAD unsupported): %w", pe, e)
	}
	if proxy == nil {
		return nil, fmt.Errorf("direct connection failed; no applicable Windows static proxy (PAC/WPAD unsupported): %w", e)
	}
	transport := t.direct.Clone()
	transport.Proxy = http.ProxyURL(proxy)
	defer transport.CloseIdleConnections()
	response, pe = transport.RoundTrip(req)
	if pe != nil {
		return nil, fmt.Errorf("direct and Windows static-proxy connections failed (PAC/WPAD unsupported): %w", pe)
	}
	return response, nil
}
func configureProxy(t *http.Transport, explicit string, read ProxyReader) (http.RoundTripper, error) {
	environment := httpproxy.FromEnvironment()
	if explicit != "" {
		u, e := url.Parse(explicit)
		if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return nil, fmt.Errorf("invalid proxy URL")
		}
		t.Proxy = http.ProxyURL(u)
		return t, nil
	}
	envProxy := environment.ProxyFunc()
	t.Proxy = func(r *http.Request) (*url.URL, error) { return envProxy(r.URL) }
	if environment.HTTPProxy != "" || environment.HTTPSProxy != "" || environment.NoProxy != "" || read == nil {
		return t, nil
	}
	return &proxyFallback{direct: t, read: read}, nil
}
