package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/overlay"
)

var ErrIntegrity = errors.New("integrity check failed")

type Client struct {
	HTTP     *http.Client
	Insecure bool
	Progress func(int64, int64)
}

func New(proxy string, insecure bool) (*Client, error) {
	t := http.DefaultTransport.(*http.Transport).Clone()
	transport, e := configureProxy(t, proxy, systemProxyReader())
	if e != nil {
		return nil, e
	}
	h := &http.Client{Transport: transport, Timeout: 5 * time.Minute}

	h.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		if req.URL.Scheme != "https" && !(insecure && req.URL.Scheme == "http") {
			return fmt.Errorf("insecure redirect rejected")
		}
		return nil
	}
	return &Client{HTTP: h, Insecure: insecure}, nil
}
func (c *Client) Open(ctx context.Context, source string) (io.ReadCloser, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	if strings.HasPrefix(source, overlay.Scheme+"://") {
		r, n, e := overlay.Open(source)
		if e != nil {
			return nil, 0, errs.Wrap(6, "cannot read verified offline bundle", e)
		}
		return r, n, nil
	}
	if !strings.Contains(source, "://") {
		f, e := os.Open(source)
		if e != nil {
			return nil, 0, e
		}
		i, e := f.Stat()
		if e != nil {
			_ = f.Close() // Read-only or already-failed operation; preserve the primary error.
			return nil, 0, e
		}
		return f, i.Size(), nil
	}
	u, e := url.Parse(source)
	if e != nil || u.Host == "" {
		return nil, 0, fmt.Errorf("invalid source URL")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && c.Insecure) {
		return nil, 0, fmt.Errorf("source must use HTTPS (HTTP requires --allow-insecure-source)")
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		req, e := http.NewRequestWithContext(ctx, "GET", source, nil)
		if e != nil {
			return nil, 0, e
		}
		req.Header.Set("User-Agent", "pftw/0.1")
		r, e := c.HTTP.Do(req)
		if e == nil {
			if r.StatusCode == 200 {
				return r.Body, r.ContentLength, nil
			}
			_ = r.Body.Close() // Read-only or already-failed operation; preserve the primary error.
			last = fmt.Errorf("HTTP %d", r.StatusCode)
			if r.StatusCode < 500 {
				break
			}
		} else {
			last = e
		}
		if attempt < 2 {
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(time.Duration(1<<attempt) * 100 * time.Millisecond):
			}
		}
	}
	return nil, 0, errs.Wrap(5, "download failed", last)
}
func ValidHash(s string) bool { b, e := hex.DecodeString(s); return e == nil && len(b) == 32 }
func Matches(path, hash string) bool {
	f, e := os.Open(path)
	if e != nil {
		return false
	}
	defer func() { _ = f.Close() }() // Read-only handle; read errors are checked above.
	h := sha256.New()
	_, e = io.Copy(h, f)
	return e == nil && strings.EqualFold(hex.EncodeToString(h.Sum(nil)), hash)
}

type meter struct {
	n, total int64
	f        func(int64, int64)
}

func (m *meter) Write(b []byte) (int, error) {
	m.n += int64(len(b))
	if m.f != nil {
		m.f(m.n, m.total)
	}
	return len(b), nil
}
func (c *Client) Fetch(ctx context.Context, source, hash, cache string) (path string, resultErr error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !ValidHash(hash) {
		return "", errs.Wrap(6, "missing or invalid sha256", ErrIntegrity)
	}
	if e := os.MkdirAll(cache, 0700); e != nil {
		return "", e
	}
	dest := filepath.Join(cache, strings.ToLower(hash)+".zip")
	if Matches(dest, hash) {
		return dest, nil
	}
	if e := os.Remove(dest); e != nil && !os.IsNotExist(e) {
		return "", e
	}
	r, total, e := c.Open(ctx, source)
	if e != nil {
		var coded *errs.Error
		if errors.As(e, &coded) && coded.Code == 6 {
			return "", fmt.Errorf("cannot read runtime: %w", e)
		}
		return "", errs.Wrap(5, "cannot read runtime", e)
	}
	defer func() { _ = r.Close() }() // Read-only handle; read errors are checked above.
	f, e := os.CreateTemp(cache, "download-*.part")
	if e != nil {
		return "", e
	}
	tmp := f.Name()
	defer func() {
		if cleanupErr := os.Remove(tmp); cleanupErr != nil && !os.IsNotExist(cleanupErr) {
			resultErr = errors.Join(resultErr, fmt.Errorf("remove partial download %s: %w", tmp, cleanupErr))
		}
	}()
	h := sha256.New()
	_, e = io.Copy(io.MultiWriter(f, h, &meter{total: total, f: c.Progress}), contextReader{ctx: ctx, r: r})
	if e == nil {
		e = ctx.Err()
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return "", errs.Wrap(5, "incomplete download", e)
	}
	if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), hash) {
		return "", errs.Wrap(6, "runtime sha256 mismatch", ErrIntegrity)
	}
	if e = os.Rename(tmp, dest); e != nil {
		return "", e
	}
	return dest, nil
}

// Local files and bundle streams need cancellation checks as well as HTTP bodies.
type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	return r.r.Read(p)
}
