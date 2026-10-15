package resolve

import (
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/feed"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"github.com/matthewdeanmartin/python_for_the_win/internal/pyver"
	"slices"
	"sort"
	"strings"
)

type Options struct {
	Spec, Arch           string
	Threaded, Prerelease bool
}

func Suffix(arch string) string {
	return map[string]string{"amd64": "64", "arm64": "arm64", "x86": "32"}[arch]
}
func Normalize(o Options) (string, pyver.Version, error) {
	s := o.Spec
	if s == "" || s == "latest" {
		s = "3"
	}
	arch := Suffix(o.Arch)
	if arch == "" {
		return "", pyver.Version{}, fmt.Errorf("unsupported architecture %q", o.Arch)
	}
	for _, a := range []string{"-arm64", "-64", "-32"} {
		if strings.HasSuffix(s, a) {
			arch = strings.TrimPrefix(a, "-")
			s = strings.TrimSuffix(s, a)
			break
		}
	}
	if o.Threaded && !strings.HasSuffix(s, "t") {
		s += "t"
	}
	v, e := pyver.Parse(s)
	return s + "-" + arch, v, e
}
func Pick(entries []feed.Entry, o Options) (feed.Entry, error) {
	spec, w, e := Normalize(o)
	if e != nil {
		return feed.Entry{}, e
	}
	var candidates []feed.Entry
	finalExists := false
	minors := map[string]bool{}
	for _, x := range entries {
		v, e := pyver.Parse(x.Version)
		if e != nil {
			continue
		}
		minors[fmt.Sprintf("3.%d", v.Minor)] = true
		if v.Minor == w.Minor && !v.Pre() {
			finalExists = true
		}
		if x.Installable() && layout.ValidKey(x.Key()) && slices.Contains(x.InstallFor, spec) {
			candidates = append(candidates, x)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		a, _ := pyver.Parse(candidates[i].Version)
		b, _ := pyver.Parse(candidates[j].Version)
		return a.Compare(b) > 0
	})
	for _, x := range candidates {
		v, _ := pyver.Parse(x.Version)
		if !v.Pre() || o.Prerelease || w.Pre() || (w.Parts == 2 && !finalExists) {
			return x, nil
		}
	}
	available := []string{}
	for m := range minors {
		available = append(available, m)
	}
	sort.Strings(available)
	return feed.Entry{}, fmt.Errorf("no eligible precompiled release for %s; available minors: %s", spec, strings.Join(available, ", "))
}
func Available(entries []feed.Entry, arch string, allArch, pre bool, showUnavailable ...bool) []feed.Entry {
	entries = slices.Clone(entries)
	show := len(showUnavailable) > 0 && showUnavailable[0]
	final := map[int]bool{}
	for _, x := range entries {
		v, e := pyver.Parse(x.Version)
		if e == nil && !v.Pre() {
			final[v.Minor] = true
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		a, _ := pyver.Parse(entries[i].Version)
		b, _ := pyver.Parse(entries[j].Version)
		return a.Compare(b) > 0
	})
	usable := map[string]bool{}
	for _, x := range entries {
		v, e := pyver.Parse(x.Version)
		if e == nil && x.Installable() && (!v.Pre() || !final[v.Minor]) {
			usable[x.Key()] = true
		}
	}
	out := []feed.Entry{}
	seen := map[string]bool{}
	for _, x := range entries {
		if !x.Installable() && (!show || (!pre && usable[x.Key()])) {
			continue
		}
		v, e := pyver.Parse(x.Version)
		if e != nil {
			continue
		}
		if !allArch && !strings.HasSuffix(x.Key(), "-"+Suffix(arch)) {
			continue
		}
		if !pre {
			if v.Pre() && final[v.Minor] {
				continue
			}
			if seen[x.Key()] {
				continue
			}
			seen[x.Key()] = true
		}
		out = append(out, x)
	}
	return out
}
