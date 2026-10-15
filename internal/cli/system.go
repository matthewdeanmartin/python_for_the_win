package cli

import (
	"bufio"
	"fmt"
	"github.com/matthewdeanmartin/python_for_the_win/internal/envpath"
	"github.com/matthewdeanmartin/python_for_the_win/internal/errs"
	"github.com/matthewdeanmartin/python_for_the_win/internal/install"
	system "github.com/matthewdeanmartin/python_for_the_win/internal/integration"
	"github.com/matthewdeanmartin/python_for_the_win/internal/layout"
	"github.com/matthewdeanmartin/python_for_the_win/internal/platform"
	"github.com/matthewdeanmartin/python_for_the_win/internal/preflight"
	"github.com/matthewdeanmartin/python_for_the_win/internal/resolve"
	"github.com/matthewdeanmartin/python_for_the_win/internal/shims"
	"github.com/matthewdeanmartin/python_for_the_win/internal/state"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func (o *options) rootPath() (string, error) {
	if o.root == "" {
		exe := o.self
		if exe == "" {
			exe, _ = os.Executable()
		}
		if strings.EqualFold(filepath.Base(filepath.Dir(exe)), "shims") {
			candidate := filepath.Dir(filepath.Dir(exe))
			if _, e := os.Stat(filepath.Join(candidate, "pftw.json")); e == nil {
				return layout.Root(candidate)
			}
		}
	}
	return layout.Root(o.root)
}
func (o *options) interactive() bool {
	return !o.dryRun && !o.yes && !o.json && platform.IsTerminal(os.Stdin) && platform.IsTerminal(os.Stdout)
}
func (o *options) choosePath(c *cobra.Command, s *state.State) (string, error) {
	if o.path == "none" || s.Portable() {
		return "none", nil
	}
	if strings.ContainsAny(o.root, `;"`) {
		return "", errs.New(3, "Installation root contains characters that cannot be represented safely in PATH; choose another root or --path none")
	}
	systemPath, se := o.info.SystemPath()
	user, ue := o.store.Get()
	if se != nil || ue != nil {
		return "", errs.New(4, "PATH information is unavailable; pass --path none to install without changing it")
	}
	hits := preflight.FindPythons(systemPath.Text, user.Text, preflight.Owned(o.root, s), preflight.FileLookup(o.info))
	decision := preflight.PathDecision(hits, o.path)
	if decision == "AliasBlock" {
		return "", errs.New(3, "A Python app execution alias would still take precedence. Turn off the conflicting Python app execution alias, or use --path none.")
	}
	if decision == "AddNoClash" {
		return "first", nil
	}
	if decision == "OnlyNone" {
		if o.path == "first" || !o.interactive() {
			return "", errs.New(4, "Another Python is on system PATH. Pass --path none to explicitly install without changing which Python runs.")
		}
		fmt.Fprint(c.ErrOrStderr(), "Another Python will still run. Continue installing without changing PATH? [y/N]: ")
		line, e := bufio.NewReader(c.InOrStdin()).ReadString('\n')
		if e != nil {
			return "", errs.New(4, "No decision received; nothing installed")
		}
		if strings.EqualFold(strings.TrimSpace(line), "y") || strings.EqualFold(strings.TrimSpace(line), "yes") {
			return "none", nil
		}
		return "", errs.New(4, "Installation cancelled; PATH was not changed")
	}
	if o.path == "first" {
		return "first", nil
	}
	if !o.interactive() {
		return "", errs.New(4, "Another Python is on user PATH. Choose --path first or --path none; --yes cannot choose for you.")
	}
	reader := bufio.NewReader(c.InOrStdin())
	for {
		fmt.Fprint(c.ErrOrStderr(), "Another Python is on user PATH. Choose 1: put pftw first, or 2: install without PATH changes: ")
		line, e := reader.ReadString('\n')
		if e != nil {
			return "", errs.New(4, "No decision received")
		}
		switch strings.TrimSpace(line) {
		case "1":
			return "first", nil
		case "2":
			return "none", nil
		}
	}
}
func (o *options) sync(c *cobra.Command, s *state.State, enable bool) error {
	self := o.self
	if self == "" {
		var e error
		self, e = os.Executable()
		if e != nil {
			return e
		}
	}
	if e := shims.Sync(o.root, self, s); e != nil {
		return e
	}
	return system.Apply(o.root, s, o.store, enable)
}
func (o *options) checks(c *cobra.Command, checks []preflight.Check) error {
	for _, check := range checks {
		if check.Status == "Block" || check.Status == "Warn" || o.verbose {
			fmt.Fprintf(c.ErrOrStderr(), "%s: %s\n", check.Status, check.Title)
			if o.verbose && check.Detail != "" {
				fmt.Fprintln(c.ErrOrStderr(), check.Detail)
			}
		}
	}
	if message := preflight.Blocking(checks); message != "" {
		return errs.New(3, "%s", message)
	}
	return nil
}
func installedKey(s *state.State, spec string) (string, error) {
	key, w, e := resolve.Normalize(resolve.Options{Spec: spec, Arch: platform.NativeArch()})
	if e != nil {
		return "", e
	}
	if _, ok := s.Installs[key]; ok {
		return key, nil
	}
	for k := range s.Installs {
		v, e := s.Get(k)
		if e == nil && w.Parts == 3 && strings.TrimSuffix(key, "-"+resolve.Suffix(v.Arch)) == v.Version {
			return k, nil
		}
	}
	return "", errs.New(2, "That version is not installed; run pftw list")
}
func (o *options) setDefault(c *cobra.Command, a []string) error {
	return o.mutate(c, func(s *state.State, m *install.Manager) error {
		key, e := installedKey(s, a[0])
		if e != nil {
			return e
		}
		s.Default = key
		if e = o.sync(c, s, s.Path().Mode == "first"); e != nil {
			return e
		}
		message := "Default set to " + key + ". PATH is not managed; use the full interpreter path."
		if o.pathResolvesDefault(s) {
			message = "Default set to " + key + ". Open a new terminal, then run: python --version"
		}
		return o.output(c, map[string]string{"default": key}, message)
	})
}

func (o *options) chooseReplacement(c *cobra.Command, s *state.State, removed []string) (string, error) {
	if !o.interactive() {
		return "", nil
	}
	removingDefault := false
	deleted := map[string]bool{}
	for _, k := range removed {
		deleted[k] = true
		if k == s.Default {
			removingDefault = true
		}
	}
	if !removingDefault {
		return "", nil
	}
	remaining := []string{}
	for k := range s.Installs {
		if !deleted[k] {
			remaining = append(remaining, k)
		}
	}
	sort.Strings(remaining)
	if len(remaining) == 0 {
		return "", nil
	}
	fmt.Fprintln(c.ErrOrStderr(), "Choose which installed Python becomes the default:")
	for i, k := range remaining {
		fmt.Fprintf(c.ErrOrStderr(), "  %d: %s\n", i+1, k)
	}
	reader := bufio.NewReader(c.InOrStdin())
	for {
		fmt.Fprint(c.ErrOrStderr(), "Selection: ")
		line, e := reader.ReadString('\n')
		if e != nil {
			return "", errs.New(4, "No replacement default selected; nothing removed")
		}
		n, e := strconv.Atoi(strings.TrimSpace(line))
		if e == nil && n >= 1 && n <= len(remaining) {
			return remaining[n-1], nil
		}
	}
}

func (o *options) pathResolvesDefault(s *state.State) bool {
	if s.Default == "" || s.Path().Mode != "first" {
		return false
	}
	system, e := o.info.SystemPath()
	if e != nil {
		return false
	}
	user, e := o.store.Get()
	if e != nil {
		return false
	}
	actual := lookupCommand(system.Text+";"+user.Text, "python")
	return envpath.Equal(actual, filepath.Join(o.root, "versions", s.Default, "python.exe"))
}
