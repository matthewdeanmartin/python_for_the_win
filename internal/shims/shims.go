package shims

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"github.com/matthewdeanmartin/python_for_the_win/internal/overlay"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/resolve"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var pattern = regexp.MustCompile(`^pythonw?3(?:\.\d+)?t?(?:-(?:64|32|arm64))?\.exe$`)

func IsName(name string) bool { return pattern.MatchString(strings.ToLower(name)) }
func Resolve(s *state.State, name, arch string) (string, error) {
	name = strings.ToLower(name)
	name = strings.Replace(name, "pythonw", "python", 1)
	if !IsName(name) {
		return "", fmt.Errorf("unsupported shim name")
	}
	key := s.Default
	if name != "python3.exe" {
		spec := strings.TrimSuffix(strings.TrimPrefix(name, "python"), ".exe")
		var e error
		key, _, e = resolve.Normalize(resolve.Options{Spec: spec, Arch: arch})
		if e != nil {
			return "", e
		}
	}
	if _, e := s.Get(key); e != nil {
		return "", e
	}
	return key, nil
}
func Names(s *state.State) []string {
	names := map[string]bool{}
	if s.Default != "" {
		names["python3.exe"] = true
	}
	for key := range s.Installs {
		v, e := s.Get(key)
		if e != nil {
			continue
		}
		names["python"+key+".exe"] = true
		for _, n := range v.Aliases {
			n = strings.ToLower(n)
			if IsName(n) && (strings.HasPrefix(n, "python3.") || strings.HasPrefix(n, "pythonw3.")) {
				names[n] = true
			}
		}
		if v.Arch == platform.NativeArch() {
			minor := strings.TrimSuffix(key, "-"+resolve.Suffix(v.Arch))
			names["python"+minor+".exe"] = true
		}
	}
	out := []string{}
	for n := range names {
		if _, e := Resolve(s, n, platform.NativeArch()); e == nil {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}
func digest(path string) ([32]byte, error) {
	f, e := openSource(path)
	if e != nil {
		return [32]byte{}, e
	}
	defer f.Close()
	h := sha256.New()
	_, e = io.Copy(h, f)
	var sum [32]byte
	copy(sum[:], h.Sum(nil))
	return sum, e
}
func copyFile(src, dest string) error {
	in, e := openSource(src)
	if e != nil {
		return e
	}
	defer in.Close()
	out, e := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if e != nil {
		return e
	}
	_, e = io.Copy(out, in)
	if e == nil {
		e = out.Sync()
	}
	ce := out.Close()
	if e == nil {
		e = ce
	}
	return e
}

type Linker interface{ Link(string, string) error }
type NativeLinker struct{}

func (NativeLinker) Link(old, new string) error    { return os.Link(old, new) }
func Sync(root, self string, s *state.State) error { return SyncWith(root, self, s, NativeLinker{}) }
func SyncWith(root, self string, s *state.State, linker Linker) error {
	dir, e := layout.Safe(root, "shims")
	if e != nil {
		return e
	}
	if e = os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	dest, e := layout.Safe(root, "shims", "pftw.exe")
	if e != nil {
		return e
	}
	sourceHash, e := digest(self)
	if e != nil {
		return e
	}
	destHash, de := digest(dest)
	manifest := filepath.Join(dir, "owned.json")
	old := []string{}
	b, e := os.ReadFile(manifest)
	hasManifest := e == nil
	if e == nil {
		if e = json.Unmarshal(b, &old); e != nil {
			return e
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	if de != nil && !os.IsNotExist(de) {
		return de
	}
	if de == nil && !hasManifest && sourceHash != destHash {
		return fmt.Errorf("unowned shims/pftw.exe exists")
	}
	if de != nil || sourceHash != destHash {
		tmp := dest + ".new"
		if _, e = layout.Safe(root, "shims", "pftw.exe.new"); e != nil {
			return e
		}
		_ = os.Remove(tmp)
		if e = copyFile(self, tmp); e != nil {
			return e
		}
		if de == nil {
			_ = os.Remove(dest + ".old")
			if e = os.Rename(dest, dest+".old"); e != nil {
				return e
			}
		}
		if e = os.Rename(tmp, dest); e != nil {
			return e
		}
	}
	windowed, e := syncWindowed(root, self)
	if e != nil {
		return e
	}
	names := []string{}
	for _, n := range Names(s) {
		if !Windowed(n) || windowed != "" {
			names = append(names, n)
		}
	}
	owned := map[string]bool{}
	for _, n := range old {
		owned[n] = true
	}
	for _, n := range names {
		if !owned[n] {
			p, e := layout.Safe(root, "shims", n)
			if e != nil {
				return e
			}
			if _, e = os.Lstat(p); !os.IsNotExist(e) {
				return fmt.Errorf("unowned shim already exists: %s", p)
			}
		}
	}
	pending := append([]string{}, old...)
	for _, n := range names {
		if !owned[n] {
			pending = append(pending, n)
		}
	}
	if e = state.Write(dir, "owned.json", pending); e != nil {
		return e
	}
	for _, n := range old {
		if !IsName(n) {
			return fmt.Errorf("invalid owned shim name")
		}
		p, e := layout.Safe(root, "shims", n)
		if e != nil {
			return e
		}
		if e = os.Remove(p); e != nil && !os.IsNotExist(e) {
			return e
		}
	}
	for _, n := range names {
		p, e := layout.Safe(root, "shims", n)
		if e != nil {
			return e
		}
		if _, e = os.Lstat(p); !os.IsNotExist(e) {
			return fmt.Errorf("unowned shim already exists: %s", p)
		}
		source := dest
		if Windowed(n) {
			source = windowed
		}
		if e = linker.Link(source, p); e != nil {
			if e = copyFile(source, p); e != nil {
				return e
			}
		}
	}
	if e = state.Write(dir, "owned.json", names); e != nil {
		return e
	}
	_ = os.Remove(dest + ".old")
	return nil
}
func Run(self string) int {
	root := filepath.Dir(filepath.Dir(self))
	s, e := state.Load(root)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		return 1
	}
	key, e := Resolve(s, filepath.Base(self), platform.NativeArch())
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		return 1
	}
	executable := "python.exe"
	if Windowed(filepath.Base(self)) {
		executable = "pythonw.exe"
	}
	target, e := layout.Safe(root, "versions", key, executable)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		return 1
	}
	code, e := launch(target)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		return 1
	}
	return code
}

// Tail removes only argv[0]; subsequent quoting is passed to Python unchanged.
func Tail(raw string) string {
	quoted := false
	i := 0
	for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t') {
		i++
	}
	for ; i < len(raw); i++ {
		if raw[i] == '"' {
			quoted = !quoted
			continue
		}
		if !quoted && (raw[i] == ' ' || raw[i] == '\t') {
			break
		}
	}
	return raw[i:]
}

func openSource(path string) (io.ReadCloser, error) {
	if strings.HasPrefix(path, overlay.Scheme+"://") {
		r, _, e := overlay.Open(path)
		return r, e
	}
	r, _, e := overlay.Base(path)
	return r, e
}
