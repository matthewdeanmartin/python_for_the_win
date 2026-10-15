package feed

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/download"
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/overlay"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultSource = "https://www.python.org/ftp/python/index-windows.json"

type Alias struct {
	Name     string `json:"name"`
	Target   string `json:"target"`
	Windowed int    `json:"windowed"`
}
type Entry struct {
	Schema      int               `json:"schema"`
	ID          string            `json:"id"`
	Version     string            `json:"sort-version"`
	Company     string            `json:"company"`
	Tag         string            `json:"tag"`
	InstallFor  []string          `json:"install-for"`
	Aliases     []Alias           `json:"alias"`
	DisplayName string            `json:"display-name"`
	Executable  string            `json:"executable"`
	URL         string            `json:"url"`
	Hash        map[string]string `json:"hash"`
}

func (e Entry) Key() string { return strings.ReplaceAll(e.Tag, "-dev", "") }
func (e Entry) UnavailableReasons() []string {
	reasons := []string{}
	if e.URL == "" {
		reasons = append(reasons, "feed has no download URL")
	} else {
		path := e.URL
		if u, err := url.Parse(e.URL); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
			path = u.Path
		}
		if strings.EqualFold(filepath.Ext(path), ".nupkg") {
			reasons = append(reasons, "NuGet package layout is not supported")
		} else if !strings.EqualFold(filepath.Ext(path), ".zip") {
			reasons = append(reasons, "payload is not a supported runtime ZIP")
		}
	}
	if e.Hash["sha256"] == "" {
		reasons = append(reasons, "feed has no SHA-256 checksum")
	} else if !download.ValidHash(e.Hash["sha256"]) {
		reasons = append(reasons, "feed SHA-256 checksum is invalid")
	}
	return reasons
}
func (e Entry) Installable() bool { return len(e.UnavailableReasons()) == 0 }

type Index struct {
	Versions []Entry `json:"versions"`
	Next     string  `json:"next"`
}

func Source(flag, env, exeDir, configured string) (string, string) {
	if flag != "" {
		return flag, "flag"
	}
	if env != "" {
		return env, "environment"
	}
	if exeDir != "" {
		p := filepath.Join(exeDir, "index.json")
		if i, e := os.Stat(p); e == nil && !i.IsDir() {
			return p, "bundle"
		}
	}
	if configured != "" {
		return configured, "config"
	}
	return DefaultSource, "default"
}
func Normalize(source string) (string, error) {
	if strings.HasPrefix(source, overlay.Scheme+"://") {
		_, _, e := overlay.Parse(source)
		return source, e
	}
	if strings.Contains(source, "://") {
		u, e := url.Parse(source)
		if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return "", fmt.Errorf("invalid feed URL")
		}
		return source, nil
	}
	p, e := filepath.Abs(source)
	if e != nil {
		return "", e
	}
	if i, e := os.Stat(p); e == nil && i.IsDir() {
		p = filepath.Join(p, "index.json")
	}
	return p, nil
}
func relative(base, ref string) (string, error) {
	if ref == "" {
		return "", nil
	}
	if strings.HasPrefix(base, overlay.Scheme+"://") {
		return overlay.Relative(base, ref)
	}
	if strings.Contains(base, "://") {
		b, _ := url.Parse(base)
		u, e := url.Parse(ref)
		if e != nil {
			return "", e
		}
		r := b.ResolveReference(u)
		if r.Scheme != "http" && r.Scheme != "https" {
			return "", fmt.Errorf("remote index cannot reference local files")
		}
		return r.String(), nil
	}
	if strings.Contains(ref, "://") || filepath.IsAbs(ref) {
		return ref, nil
	}
	return filepath.Join(filepath.Dir(base), filepath.FromSlash(ref)), nil
}

type Iterator struct {
	Client  *download.Client
	Cache   string
	Refresh bool
	next    string
	seen    map[string]bool
	pages   int
}

func Load(c *download.Client, source, cache string, refresh bool) (*Iterator, error) {
	s, e := Normalize(source)
	if e != nil {
		return nil, e
	}
	if strings.HasPrefix(s, "http://") && !c.Insecure {
		return nil, fmt.Errorf("HTTP source requires --allow-insecure-source")
	}
	if strings.HasPrefix(s, overlay.Scheme+"://") {
		cache = ""
	}
	return &Iterator{Client: c, Cache: cache, Refresh: refresh, next: s, seen: map[string]bool{}}, nil
}
func (i *Iterator) Next(ctx context.Context) ([]Entry, error) {
	if i.next == "" {
		return nil, io.EOF
	}
	source := i.next
	if i.seen[source] || i.pages >= 100 {
		return nil, fmt.Errorf("feed paging cycle or page limit reached")
	}
	i.seen[source] = true
	i.pages++
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(source)))
	cache := filepath.Join(i.Cache, "index-"+hash[:12]+".json")
	fetched := false
	var data []byte
	var e error
	if i.pages == 1 && !i.Refresh && i.Cache != "" {
		if info, err := os.Stat(cache); err == nil && time.Since(info.ModTime()) < time.Hour {
			data, _ = os.ReadFile(cache)
		}
	}
	if data == nil {
		fetched = true
		r, _, err := i.Client.Open(ctx, source)
		if err != nil {
			var coded *errs.Error
			if errors.As(err, &coded) && coded.Code == 6 {
				return nil, fmt.Errorf("cannot read Python index: %w", err)
			}
			return nil, errs.Wrap(5, "cannot read Python index", err)
		}
		data, e = io.ReadAll(io.LimitReader(r, (16<<20)+1))
		r.Close()
		if e != nil {
			return nil, e
		}
		if len(data) > 16<<20 {
			return nil, fmt.Errorf("index exceeds size limit")
		}
	}
	var idx Index
	if e = json.Unmarshal(data, &idx); e != nil {
		return nil, fmt.Errorf("invalid Python index: %w", e)
	}
	if idx.Versions == nil {
		return nil, fmt.Errorf("invalid Python index: missing versions array")
	}
	if fetched && i.pages == 1 && i.Cache != "" {
		if e = os.MkdirAll(i.Cache, 0700); e == nil {
			f, err := os.CreateTemp(i.Cache, "index-*.tmp")
			if err == nil {
				name := f.Name()
				_, err = f.Write(data)
				ce := f.Close()
				if err == nil && ce == nil {
					_ = os.Rename(name, cache)
				}
				_ = os.Remove(name)
			}
		}
	}
	i.next, e = relative(source, idx.Next)
	if e != nil {
		return nil, e
	}
	out := []Entry{}
	for _, v := range idx.Versions {
		if v.Schema != 1 || v.Company != "PythonCore" {
			continue
		}
		v.URL, e = relative(source, v.URL)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, nil
}
func (i *Iterator) All(ctx context.Context) ([]Entry, error) {
	out := []Entry{}
	for {
		v, e := i.Next(ctx)
		if e == io.EOF {
			return out, nil
		}
		if e != nil {
			return nil, e
		}
		out = append(out, v...)
	}
}
