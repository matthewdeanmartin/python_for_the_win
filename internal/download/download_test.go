package download

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestFetch(t *testing.T) {
	for _, status := range []int{200, 404, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if status == 404 || status == 500 && calls == 1 {
					w.WriteHeader(status)
					return
				}
				fmt.Fprint(w, "payload")
			}))
			defer s.Close()
			c, _ := New("", true)
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte("payload")))
			cache := t.TempDir()
			p, e := c.Fetch(context.Background(), s.URL, hash, cache)
			if status == 404 {
				if e == nil || calls != 1 {
					t.Fatal(e, calls)
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			before := calls
			if _, e = c.Fetch(context.Background(), s.URL, hash, cache); e != nil || calls != before {
				t.Fatal("cache miss", e)
			}
			if !Matches(p, hash) {
				t.Fatal("bad payload")
			}
			if _, e = c.Fetch(context.Background(), s.URL, fmt.Sprintf("%064d", 0), cache); e == nil {
				t.Fatal("hash mismatch accepted")
			}
			parts, _ := filepath.Glob(filepath.Join(cache, "*.part"))
			if len(parts) != 0 {
				t.Fatal(parts)
			}
		})
	}
}
func TestHTTPSRequired(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer s.Close()
	c, _ := New("", false)
	if _, _, e := c.Open(context.Background(), s.URL); e == nil {
		t.Fatal("HTTP allowed")
	}
}
