package preflight

import (
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/envpath"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"os"
	"path/filepath"
	"strings"
)

type Check struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
	Fix    string `json:"fix,omitempty"`
}
type Hit struct {
	Scope, Dir, Name string
	Alias            bool
	Unknown          bool
}
type Lookup func(string) (exists, alias, unknown bool)

func FindPythons(system, user string, owned []string, lookup Lookup) []Hit {
	out := []Hit{}
	for _, part := range []struct{ scope, value string }{{"system", system}, {"user", user}} {
		seen := []string{}
		for _, dir := range envpath.Split(part.value) {
			dir = envpath.Expand(strings.Trim(dir, `"`))
			if dir == "" || envpath.Contains(seen, dir) || envpath.Contains(owned, dir) {
				continue
			}
			seen = append(seen, dir)
			for _, name := range []string{"python.exe", "python3.exe", "py.exe"} {
				exists, alias, unknown := lookup(filepath.Join(dir, name))
				if exists {
					out = append(out, Hit{part.scope, dir, name, alias, unknown})
				}
			}
		}
	}
	return out
}
func FileLookup(info platform.Info) Lookup {
	return func(p string) (bool, bool, bool) {
		if _, e := os.Lstat(p); e != nil {
			return false, false, !os.IsNotExist(e)
		}
		alias, e := info.IsAppExecLink(p)
		return true, alias, e != nil
	}
}
func PathDecision(hits []Hit, mode string) string {
	if mode == "none" {
		return "AddNoClash"
	}
	user := false
	for _, h := range hits {
		if h.Name == "py.exe" {
			continue
		}
		if h.Scope == "system" {
			if h.Alias {
				return "AliasBlock"
			}
			return "OnlyNone"
		}
		if !h.Alias {
			user = true
		}
	}
	if user {
		return "MustChoose"
	}
	return "AddNoClash"
}
func Owned(root string, s *state.State) []string {
	out := []string{filepath.Join(root, "shims")}
	for key := range s.Installs {
		out = append(out, filepath.Join(root, "versions", key), filepath.Join(root, "versions", key, "Scripts"))
	}
	return out
}
func Run(info platform.Info, root, key string, s *state.State, allowAdmin bool) []Check {
	out := []Check{}
	add := func(id, status, title, detail, fix string) { out = append(out, Check{id, status, title, detail, fix}) }
	elevated, e := info.IsElevated()
	if e != nil {
		add("elevation", "Info", "Elevation status unknown", e.Error(), "")
	} else if elevated && !allowAdmin {
		add("elevation", "Block", "Run pftw as yourself, not as administrator.", "", "Use a normal terminal, or explicitly pass --allow-admin.")
	}
	free, e := info.FreeBytes(root)
	if e != nil {
		add("disk", "Info", "Available disk space unknown", e.Error(), "")
	} else if free < 300*1024*1024 {
		add("disk", "Block", "Not enough free disk space", fmt.Sprintf("%d MB available", free/1024/1024), "Free at least 300 MB and try again.")
	}
	network, e := info.IsNetworkPath(root)
	if e != nil {
		add("network-root", "Info", "Root type unknown", e.Error(), "")
	} else if network {
		add("network-root", "Warn", "Installation root is on a network share", "File locking and executable access may differ.", "")
	}
	if key != "" {
		if _, owned := s.Installs[key]; !owned {
			if _, e := os.Lstat(filepath.Join(root, "versions", key)); !os.IsNotExist(e) {
				add("target", "Block", "Installation target is not owned by pftw", key, "Choose another root or move the conflicting directory yourself.")
			}
		}
	}
	return out
}
func Blocking(checks []Check) string {
	messages := []string{}
	for _, c := range checks {
		if c.Status == "Block" {
			messages = append(messages, c.Title+" "+c.Fix)
		}
	}
	return strings.Join(messages, "\n")
}
