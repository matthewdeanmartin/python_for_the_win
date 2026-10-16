package download

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestFetchSizeLimit(t *testing.T) {
	for _, chunked := range []bool{false, true} {
		for _, size := range []int{8, 9} {
			t.Run(fmt.Sprintf("chunked=%t/size=%d", chunked, size), func(t *testing.T) {
				payload := make([]byte, size)
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if chunked {
						w.(http.Flusher).Flush()
					}
					_, _ = w.Write(payload)
				}))
				defer s.Close()
				c, err := New("", true)
				if err != nil {
					t.Fatal(err)
				}
				cache := t.TempDir()
				hash := fmt.Sprintf("%x", sha256.Sum256(payload))
				path, err := c.fetch(context.Background(), s.URL, hash, cache, 8)
				if size == 8 {
					if err != nil || !Matches(path, hash) {
						t.Fatalf("boundary payload rejected: %v", err)
					}
				} else {
					if !errors.Is(err, ErrIntegrity) || path != "" {
						t.Fatalf("oversize accepted: path=%q err=%v", path, err)
					}
					entries, err := os.ReadDir(cache)
					if err != nil || len(entries) != 0 {
						t.Fatalf("partial download left behind: %v %v", entries, err)
					}
				}
			})
		}
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
