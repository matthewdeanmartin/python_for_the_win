// Package selfupdate verifies release archives before replacing executable files.
package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/archive"
	"github.com/matthewdeanmartin/python_for_the_win/internal/download"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"golang.org/x/mod/semver"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const API = "https://api.github.com/repos/matthewdeanmartin/python_for_the_win/releases"

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}
type Release struct {
	Tag    string  `json:"tag_name"`
	Draft  bool    `json:"draft"`
	Pre    bool    `json:"prerelease"`
	Assets []Asset `json:"assets"`
}
type Client struct {
	Preview  bool
	Download *download.Client
	API      string
}

func Version(v string) string {
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}
func Scoop(path string) bool {
	return strings.Contains(strings.ToLower(strings.ReplaceAll(path, string(rune(92)), "/")), "/scoop/apps/pftw/")
}
func (c Client) read(ctx context.Context, url string, limit int64) ([]byte, error) {
	r, _, e := c.Download.Open(ctx, url)
	if e != nil {
		if strings.Contains(e.Error(), "403") {
			return nil, fmt.Errorf("GitHub refused the request (403); the unauthenticated API rate limit may be exhausted. Retry later: %w", e)
		}
		return nil, e
	}
	defer r.Close()
	b, e := io.ReadAll(io.LimitReader(r, limit+1))
	if e == nil && int64(len(b)) > limit {
		e = fmt.Errorf("release metadata exceeds size limit")
	}
	return b, e
}
func (c Client) Latest(ctx context.Context, current, arch string, pre, force bool, root string) (exe, version string, err error) {
	if current == "dev" && !force {
		return "", "", fmt.Errorf("development/go install builds do not self-update; use --force to replace this build")
	}
	if current != "dev" && !semver.IsValid(Version(current)) {
		return "", "", fmt.Errorf("invalid current version %q", current)
	}
	api := c.API
	if api == "" {
		api = API
	}
	if !pre {
		api += "/latest"
	}
	b, e := c.read(ctx, api, 4<<20)
	if e != nil {
		return "", "", e
	}
	releases := []Release{}
	if pre {
		e = json.Unmarshal(b, &releases)
	} else {
		var r Release
		e = json.Unmarshal(b, &r)
		releases = append(releases, r)
	}
	if e != nil {
		return "", "", e
	}
	best := Release{}
	for _, r := range releases {
		if !r.Draft && (pre || !r.Pre && semver.Prerelease(Version(r.Tag)) == "") && semver.IsValid(Version(r.Tag)) && (best.Tag == "" || semver.Compare(Version(r.Tag), Version(best.Tag)) > 0) {
			best = r
		}
	}
	if best.Tag == "" {
		return "", "", fmt.Errorf("no eligible pftw release is available")
	}
	if current != "dev" && semver.Compare(Version(best.Tag), Version(current)) <= 0 {
		return "", best.Tag, nil
	}
	name := "pftw_" + strings.TrimPrefix(best.Tag, "v") + "_windows_" + arch + ".zip"
	url, checks := "", ""
	for _, a := range best.Assets {
		if a.Name == name {
			url = a.URL
		}
		if a.Name == "checksums.txt" {
			checks = a.URL
		}
	}
	if url == "" || checks == "" {
		return "", "", fmt.Errorf("Release %s is missing %s or checksums.txt", best.Tag, name)
	}
	b, e = c.read(ctx, checks, 4<<20)
	if e != nil {
		return "", "", e
	}
	hash := ""
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			if hash != "" {
				return "", "", fmt.Errorf("duplicate checksum entry")
			}
			hash = f[0]
		}
	}
	if hash == "" {
		return "", "", fmt.Errorf("no checksum for %s", name)
	}
	if !download.ValidHash(hash) {
		return "", "", fmt.Errorf("invalid release checksum")
	}
	if c.Preview {
		return url, best.Tag, nil
	}
	cache, e := layout.Safe(root, "cache")
	if e != nil {
		return "", "", e
	}
	zip, e := c.Download.Fetch(ctx, url, hash, cache)
	if e != nil {
		return "", "", e
	}
	staging, e := layout.Safe(root, "staging")
	if e != nil {
		return "", "", e
	}
	if e = os.MkdirAll(staging, 0700); e != nil {
		return "", "", e
	}
	dir, e := os.MkdirTemp(staging, "pftw-update-")
	if e != nil {
		return "", "", e
	}
	if e = archive.Extract(zip, dir); e != nil {
		_ = os.RemoveAll(dir)
		return "", "", e
	}
	exe = filepath.Join(dir, "pftw.exe")
	if i, e := os.Stat(exe); e != nil || !i.Mode().IsRegular() {
		_ = os.RemoveAll(dir)
		return "", "", fmt.Errorf("release archive has no pftw.exe")
	}
	return exe, best.Tag, nil
}

// Replace stages bytes on the same volume, then retains the running executable as .old.
func Replace(source, target string) error {
	parent := filepath.Dir(target)
	name := filepath.Base(target)
	for _, n := range []string{name, name + ".new", name + ".old"} {
		if _, e := layout.Safe(parent, n); e != nil {
			return e
		}
	}
	in, e := os.Open(source)
	if e != nil {
		return e
	}
	defer in.Close()
	tmp, e := os.OpenFile(target+".new", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if e != nil {
		return e
	}
	_, e = io.Copy(tmp, in)
	if e == nil {
		e = tmp.Sync()
	}
	ce := tmp.Close()
	if e == nil {
		e = ce
	}
	defer os.Remove(target + ".new")
	if e != nil {
		return e
	}
	if _, e = os.Lstat(target + ".old"); !os.IsNotExist(e) {
		return fmt.Errorf("previous executable backup exists; restart pftw before updating")
	}
	if e = os.Rename(target, target+".old"); e != nil {
		return e
	}
	if e = os.Rename(target+".new", target); e != nil {
		re := os.Rename(target+".old", target)
		return fmt.Errorf("publish failed: %w (restore: %v)", e, re)
	}
	return nil
}

// Cleanup removes only the executable's exact backup, never a wildcard or directory.
func Cleanup(target string) {
	if i, e := os.Stat(target); e != nil || !i.Mode().IsRegular() {
		return
	}
	p, e := layout.Safe(filepath.Dir(target), filepath.Base(target)+".old")
	if e != nil {
		return
	}
	if i, e := os.Lstat(p); e == nil && i.Mode().IsRegular() {
		_ = os.Remove(p)
	}
}
