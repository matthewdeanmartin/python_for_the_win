package feed

import (
	"context"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/download"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPagingCacheAndFiltering(t *testing.T) {
	calls := map[string]int{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls[r.URL.Path]++
		if r.URL.Path == "/next.json" {
			fmt.Fprint(w, `{"versions":[]}`)
			return
		}
		fmt.Fprint(w, `{"future":true,"versions":[{"schema":1,"company":"PythonCore","url":"runtime.zip"},{"schema":1,"company":"PythonEmbed"},{"schema":2,"company":"PythonCore"}],"next":"next.json"}`)
	}))
	defer s.Close()
	c, _ := download.New("", true)
	cache := t.TempDir()
	it, _ := Load(c, s.URL+"/index.json", cache, false)
	v, e := it.Next(context.Background())
	if e != nil || len(v) != 1 || v[0].URL != s.URL+"/runtime.zip" || v[0].Installable() {
		t.Fatal(v, e)
	}
	if calls["/next.json"] != 0 {
		t.Fatal("eager page")
	}
	if _, e = it.Next(context.Background()); e != nil {
		t.Fatal(e)
	}
	if _, e = it.Next(context.Background()); e != io.EOF {
		t.Fatal(e)
	}
	it, _ = Load(c, s.URL+"/index.json", cache, false)
	_, _ = it.Next(context.Background())
	if calls["/index.json"] != 1 {
		t.Fatal(calls)
	}
	files, _ := filepath.Glob(filepath.Join(cache, "index-*.json"))
	for _, p := range files {
		_ = os.Chtimes(p, time.Now().Add(-2*time.Hour), time.Now().Add(-2*time.Hour))
	}
	it, _ = Load(c, s.URL+"/index.json", cache, false)
	_, _ = it.Next(context.Background())
	if calls["/index.json"] != 2 {
		t.Fatal(calls)
	}
	it, _ = Load(c, s.URL+"/index.json", cache, true)
	_, _ = it.Next(context.Background())
	if calls["/index.json"] != 3 {
		t.Fatal(calls)
	}
}
func TestLocalAndMalformed(t *testing.T) {
	r := t.TempDir()
	c, _ := download.New("", false)
	p := filepath.Join(r, "index.json")
	for _, data := range []string{`{`, `{}`, `null`} {
		_ = os.WriteFile(p, []byte(data), 0600)
		it, _ := Load(c, r, "", true)
		if _, e := it.Next(context.Background()); e == nil {
			t.Fatal("accepted", data)
		}
	}
	_ = os.WriteFile(p, []byte(`{"versions":[{"schema":1,"company":"PythonCore","url":"payload.zip"}],"next":"index.json"}`), 0600)
	it, _ := Load(c, r, "", true)
	v, e := it.Next(context.Background())
	if e != nil || v[0].URL != filepath.Join(r, "payload.zip") {
		t.Fatal(v, e)
	}
	if _, e = it.Next(context.Background()); e == nil {
		t.Fatal("cycle accepted")
	}
}
func TestSourcePrecedence(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.json"), []byte(`{}`), 0600)
	for _, tc := range []struct{ f, e, d, c, w string }{{"flag", "env", dir, "config", "flag"}, {"", "env", dir, "config", "env"}, {"", "", dir, "config", filepath.Join(dir, "index.json")}, {"", "", "", "config", "config"}, {"", "", "", "", DefaultSource}} {
		got, _ := Source(tc.f, tc.e, tc.d, tc.c)
		if got != tc.w {
			t.Fatal(got, tc.w)
		}
	}
}

func TestInsecureCacheCannotBypassPolicy(t *testing.T) {
	c, _ := download.New("", false)
	if _, e := Load(c, "http://example.invalid/index.json", t.TempDir(), false); e == nil {
		t.Fatal("insecure source accepted before cache lookup")
	}
}

func TestExactUnavailableReasons(t *testing.T) {
	e := Entry{URL: "https://example.test/python.nupkg"}
	reasons := e.UnavailableReasons()
	if len(reasons) != 2 || reasons[1] != "feed has no SHA-256 checksum" {
		t.Fatal(reasons)
	}
	e.URL = ""
	if reasons = e.UnavailableReasons(); reasons[0] != "feed has no download URL" {
		t.Fatal(reasons)
	}
	e.URL = "https://example.test/python.zip"
	e.Hash = map[string]string{"sha256": "bad"}
	if reasons = e.UnavailableReasons(); len(reasons) != 1 || reasons[0] != "feed SHA-256 checksum is invalid" {
		t.Fatal(reasons)
	}
}

func TestLocalWindowsPayloadIsInstallable(t *testing.T) {
	e := Entry{URL: `C:\bundle\runtime.zip`, Hash: map[string]string{"sha256": strings.Repeat("a", 64)}}
	if !e.Installable() {
		t.Fatal(e.UnavailableReasons())
	}
}
