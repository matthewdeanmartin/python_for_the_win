package cli

import (
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/install"
	"github.com/matthewdeanmartin/python_for_the_win/internal/pyver"
	"github.com/matthewdeanmartin/python_for_the_win/internal/resolve"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"github.com/spf13/cobra"
	"sort"
	"strings"
)

func (o *options) upgrade(c *cobra.Command, args []string) error {
	if (len(args) == 0) == !o.all {
		return errs.New(2, "choose SPEC or --all")
	}
	return o.mutate(c, func(s *state.State, m *install.Manager) error {
		keys := []string{}
		if o.all {
			for k := range s.Installs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
		} else {
			k, e := installedKey(s, args[0])
			if e != nil {
				return e
			}
			keys = append(keys, k)
		}
		entries, d, source, e := o.entries(c, s)
		if e != nil {
			return e
		}
		m.Download = d
		m.Fresh = o.fresh
		messages := []string{}
		for _, key := range keys {
			old, e := s.Get(key)
			if e != nil {
				return e
			}
			pv, e := pyver.Parse(old.Version)
			if e != nil {
				return e
			}
			v, e := resolve.Pick(entries, resolve.Options{Spec: key, Arch: old.Arch, Threaded: old.Threaded, Prerelease: pv.Pre()})
			if e != nil {
				return e
			}
			next, e := pyver.Parse(v.Version)
			if e != nil {
				return e
			}
			// A final install must never fall back to a prerelease, even with a partial source.
			if !pv.Pre() && next.Pre() {
				return errs.New(3, "No final upgrade is available for %s", key)
			}
			message, e := m.Install(c.Context(), v, source, !old.Pip)
			if e != nil {
				return e
			}
			messages = append(messages, message)
		}
		s, e = state.Load(o.root)
		if e != nil {
			return e
		}
		if e = o.sync(c, s, s.Path().Mode == "first"); e != nil {
			return e
		}
		return o.output(c, map[string]any{"results": messages}, strings.Join(messages, string(rune(10))))
	})
}
