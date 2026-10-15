package download

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestStaticProxySelection(t *testing.T) {
	p := StaticProxy{Enabled: true, Server: "http=plain:80;https=secure:8080", Bypass: "<local>;*.internal;exact.example"}
	for _, tc := range []struct{ url, want string }{{"https://python.org/x", "http://secure:8080"}, {"http://python.org/x", "http://plain:80"}, {"https://printer/x", ""}, {"https://a.internal/x", ""}, {"https://exact.example/x", ""}, {"https://notinternal/x", ""}} {
		u, _ := url.Parse(tc.url)
		got, e := p.URL(u)
		if e != nil {
			t.Fatal(e)
		}
		text := ""
		if got != nil {
			text = got.String()
		}
		if text != tc.want {
			t.Fatal(tc, text)
		}
	}
	p.Server = "socks=foo:10"
	u, _ := url.Parse("https://python.org")
	if got, e := p.URL(u); e != nil || got != nil {
		t.Fatal(got, e)
	}
	p.Server = "http://user:secret@proxy:80"
	if _, e := p.URL(u); e == nil || strings.Contains(e.Error(), "secret") {
		t.Fatal(e)
	}
}
func clearProxyEnv(t *testing.T) {
	for _, k := range []string{"HTTP_PROXY", "http_proxy", "HTTPS_PROXY", "https_proxy", "NO_PROXY", "no_proxy", "REQUEST_METHOD"} {
		t.Setenv(k, "")
	}
}
func TestFallbackAfterDirectFailure(t *testing.T) {
	clearProxyEnv(t)
	reads, hits := 0, 0
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Host != "offline.invalid" {
			t.Error(r.URL)
		}
		fmt.Fprint(w, "via proxy")
	}))
	defer proxy.Close()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if strings.HasPrefix(address, "offline.invalid:") {
			return nil, fmt.Errorf("direct network unavailable")
		}
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}
	rt, e := configureProxy(transport, "", func() (StaticProxy, error) { reads++; return StaticProxy{Enabled: true, Server: proxy.URL}, nil })
	if e != nil {
		t.Fatal(e)
	}
	client := &http.Client{Transport: rt}
	response, e := client.Get("http://offline.invalid/payload")
	if e != nil {
		t.Fatal(e)
	}
	b, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if string(b) != "via proxy" || reads != 1 || hits != 1 {
		t.Fatal(string(b), reads, hits)
	}
}
func TestProxyDoesNotReadRegistryOnSuccessOrHTTPFailure(t *testing.T) {
	for _, status := range []int{200, 404, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			clearProxyEnv(t)
			reads := 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
			defer s.Close()
			rt, e := configureProxy(http.DefaultTransport.(*http.Transport).Clone(), "", func() (StaticProxy, error) { reads++; return StaticProxy{}, nil })
			if e != nil {
				t.Fatal(e)
			}
			r, e := (&http.Client{Transport: rt}).Get(s.URL)
			if e != nil {
				t.Fatal(e)
			}
			r.Body.Close()
			if reads != 0 {
				t.Fatal("read registry without connection failure")
			}
		})
	}
}
func TestConfiguredProxyDisablesFallback(t *testing.T) {
	for _, kind := range []string{"flag", "HTTPS_PROXY", "NO_PROXY"} {
		t.Run(kind, func(t *testing.T) {
			clearProxyEnv(t)
			explicit := ""
			if kind == "flag" {
				explicit = "http://127.0.0.1:9"
			} else {
				t.Setenv(kind, "example.org")
			}
			rt, e := configureProxy(http.DefaultTransport.(*http.Transport).Clone(), explicit, func() (StaticProxy, error) { t.Fatal("unexpected settings read"); return StaticProxy{}, nil })
			if e != nil {
				t.Fatal(e)
			}
			if _, ok := rt.(*proxyFallback); ok {
				t.Fatal("fallback enabled despite configuration")
			}
		})
	}
}
